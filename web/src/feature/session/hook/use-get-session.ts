import { useInfiniteQuery } from '@connectrpc/connect-query'
import { getSession } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetSessionOptions defines the input parameters for the hook.
interface UseGetSessionOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // sessionId is the session's unique identifier (required for API call)
  sessionId: string
  // pageSize is the number of transcripts per page
  pageSize?: number
}

// useGetSession provides an infinite query hook for fetching session details with pagination.
// Returns a TanStack Query infinite query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetSession = ({
  organizationId,
  sessionId,
  pageSize = 50,
}: UseGetSessionOptions) => {
  return useInfiniteQuery(
    getSession,
    {
      organizationId: organizationId || '',
      sessionId,
      pagination: { page: 1, pageSize },
    },
    {
      enabled: !!organizationId,
      pageParamKey: 'pagination',
      getNextPageParam: (lastPage) => {
        const pagination = lastPage.transcriptPagination
        if (!pagination || pagination.currentPage >= pagination.totalPages) {
          return undefined
        }
        return { page: pagination.currentPage + 1, pageSize }
      },
    },
  )
}
