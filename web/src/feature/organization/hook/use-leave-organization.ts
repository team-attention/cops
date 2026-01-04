import { useMutation } from '@connectrpc/connect-query'
import { leaveOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useLeaveOrganization provides a mutation hook for leaving an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useLeaveOrganization = () => {
  return useMutation(leaveOrganization, { transport })
}
