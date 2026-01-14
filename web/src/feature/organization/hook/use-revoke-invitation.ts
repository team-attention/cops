import { useMutation } from '@connectrpc/connect-query'
import { revokeInvitation } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useRevokeInvitation provides a mutation hook for revoking a pending invitation.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useRevokeInvitation = () => {
  return useMutation(revokeInvitation, { transport })
}
