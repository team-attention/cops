import { useMutation } from '@connectrpc/connect-query'
import { issueAPIKey } from '@/gen/grpcstub/apikey/v1/apikey-APIKeyService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useIssueAPIKey provides a mutation hook for creating new API keys.
// Returns TanStack Query mutation object with mutate, isLoading, error states.
export const useIssueAPIKey = () => {
  return useMutation(issueAPIKey, { transport })
}
