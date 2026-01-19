import { useMutation } from '@connectrpc/connect-query'
import { deleteAccount } from '@/gen/grpcstub/user/v1/user-UserService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useDeleteAccount provides a mutation hook for deleting the current user's account.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useDeleteAccount = () => {
  return useMutation(deleteAccount, { transport })
}
