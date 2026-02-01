import { useQuery } from '@connectrpc/connect-query'
import { getSessionSegments } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetSessionSegmentsOptions defines the input parameters for the hook.
interface UseGetSessionSegmentsOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // sessionId is the session's unique identifier (required for API call)
  sessionId: string
}

// useGetSessionSegments provides a query hook for fetching session segment summaries.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetSessionSegments = ({
  organizationId,
  sessionId,
}: UseGetSessionSegmentsOptions) => {
  return useQuery(
    getSessionSegments,
    { organizationId: organizationId || '', sessionId },
    { enabled: !!organizationId },
  )
}
