import { useQuery } from '@connectrpc/connect-query'
import { listSessions } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseListSessionsOptions defines the input parameters for the hook.
interface UseListSessionsOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // projectId filters sessions by project (optional)
  projectId?: string
  // page is the pagination page number (defaults to 1)
  page?: number
  // pageSize is the number of items per page (defaults to 20)
  pageSize?: number
  // sortBy is the field to sort by (defaults to "started_at")
  sortBy?: string
  // sortDesc indicates descending sort order (defaults to true)
  sortDesc?: boolean
  // enabled controls whether the query should run (in addition to organizationId check)
  enabled?: boolean
}

// useListSessions provides a query hook for fetching sessions list.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty or enabled is false.
export const useListSessions = ({
  organizationId,
  projectId,
  page = 1,
  pageSize = 20,
  sortBy = 'started_at',
  sortDesc = true,
  enabled = true,
}: UseListSessionsOptions) => {
  return useQuery(
    listSessions,
    {
      organizationId: organizationId || '',
      projectId: projectId || '',
      pagination: { page, pageSize },
      sortBy,
      sortDesc,
    },
    { enabled: enabled && !!organizationId },
  )
}
