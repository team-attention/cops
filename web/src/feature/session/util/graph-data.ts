import type { Session } from '@/gen/grpcstub/session/v1/session_pb'
import type { SessionSegment } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import type {
  AgentSegment,
  SegmentTimelineData,
  SubAgentInfo,
  TimelineSegment,
} from '../type/graph'

// LANE_SPACING defines the vertical gap between lanes
const LANE_SPACING = 60

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

// getFirstTimestamp returns the earliest timestamp from sessions
const getFirstTimestamp = (sessions: Array<Session>): Timestamp | undefined => {
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

// extractSubAgentInfo extracts SubAgent information from grouped sessions
const extractSubAgentInfo = (
  groupedSessions: Map<string, Array<Session>>,
): Array<SubAgentInfo> => {
  const subAgents: Array<SubAgentInfo> = []

  for (const [agentId, sessions] of groupedSessions) {
    if (agentId === 'main') {
      continue
    }

    const timestamp = getFirstTimestamp(sessions)
    subAgents.push({
      agentId,
      timestamp,
    })
  }

  // Sort by timestamp for consistent ordering
  subAgents.sort((a, b) => {
    if (!a.timestamp || !b.timestamp) {
      return 0
    }
    return Number(a.timestamp.seconds) - Number(b.timestamp.seconds)
  })

  return subAgents
}

// groupSessionsByAgent groups sessions by their agentId
const groupSessionsByAgent = (
  sessions: Array<Session>,
): Map<string, Array<Session>> => {
  const grouped = new Map<string, Array<Session>>()

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const agentId = metadata?.agentId || 'main'

    const existing = grouped.get(agentId) || []
    existing.push(session)
    grouped.set(agentId, existing)
  }

  return grouped
}

// getTimeRange calculates min and max timestamps from sessions
const getTimeRange = (
  sessions: Array<Session>,
): { minTime: Timestamp | undefined; maxTime: Timestamp | undefined } => {
  let minTime: Timestamp | undefined
  let maxTime: Timestamp | undefined

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const ts = metadata?.timestamp
    if (ts) {
      if (!minTime || Number(ts.seconds) < Number(minTime.seconds)) {
        minTime = ts
      }
      if (!maxTime || Number(ts.seconds) > Number(maxTime.seconds)) {
        maxTime = ts
      }
    }
  }

  return { minTime, maxTime }
}

// buildSegmentData converts sessions into SegmentTimelineData for visualization
export const buildSegmentData = (
  sessions: Array<Session>,
): SegmentTimelineData => {
  const groupedSessions = groupSessionsByAgent(sessions)
  const subAgentInfos = extractSubAgentInfo(groupedSessions)

  // Calculate overall time range
  let minTimeSeconds = Number.MAX_SAFE_INTEGER
  let maxTimeSeconds = 0

  for (const session of sessions) {
    const metadata = getSessionMetadata(session)
    const ts = metadata?.timestamp
    if (ts) {
      const seconds = Number(ts.seconds)
      if (seconds < minTimeSeconds) minTimeSeconds = seconds
      if (seconds > maxTimeSeconds) maxTimeSeconds = seconds
    }
  }

  // Handle edge case with no timestamps
  if (minTimeSeconds === Number.MAX_SAFE_INTEGER) {
    minTimeSeconds = 0
    maxTimeSeconds = 0
  }

  const timeRange = { start: minTimeSeconds, end: maxTimeSeconds }
  const totalDuration = maxTimeSeconds - minTimeSeconds

  // Build segments
  const segments: Array<AgentSegment> = []

  // Main segment (center position, y = 0)
  const mainSessions = groupedSessions.get('main') || []
  const mainSessionsWithTimestamp = mainSessions.filter((session) => {
    const metadata = getSessionMetadata(session)
    return metadata?.timestamp !== undefined
  })

  if (mainSessionsWithTimestamp.length > 0) {
    const { minTime, maxTime } = getTimeRange(mainSessionsWithTimestamp)
    if (minTime && maxTime) {
      segments.push({
        id: 'main',
        label: 'Main',
        startTime: minTime,
        endTime: maxTime,
        sessions: mainSessionsWithTimestamp,
        yPosition: 0,
        messageCount: mainSessionsWithTimestamp.length,
      })
    }
  }

  // SubAgent segments (all below main, in order of first activity)
  subAgentInfos.forEach((info, index) => {
    const agentSessions = groupedSessions.get(info.agentId) || []
    const agentSessionsWithTimestamp = agentSessions.filter((session) => {
      const metadata = getSessionMetadata(session)
      return metadata?.timestamp !== undefined
    })

    if (agentSessionsWithTimestamp.length > 0) {
      const { minTime, maxTime } = getTimeRange(agentSessionsWithTimestamp)
      if (minTime && maxTime) {
        // All SubAgents below main, in sequential order
        const yPosition = (index + 1) * LANE_SPACING

        segments.push({
          id: info.agentId,
          label: `SubAgent ${index + 1}`,
          startTime: minTime,
          endTime: maxTime,
          sessions: agentSessionsWithTimestamp,
          yPosition,
          messageCount: agentSessionsWithTimestamp.length,
        })
      }
    }
  })

  return { segments, timeRange, totalDuration }
}

