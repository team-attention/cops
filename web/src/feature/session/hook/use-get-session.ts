import { useQuery } from '@connectrpc/connect-query'
import { getSession } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

export const useGetSession = (sessionId: string) => {
  return useQuery(getSession, { sessionId })
}
