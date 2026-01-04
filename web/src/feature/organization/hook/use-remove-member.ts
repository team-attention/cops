import { useMutation } from '@connectrpc/connect-query'
import { removeMember } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useRemoveMember provides a mutation hook for removing a member from an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useRemoveMember = () => {
  return useMutation(removeMember, { transport })
}
