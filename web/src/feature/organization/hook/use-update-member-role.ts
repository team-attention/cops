import { useMutation } from '@connectrpc/connect-query'
import { updateMemberRole } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useUpdateMemberRole provides a mutation hook for changing a member's role.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useUpdateMemberRole = () => {
  return useMutation(updateMemberRole, { transport })
}
