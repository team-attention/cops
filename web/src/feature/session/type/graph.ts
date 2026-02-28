import type { Session } from '@/gen/grpcstub/session/v1/session_pb'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

// AgentSegment represents an agent's session span in the timeline (with sessions data)
export interface AgentSegment {
  // 'main' or agentId
  id: string
  // Display label for the segment
  label: string
  // Start timestamp of the agent's first session
  startTime: Timestamp
  // End timestamp of the agent's last session
  endTime: Timestamp
  // All sessions belonging to this agent
  sessions: Array<Session>
  // Calculated Y position for rendering
  yPosition: number
  // Total number of messages in this segment
  messageCount: number
}

// TimelineSegment represents lightweight segment info from API (without sessions)
export interface TimelineSegment {
  // 'main' or agentId
  id: string
  // Display label for the segment
  label: string
  // Start timestamp of the agent's first message
  startTime: Timestamp
  // End timestamp of the agent's last message
  endTime: Timestamp
  // Calculated Y position for rendering
  yPosition: number
  // Total number of messages in this segment
  messageCount: number
}

// SegmentTimelineData contains all data needed to render the segment-based timeline
export interface SegmentTimelineData {
  // All segments (Main + SubAgents)
  segments: Array<AgentSegment>
  // Time range in seconds for X axis calculation
  timeRange: { start: number; end: number }
  // Total duration in seconds
  totalDuration: number
}

// SubAgentInfo extracted from session metadata
export interface SubAgentInfo {
  // SubAgent's unique identifier
  agentId: string
  // First timestamp from SubAgent's sessions
  timestamp?: Timestamp
}