// timestampToX converts a timestamp to X coordinate
export const timestampToX = (
  timestamp: Timestamp,
  timeRange: { start: number; end: number },
  width: number,
  padding = 40,
): number => {
  const seconds = Number(timestamp.seconds)
  const duration = timeRange.end - timeRange.start

  if (duration === 0) {
    return padding + (width - 2 * padding) / 2
  }

  const ratio = (seconds - timeRange.start) / duration
  return padding + ratio * (width - 2 * padding)
}

// Helper to get the message UUID from a session for navigation
export const getSessionMessageId = (session: Session): string | undefined => {
  const metadata = getSessionMetadata(session)
  return metadata?.uuid
}

// TimelineData contains processed segment data for timeline rendering
export interface TimelineData {
  // All segments with calculated yPosition
  segments: Array<TimelineSegment>
  // Time range in seconds for X axis calculation
  timeRange: { start: number; end: number }
  // Total duration in seconds
  totalDuration: number
}

// convertApiSegmentsToTimeline converts API SessionSegment array to TimelineData
export const convertApiSegmentsToTimeline = (
  apiSegments: Array<SessionSegment>,
  startTime: Timestamp | undefined,
  endTime: Timestamp | undefined,
  totalDurationSeconds: bigint,
): TimelineData => {
  // Calculate time range
  const startSeconds = startTime ? Number(startTime.seconds) : 0
  const endSeconds = endTime ? Number(endTime.seconds) : 0
  const timeRange = { start: startSeconds, end: endSeconds }
  const totalDuration = Number(totalDurationSeconds)

  // Separate main and subagent segments
  const mainSegment = apiSegments.find((s) => s.id === 'main')
  const subAgentSegments = apiSegments.filter((s) => s.id !== 'main')

  // Sort subagents by start time for consistent positioning
  subAgentSegments.sort((a, b) => {
    const aStart = a.startTime ? Number(a.startTime.seconds) : 0
    const bStart = b.startTime ? Number(b.startTime.seconds) : 0
    return aStart - bStart
  })

  // Build timeline segments with y positions
  const segments: Array<TimelineSegment> = []

  // Main segment at center (y = 0)
  if (mainSegment && mainSegment.startTime && mainSegment.endTime) {
    segments.push({
      id: mainSegment.id,
      label: mainSegment.label,
      startTime: mainSegment.startTime,
      endTime: mainSegment.endTime,
      yPosition: 0,
      messageCount: mainSegment.messageCount,
    })
  }

  // SubAgent segments (all below main, in order of start time)
  subAgentSegments.forEach((apiSeg, index) => {
    if (!apiSeg.startTime || !apiSeg.endTime) return

    // All SubAgents below main, in sequential order
    const yPosition = (index + 1) * LANE_SPACING

    segments.push({
      id: apiSeg.id,
      label: apiSeg.label,
      startTime: apiSeg.startTime,
      endTime: apiSeg.endTime,
      yPosition,
      messageCount: apiSeg.messageCount,
    })
  })

  return { segments, timeRange, totalDuration }
}
