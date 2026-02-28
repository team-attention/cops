import { useQuery } from '@connectrpc/connect-query'
import { getInvitationByToken } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

interface UseGetInvitationByTokenInput {
  token: string
}

// useGetInvitationByToken provides a query hook for fetching invitation details by token.
// This is a public endpoint (no auth required for viewing).
// Returns a TanStack Query object with data, isLoading, error states.
export const useGetInvitationByToken = (
  input: UseGetInvitationByTokenInput,
) => {
  return useQuery(getInvitationByToken, input, { transport })
}
