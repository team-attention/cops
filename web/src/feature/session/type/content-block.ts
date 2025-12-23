import type { Timestamp } from '@bufbuild/protobuf/wkt'
import type { UsageMetadata } from '@/gen/grpcstub/collector/v1/collector_pb'

// Content block types for assistant messages
export interface TextContentBlock {
  type: 'text'
  text: string
}

export interface ToolUseContentBlock {
  type: 'tool_use'
  id: string
  name: string
  input: Record<string, unknown>
}

export interface ToolResultContentBlock {
  type: 'tool_result'
  tool_use_id: string
  content: string
  is_error?: boolean
}

export type AssistantContentBlock = TextContentBlock | ToolUseContentBlock
export type ContentBlock = TextContentBlock | ToolUseContentBlock | ToolResultContentBlock

// Parsed message structure for rendering
export interface ParsedMessage {
  uuid: string
  type: 'user' | 'assistant' | 'system' | 'tool_result'
  timestamp?: Timestamp
  isMeta: boolean
  isSidechain: boolean
  usage?: UsageMetadata
  content: ContentBlock[]
  // For tool_result messages
  toolName?: string
  parentToolUseId?: string
}

// Tool call with linked result for panel display
export interface LinkedToolCall {
  toolUse: ToolUseContentBlock
  toolResult?: ToolResultContentBlock
  sourceMessageUuid: string
  timestamp?: Timestamp
}
