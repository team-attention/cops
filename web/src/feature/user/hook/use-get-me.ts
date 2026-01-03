import { useQuery } from '@connectrpc/connect-query'
import { getMe } from '@/gen/grpcstub/user/v1/user-UserService_connectquery'
import { transport } from '@/shared/service/connect-transport'

export const useGetMe = (options?: { enabled?: boolean }) => {
  return useQuery(getMe, {}, { transport, ...options })
}
