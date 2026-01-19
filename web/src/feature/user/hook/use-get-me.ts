import { useQuery } from '@tanstack/react-query'
import { getMeQueryOptions } from '@/feature/user/hook/get-me-query-options'

// useGetMe fetches the current authenticated user's data.
// Uses shared queryOptions for consistency with route loaders.
export const useGetMe = (options?: { enabled?: boolean }) => {
  return useQuery({
    ...getMeQueryOptions(),
    ...options,
  })
}
