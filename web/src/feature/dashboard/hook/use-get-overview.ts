import { useQuery } from '@connectrpc/connect-query'
import { getOverview } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetOverviewOptions defines the input parameters for the hook.
interface UseGetOverviewOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
}

// useGetOverview provides a query hook for fetching dashboard overview data.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null or empty.
export const useGetOverview = ({ organizationId }: UseGetOverviewOptions) => {
  return useQuery(
    getOverview,
    { organizationId: organizationId || '' },
    { enabled: !!organizationId },
  )
}
