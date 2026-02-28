import { useLayoutEffect, useRef, useState } from 'react'
import {
  AlertCircle,
  Bot,
  Check,
  ChevronRight,
  Copy,
  Play,
  User,
  Wrench,
  Zap,
} from 'lucide-react'
import type { ParsedMessage } from '../type/content-block'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import { Badge } from '@/gen/shadcn/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/gen/shadcn/ui/tooltip'
import { truncateId } from '@/shared/util/format'

interface MessageBubbleProps {
  message: ParsedMessage
  isSelected?: boolean
  onSelect?: () => void
  onToolClick?: (toolUseId: string) => void
}

const formatTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return ''
  const date = new Date(Number(timestamp.seconds) * 1000)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const formatTokens = (value: number | undefined): string => {
  if (!value) return '0'
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return value.toString()
}

// Progress type to label mapping
const getProgressLabel = (type: string | undefined) => {
  switch (type) {
    case 'agent':
      return 'Agent'
    case 'skill':
      return 'Skill'
    case 'hook':
      return 'Hook'
    case 'bash':
      return 'Bash'
    case 'mcp':
      return 'MCP'
    default:
      return 'Unknown'
  }
}

// Tool name to icon mapping
const getToolIcon = (toolName: string) => {
  const name = toolName.toLowerCase()
  if (name.includes('read') || name.includes('glob') || name.includes('grep')) {
    return '📖'
  }
  if (name.includes('write') || name.includes('edit')) {
    return '✏️'
  }
  if (name.includes('bash') || name.includes('shell')) {
    return '⚡'
  }
  if (name.includes('task') || name.includes('agent')) {
    return '🤖'
  }
  if (name.includes('web') || name.includes('fetch')) {
    return '🌐'
  }
  return '🔧'
}

