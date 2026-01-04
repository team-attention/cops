import { useMutation } from '@connectrpc/connect-query'
import { googleAuth } from '@/gen/grpcstub/auth/v1/auth-AuthService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useGoogleAuth provides mutation for exchanging Google auth code for tokens.
// Returns a TanStack Query mutation object.
export const useGoogleAuth = () => {
  return useMutation(googleAuth, { transport })
}
