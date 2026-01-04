import { useQuery } from '@connectrpc/connect-query'
import { getProject } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetProjectOptions defines the input parameters for the hook.
interface UseGetProjectOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // projectId is the project's unique identifier (required for API call)
  projectId: string
}

// useGetProject provides a query hook for fetching a single project's details.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetProject = ({
  organizationId,
  projectId,
}: UseGetProjectOptions) => {
  return useQuery(
    getProject,
    {
      organizationId: organizationId || '',
      projectId,
    },
    { enabled: !!organizationId },
  )
}
