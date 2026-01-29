import { useMemo } from 'react'
import { X } from 'lucide-react'
import { ScrollArea } from '@/gen/shadcn/ui/scroll-area'
import { MessageBubble } from './message-bubble'
import {
  enrichToolResultMessages,
  extractToolCalls,
  filterSessionsForChat,
  parseMessageContent,
} from '../util/parse-content'
import type { Session } from '@/gen/grpcstub/session/v1/session_pb'

interface MessagePopoverProps {
  // Sessions to display in the popover
  sessions: Session[]
  // Node label for the header
  nodeLabel: string
  // Position relative to viewport
  position: { x: number; y: number }
  // Callback when popover is closed
  onClose: () => void
}

// MessagePopover displays a list of messages for a selected node.
// Reuses existing message parsing and rendering infrastructure.
export const MessagePopover = ({
  sessions,
  nodeLabel,
  position,
  onClose,
}: MessagePopoverProps) => {
  // Parse and enrich messages
  const parsedMessages = useMemo(() => {
    const toolCalls = extractToolCalls(sessions)
    const filtered = filterSessionsForChat(sessions)
    const parsed = filtered.map(parseMessageContent)
    return enrichToolResultMessages(parsed, toolCalls)
  }, [sessions])

  // Calculate position to keep popover within viewport
  const adjustedPosition = useMemo(() => {
    const popoverWidth = 480
    const popoverHeight = 400
    const margin = 20

    let x = position.x + 20 // Offset from click
    let y = position.y - popoverHeight / 2 // Center vertically

    // Adjust if going off right edge
    if (x + popoverWidth > window.innerWidth - margin) {
      x = position.x - popoverWidth - 20
    }

    // Adjust if going off bottom
    if (y + popoverHeight > window.innerHeight - margin) {
      y = window.innerHeight - popoverHeight - margin
    }

    // Adjust if going off top
    if (y < margin) {
      y = margin
    }

    return { x, y }
  }, [position])

  return (
    <>
      {/* Overlay to capture outside clicks */}
      <div
        className="fixed inset-0 z-40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Popover container */}
      <div
        className="fixed z-50 flex max-h-[400px] w-[480px] flex-col overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900 shadow-2xl"
        style={{
          left: adjustedPosition.x,
          top: adjustedPosition.y,
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-3">
          <span className="font-mono text-sm font-medium text-zinc-200">
            {nodeLabel}
          </span>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-300"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Content */}
        <ScrollArea className="flex-1">
          <div className="p-4">
            {parsedMessages.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-8 text-zinc-500">
                <p className="font-mono text-sm">No messages</p>
                <p className="mt-1 font-mono text-xs text-zinc-600">
                  This node has no displayable messages
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {parsedMessages.map((message) => (
                  <MessageBubble key={message.uuid} message={message} />
                ))}
              </div>
            )}
          </div>
        </ScrollArea>
      </div>
    </>
  )
}
