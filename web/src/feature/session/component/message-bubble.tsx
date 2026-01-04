import { AlertCircle, Bot, User, Wrench, Zap } from 'lucide-react'
import type { ParsedMessage } from '../type/content-block'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import { Badge } from '@/gen/shadcn/ui/badge'

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
  const isUser = message.type === 'user'
  const isAssistant = message.type === 'assistant'
  const isSystem = message.type === 'system' || message.isMeta
  const isToolResult = message.type === 'tool_result'

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
    group relative rounded-xl border p-4 transition-all duration-200 cursor-pointer
    ${isSelected ? 'ring-2 ring-cyan-500/50' : ''}
    ${
      isSystem
        ? 'border-dashed border-zinc-700/50 bg-zinc-900/30 opacity-50 hover:opacity-75'
        : isUser
          ? 'border-cyan-500/20 bg-gradient-to-br from-cyan-950/30 to-zinc-900/80 hover:border-cyan-500/40'
          : isAssistant
            ? 'border-violet-500/20 bg-gradient-to-br from-violet-950/30 to-zinc-900/80 hover:border-violet-500/40'
            : 'border-zinc-800/50 bg-zinc-900/50 hover:border-zinc-700/50'
    }
  `

  return (
    <div className={containerStyles} onClick={onSelect}>
      {/* Glow effect on hover */}
      {!isSystem && (
        <div
          className={`pointer-events-none absolute inset-0 rounded-xl opacity-0 blur-xl transition-opacity duration-300 group-hover:opacity-20 ${
            isUser ? 'bg-cyan-500' : 'bg-violet-500'
          }`}
        />
      )}

      {/* Header */}
      <div className="relative mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          {/* Avatar */}
          <div
            className={`rounded-lg p-1.5 ${
              isSystem
                ? 'bg-zinc-800'
                : isUser
                  ? 'bg-cyan-500/10 text-cyan-400'
                  : 'bg-violet-500/10 text-violet-400'
            }`}
          >
            {isSystem ? (
              <AlertCircle className="h-4 w-4 text-zinc-500" />
            ) : isUser ? (
              <User className="h-4 w-4" />
            ) : isToolResult ? (
              <Wrench className="h-4 w-4 text-amber-400" />
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
                    : 'text-violet-400'
            }`}
          >
            {isSystem
              ? 'System'
              : isUser
                ? 'Human'
                : isToolResult
                  ? message.toolName || 'Tool Result'
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

        {/* Timestamp */}
        <span className="font-mono text-[10px] text-zinc-600">
          {formatTime(message.timestamp)}
        </span>
      </div>

      {/* Content */}
      <div className="relative">
        {textContent && (
          <div
            className={`whitespace-pre-wrap break-words font-mono text-sm leading-relaxed ${
              isSystem ? 'text-zinc-500' : 'text-zinc-300'
            }`}
          >
            {textContent.length > 1000 ? (
              <>
                {textContent.slice(0, 1000)}
                <span className="text-zinc-600">... (truncated)</span>
              </>
            ) : (
              textContent
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

        {/* Tool Result Content */}
        {isToolResult && message.content[0]?.type === 'tool_result' && (
          <div className="mt-2 rounded-lg border border-zinc-800 bg-zinc-950/50 p-3">
            <pre className="max-h-40 overflow-auto font-mono text-xs text-zinc-400">
              {(message.content[0] as { content: string }).content.slice(
                0,
                500,
              )}
              {(message.content[0] as { content: string }).content.length >
                500 && (
                <span className="text-zinc-600">... (see tool panel)</span>
              )}
            </pre>
          </div>
        )}
      </div>

      {/* Selection indicator */}
      {isSelected && (
        <div className="absolute -left-px top-4 h-8 w-1 rounded-r-full bg-cyan-500" />
      )}
    </div>
  )
}
