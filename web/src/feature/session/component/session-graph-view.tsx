import { useCallback, useMemo, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { NodeMouseHandler, NodeTypes } from '@xyflow/react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Network } from 'lucide-react'
import { SessionNode } from './session-node'
import { MessagePopover } from './message-popover'
import { buildGraphElements } from '../util/graph-data'
import type { Session } from '@/gen/grpcstub/session/v1/session_pb'
import type { SessionNode as SessionNodeType, AppNode } from '../type/graph'

// nodeTypes registers custom node components
const nodeTypes: NodeTypes = {
  sessionNode: SessionNode,
}

// Default edge styling for dark theme
const defaultEdgeOptions = {
  style: { stroke: '#6366f1', strokeWidth: 2 },
  animated: false,
}

interface SessionGraphViewProps {
  // All session entries (Main + SubAgent combined)
  sessions: Session[]
}

// SessionGraphView renders an interactive graph of Main and SubAgent sessions.
export const SessionGraphView = ({ sessions }: SessionGraphViewProps) => {
  // Build graph elements from sessions
  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => buildGraphElements(sessions),
    [sessions],
  )

  // React Flow state
  const [nodes, , onNodesChange] = useNodesState<AppNode>(initialNodes)
  const [edges, , onEdgesChange] = useEdgesState(initialEdges)

  // State for selected node and popover
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [popoverPosition, setPopoverPosition] = useState<{
    x: number
    y: number
  } | null>(null)

  // Handle node click
  const onNodeClick: NodeMouseHandler = useCallback((event, node) => {
    setSelectedNodeId(node.id)
    setPopoverPosition({
      x: event.clientX,
      y: event.clientY,
    })
  }, [])

  // Handle popover close
  const handlePopoverClose = useCallback(() => {
    setSelectedNodeId(null)
    setPopoverPosition(null)
  }, [])

  // Helper to check if node is a session node
  const isSessionNode = (node: AppNode): node is SessionNodeType => {
    return node.type === 'sessionNode'
  }

  // Get selected node's sessions for popover
  const selectedNodeSessions = useMemo(() => {
    if (!selectedNodeId) return []
    const node = nodes.find((n) => n.id === selectedNodeId)
    if (!node || !isSessionNode(node)) return []
    return node.data.sessions
  }, [nodes, selectedNodeId])

  // Get selected node's label for popover
  const selectedNodeLabel = useMemo(() => {
    if (!selectedNodeId) return ''
    const node = nodes.find((n) => n.id === selectedNodeId)
    if (!node || !isSessionNode(node)) return ''
    return node.data.label
  }, [nodes, selectedNodeId])

  // Count SubAgent nodes
  const subAgentCount = nodes.filter((n) => {
    if (!isSessionNode(n)) return false
    return n.data.agentId
  }).length

  // MiniMap node color function
  const getNodeColor = useCallback((node: AppNode) => {
    if (!isSessionNode(node)) return '#3f3f46'
    return node.data.agentId ? '#22d3ee' : '#8b5cf6'
  }, [])

  return (
    <Card className="group relative flex min-h-[500px] flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient glow */}
      <div className="pointer-events-none absolute -right-16 -top-16 h-32 w-32 rounded-full bg-violet-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-violet-500/10" />

      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-violet-500/20 bg-violet-500/10 p-2">
              <Network className="h-4 w-4 text-violet-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Session Graph
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Main and SubAgent relationships
              </p>
            </div>
          </div>
          {subAgentCount > 0 && (
            <Badge
              variant="outline"
              className="border-cyan-500/20 bg-cyan-500/5 font-mono text-xs text-cyan-400"
            >
              {subAgentCount} SubAgent{subAgentCount !== 1 ? 's' : ''}
            </Badge>
          )}
        </div>
      </CardHeader>

      <CardContent className="relative flex-1 overflow-hidden p-0">
        <div className="h-[450px] w-full">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={onNodeClick}
            nodeTypes={nodeTypes}
            defaultEdgeOptions={defaultEdgeOptions}
            fitView
            fitViewOptions={{ padding: 0.3 }}
            className="bg-zinc-900"
            proOptions={{ hideAttribution: true }}
          >
            <Background color="#3f3f46" gap={16} />
            <Controls
              className="!rounded-xl !border-zinc-700 !bg-zinc-800"
              showInteractive={false}
            />
            <MiniMap
              className="!rounded-xl !border-zinc-700 !bg-zinc-900"
              nodeColor={getNodeColor}
              maskColor="rgba(24, 24, 27, 0.8)"
            />
          </ReactFlow>
        </div>

        {/* Message Popover when node is selected */}
        {selectedNodeId && popoverPosition && (
          <MessagePopover
            sessions={selectedNodeSessions}
            nodeLabel={selectedNodeLabel}
            position={popoverPosition}
            onClose={handlePopoverClose}
          />
        )}
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-violet-500 to-cyan-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
