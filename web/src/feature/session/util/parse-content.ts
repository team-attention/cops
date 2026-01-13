import type {
  AssistantContentBlock,
  Transcript,
  UserTranscriptData,
} from '@/gen/grpcstub/transcript/v1/transcript_pb'
import { TranscriptType } from '@/gen/grpcstub/transcript/v1/transcript_pb'
import type {
  ContentBlock,
  LinkedToolCall,
  ParsedMessage,
  ToolResultContentBlock,
  ToolUseContentBlock,
} from '../type/content-block'

// Helper to extract user message text content from UserTranscriptData
const extractUserMessageText = (userData: UserTranscriptData): string => {
  if (!userData.message) {
    return ''
  }

  const { content } = userData.message

  if (content.case === 'text') {
    return content.value
  }

  if (content.case === 'blocks') {
    return content.value.blocks
      .filter((block) => block.type === 'text')
      .map((block) => block.text)
      .join('')
  }

  return ''
}

// Helper to convert AssistantContentBlock[] to ContentBlock[]
const convertAssistantContent = (
  content: Array<AssistantContentBlock>,
): Array<ContentBlock> => {
  const blocks: Array<ContentBlock> = []

  for (const item of content) {
    if (item.type === 'text') {
      blocks.push({
        type: 'text',
        text: item.text,
      })
    } else if (item.type === 'thinking') {
      blocks.push({
        type: 'text',
        text: item.thinking,
      })
    } else if (item.type === 'tool_use') {
      let input: globalThis.Record<string, unknown> = {}
      try {
        input = JSON.parse(item.inputJson)
      } catch {
        // Keep empty object if parsing fails
      }

      blocks.push({
        type: 'tool_use',
        id: item.id,
        name: item.name,
        input,
      })
    }
  }

  return blocks
}

// Parse a single Transcript into a renderable ParsedMessage
export const parseMessageContent = (transcript: Transcript): ParsedMessage => {
  if (
    transcript.type === TranscriptType.USER &&
    transcript.data.case === 'userData'
  ) {
    const userData = transcript.data.value
    const metadata = userData.metadata

    return {
      uuid: metadata?.uuid || '',
      type: 'user',
      timestamp: metadata?.timestamp,
      isMeta: userData.isMeta,
      isSidechain: metadata?.isSidechain || false,
      content: [
        {
          type: 'text',
          text: extractUserMessageText(userData),
        },
      ],
    }
  }

  if (
    transcript.type === TranscriptType.ASSISTANT &&
    transcript.data.case === 'assistantData'
  ) {
    const assistantData = transcript.data.value
    const metadata = assistantData.metadata

    return {
      uuid: metadata?.uuid || '',
      type: 'assistant',
      timestamp: metadata?.timestamp,
      isMeta: false,
      isSidechain: metadata?.isSidechain || false,
      usage: assistantData.message?.usage,
      content: assistantData.message?.content
        ? convertAssistantContent(assistantData.message.content)
        : [],
    }
  }

  if (
    transcript.type === TranscriptType.SYSTEM &&
    transcript.data.case === 'systemData'
  ) {
    const systemData = transcript.data.value
    const metadata = systemData.metadata

    return {
      uuid: metadata?.uuid || '',
      type: 'system',
      timestamp: metadata?.timestamp,
      isMeta: systemData.isMeta,
      isSidechain: metadata?.isSidechain || false,
      content: [],
    }
  }

  if (transcript.type === TranscriptType.FILE_HISTORY_SNAPSHOT) {
    return {
      uuid: '',
      type: 'system',
      isMeta: false,
      isSidechain: false,
      content: [],
    }
  }

  // Fallback for UNSPECIFIED, SUMMARY, or unknown types
  return {
    uuid: '',
    type: 'system',
    isMeta: false,
    isSidechain: false,
    content: [],
  }
}

// Extract and link tool calls from transcripts
export const extractToolCalls = (
  transcripts: Array<Transcript>,
): Array<LinkedToolCall> => {
  const toolUseMap = new Map<
    string,
    {
      block: ToolUseContentBlock
      sourceUuid: string
      timestamp?: ParsedMessage['timestamp']
    }
  >()
  const toolResults = new Map<string, ToolResultContentBlock>()

  // First pass - collect tool_use blocks from assistant transcripts
  for (const transcript of transcripts) {
    if (
      transcript.type === TranscriptType.ASSISTANT &&
      transcript.data.case === 'assistantData'
    ) {
      const assistantData = transcript.data.value
      const metadata = assistantData.metadata
      const content = assistantData.message?.content || []

      for (const item of content) {
        if (item.type === 'tool_use') {
          let input: globalThis.Record<string, unknown> = {}
          try {
            input = JSON.parse(item.inputJson)
          } catch {
            // Keep empty object if parsing fails
          }

          toolUseMap.set(item.id, {
            block: {
              type: 'tool_use',
              id: item.id,
              name: item.name,
              input,
            },
            sourceUuid: metadata?.uuid || '',
            timestamp: metadata?.timestamp,
          })
        }
      }
    }
  }

  // Second pass - collect tool_result blocks from user transcripts
  for (const transcript of transcripts) {
    if (
      transcript.type === TranscriptType.USER &&
      transcript.data.case === 'userData'
    ) {
      const userData = transcript.data.value
      const message = userData.message
      if (message?.content.case === 'blocks') {
        for (const block of message.content.value.blocks) {
          if (block.type === 'tool_result' && block.toolResult) {
            toolResults.set(block.toolResult.toolUseId, {
              type: 'tool_result',
              tool_use_id: block.toolResult.toolUseId,
              content: block.toolResult.content,
            })
          }
        }
      }
    }
  }

  // Link tool uses with results
  return Array.from(toolUseMap.entries()).map(
    ([id, { block, sourceUuid, timestamp }]) => ({
      toolUse: block,
      toolResult: toolResults.get(id),
      sourceMessageUuid: sourceUuid,
      timestamp,
    }),
  )
}

// Filter transcripts for chat view display
export const filterTranscriptsForChat = (
  transcripts: Array<Transcript>,
): Array<Transcript> => {
  return transcripts.filter(
    (transcript) => transcript.type !== TranscriptType.FILE_HISTORY_SNAPSHOT,
  )
}
