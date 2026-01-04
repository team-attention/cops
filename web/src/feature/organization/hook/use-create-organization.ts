import { useMutation } from '@connectrpc/connect-query'
import { createOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useCreateOrganization provides a mutation hook for creating a new organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useCreateOrganization = () => {
  return useMutation(createOrganization, { transport })
}
