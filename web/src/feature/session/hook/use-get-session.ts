import { useQuery } from '@connectrpc/connect-query'
import { getSession } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetSessionOptions defines the input parameters for the hook.
interface UseGetSessionOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // sessionId is the session's unique identifier (required for API call)
  sessionId: string
}

// useGetSession provides a query hook for fetching a single session's details.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetSession = ({
  organizationId,
  sessionId,
}: UseGetSessionOptions) => {
  return useQuery(
    getSession,
    {
      organizationId: organizationId || '',
      sessionId,
    },
    { enabled: !!organizationId },
  )
}
