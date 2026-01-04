import { useMutation } from '@connectrpc/connect-query'
import { deviceCodeApprove } from '@/gen/grpcstub/auth/v1/auth-AuthPrivateService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useApproveDevice provides a mutation hook for approving device codes.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useApproveDevice = () => {
  return useMutation(deviceCodeApprove, { transport })
}
