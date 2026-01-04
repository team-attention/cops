import { useQuery } from '@connectrpc/connect-query'
import { getOrganizationMembers } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

interface UseGetOrganizationMembersInput {
  organizationId: string
}

// useGetOrganizationMembers provides a query hook for fetching organization members.
// Returns a TanStack Query object with data, isLoading, error states.
export const useGetOrganizationMembers = (input: UseGetOrganizationMembersInput) => {
  return useQuery(getOrganizationMembers, input, { transport })
}
