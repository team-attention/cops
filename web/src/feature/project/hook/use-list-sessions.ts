import { useQuery } from '@connectrpc/connect-query'
import { listSessions } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListSessionsOptions {
  projectId: string
  page?: number
  pageSize?: number
}

export const useListSessions = ({ projectId, page = 1, pageSize = 20 }: UseListSessionsOptions) => {
  return useQuery(listSessions, {
    projectId,
    pagination: { page, pageSize },
  })
}
