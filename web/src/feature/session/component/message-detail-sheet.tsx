import { useState } from 'react'
import {
  AlertCircle,
  Bot,
  Check,
  Copy,
  Play,
  User,
  Wrench,
} from 'lucide-react'
import { ContentPanel } from './content-panel'
import { ToolCallItem } from './tool-call-item'
import type { LinkedToolCall, ParsedMessage } from '../type/content-block'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/gen/shadcn/ui/sheet'
import { Badge } from '@/gen/shadcn/ui/badge'
import { ScrollArea } from '@/gen/shadcn/ui/scroll-area'

interface MessageDetailSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  message: ParsedMessage | null
  relatedToolCalls: Array<LinkedToolCall>
}

const formatTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return ''
  const date = new Date(Number(timestamp.seconds) * 1000)
  return date.toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

const formatDate = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return ''
  const date = new Date(Number(timestamp.seconds) * 1000)
  return date.toLocaleDateString([], {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// Progress type to label mapping
const getProgressLabel = (type: string | undefined) => {
  switch (type) {
    case 'agent':
      return 'Agent'
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

export const MessageDetailSheet = ({
  open,
  onOpenChange,
  message,
  relatedToolCalls,
}: MessageDetailSheetProps) => {
  const [copied, setCopied] = useState(false)

  if (!message) return null

  const handleCopyUuid = async () => {
    await navigator.clipboard.writeText(message.uuid)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const isUser = message.type === 'user'
  const isSystem = message.type === 'system' || message.isMeta
  const isToolResult = message.type === 'tool_result'
  const isProgress = message.type === 'progress'

  // Extract full text content (no truncation)
  const textContent = message.content
    .filter((block) => block.type === 'text')
    .map((block) => (block as { type: 'text'; text: string }).text)
    .join('\n')

  // Get message type info
  const getTypeInfo = () => {
    if (isSystem) return { label: 'System', icon: AlertCircle, color: 'zinc' }
    if (isUser) return { label: 'Human', icon: User, color: 'cyan' }
    if (isToolResult)
      return {
        label: message.toolName || 'Tool Result',
        icon: Wrench,
        color: 'amber',
      }
    if (isProgress)
      return {
        label: `${getProgressLabel(message.progressType)} Progress`,
        icon: Play,
        color: 'emerald',
      }
    return { label: 'Assistant', icon: Bot, color: 'violet' }
  }

  const typeInfo = getTypeInfo()
  const TypeIcon = typeInfo.icon

  const colorClasses: Record<string, { bg: string; text: string; border: string }> = {
    zinc: {
      bg: 'bg-zinc-800',
      text: 'text-zinc-500',
      border: 'border-zinc-700',
    },
    cyan: {
      bg: 'bg-cyan-500/10',
      text: 'text-cyan-400',
      border: 'border-cyan-500/30',
    },
    amber: {
      bg: 'bg-amber-500/10',
      text: 'text-amber-400',
      border: 'border-amber-500/30',
    },
    emerald: {
      bg: 'bg-emerald-500/10',
      text: 'text-emerald-400',
      border: 'border-emerald-500/30',
    },
    violet: {
      bg: 'bg-violet-500/10',
      text: 'text-violet-400',
      border: 'border-violet-500/30',
    },
  }

  const colors = colorClasses[typeInfo.color]

  return (
    <Sheet open={open} onOpenChange={onOpenChange} modal={false}>
      <SheetContent
        side="right"
        className="w-full border-zinc-800 bg-zinc-900/95 backdrop-blur-sm sm:max-w-lg lg:max-w-xl"
      >
        <SheetHeader className="border-b border-zinc-800/50 pb-4">
          <div className="flex items-center gap-3">
            <div className={`rounded-lg p-2 ${colors.bg}`}>
              <TypeIcon className={`h-5 w-5 ${colors.text}`} />
            </div>
            <div className="flex-1">
              <SheetTitle className="text-zinc-100">
                <Badge
                  variant="outline"
                  className={`${colors.border} ${colors.bg} font-mono text-xs ${colors.text}`}
                >
                  {typeInfo.label}
                </Badge>
              </SheetTitle>
              <SheetDescription className="mt-1 font-mono text-xs text-zinc-500">
                {formatDate(message.timestamp)} at{' '}
                {formatTime(message.timestamp)}
              </SheetDescription>
            </div>
          </div>

          {/* UUID */}
          <div className="mt-3 flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
              UUID
            </span>
            <button
              onClick={handleCopyUuid}
              className="flex items-center gap-1.5 rounded-md border border-zinc-800 bg-zinc-900/50 px-2 py-1 font-mono text-xs text-zinc-400 transition-colors hover:border-zinc-700 hover:bg-zinc-800/50 hover:text-zinc-300"
            >
              {copied ? (
                <>
                  <Check className="h-3 w-3 text-emerald-400" />
                  <span className="text-emerald-400">Copied!</span>
                </>
              ) : (
                <>
                  <Copy className="h-3 w-3" />
                  <span className="max-w-[280px] truncate">{message.uuid}</span>
                </>
              )}
            </button>
          </div>
        </SheetHeader>

        <ScrollArea className="flex-1 min-h-0 px-4">
          <div className="space-y-6 py-4">
            {/* Full Message Content */}
            {textContent && (
              <div>
                <div className="mb-2 flex items-center gap-2">
                  <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-500">
                    Content
                  </span>
                </div>
                <ContentPanel>{textContent}</ContentPanel>
              </div>
            )}

            {/* Tool Calls Section */}
            {relatedToolCalls.length > 0 && (
              <div>
                <div className="mb-3 flex items-center gap-2">
                  <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-500">
                    Tool Calls
                  </span>
                  <Badge
                    variant="outline"
                    className="border-zinc-700 bg-zinc-800/50 font-mono text-[10px] text-zinc-400"
                  >
                    {relatedToolCalls.length}
                  </Badge>
                </div>
                <div className="space-y-2">
                  {relatedToolCalls.map((toolCall) => (
                    <ToolCallItem
                      key={toolCall.toolUse.id}
                      toolCall={toolCall}
                      isHighlighted={false}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* Empty state */}
            {!textContent && relatedToolCalls.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-zinc-500">
                <AlertCircle className="mb-3 h-8 w-8 opacity-30" />
                <p className="font-mono text-sm">No content available</p>
              </div>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  )
}
