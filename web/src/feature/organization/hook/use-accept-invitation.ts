import { useMutation } from '@connectrpc/connect-query'
import { acceptInvitation } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useAcceptInvitation provides a mutation hook for accepting a member invitation.
// Requires authentication - verifies email matches the invitation.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useAcceptInvitation = () => {
  return useMutation(acceptInvitation, { transport })
}
