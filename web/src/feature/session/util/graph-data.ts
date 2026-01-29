import type {
  Session,
  TokenUsage,
} from '@/gen/grpcstub/session/v1/session_pb'
import { SessionType, ProgressType } from '@/gen/grpcstub/session/v1/session_pb'
import { create } from '@bufbuild/protobuf'
import { TokenUsageSchema } from '@/gen/grpcstub/session/v1/session_pb'
import { Position } from '@xyflow/react'
import type {
  SessionNode,
  SessionEdge,
  SubAgentInfo,
  SessionNodeData,
} from '../type/graph'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

// NODE_SPACING defines the horizontal gap between nodes
const NODE_SPACING_X = 300
// NODE_SPACING_Y defines the vertical gap for parallel nodes
const NODE_SPACING_Y = 150

// Helper to get metadata from a session
const getSessionMetadata = (session: Session) => {
  switch (session.data.case) {
    case 'humanData':
      return session.data.value.metadata
    case 'agentData':
      return session.data.value.metadata
    case 'toolExecutionData':
      return session.data.value.metadata
    case 'systemData':
      return session.data.value.metadata
    case 'progressData':
      return session.data.value.metadata
    default:
      return undefined
  }
}

// extractSubAgentInfo extracts SubAgent spawn information from Progress entries.
// Looks for PROGRESS_TYPE_AGENT entries in the Main session (no agentId).
const extractSubAgentInfo = (sessions: Session[]): SubAgentInfo[] => {
  const subAgents: SubAgentInfo[] = []

  for (const session of sessions) {
    // Only look at Progress entries
    if (
      session.type !== SessionType.PROGRESS ||
      session.data.case !== 'progressData'
    ) {
      continue
    }

    const progress = session.data.value
    const data = progress.data
    const metadata = progress.metadata

    // Only PROGRESS_TYPE_AGENT entries
    if (data?.type !== ProgressType.AGENT) {
      continue
    }

    // Only Main session entries (no agentId in metadata)
    if (metadata?.agentId) {
      continue
    }

    // Extract SubAgent info
    const agentId = data.agentId
    if (!agentId) {
      continue
    }

    subAgents.push({
      agentId,
      spawnedByToolUseId: progress.toolExecutionId,
      prompt: data.prompt,
      timestamp: metadata?.timestamp,
    })
  }

  return subAgents
}

// groupSessionsByAgent groups sessions by their agentId.
// Main session entries (agentId undefined/empty) go to 'main' key.
const groupSessionsByAgent = (sessions: Session[]): Map<string, Session[]> => {
  const grouped = new Map<string, Session[]>()

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const agentId = metadata?.agentId || 'main'

    const existing = grouped.get(agentId) || []
    existing.push(session)
    grouped.set(agentId, existing)
  }

  return grouped
}

// aggregateTokenUsage aggregates token usage across sessions.
const aggregateTokenUsage = (sessions: Session[]): TokenUsage | undefined => {
  let inputTokens = 0
  let outputTokens = 0
  let cacheCreationInputTokens = 0
  let cacheReadInputTokens = 0
  let hasUsage = false

  for (const session of sessions) {
    if (
      session.type === SessionType.AGENT &&
      session.data.case === 'agentData'
    ) {
      const usage = session.data.value.usage
      if (usage) {
        hasUsage = true
        inputTokens += usage.inputTokens ?? 0
        outputTokens += usage.outputTokens ?? 0
        cacheCreationInputTokens += usage.cacheCreationInputTokens ?? 0
        cacheReadInputTokens += usage.cacheReadInputTokens ?? 0
      }
    }
  }

  if (!hasUsage) {
    return undefined
  }

  return create(TokenUsageSchema, {
    inputTokens,
    outputTokens,
    cacheCreationInputTokens,
    cacheReadInputTokens,
  })
}

// getFirstTimestamp returns the earliest timestamp from sessions
const getFirstTimestamp = (sessions: Session[]): Timestamp | undefined => {
  let earliest: Timestamp | undefined

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const ts = metadata?.timestamp
    if (ts) {
      if (!earliest || Number(ts.seconds) < Number(earliest.seconds)) {
        earliest = ts
      }
    }
  }

  return earliest
}

