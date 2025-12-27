import { useQuery } from '@connectrpc/connect-query'
import { listProjects } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListProjectsOptions {
  page?: number
  pageSize?: number
  enabled?: boolean
}

export const useListProjects = ({
  page = 1,
  pageSize = 20,
  enabled = true,
}: UseListProjectsOptions = {}) => {
  return useQuery(
    listProjects,
    {
      pagination: { page, pageSize },
    },
    { enabled }
  )
}
