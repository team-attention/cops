import { useCallback, useMemo, useState } from 'react'
import { createFileRoute, redirect } from '@tanstack/react-router'
import { Loader2, MessageSquare, RefreshCw } from 'lucide-react'
import { useGetSession } from '@/feature/session/hook/use-get-session'
import { useGetSessionSegments } from '@/feature/session/hook/use-get-session-segments'
import { SessionHeader } from '@/feature/session/component/session-header'
import { SessionTimelineView } from '@/feature/session/component/session-timeline-view'
import { SessionDetailPanel } from '@/feature/session/component/session-detail-panel'
import { MessageDetailSheet } from '@/feature/session/component/message-detail-sheet'
import {
  enrichToolResultMessages,
  extractToolCalls,
  filterSessionsForChat,
  parseMessageContent,
} from '@/feature/session/util/parse-content'
import { convertApiSegmentsToTimeline } from '@/feature/session/util/graph-data'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { Card } from '@/gen/shadcn/ui/card'
import { useUserStore } from '@/shared/store/user-store'
import { useAuthStore } from '@/shared/store/auth-store'
import { APP_VERSION } from '@/shared/config/version'
import type { TimelineSegment } from '@/feature/session/type/graph'

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
  const [selectedSegment, setSelectedSegment] =
    useState<TimelineSegment | null>(null)

  // Handle segment click - open detail panel for the segment
  const handleSegmentClick = useCallback((segment: TimelineSegment) => {
    setSelectedSegment(segment)
  }, [])

  const {
    data,
    isLoading,
    isError,
    refetch,
    isFetching,
  } = useGetSession({
    organizationId: selectedOrganizationId,
    sessionId,
  })

  // Fetch lightweight segments for timeline view
  const { data: segmentsData } = useGetSessionSegments({
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

  // Convert API segments to timeline data for Graph view
  const timelineData = useMemo(() => {
    if (!segmentsData) return null
    return convertApiSegmentsToTimeline(
      segmentsData.segments,
      segmentsData.startTime,
      segmentsData.endTime,
      segmentsData.totalDurationSeconds,
    )
  }, [segmentsData])

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

            {/* Timeline View */}
            {timelineData ? (
              <div className="flex gap-6">
                <div className="min-w-0 w-1/2">
                  <SessionTimelineView
                    timelineData={timelineData}
                    onSegmentClick={handleSegmentClick}
                    selectedSegmentId={selectedSegment?.id}
                  />
                </div>
                <div className="min-w-0 w-1/2">
                  {selectedSegment ? (
                    <SessionDetailPanel
                      organizationId={selectedOrganizationId || ''}
                      sessionId={sessionId}
                      segment={selectedSegment}
                      onClose={() => setSelectedSegment(null)}
                      onToolClick={handleToolClick}
                      selectedMessageId={selectedMessageId}
                      onSelectMessage={setSelectedMessageId}
                    />
                  ) : (
                    <Card className="flex h-[600px] items-center justify-center border-zinc-800/50 bg-zinc-900/80">
                      <p className="font-mono text-sm text-zinc-500">
                        Select a segment to view messages
                      </p>
                    </Card>
                  )}
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="h-6 w-6 animate-spin text-zinc-500" />
              </div>
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
