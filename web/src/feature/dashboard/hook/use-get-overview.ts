import { useQuery } from '@connectrpc/connect-query'
import { getOverview } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

export const useGetOverview = () => {
  return useQuery(getOverview, {})
}
