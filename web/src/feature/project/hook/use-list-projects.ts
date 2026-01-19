import { useQuery } from '@connectrpc/connect-query'
import { listProjects } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseListProjectsOptions defines the input parameters for the hook.
interface UseListProjectsOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // page is the pagination page number (defaults to 1)
  page?: number
  // pageSize is the number of items per page (defaults to 20)
  pageSize?: number
  // enabled controls whether the query should run (in addition to organizationId check)
  enabled?: boolean
}

// useListProjects provides a query hook for fetching projects list.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty or enabled is false.
export const useListProjects = ({
  organizationId,
  page = 1,
  pageSize = 20,
  enabled = true,
}: UseListProjectsOptions) => {
  return useQuery(
    listProjects,
    {
      organizationId: organizationId || '',
      pagination: { page, pageSize },
    },
    { enabled: enabled && !!organizationId },
  )
}
