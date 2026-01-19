import type {
  AssistantContentBlock,
  Transcript,
  UserTranscriptData,
} from '@/gen/grpcstub/transcript/v1/transcript_pb'
import {
  TranscriptType,
  ProgressDataType,
} from '@/gen/grpcstub/transcript/v1/transcript_pb'
import type { Value } from '@bufbuild/protobuf/wkt'
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

// Helper to format toolUseResult (Value type) to displayable string
const formatToolUseResult = (value: Value | undefined): string => {
  if (!value) return ''

  switch (value.kind.case) {
    case 'stringValue':
      return value.kind.value
    case 'structValue':
      return JSON.stringify(value.kind.value?.fields, null, 2)
    case 'listValue':
      return JSON.stringify(value.kind.value?.values, null, 2)
    case 'numberValue':
      return String(value.kind.value)
    case 'boolValue':
      return String(value.kind.value)
    case 'nullValue':
      return 'null'
    default:
      return ''
  }
}

// Parse a single Transcript into a renderable ParsedMessage
export const parseMessageContent = (transcript: Transcript): ParsedMessage => {
  if (
    transcript.type === TranscriptType.USER &&
    transcript.data.case === 'userData'
  ) {
    const userData = transcript.data.value
    const metadata = userData.metadata
    const message = userData.message

    // Check for tool_result-only user messages
    if (message?.content.case === 'blocks') {
      const blocks = message.content.value.blocks
      const hasText = blocks.some(
        (block) => block.type === 'text' && block.text.trim(),
      )
      const toolResultBlocks = blocks.filter(
        (block) => block.type === 'tool_result' && block.toolResult,
      )

      // If no text and has tool_result blocks, return as tool_result type
      if (!hasText && toolResultBlocks.length > 0) {
        const firstToolResult = toolResultBlocks[0].toolResult!
        return {
          uuid: metadata?.uuid || '',
          type: 'tool_result',
          timestamp: metadata?.timestamp,
          isMeta: userData.isMeta,
          isSidechain: metadata?.isSidechain || false,
          parentToolUseId: firstToolResult.toolUseId,
          content: [
            {
              type: 'tool_result',
              tool_use_id: firstToolResult.toolUseId,
              content: firstToolResult.content,
            },
          ],
        }
      }
    }

    // Extract text content first
    const textContent = extractUserMessageText(userData)

    // If no text content but has toolUseResult, display that instead
    if (!textContent && userData.toolUseResult) {
      const resultText = formatToolUseResult(userData.toolUseResult)
      return {
        uuid: metadata?.uuid || '',
        type: 'user',
        timestamp: metadata?.timestamp,
        isMeta: userData.isMeta,
        isSidechain: metadata?.isSidechain || false,
        isToolUseResult: true,
        content: [
          {
            type: 'text',
            text: resultText,
          },
        ],
      }
    }

    return {
      uuid: metadata?.uuid || '',
      type: 'user',
      timestamp: metadata?.timestamp,
      isMeta: userData.isMeta,
      isSidechain: metadata?.isSidechain || false,
      content: [
        {
          type: 'text',
          text: textContent,
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

  if (
    transcript.type === TranscriptType.PROGRESS &&
    transcript.data.case === 'progressData'
  ) {
    const progressData = transcript.data.value
    const metadata = progressData.metadata
    const data = progressData.data

    return {
      uuid: metadata?.uuid || '',
      type: 'progress',
      timestamp: metadata?.timestamp,
      isMeta: false,
      isSidechain: metadata?.isSidechain || false,
      content: [
        {
          type: 'text',
          text: data?.prompt || '',
        },
      ],
      progressType:
        data?.type === ProgressDataType.AGENT ? 'agent' : 'skill',
      prompt: data?.prompt,
      agentId: data?.agentId,
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

// Enrich tool_result messages with tool names from linked tool calls
export const enrichToolResultMessages = (
  parsedMessages: Array<ParsedMessage>,
  toolCalls: Array<LinkedToolCall>,
): Array<ParsedMessage> => {
  const toolUseIdToName = new Map(
    toolCalls.map((tc) => [tc.toolUse.id, tc.toolUse.name]),
  )

  return parsedMessages.map((message) => {
    if (message.type === 'tool_result' && message.parentToolUseId) {
      return {
        ...message,
        toolName: toolUseIdToName.get(message.parentToolUseId),
      }
    }
    return message
  })
}
