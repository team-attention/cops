import type {
  AssistantMessageContent,
  Record,
  UserRecordData,
} from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type {
  ContentBlock,
  LinkedToolCall,
  ParsedMessage,
  ToolResultContentBlock,
  ToolUseContentBlock,
} from '../type/content-block'
import { RecordType } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'

// Helper to extract user message text content from UserRecordData
const extractUserMessageText = (userData: UserRecordData): string => {
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

// Helper to convert AssistantMessageContent[] to ContentBlock[]
const convertAssistantContent = (
  content: Array<AssistantMessageContent>,
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
        input = JSON.parse(item.toolUseInputJson)
      } catch {
        // Keep empty object if parsing fails
      }

      blocks.push({
        type: 'tool_use',
        id: item.toolUseId,
        name: item.toolUseName,
        input,
      })
    } else if (item.type === 'tool_result') {
      blocks.push({
        type: 'tool_result',
        tool_use_id: item.toolUseId,
        content: item.text,
      })
    }
  }

  return blocks
}

// Parse a single Record into a renderable ParsedMessage
export const parseMessageContent = (record: Record): ParsedMessage => {
  if (record.type === RecordType.USER && record.data.case === 'userData') {
    const userData = record.data.value
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
    record.type === RecordType.ASSISTANT &&
    record.data.case === 'assistantData'
  ) {
    const assistantData = record.data.value
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

  if (record.type === RecordType.FILE_HISTORY_SNAPSHOT) {
    return {
      uuid: '',
      type: 'system',
      isMeta: false,
      isSidechain: false,
      content: [],
    }
  }

  // Fallback for UNSPECIFIED or unknown types
  return {
    uuid: '',
    type: 'system',
    isMeta: false,
    isSidechain: false,
    content: [],
  }
}

// Extract and link tool calls from records
export const extractToolCalls = (
  records: Array<Record>,
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

  // First pass - collect tool_use blocks from assistant records
  for (const record of records) {
    if (
      record.type === RecordType.ASSISTANT &&
      record.data.case === 'assistantData'
    ) {
      const assistantData = record.data.value
      const metadata = assistantData.metadata
      const content = assistantData.message?.content || []

      for (const item of content) {
        if (item.type === 'tool_use') {
          let input: globalThis.Record<string, unknown> = {}
          try {
            input = JSON.parse(item.toolUseInputJson)
          } catch {
            // Keep empty object if parsing fails
          }

          toolUseMap.set(item.toolUseId, {
            block: {
              type: 'tool_use',
              id: item.toolUseId,
              name: item.toolUseName,
              input,
            },
            sourceUuid: metadata?.uuid || '',
            timestamp: metadata?.timestamp,
          })
        }
      }
    }
  }

  // Second pass - collect tool_result blocks
  for (const record of records) {
    if (
      record.type === RecordType.ASSISTANT &&
      record.data.case === 'assistantData'
    ) {
      const assistantData = record.data.value
      const content = assistantData.message?.content || []

      for (const item of content) {
        if (item.type === 'tool_result') {
          toolResults.set(item.toolUseId, {
            type: 'tool_result',
            tool_use_id: item.toolUseId,
            content: item.text,
          })
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

// Filter records for chat view display
export const filterRecordsForChat = (records: Array<Record>): Array<Record> => {
  return records.filter(
    (record) => record.type !== RecordType.FILE_HISTORY_SNAPSHOT,
  )
}
