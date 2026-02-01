import { useMemo } from 'react'
import { Loader2, MessageSquare, X } from 'lucide-react'
import {
  enrichToolResultMessages,
  extractToolCalls,
  filterSessionsForChat,
  parseMessageContent,
} from '../util/parse-content'
import { useGetSession } from '../hook/use-get-session'
import { MessageBubble } from './message-bubble'
import type { TimelineSegment } from '../type/graph'
import type { LinkedToolCall, ParsedMessage } from '../type/content-block'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Button } from '@/gen/shadcn/ui/button'

interface SessionDetailPanelProps {
  organizationId: string
  sessionId: string
  segment: TimelineSegment
  onClose: () => void
  onToolClick?: (toolUseId: string) => void
  selectedMessageId?: string
  onSelectMessage?: (messageId: string) => void
}

export const SessionDetailPanel = ({
  organizationId,
  sessionId,
  segment,
  onClose,
  onToolClick,
  selectedMessageId,
  onSelectMessage,
}: SessionDetailPanelProps) => {
  // Fetch segment-specific messages using agentId filter
  const { data, isLoading } = useGetSession({
    organizationId,
    sessionId,
    agentId: segment.id,
  })

  // Merge all pages' session data
  const sessions = useMemo(() => {
    return data?.pages.flatMap((page) => page.session?.sessions ?? []) ?? []
  }, [data])

  // Extract tool calls from segment sessions
  const toolCalls: Array<LinkedToolCall> = useMemo(() => {
    return extractToolCalls(sessions)
  }, [sessions])

  // Parse and enrich messages
  const parsedMessages: Array<ParsedMessage> = useMemo(() => {
    const filtered = filterSessionsForChat(sessions)
    const parsed = filtered.map(parseMessageContent)
    return enrichToolResultMessages(parsed, toolCalls)
  }, [sessions, toolCalls])

  // Use segment's message count from API
  const messageCount = segment.messageCount

  // Color based on segment type
  const isMain = segment.id === 'main'

  return (
    <Card className="flex h-[600px] w-full flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-3 pt-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={`rounded-lg border p-2 ${isMain
                ? 'border-violet-500/20 bg-violet-500/10'
                : 'border-cyan-500/20 bg-cyan-500/10'
                }`}
            >
              <MessageSquare
                className={`h-4 w-4 ${isMain ? 'text-violet-400' : 'text-cyan-400'}`}
              />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                {segment.label}
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Agent messages
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge
              variant="outline"
              className={`font-mono text-xs ${isMain
                ? 'border-violet-500/20 bg-violet-500/5 text-violet-400'
                : 'border-cyan-500/20 bg-cyan-500/5 text-cyan-400'
                }`}
            >
              {messageCount} msgs
            </Badge>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-zinc-500 hover:bg-zinc-800 hover:text-zinc-300"
              onClick={onClose}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="flex-1 overflow-hidden p-0">
        {isLoading ? (
          <div className="flex h-full flex-col items-center justify-center py-12 text-zinc-500">
            <Loader2 className="mb-3 h-8 w-8 animate-spin opacity-50" />
            <p className="font-mono text-sm">Loading messages...</p>
          </div>
        ) : parsedMessages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center py-12 text-zinc-500">
            <MessageSquare className="mb-3 h-10 w-10 opacity-30" />
            <p className="font-mono text-sm">No messages</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              This segment has no displayable messages
            </p>
          </div>
        ) : (
          <div className="h-full w-full overflow-y-auto">
            <div className="space-y-3 p-3">
              {parsedMessages.map((message) => (
                <MessageBubble
                  key={message.uuid}
                  message={message}
                  isSelected={selectedMessageId === message.uuid}
                  onSelect={() => onSelectMessage?.(message.uuid)}
                  onToolClick={onToolClick}
                />
              ))}
            </div>
          </div>
        )}
      </CardContent>

      {/* Bottom accent */}
      <div
        className={`h-[2px] w-full bg-gradient-to-r ${isMain
          ? 'from-violet-500 to-violet-600'
          : 'from-cyan-500 to-cyan-600'
          }`}
      />
    </Card>
  )
}
