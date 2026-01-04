import { useEffect, useMemo, useRef } from 'react'
import { MessageSquare } from 'lucide-react'
import {
  filterRecordsForChat,
  parseMessageContent,
} from '../util/parse-content'
import { MessageBubble } from './message-bubble'
import type { Record } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { ScrollArea } from '@/gen/shadcn/ui/scroll-area'
import { Badge } from '@/gen/shadcn/ui/badge'

interface ChatViewProps {
  records: Array<Record>
  selectedMessageId?: string
  onSelectMessage?: (messageId: string) => void
  onToolClick?: (toolUseId: string) => void
}

export const ChatView = ({
  records,
  selectedMessageId,
  onSelectMessage,
  onToolClick,
}: ChatViewProps) => {
  const scrollRef = useRef<HTMLDivElement>(null)

  // Filter and parse records
  const parsedMessages = useMemo(() => {
    const filtered = filterRecordsForChat(records)
    return filtered.map(parseMessageContent)
  }, [records])

  // Count non-meta messages
  const messageCount = parsedMessages.filter(
    (m) => !m.isMeta && !m.isSidechain,
  ).length

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [parsedMessages.length])

  return (
    <Card className="group relative flex h-[calc(100vh-320px)] min-h-[400px] flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient glow */}
      <div className="pointer-events-none absolute -left-16 -top-16 h-32 w-32 rounded-full bg-cyan-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-cyan-500/10" />

      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/10 p-2">
              <MessageSquare className="h-4 w-4 text-cyan-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Conversation
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Session interaction log
              </p>
            </div>
          </div>
          <Badge
            variant="outline"
            className="border-zinc-700 bg-zinc-800/50 font-mono text-xs text-zinc-400"
          >
            {messageCount} messages
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="flex-1 overflow-hidden p-0">
        {parsedMessages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center py-12 text-zinc-500">
            <MessageSquare className="mb-3 h-10 w-10 opacity-30" />
            <p className="font-mono text-sm">No messages yet</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              Messages will appear here
            </p>
          </div>
        ) : (
          <ScrollArea className="h-full" ref={scrollRef}>
            <div className="space-y-3 p-4">
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
          </ScrollArea>
        )}
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-cyan-500 to-violet-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
