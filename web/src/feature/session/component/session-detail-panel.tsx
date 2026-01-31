import { useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { ScrollArea } from '@/gen/shadcn/ui/scroll-area'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Button } from '@/gen/shadcn/ui/button'
import { Loader2, MessageSquare, X } from 'lucide-react'
import type { Session } from '@/gen/grpcstub/session/v1/session_pb'
import type { TimelineSegment } from '../type/graph'
import type { LinkedToolCall, ParsedMessage } from '../type/content-block'
import { MessageBubble } from './message-bubble'
import {
  enrichToolResultMessages,
  extractToolCalls,
  filterSessionsForChat,
  parseMessageContent,
} from '../util/parse-content'

interface SessionDetailPanelProps {
  segment: TimelineSegment
  sessions: Session[]
  isLoading?: boolean
  onClose: () => void
  onToolClick?: (toolUseId: string) => void
}

export const SessionDetailPanel = ({
  segment,
  sessions,
  isLoading = false,
  onClose,
  onToolClick,
}: SessionDetailPanelProps) => {
  // Filter sessions for this specific segment (by agentId)
  const segmentSessions = useMemo(() => {
    return sessions.filter((session) => {
      // Get agentId from session metadata
      let agentId: string | undefined
      switch (session.data.case) {
        case 'humanData':
          agentId = session.data.value.metadata?.agentId
          break
        case 'agentData':
          agentId = session.data.value.metadata?.agentId
          break
        case 'toolExecutionData':
          agentId = session.data.value.metadata?.agentId
          break
        case 'systemData':
          agentId = session.data.value.metadata?.agentId
          break
        case 'progressData':
          agentId = session.data.value.metadata?.agentId
          break
      }
      // Main segment has no agentId, SubAgents have agentId set
      const sessionAgentId = agentId || 'main'
      return sessionAgentId === segment.id
    })
  }, [sessions, segment.id])

  // Extract tool calls from segment sessions
  const toolCalls: LinkedToolCall[] = useMemo(() => {
    return extractToolCalls(segmentSessions)
  }, [segmentSessions])

  // Parse and enrich messages
  const parsedMessages: ParsedMessage[] = useMemo(() => {
    const filtered = filterSessionsForChat(segmentSessions)
    const parsed = filtered.map(parseMessageContent)
    return enrichToolResultMessages(parsed, toolCalls)
  }, [segmentSessions, toolCalls])

  // Use segment's message count from API
  const messageCount = segment.messageCount

  // Color based on segment type
  const isMain = segment.id === 'main'

  return (
    <Card className="flex h-[600px] w-[400px] flex-shrink-0 flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-3 pt-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div
              className={`rounded-lg border p-2 ${
                isMain
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
              className={`font-mono text-xs ${
                isMain
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
          <ScrollArea className="h-full">
            <div className="space-y-3 p-3">
              {parsedMessages.map((message) => (
                <MessageBubble
                  key={message.uuid}
                  message={message}
                  isSelected={false}
                  onSelect={() => {}}
                  onToolClick={onToolClick}
                />
              ))}
            </div>
          </ScrollArea>
        )}
      </CardContent>

      {/* Bottom accent */}
      <div
        className={`h-[2px] w-full bg-gradient-to-r ${
          isMain
            ? 'from-violet-500 to-violet-600'
            : 'from-cyan-500 to-cyan-600'
        }`}
      />
    </Card>
  )
}
