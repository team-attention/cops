import { useQuery } from '@connectrpc/connect-query'
import { listInvitations } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

interface UseListInvitationsInput {
  organizationId: string
}

// useListInvitations provides a query hook for fetching pending invitations.
// Returns a TanStack Query object with data, isLoading, error states.
export const useListInvitations = (input: UseListInvitationsInput) => {
  return useQuery(listInvitations, input, { transport })
}
