import { useMutation } from '@connectrpc/connect-query'
import { createInvitation } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useCreateInvitation provides a mutation hook for creating a member invitation.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useCreateInvitation = () => {
  return useMutation(createInvitation, { transport })
}
