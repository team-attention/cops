import { queryOptions } from '@tanstack/react-query'
import { callUnaryMethod } from '@connectrpc/connect-query'
import { getMe } from '@/gen/grpcstub/user/v1/user-UserService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// getMeQueryOptions returns a queryOptions configuration for fetching the current user.
// This can be used with useQuery, ensureQueryData, or prefetchQuery.
//
// Note: We use callUnaryMethod instead of the connect-query useQuery pattern because
// queryOptions needs a plain queryFn that returns a Promise, not a hook. The generated
// getMe method is designed for use with connect-query's useQuery hook, so we need
// callUnaryMethod for standalone query execution in route loaders.
export const getMeQueryOptions = () => {
  return queryOptions({
    queryKey: ['user', 'getMe'] as const,
    queryFn: async () => {
      return callUnaryMethod(transport, getMe, {}, undefined)
    },
    staleTime: 5 * 60 * 1000, // 5 minutes to match existing query client defaults
  })
}
