import { useEffect, useMemo, useRef, useState } from 'react'
import { createFileRoute, redirect } from '@tanstack/react-router'
import { Loader2, MessageSquare, RefreshCw } from 'lucide-react'
import { useGetSession } from '@/feature/session/hook/use-get-session'
import { SessionHeader } from '@/feature/session/component/session-header'
import { ChatView } from '@/feature/session/component/chat-view'
import { SessionGraphView } from '@/feature/session/component/session-graph-view'
import { ViewToggle } from '@/feature/session/component/view-toggle'
import { MessageDetailSheet } from '@/feature/session/component/message-detail-sheet'
import {
  enrichToolResultMessages,
  extractToolCalls,
  filterSessionsForChat,
  parseMessageContent,
} from '@/feature/session/util/parse-content'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useUserStore } from '@/shared/store/user-store'
import { useAuthStore } from '@/shared/store/auth-store'
import { APP_VERSION } from '@/shared/config/version'
import type { ViewMode } from '@/feature/session/type/graph'

export const Route = createFileRoute('/sessions/$sessionId')({
  beforeLoad: ({ location }) => {
    // Check authentication status
    const { isAuthenticated } = useAuthStore.getState()
    if (!isAuthenticated) {
      throw redirect({ to: '/auth', search: { redirect: location.href } })
    }
  },
  component: SessionDetailPage,
})

const LoadingSkeleton = () => (
  <div className="space-y-6">
    {/* Breadcrumb skeleton */}
    <Skeleton className="h-4 w-64 bg-zinc-800/50" />

    {/* Header skeleton */}
    <Skeleton className="h-44 bg-zinc-800/50" />

    {/* Content skeleton - Full width */}
    <Skeleton className="h-[500px] bg-zinc-800/50" />
  </div>
)

function SessionDetailPage() {
  const { sessionId } = Route.useParams()
  const { selectedOrganizationId } = useUserStore()
  const [selectedMessageId, setSelectedMessageId] = useState<string>()
  const [viewMode, setViewMode] = useState<ViewMode>('chat')
  const loadMoreRef = useRef<HTMLDivElement>(null)

  const {
    data,
    isLoading,
    isError,
    refetch,
    isFetching,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useGetSession({
    organizationId: selectedOrganizationId,
    sessionId,
  })

  // Merge all pages' session data
  const session = useMemo(() => {
    if (!data?.pages?.length) return null
    const firstPage = data.pages[0]
    if (!firstPage.session) return null
    const allSessions = data.pages.flatMap(
      (page) => page.session?.sessions ?? [],
    )
    return {
      ...firstPage.session,
      sessions: allSessions,
    }
  }, [data])

  // Extract total count from pagination metadata
  const totalMessageCount = useMemo(() => {
    const firstPage = data?.pages?.[0]
    if (!firstPage?.transcriptPagination?.totalCount) return undefined
    return Number(firstPage.transcriptPagination.totalCount)
  }, [data])

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    if (!loadMoreRef.current || !hasNextPage) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 0.1 },
    )

    observer.observe(loadMoreRef.current)
    return () => observer.disconnect()
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  // Extract tool calls from session entries
  const toolCalls = useMemo(() => {
    if (!session?.sessions) return []
    return extractToolCalls(session.sessions)
  }, [session?.sessions])

  // Calculate parsed messages at page level for Sheet usage
  const parsedMessages = useMemo(() => {
    if (!session?.sessions) return []
    const filtered = filterSessionsForChat(session.sessions)
    const parsed = filtered.map(parseMessageContent)
    return enrichToolResultMessages(parsed, toolCalls)
  }, [session?.sessions, toolCalls])

  // Sheet state
  const isSheetOpen = selectedMessageId !== undefined

  const selectedMessage = useMemo(() => {
    if (!selectedMessageId) return null
    return parsedMessages.find((m) => m.uuid === selectedMessageId) ?? null
  }, [parsedMessages, selectedMessageId])

  const selectedMessageToolCalls = useMemo(() => {
    if (!selectedMessageId) return []
    return toolCalls.filter((tc) => tc.sourceMessageUuid === selectedMessageId)
  }, [toolCalls, selectedMessageId])

  const handleSheetClose = () => setSelectedMessageId(undefined)

  // Handle tool click from chat - open Sheet for that message
  const handleToolClick = (toolUseId: string) => {
    // Find the message that contains this tool
    const toolCall = toolCalls.find((tc) => tc.toolUse.id === toolUseId)
    if (toolCall) {
      setSelectedMessageId(toolCall.sourceMessageUuid)
    }
  }

  return (
    <div className="relative">
      <div className="mx-auto w-full max-w-[1800px] px-4 py-6 sm:px-6 lg:px-8 xl:px-10">
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
            <SessionHeader session={session} totalMessageCount={totalMessageCount} />

            {/* View Toggle */}
            <div className="flex justify-end">
              <ViewToggle value={viewMode} onChange={setViewMode} />
            </div>

            {/* Conditional View Rendering */}
            {viewMode === 'chat' ? (
              <>
                {/* Full-width Conversation List */}
                <ChatView
                  sessions={session.sessions ?? []}
                  toolCalls={toolCalls}
                  parsedMessages={parsedMessages}
                  selectedMessageId={selectedMessageId}
                  onSelectMessage={setSelectedMessageId}
                  onToolClick={handleToolClick}
                />

                {/* Load more trigger for infinite scroll */}
                <div ref={loadMoreRef} className="flex h-10 items-center justify-center">
                  {isFetchingNextPage && (
                    <div className="flex items-center gap-2 text-zinc-500">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span className="font-mono text-xs">Loading more...</span>
                    </div>
                  )}
                </div>
              </>
            ) : (
              <SessionGraphView sessions={session.sessions ?? []} />
            )}

            {/* Message Detail Sheet */}
            <MessageDetailSheet
              open={isSheetOpen}
              onOpenChange={(open) => !open && handleSheetClose()}
              message={selectedMessage}
              relatedToolCalls={selectedMessageToolCalls}
            />
          </div>
        )}

        {/* Footer */}
        <div className="mt-12 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">
            C-Ops v{APP_VERSION}
          </span>
          <div className="h-px flex-1 bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </div>
    </div>
  )
}
