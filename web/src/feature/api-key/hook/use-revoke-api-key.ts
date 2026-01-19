import { useMutation } from '@connectrpc/connect-query'
import { revokeAPIKey } from '@/gen/grpcstub/apikey/v1/apikey-APIKeyService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useRevokeAPIKey provides a mutation hook for revoking API keys.
// Returns TanStack Query mutation object with mutate, isLoading, error states.
export const useRevokeAPIKey = () => {
  return useMutation(revokeAPIKey, { transport })
}
