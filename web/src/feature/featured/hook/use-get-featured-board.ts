import { useQuery } from '@connectrpc/connect-query'
import { getFeaturedBoard } from '@/gen/grpcstub/dashboard/v1/dashboard-FeaturedBoardService_connectquery'

// UseGetFeaturedBoardOptions defines the input parameters for the hook.
interface UseGetFeaturedBoardOptions {
  // orgSlug is the organization slug (required for API call)
  orgSlug: string
  // sinceUnix is the Unix timestamp to filter activity since
  sinceUnix: bigint
}

// useGetFeaturedBoard provides a query hook for fetching featured board data.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when orgSlug is empty.
// Polling is enabled at 30-second intervals for live updates.
export const useGetFeaturedBoard = ({
  orgSlug,
  sinceUnix,
}: UseGetFeaturedBoardOptions) => {
  return useQuery(
    getFeaturedBoard,
    { orgSlug: orgSlug || '', sinceUnix },
    {
      enabled: !!orgSlug,
      refetchInterval: 30_000,
    },
  )
}
