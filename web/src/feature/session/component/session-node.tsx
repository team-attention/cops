import { memo } from 'react'
import { Handle, Position } from '@xyflow/react'
import type { NodeProps } from '@xyflow/react'
import { Bot, MessageSquare, User, Zap } from 'lucide-react'
import { Badge } from '@/gen/shadcn/ui/badge'
import type { SessionNode as SessionNodeType } from '../type/graph'

// formatTokenCount formats token count for display (e.g., 1.5K, 2.3M).
const formatTokenCount = (value: number | undefined): string => {
  if (!value || value === 0) return '0'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toString()
}

// SessionNode renders a custom node for Main or SubAgent sessions.
// Props include data (SessionNodeData) and selected state.
// Main vs SubAgent is determined by checking if agentId is present.
const SessionNode = memo(({ data, selected }: NodeProps<SessionNodeType>) => {
  const { agentId, label, messageCount, usage } = data
  const isMain = !agentId

  // Calculate total tokens from usage
  const totalTokens = usage
    ? (usage.inputTokens ?? 0) + (usage.outputTokens ?? 0)
    : 0

  // Theme colors based on node type
  const themeColors = isMain
    ? {
        border: 'border-violet-500',
        borderHover: 'hover:border-violet-400',
        bg: 'bg-violet-500/10',
        iconBg: 'bg-violet-500',
        text: 'text-violet-400',
        ring: 'ring-violet-500/50',
      }
    : {
        border: 'border-cyan-500',
        borderHover: 'hover:border-cyan-400',
        bg: 'bg-cyan-500/10',
        iconBg: 'bg-cyan-500',
        text: 'text-cyan-400',
        ring: 'ring-cyan-500/50',
      }

  return (
    <>
      {/* Target handle on left for incoming connections */}
      <Handle
        type="target"
        position={Position.Left}
        className="!h-3 !w-3 !border-2 !border-zinc-800 !bg-zinc-600"
      />

      {/* Node container */}
      <div
        className={`
          min-w-[180px] max-w-[220px] rounded-xl border-2 p-4 transition-all duration-200
          ${themeColors.border} ${themeColors.borderHover} ${themeColors.bg}
          ${selected ? `ring-2 ${themeColors.ring} border-opacity-100` : 'border-opacity-50'}
          bg-zinc-900/90 backdrop-blur-sm
        `}
      >
        {/* Header with icon and label */}
        <div className="mb-3 flex items-center gap-2">
          <div className={`rounded-lg p-1.5 ${themeColors.iconBg}`}>
            {isMain ? (
              <User className="h-4 w-4 text-white" />
            ) : (
              <Bot className="h-4 w-4 text-white" />
            )}
          </div>
          <span className={`font-mono text-xs font-medium ${themeColors.text}`}>
            {isMain ? 'Main Session' : label}
          </span>
        </div>

        {/* Stats row with message count and token usage */}
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className="border-zinc-700 bg-zinc-800/50 font-mono text-[10px] text-zinc-400"
          >
            <MessageSquare className="mr-1 h-2.5 w-2.5" />
            {messageCount}
          </Badge>

          {totalTokens > 0 && (
            <Badge
              variant="outline"
              className={`border-zinc-700 bg-zinc-800/50 font-mono text-[10px] ${themeColors.text}`}
            >
              <Zap className="mr-1 h-2.5 w-2.5" />
              {formatTokenCount(totalTokens)}
            </Badge>
          )}
        </div>
      </div>

      {/* Source handle on right for outgoing connections */}
      <Handle
        type="source"
        position={Position.Right}
        className="!h-3 !w-3 !border-2 !border-zinc-800 !bg-zinc-600"
      />
    </>
  )
})

// Display name for React DevTools
SessionNode.displayName = 'SessionNode'

export { SessionNode }
