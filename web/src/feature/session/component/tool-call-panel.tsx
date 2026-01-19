import { useMemo } from 'react'
import { Wrench } from 'lucide-react'
import { ToolCallItem } from './tool-call-item'
import type { LinkedToolCall } from '../type/content-block'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { ScrollArea } from '@/gen/shadcn/ui/scroll-area'
import { Badge } from '@/gen/shadcn/ui/badge'

interface ToolCallPanelProps {
  toolCalls: Array<LinkedToolCall>
  highlightedMessageId?: string
}

export const ToolCallPanel = ({
  toolCalls,
  highlightedMessageId,
}: ToolCallPanelProps) => {
  // Group tool calls by whether they belong to highlighted message
  const highlightedTools = useMemo(() => {
    if (!highlightedMessageId) return new Set<string>()
    return new Set(
      toolCalls
        .filter((tc) => tc.sourceMessageUuid === highlightedMessageId)
        .map((tc) => tc.toolUse.id),
    )
  }, [toolCalls, highlightedMessageId])

  // Count successful vs error tools
  const successCount = toolCalls.filter(
    (tc) => tc.toolResult && !tc.toolResult.is_error,
  ).length
  const errorCount = toolCalls.filter((tc) => tc.toolResult?.is_error).length

  return (
    <Card className="group relative flex h-[calc(100vh-240px)] min-h-[500px] flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient glow */}
      <div className="pointer-events-none absolute -right-16 -top-16 h-32 w-32 rounded-full bg-amber-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-amber-500/10" />

      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-2">
              <Wrench className="h-4 w-4 text-amber-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Tool Calls
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Execution history
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {errorCount > 0 && (
              <Badge
                variant="outline"
                className="border-red-500/30 bg-red-500/10 font-mono text-[10px] text-red-400"
              >
                {errorCount} errors
              </Badge>
            )}
            <Badge
              variant="outline"
              className="border-zinc-700 bg-zinc-800/50 font-mono text-xs text-zinc-400"
            >
              {toolCalls.length} calls
            </Badge>
          </div>
        </div>

        {/* Stats bar */}
        {toolCalls.length > 0 && (
          <div className="mt-3 flex items-center gap-2">
            <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-800">
              <div
                className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-emerald-400"
                style={{
                  width: `${(successCount / toolCalls.length) * 100}%`,
                }}
              />
            </div>
            <span className="font-mono text-[10px] text-zinc-500">
              {Math.round((successCount / toolCalls.length) * 100)}% success
            </span>
          </div>
        )}
      </CardHeader>

      <CardContent className="flex-1 overflow-hidden p-0">
        {toolCalls.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center py-12 text-zinc-500">
            <Wrench className="mb-3 h-10 w-10 opacity-30" />
            <p className="font-mono text-sm">No tool calls</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              Tools will appear when used
            </p>
          </div>
        ) : (
          <ScrollArea className="h-full">
            <div className="space-y-2 p-4">
              {toolCalls.map((toolCall) => (
                <ToolCallItem
                  key={toolCall.toolUse.id}
                  toolCall={toolCall}
                  isHighlighted={highlightedTools.has(toolCall.toolUse.id)}
                />
              ))}
            </div>
          </ScrollArea>
        )}
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-amber-500 to-orange-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