export const MessageBubble = ({
  message,
  isSelected = false,
  onSelect,
  onToolClick,
}: MessageBubbleProps) => {
  const [copied, setCopied] = useState(false)
  const [isTruncated, setIsTruncated] = useState(false)
  const textRef = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const element = textRef.current
    if (element) {
      setIsTruncated(element.scrollHeight > element.clientHeight)
    }
  }, [message.content])

  const handleCopyUuid = async (e: React.MouseEvent) => {
    e.stopPropagation()
    await navigator.clipboard.writeText(message.uuid)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const isUser = message.type === 'user'
  const isAssistant = message.type === 'assistant'
  const isSystem = message.type === 'system' || message.isMeta
  const isToolResult = message.type === 'tool_result'
  const isProgress = message.type === 'progress'
  const isUserToolResult = isUser && message.isToolUseResult

  // Extract text content and tool uses
  const textContent = message.content
    .filter((block) => block.type === 'text')
    .map((block) => (block as { type: 'text'; text: string }).text)
    .join('\n')

  const toolUses = message.content.filter((block) => block.type === 'tool_use')

  // Calculate total tokens for assistant messages
  const totalTokens = message.usage
    ? (message.usage.inputTokens ?? 0) + (message.usage.outputTokens ?? 0)
    : 0

  // Base styles
  const containerStyles = `
    w-full group relative rounded-xl border p-3 transition-all duration-200 cursor-pointer
    ${isSelected ? 'ring-2 ring-cyan-500/50' : ''}
    ${
      isSystem
        ? 'border-dashed border-zinc-700/50 bg-zinc-900/30 opacity-50 hover:opacity-75'
        : isUser
          ? 'border-cyan-500/20 bg-gradient-to-br from-cyan-950/30 to-zinc-900/80 hover:border-cyan-500/40'
          : isAssistant
            ? 'border-violet-500/20 bg-gradient-to-br from-violet-950/30 to-zinc-900/80 hover:border-violet-500/40'
            : isToolResult
              ? 'border-amber-500/20 bg-gradient-to-br from-amber-950/30 to-zinc-900/80 hover:border-amber-500/40'
              : isProgress
                ? 'border-emerald-500/20 bg-gradient-to-br from-emerald-950/30 to-zinc-900/80 hover:border-emerald-500/40'
                : 'border-zinc-800/50 bg-zinc-900/50 hover:border-zinc-700/50'
    }
  `

  return (
    <div className={containerStyles} onClick={onSelect}>
      {/* Glow effect on hover */}
      {!isSystem && (
        <div
          className={`pointer-events-none absolute inset-0 rounded-xl opacity-0 blur-xl transition-opacity duration-300 group-hover:opacity-20 ${
            isUser
              ? 'bg-cyan-500'
              : isToolResult
                ? 'bg-amber-500'
                : isProgress
                  ? 'bg-emerald-500'
                  : 'bg-violet-500'
          }`}
        />
      )}

      {/* Header */}
      <div className="relative mb-2 flex min-w-0 items-center justify-between">
        <div className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
          {/* Avatar */}
          <div
            className={`rounded-lg p-1.5 ${
              isSystem
                ? 'bg-zinc-800'
                : isUser
                  ? 'bg-cyan-500/10 text-cyan-400'
                  : isToolResult
                    ? 'bg-amber-500/10 text-amber-400'
                    : isProgress
                      ? 'bg-emerald-500/10 text-emerald-400'
                      : 'bg-violet-500/10 text-violet-400'
            }`}
          >
            {isSystem ? (
              <AlertCircle className="h-4 w-4 text-zinc-500" />
            ) : isUserToolResult ? (
              <Wrench className="h-4 w-4" />
            ) : isUser ? (
              <User className="h-4 w-4" />
            ) : isToolResult ? (
              <Wrench className="h-4 w-4 text-amber-400" />
            ) : isProgress ? (
              <Play className="h-4 w-4" />
            ) : (
              <Bot className="h-4 w-4" />
            )}
          </div>

          {/* Role label */}
          <span
            className={`font-mono text-xs font-medium ${
              isSystem
                ? 'text-zinc-600'
                : isUser
                  ? 'text-cyan-400'
                  : isToolResult
                    ? 'text-amber-400'
                    : isProgress
                      ? 'text-emerald-400'
                      : 'text-violet-400'
            }`}
          >
            {isSystem
              ? 'System'
              : isUserToolResult
                ? 'Tool Result'
                : isUser
                  ? 'Human'
                  : isToolResult
                    ? message.toolName || 'Tool Result'
                    : isProgress
                      ? `${getProgressLabel(message.progressType)} Progress`
                      : 'Assistant'}
          </span>

          {/* Token usage for assistant */}
          {isAssistant && totalTokens > 0 && (
            <Badge
              variant="outline"
              className="ml-2 border-violet-500/20 bg-violet-500/5 font-mono text-[10px] text-violet-300"
            >
              <Zap className="mr-1 h-2.5 w-2.5" />
              {formatTokens(totalTokens)}
            </Badge>
          )}
        </div>

        {/* UUID, Timestamp, and Click Hint */}
        <div className="flex flex-shrink-0 items-center gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  onClick={handleCopyUuid}
                  className="flex items-center gap-1 rounded px-1 py-0.5 font-mono text-[10px] text-zinc-600 transition-colors hover:bg-zinc-800/50 hover:text-zinc-400"
                >
                  {copied ? (
                    <Check className="h-2.5 w-2.5 text-emerald-400" />
                  ) : (
                    <Copy className="h-2.5 w-2.5" />
                  )}
                  <span className={copied ? 'text-emerald-400' : ''}>
                    {truncateId(message.uuid, 12)}
                  </span>
                </button>
              </TooltipTrigger>
              <TooltipContent side="top" className="font-mono text-xs">
                {message.uuid}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <span className="font-mono text-[10px] text-zinc-600">
            {formatTime(message.timestamp)}
          </span>
          {/* Click hint - visible on hover */}
          <ChevronRight className="h-4 w-4 text-zinc-600 opacity-0 transition-opacity group-hover:opacity-100" />
        </div>
      </div>

      {/* Content */}
      <div className="relative min-w-0">
        {textContent && (
          <div className="relative">
            <div
              ref={textRef}
              className={`line-clamp-3 whitespace-pre-wrap break-words font-mono text-sm leading-relaxed tracking-wide ${
                isSystem ? 'text-zinc-500' : 'text-zinc-200'
              }`}
            >
              {textContent}
            </div>
            {/* Fade-out gradient when truncated */}
            {isTruncated && (
              <div
                className={`pointer-events-none absolute bottom-0 left-0 right-0 h-6 ${
                  isUser
                    ? 'bg-gradient-to-t from-cyan-950/80 to-transparent'
                    : isToolResult
                      ? 'bg-gradient-to-t from-amber-950/80 to-transparent'
                      : isProgress
                        ? 'bg-gradient-to-t from-emerald-950/80 to-transparent'
                        : isSystem
                          ? 'bg-gradient-to-t from-zinc-900/80 to-transparent'
                          : 'bg-gradient-to-t from-violet-950/80 to-transparent'
                }`}
              />
            )}
          </div>
        )}

        {/* Tool Use Chips */}
        {toolUses.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {toolUses.map((tool) => (
              <button
                key={tool.id}
                onClick={(e) => {
                  e.stopPropagation()
                  onToolClick?.(tool.id)
                }}
                className="group/tool flex items-center gap-1.5 rounded-lg border border-amber-500/20 bg-amber-500/5 px-2.5 py-1 font-mono text-xs text-amber-300 transition-all hover:border-amber-500/40 hover:bg-amber-500/10"
              >
                <span>{getToolIcon(tool.name)}</span>
                <span>{tool.name}</span>
                <span className="text-amber-500/50 transition-colors group-hover/tool:text-amber-400">
                  →
                </span>
              </button>
            ))}
          </div>
        )}

        {/* Tool Result Content - Compact Indicator */}
        {isToolResult && message.content[0]?.type === 'tool_result' && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              if (message.parentToolUseId) {
                onToolClick?.(message.parentToolUseId)
              }
            }}
            className="group/result mt-2 flex items-center gap-2 rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 transition-all hover:border-amber-500/40 hover:bg-amber-500/10"
          >
            <span className="font-mono text-xs text-amber-300">
              {message.toolName ? getToolIcon(message.toolName) : '📤'}
            </span>
            <span className="font-mono text-xs text-zinc-400">
              Result returned
            </span>
            <span className="font-mono text-xs text-amber-500/50 transition-colors group-hover/result:text-amber-400">
              View in panel →
            </span>
          </button>
        )}
      </div>

      {/* Selection indicator */}
      {isSelected && (
        <div className="absolute -left-px top-4 h-8 w-1 rounded-r-full bg-cyan-500" />
      )}
    </div>
  )
}
