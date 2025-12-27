import { SessionType } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type { SessionRecord } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type {
  ParsedMessage,
  ContentBlock,
  LinkedToolCall,
  ToolUseContentBlock,
  ToolResultContentBlock,
} from '../type/content-block'

// Safely parse JSON with fallback
const tryParseJSON = (str: string): unknown => {
  try {
    return JSON.parse(str)
  } catch {
    return str
  }
}

// Type guard for content blocks
const isValidContentBlock = (block: unknown): block is ContentBlock => {
  if (typeof block !== 'object' || block === null) return false
  const b = block as Record<string, unknown>
  return b.type === 'text' || b.type === 'tool_use' || b.type === 'tool_result'
}

// Parse a single SessionRecord into a renderable ParsedMessage
export const parseMessageContent = (record: SessionRecord): ParsedMessage => {
  const contentStr = record.message?.content || ''
  const content = tryParseJSON(contentStr)

  if (record.type === SessionType.USER) {
    const text = typeof content === 'string' ? content : contentStr
    return {
      uuid: record.uuid,
      type: 'user',
      timestamp: record.timestamp,
      isMeta: record.isMeta,
      isSidechain: record.isSidechain,
      content: [{ type: 'text', text }],
    }
  }

  if (record.type === SessionType.ASSISTANT) {
    const blocks = Array.isArray(content)
      ? content.filter(isValidContentBlock)
      : [{ type: 'text' as const, text: contentStr }]
    return {
      uuid: record.uuid,
      type: 'assistant',
      timestamp: record.timestamp,
      isMeta: record.isMeta,
      isSidechain: record.isSidechain,
      usage: record.message?.usage,
      content: blocks,
    }
  }

  // Note: tool_result is not a separate SessionType in aggregation schema
  // Tool results are embedded in assistant message content blocks
  // This code path may not be reached with aggregation.v1.SessionRecord

  // Fallback for system/summary/other types
  return {
    uuid: record.uuid,
    type: 'system',
    timestamp: record.timestamp,
    isMeta: record.isMeta,
    isSidechain: record.isSidechain,
    content: [{ type: 'text', text: contentStr }],
  }
}

// Extract and link tool calls from session records
export const extractToolCalls = (records: SessionRecord[]): LinkedToolCall[] => {
  const toolUseMap = new Map<string, {
    block: ToolUseContentBlock
    sourceUuid: string
    timestamp?: SessionRecord['timestamp']
  }>()
  const toolResults = new Map<string, ToolResultContentBlock>()

  // First pass: collect all tool_use blocks from assistant messages
  for (const record of records) {
    if (record.type === SessionType.ASSISTANT) {
      const contentStr = record.message?.content || ''
      const content = tryParseJSON(contentStr)
      if (Array.isArray(content)) {
        for (const block of content) {
          if (isValidContentBlock(block) && block.type === 'tool_use') {
            toolUseMap.set(block.id, {
              block: block as ToolUseContentBlock,
              sourceUuid: record.uuid,
              timestamp: record.timestamp,
            })
          }
        }
      }
    }
  }

  // Second pass: collect tool_result blocks from assistant messages
  // In aggregation schema, tool_result is embedded in content, not a separate record type
  for (const record of records) {
    if (record.type === SessionType.ASSISTANT) {
      const contentStr = record.message?.content || ''
      const content = tryParseJSON(contentStr)
      if (Array.isArray(content)) {
        for (const block of content) {
          if (isValidContentBlock(block) && block.type === 'tool_result') {
            const resultBlock = block as ToolResultContentBlock
            toolResults.set(resultBlock.tool_use_id, resultBlock)
          }
        }
      }
    }
  }

  // Link them together and return as array
  return Array.from(toolUseMap.entries()).map(([id, { block, sourceUuid, timestamp }]) => ({
    toolUse: block,
    toolResult: toolResults.get(id),
    sourceMessageUuid: sourceUuid,
    timestamp,
  }))
}

// Filter records for chat view display
export const filterRecordsForChat = (records: SessionRecord[]): SessionRecord[] => {
  return records.filter((record) => {
    // Exclude summary and file-history-snapshot types
    if (record.type === SessionType.SUMMARY || record.type === SessionType.FILE_HISTORY_SNAPSHOT) {
      return false
    }
    // Exclude queue-operation types
    if (record.type === SessionType.QUEUE_OPERATION) {
      return false
    }
    return true
  })
}
