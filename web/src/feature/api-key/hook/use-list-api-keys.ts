import { useQuery } from '@connectrpc/connect-query'
import { listAPIKeys } from '@/gen/grpcstub/apikey/v1/apikey-APIKeyService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// UseListAPIKeysOptions defines input parameters for the hook.
interface UseListAPIKeysOptions {
  // enabled controls whether the query should run
  enabled?: boolean
}

// useListAPIKeys provides a query hook for fetching API keys list.
// Returns TanStack Query object with data, isLoading, error states.
export const useListAPIKeys = ({
  enabled = true,
}: UseListAPIKeysOptions = {}) => {
  return useQuery(listAPIKeys, {}, { transport, enabled })
}
