import { useMutation } from '@connectrpc/connect-query'
import { updateOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useUpdateOrganization provides a mutation hook for updating an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useUpdateOrganization = () => {
  return useMutation(updateOrganization, { transport })
}
