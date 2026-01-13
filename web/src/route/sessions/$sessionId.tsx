import { useMemo, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { MessageSquare, RefreshCw } from 'lucide-react'
import { useGetSession } from '@/feature/session/hook/use-get-session'
import { SessionHeader } from '@/feature/session/component/session-header'
import { ChatView } from '@/feature/session/component/chat-view'
import { ToolCallPanel } from '@/feature/session/component/tool-call-panel'
import { extractToolCalls } from '@/feature/session/util/parse-content'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useUserStore } from '@/shared/store/user-store'

export const Route = createFileRoute('/sessions/$sessionId')({
  component: SessionDetailPage,
})

const LoadingSkeleton = () => (
  <div className="space-y-6">
    {/* Breadcrumb skeleton */}
    <Skeleton className="h-4 w-64 bg-zinc-800/50" />

    {/* Header skeleton */}
    <Skeleton className="h-44 bg-zinc-800/50" />

    {/* Content skeleton */}
    <div className="grid gap-6 lg:grid-cols-5">
      <div className="lg:col-span-3">
        <Skeleton className="h-[500px] bg-zinc-800/50" />
      </div>
      <div className="lg:col-span-2">
        <Skeleton className="h-[500px] bg-zinc-800/50" />
      </div>
    </div>
  </div>
)

function SessionDetailPage() {
  const { sessionId } = Route.useParams()
  const { selectedOrganizationId } = useUserStore()
  const [selectedMessageId, setSelectedMessageId] = useState<string>()

  const { data, isLoading, isError, refetch, isFetching } = useGetSession({
    organizationId: selectedOrganizationId,
    sessionId,
  })

  const session = data?.session

  // Extract tool calls from session transcripts
  const toolCalls = useMemo(() => {
    if (!session?.transcripts) return []
    return extractToolCalls(session.transcripts)
  }, [session?.transcripts])

  // Handle tool click from chat - scroll to and highlight the tool
  const handleToolClick = (toolUseId: string) => {
    // Find the message that contains this tool
    const toolCall = toolCalls.find((tc) => tc.toolUse.id === toolUseId)
    if (toolCall) {
      setSelectedMessageId(toolCall.sourceMessageUuid)
    }
  }

  return (
    <div className="relative">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {isLoading ? (
          <LoadingSkeleton />
        ) : isError || !session ? (
          <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
            <MessageSquare className="mb-4 h-12 w-12 opacity-50" />
            <p className="font-mono text-sm">Session not found</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              The session may have been deleted or doesn't exist
            </p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Refresh button */}
            <div className="flex justify-end">
              <button
                onClick={() => refetch()}
                disabled={isFetching}
                className="group flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-2 text-sm text-zinc-400 transition-all hover:border-zinc-700 hover:bg-zinc-800/50 hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <RefreshCw
                  className={`h-4 w-4 transition-transform ${isFetching ? 'animate-spin' : 'group-hover:rotate-90'}`}
                />
                <span className="font-mono text-xs">Refresh</span>
              </button>
            </div>

            {/* Session Header */}
            <SessionHeader session={session} />

            {/* Two-column layout: Chat View | Tool Panel */}
            <div className="grid gap-6 lg:grid-cols-5">
              <div className="lg:col-span-3">
                <ChatView
                  transcripts={session.transcripts ?? []}
                  selectedMessageId={selectedMessageId}
                  onSelectMessage={setSelectedMessageId}
                  onToolClick={handleToolClick}
                />
              </div>
              <div className="lg:col-span-2">
                <ToolCallPanel
                  toolCalls={toolCalls}
                  highlightedMessageId={selectedMessageId}
                />
              </div>
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="mt-12 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">
            C-Ops v0.1.0
          </span>
          <div className="h-px flex-1 bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </div>
    </div>
  )
}
