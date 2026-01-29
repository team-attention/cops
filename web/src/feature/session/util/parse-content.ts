import type {
  Session,
  HumanMessage,
  AgentMessage,
  ToolExecution,
  HumanContentBlock as ProtoHumanContentBlock,
  AgentContentBlock as ProtoAgentContentBlock,
} from '@/gen/grpcstub/session/v1/session_pb'
import {
  SessionType,
  ProgressType,
  AgentContentBlockType,
  HumanContentBlockType,
  SystemMessageSubtype,
} from '@/gen/grpcstub/session/v1/session_pb'
import type {
  ContentBlock,
  LinkedToolCall,
  ParsedMessage,
  ToolResultContentBlock,
  ToolUseContentBlock,
} from '../type/content-block'

// Helper to convert HumanContentBlock[] to ContentBlock[]
const convertHumanContent = (
  content: Array<ProtoHumanContentBlock>,
): Array<ContentBlock> => {
  const blocks: Array<ContentBlock> = []

  for (const item of content) {
    if (item.type === HumanContentBlockType.TEXT) {
      blocks.push({
        type: 'text',
        text: item.text,
      })
    }
    // IMAGE type is not rendered as text content
  }

  return blocks
}

// Helper to convert AgentContentBlock[] to ContentBlock[]
const convertAgentContent = (
  content: Array<ProtoAgentContentBlock>,
): Array<ContentBlock> => {
  const blocks: Array<ContentBlock> = []

  for (const item of content) {
    if (item.type === AgentContentBlockType.TEXT) {
      blocks.push({
        type: 'text',
        text: item.text,
      })
    } else if (item.type === AgentContentBlockType.THINKING && item.thinking) {
      blocks.push({
        type: 'text',
        text: item.thinking.content,
      })
    } else if (
      item.type === AgentContentBlockType.TOOL_CALL_REF &&
      item.toolCallRef
    ) {
      // Tool call references are handled separately via extractToolCalls
      // We add a placeholder that will be rendered as a tool badge
      blocks.push({
        type: 'tool_use',
        id: item.toolCallRef.toolExecutionId,
        name: item.toolCallRef.toolName,
        input: {},
      })
    }
  }

  return blocks
}

// Parse a single Session entry into a renderable ParsedMessage
export const parseMessageContent = (session: Session): ParsedMessage => {
  // Handle HUMAN type
  if (
    session.type === SessionType.HUMAN &&
    session.data.case === 'humanData'
  ) {
    const human = session.data.value as HumanMessage
    const metadata = human.metadata

    return {
      uuid: metadata?.uuid || '',
      type: 'user',
      timestamp: metadata?.timestamp,
      isMeta: human.isMeta,
      isSidechain: metadata?.isSidechain || false,
      content: convertHumanContent(human.content),
    }
  }

  // Handle AGENT type
  if (
    session.type === SessionType.AGENT &&
    session.data.case === 'agentData'
  ) {
    const agent = session.data.value as AgentMessage
    const metadata = agent.metadata

    return {
      uuid: metadata?.uuid || '',
      type: 'assistant',
      timestamp: metadata?.timestamp,
      isMeta: false,
      isSidechain: metadata?.isSidechain || false,
      usage: agent.usage,
      content: convertAgentContent(agent.content),
    }
  }

  // Handle TOOL_EXECUTION type
  if (
    session.type === SessionType.TOOL_EXECUTION &&
    session.data.case === 'toolExecutionData'
  ) {
    const toolExec = session.data.value as ToolExecution
    const metadata = toolExec.metadata
    const result = toolExec.result

    // Render as tool_result if there's a result
    if (result) {
      return {
        uuid: metadata?.uuid || '',
        type: 'tool_result',
        timestamp: metadata?.timestamp,
        isMeta: false,
        isSidechain: metadata?.isSidechain || false,
        parentToolUseId: toolExec.id,
        toolName: toolExec.toolName,
        content: [
          {
            type: 'tool_result',
            tool_use_id: toolExec.id,
            content: result.content,
            is_error: result.error !== '',
          },
        ],
      }
    }

    // Pending tool execution - render as system message
    return {
      uuid: metadata?.uuid || '',
      type: 'system',
      timestamp: metadata?.timestamp,
      isMeta: false,
      isSidechain: metadata?.isSidechain || false,
      content: [],
    }
  }

  // Handle SYSTEM type
  if (
    session.type === SessionType.SYSTEM &&
    session.data.case === 'systemData'
  ) {
    const systemData = session.data.value
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

  // Handle PROGRESS type
  if (
    session.type === SessionType.PROGRESS &&
    session.data.case === 'progressData'
  ) {
    const progress = session.data.value
    const metadata = progress.metadata
    const data = progress.data

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
      progressType: data?.type === ProgressType.AGENT ? 'agent' : 'skill',
      prompt: data?.prompt,
      agentId: data?.agentId,
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

// Extract and link tool calls from sessions
export const extractToolCalls = (
  sessions: Array<Session>,
): Array<LinkedToolCall> => {
  const toolExecutionMap = new Map<
    string,
    {
      block: ToolUseContentBlock
      sourceUuid: string
      timestamp?: ParsedMessage['timestamp']
      result?: ToolResultContentBlock
    }
  >()

  // Collect ToolExecution entries
  for (const session of sessions) {
    if (
      session.type === SessionType.TOOL_EXECUTION &&
      session.data.case === 'toolExecutionData'
    ) {
      const toolExec = session.data.value as ToolExecution
      const metadata = toolExec.metadata

      let input: globalThis.Record<string, unknown> = {}
      try {
        input = JSON.parse(toolExec.inputJson)
      } catch {
        // Keep empty object if parsing fails
      }

      const result = toolExec.result
      let toolResult: ToolResultContentBlock | undefined
      if (result) {
        toolResult = {
          type: 'tool_result',
          tool_use_id: toolExec.id,
          content: result.content,
          is_error: result.error !== '',
        }
      }

      toolExecutionMap.set(toolExec.id, {
        block: {
          type: 'tool_use',
          id: toolExec.id,
          name: toolExec.toolName,
          input,
        },
        sourceUuid: toolExec.sourceAgentUuid || metadata?.uuid || '',
        timestamp: metadata?.timestamp,
        result: toolResult,
      })
    }
  }

  // Build LinkedToolCall array
  return Array.from(toolExecutionMap.entries()).map(
    ([, { block, sourceUuid, timestamp, result }]) => ({
      toolUse: block,
      toolResult: result,
      sourceMessageUuid: sourceUuid,
      timestamp,
    }),
  )
}

// Filter sessions for chat view display
export const filterSessionsForChat = (
  sessions: Array<Session>,
): Array<Session> => {
  return sessions.filter((session) => {
    // Filter out FILE_SNAPSHOT system messages
    if (
      session.type === SessionType.SYSTEM &&
      session.data.case === 'systemData'
    ) {
      const systemData = session.data.value
      return systemData.subtype !== SystemMessageSubtype.FILE_SNAPSHOT
    }
    return true
  })
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
