import { useQuery } from '@connectrpc/connect-query'
import { getProject } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

export const useGetProject = (projectId: string) => {
  return useQuery(getProject, { projectId })
}
