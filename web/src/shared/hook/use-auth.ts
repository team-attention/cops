import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// Constants for token storage
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// useAuth provides authentication state and management functions.
// Returns authentication status and token management utilities.
export const useAuth = () => {
  // Read access token from localStorage
  const token = localStorage.getItem(ACCESS_TOKEN_KEY);

  // Determine isAuthenticated by checking if token exists and has length > 0
  const isAuthenticated = token !== null && token.length > 0;

  // Define logout function
  const logout = () => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
  };

  // Define storeTokens function
  const storeTokens = (tokens: TokenPair) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
  };

  return {
    isAuthenticated,
    logout,
    storeTokens,
  };
};