// getLastTimestamp returns the latest timestamp from sessions
const getLastTimestamp = (sessions: Session[]): Timestamp | undefined => {
  let latest: Timestamp | undefined

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const ts = metadata?.timestamp
    if (ts) {
      if (!latest || Number(ts.seconds) > Number(latest.seconds)) {
        latest = ts
      }
    }
  }

  return latest
}

// buildGraphElements builds React Flow nodes and edges from session data.
// Returns nodes positioned in left-to-right layout with parallel SubAgents vertically stacked.
// Node IDs: 'main' for Main session, agentId for SubAgent nodes.
export const buildGraphElements = (
  sessions: Session[],
): { nodes: SessionNode[]; edges: SessionEdge[] } => {
  const nodes: SessionNode[] = []
  const edges: SessionEdge[] = []

  // Extract SubAgentInfo
  const subAgentInfos = extractSubAgentInfo(sessions)

  // Group sessions by agent
  const groupedSessions = groupSessionsByAgent(sessions)

  // Create Main node
  const mainSessions = groupedSessions.get('main') || []
  const mainNodeData: SessionNodeData = {
    id: 'main',
    agentId: undefined,
    label: 'Main Session',
    sessions: mainSessions,
    usage: aggregateTokenUsage(mainSessions),
    startedAt: getFirstTimestamp(mainSessions),
    endedAt: getLastTimestamp(mainSessions),
    messageCount: mainSessions.length,
  }

  nodes.push({
    id: 'main',
    type: 'sessionNode',
    position: { x: 0, y: 0 },
    data: mainNodeData,
    sourcePosition: Position.Right,
    targetPosition: Position.Left,
  } satisfies SessionNode)

  // Group SubAgents by timestamp to identify parallel spawns
  // Use toolExecutionId for grouping since parallel calls happen in same batch
  const subAgentsBySpawnTime = new Map<string, SubAgentInfo[]>()
  for (const info of subAgentInfos) {
    const key = info.timestamp ? String(info.timestamp.seconds) : 'unknown'
    const existing = subAgentsBySpawnTime.get(key) || []
    existing.push(info)
    subAgentsBySpawnTime.set(key, existing)
  }

  // Sort spawn times for sequential positioning
  const sortedSpawnTimes = Array.from(subAgentsBySpawnTime.keys()).sort(
    (a, b) => Number(a) - Number(b),
  )

  // Create SubAgent nodes
  let xIndex = 1
  for (const spawnTime of sortedSpawnTimes) {
    const batch = subAgentsBySpawnTime.get(spawnTime) || []
    const batchSize = batch.length

    // Calculate y positions for parallel agents (vertically centered around y=0)
    const startY = -((batchSize - 1) * NODE_SPACING_Y) / 2

    batch.forEach((info, idx) => {
      const agentSessions = groupedSessions.get(info.agentId) || []
      const nodeData: SessionNodeData = {
        id: info.agentId,
        agentId: info.agentId,
        label: info.prompt
          ? info.prompt.slice(0, 30) + (info.prompt.length > 30 ? '...' : '')
          : `SubAgent`,
        sessions: agentSessions,
        usage: aggregateTokenUsage(agentSessions),
        startedAt: getFirstTimestamp(agentSessions),
        endedAt: getLastTimestamp(agentSessions),
        messageCount: agentSessions.length,
      }

      const yPosition = startY + idx * NODE_SPACING_Y

      nodes.push({
        id: info.agentId,
        type: 'sessionNode',
        position: { x: xIndex * NODE_SPACING_X, y: yPosition },
        data: nodeData,
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
      } satisfies SessionNode)

      // Create edge from Main to SubAgent
      edges.push({
        id: `e-main-${info.agentId}`,
        source: 'main',
        target: info.agentId,
        animated: false,
      })
    })

    xIndex++
  }

  return { nodes, edges }
}
