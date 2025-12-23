import { useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Copy,
  Check,
  FileText,
  Pencil,
  Terminal,
  Search,
  Globe,
  Bot,
  AlertCircle,
} from 'lucide-react'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/gen/shadcn/ui/collapsible'
import { Badge } from '@/gen/shadcn/ui/badge'
import type { LinkedToolCall } from '../type/content-block'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

interface ToolCallItemProps {
  toolCall: LinkedToolCall
  isHighlighted?: boolean
}

const formatTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return ''
  const date = new Date(Number(timestamp.seconds) * 1000)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// Get icon for tool type
const getToolIcon = (toolName: string) => {
  const name = toolName.toLowerCase()
  if (name.includes('read') || name.includes('glob')) return FileText
  if (name.includes('write') || name.includes('edit')) return Pencil
  if (name.includes('bash') || name.includes('shell')) return Terminal
  if (name.includes('grep') || name.includes('search')) return Search
  if (name.includes('web') || name.includes('fetch')) return Globe
  if (name.includes('task') || name.includes('agent')) return Bot
  return Terminal
}

// Get accent color for tool type
const getToolColor = (toolName: string): string => {
  const name = toolName.toLowerCase()
  if (name.includes('read') || name.includes('glob') || name.includes('grep')) return 'cyan'
  if (name.includes('write') || name.includes('edit')) return 'amber'
  if (name.includes('bash') || name.includes('shell')) return 'emerald'
  if (name.includes('web') || name.includes('fetch')) return 'blue'
  if (name.includes('task') || name.includes('agent')) return 'violet'
  return 'zinc'
}

export const ToolCallItem = ({ toolCall, isHighlighted = false }: ToolCallItemProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showFullOutput, setShowFullOutput] = useState(false)

  const { toolUse, toolResult } = toolCall
  const Icon = getToolIcon(toolUse.name)
  const color = getToolColor(toolUse.name)
  const hasError = toolResult?.is_error

  // Format input JSON
  const inputJson = JSON.stringify(toolUse.input, null, 2)

  // Process output
  const outputContent = toolResult?.content || ''
  const outputLines = outputContent.split('\n')
  const isLongOutput = outputLines.length > 20
  const displayOutput = showFullOutput ? outputContent : outputLines.slice(0, 20).join('\n')

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    await navigator.clipboard.writeText(outputContent)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const colorClasses = {
    cyan: {
      border: 'border-cyan-500/30',
      bg: 'bg-cyan-500/10',
      text: 'text-cyan-400',
      glow: 'bg-cyan-500',
    },
    amber: {
      border: 'border-amber-500/30',
      bg: 'bg-amber-500/10',
      text: 'text-amber-400',
      glow: 'bg-amber-500',
    },
    emerald: {
      border: 'border-emerald-500/30',
      bg: 'bg-emerald-500/10',
      text: 'text-emerald-400',
      glow: 'bg-emerald-500',
    },
    blue: {
      border: 'border-blue-500/30',
      bg: 'bg-blue-500/10',
      text: 'text-blue-400',
      glow: 'bg-blue-500',
    },
    violet: {
      border: 'border-violet-500/30',
      bg: 'bg-violet-500/10',
      text: 'text-violet-400',
      glow: 'bg-violet-500',
    },
    zinc: {
      border: 'border-zinc-700',
      bg: 'bg-zinc-800/50',
      text: 'text-zinc-400',
      glow: 'bg-zinc-500',
    },
  }

  const styles = colorClasses[color as keyof typeof colorClasses]

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div
        className={`group relative overflow-hidden rounded-lg border transition-all duration-200 ${
          isHighlighted
            ? 'border-cyan-500/50 bg-cyan-500/5 ring-1 ring-cyan-500/30'
            : `border-zinc-800/50 bg-zinc-900/50 hover:border-zinc-700/50`
        }`}
      >
        {/* Highlight indicator */}
        {isHighlighted && (
          <div className="absolute left-0 top-0 h-full w-1 bg-cyan-500" />
        )}

        <CollapsibleTrigger className="flex w-full items-center justify-between p-3 text-left">
          <div className="flex items-center gap-3">
            {/* Expand icon */}
            <div className="text-zinc-600 transition-transform duration-200">
              {isOpen ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </div>

            {/* Tool icon */}
            <div className={`rounded-md p-1.5 ${styles.bg}`}>
              <Icon className={`h-4 w-4 ${styles.text}`} />
            </div>

            {/* Tool name */}
            <span className={`font-mono text-sm font-medium ${styles.text}`}>
              {toolUse.name}
            </span>

            {/* Error indicator */}
            {hasError && (
              <Badge
                variant="outline"
                className="border-red-500/30 bg-red-500/10 text-red-400"
              >
                <AlertCircle className="mr-1 h-3 w-3" />
                Error
              </Badge>
            )}

            {/* Status indicator */}
            {toolResult && !hasError && (
              <div className="h-2 w-2 rounded-full bg-emerald-400" />
            )}
          </div>

          {/* Timestamp */}
          <span className="font-mono text-[10px] text-zinc-600">
            {formatTime(toolCall.timestamp)}
          </span>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="space-y-3 border-t border-zinc-800/50 p-3">
            {/* Input */}
            <div>
              <div className="mb-1.5 flex items-center gap-2">
                <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-500">
                  Input
                </span>
              </div>
              <pre className="max-h-32 overflow-auto rounded-lg border border-zinc-800 bg-zinc-950/50 p-2 font-mono text-xs text-zinc-400">
                {inputJson}
              </pre>
            </div>

            {/* Output */}
            {toolResult && (
              <div>
                <div className="mb-1.5 flex items-center justify-between">
                  <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-500">
                    Output
                    {isLongOutput && (
                      <span className="ml-2 text-zinc-600">
                        ({outputLines.length} lines)
                      </span>
                    )}
                  </span>
                  <button
                    onClick={handleCopy}
                    className="flex items-center gap-1 rounded px-2 py-0.5 font-mono text-[10px] text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-300"
                  >
                    {copied ? (
                      <>
                        <Check className="h-3 w-3 text-emerald-400" />
                        <span className="text-emerald-400">Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="h-3 w-3" />
                        <span>Copy</span>
                      </>
                    )}
                  </button>
                </div>
                <pre
                  className={`overflow-auto rounded-lg border p-2 font-mono text-xs ${
                    hasError
                      ? 'border-red-500/20 bg-red-950/20 text-red-300'
                      : 'border-zinc-800 bg-zinc-950/50 text-zinc-400'
                  }`}
                  style={{ maxHeight: showFullOutput ? '400px' : '200px' }}
                >
                  {displayOutput}
                </pre>
                {isLongOutput && (
                  <button
                    onClick={() => setShowFullOutput(!showFullOutput)}
                    className="mt-2 font-mono text-[10px] text-cyan-400 transition-colors hover:text-cyan-300"
                  >
                    {showFullOutput
                      ? '← Show less'
                      : `Show ${outputLines.length - 20} more lines →`}
                  </button>
                )}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}
