import type { SessionRecord } from '@/gen/grpcstub/collector/v1/collector_pb'
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
  const content = tryParseJSON(record.content)

  if (record.type === 'user') {
    const text = typeof content === 'string' ? content : record.content
    return {
      uuid: record.uuid,
      type: 'user',
      timestamp: record.timestamp,
      isMeta: record.isMeta,
      isSidechain: record.isSidechain,
      content: [{ type: 'text', text }],
    }
  }

  if (record.type === 'assistant') {
    const blocks = Array.isArray(content)
      ? content.filter(isValidContentBlock)
      : [{ type: 'text' as const, text: record.content }]
    return {
      uuid: record.uuid,
      type: 'assistant',
      timestamp: record.timestamp,
      isMeta: record.isMeta,
      isSidechain: record.isSidechain,
      usage: record.usage,
      content: blocks,
    }
  }

  if (record.type === 'tool_result') {
    const resultContent = typeof content === 'string' ? content : record.content
    return {
      uuid: record.uuid,
      type: 'tool_result',
      timestamp: record.timestamp,
      isMeta: record.isMeta,
      isSidechain: record.isSidechain,
      toolName: record.slug,
      parentToolUseId: record.parentUuid,
      content: [{
        type: 'tool_result',
        tool_use_id: record.parentUuid,
        content: resultContent,
      }],
    }
  }

  // Fallback for system/summary/other types
  return {
    uuid: record.uuid,
    type: 'system',
    timestamp: record.timestamp,
    isMeta: record.isMeta,
    isSidechain: record.isSidechain,
    content: [{ type: 'text', text: record.content }],
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
    if (record.type === 'assistant') {
      const content = tryParseJSON(record.content)
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

  // Second pass: collect tool_result records
  for (const record of records) {
    if (record.type === 'tool_result') {
      const content = tryParseJSON(record.content)
      toolResults.set(record.parentUuid, {
        type: 'tool_result',
        tool_use_id: record.parentUuid,
        content: typeof content === 'string' ? content : record.content,
      })
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
    if (record.type === 'summary' || record.type === 'file-history-snapshot') {
      return false
    }
    // Exclude queue-operation types
    if (record.type === 'queue-operation') {
      return false
    }
    return true
  })
}
