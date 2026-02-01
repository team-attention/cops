import { useQuery } from '@connectrpc/connect-query'
import { getMessage } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetMessageOptions defines the input parameters for the hook.
interface UseGetMessageOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // sessionId is the session's unique identifier (required for API call)
  sessionId: string
  // messageId is the message's UUID (required for API call)
  messageId: string | null
}

// useGetMessage provides a query hook for fetching a single message by UUID.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId or messageId is null/empty.
export const useGetMessage = ({
  organizationId,
  sessionId,
  messageId,
}: UseGetMessageOptions) => {
  return useQuery(
    getMessage,
    {
      organizationId: organizationId || '',
      sessionId,
      messageId: messageId || '',
    },
    {
      enabled: !!organizationId && !!messageId,
    },
  )
}
