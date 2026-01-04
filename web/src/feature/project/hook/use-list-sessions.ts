import { useQuery } from '@connectrpc/connect-query'
import { listSessions } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListSessionsOptions {
  projectId?: string
  page?: number
  pageSize?: number
  sortBy?: string
  sortDesc?: boolean
  enabled?: boolean
}

export const useListSessions = ({
  projectId,
  page = 1,
  pageSize = 20,
  sortBy = 'started_at',
  sortDesc = true,
  enabled = true,
}: UseListSessionsOptions = {}) => {
  return useQuery(
    listSessions,
    {
      projectId: projectId || '',
      pagination: { page, pageSize },
      sortBy,
      sortDesc,
    },
    { enabled },
  )
}
