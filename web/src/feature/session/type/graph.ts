import type { Node, Edge, BuiltInNode } from '@xyflow/react'
import type { Session, TokenUsage } from '@/gen/grpcstub/session/v1/session_pb'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

// ViewMode for switching between Chat and Graph views
export type ViewMode = 'chat' | 'graph'

// SessionNodeData holds data for each node in the graph.
// Main vs SubAgent is determined at runtime by checking if agentId is present:
// - Main: agentId is undefined/empty, id is 'main'
// - SubAgent: agentId is present, id equals agentId
export interface SessionNodeData extends Record<string, unknown> {
  // Node identifier: 'main' for Main session, agentId for SubAgent
  id: string
  // SubAgent identifier (undefined/empty for Main session)
  agentId?: string
  // Display label for the node
  label: string
  // Sessions belonging to this node (filtered by agentId)
  sessions: Session[]
  // Token usage aggregated for this node
  usage?: TokenUsage
  // First message timestamp
  startedAt?: Timestamp
  // Last message timestamp
  endedAt?: Timestamp
  // Message count for display
  messageCount: number
}

// Helper arrow function to check if node is Main session
export const isMainNode = (data: SessionNodeData): boolean => !data.agentId

// Type alias for React Flow Node with SessionNodeData
export type SessionNode = Node<SessionNodeData, 'sessionNode'>

// Type alias for React Flow Edge
export type SessionEdge = Edge

// Combined AppNode type for React Flow
export type AppNode = SessionNode | BuiltInNode

// SubAgentInfo extracted from Progress entries
export interface SubAgentInfo {
  // SubAgent's unique identifier
  agentId: string
  // Tool execution ID that spawned this SubAgent
  spawnedByToolUseId: string
  // User prompt for SubAgent
  prompt?: string
  // Timestamp when SubAgent was spawned
  timestamp?: Timestamp
}
