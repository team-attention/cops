import { useState, useCallback } from 'react';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// Constants for token storage
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// useAuth provides authentication state and management functions.
// Returns authentication status and token management utilities.
// Uses React state to ensure components re-render on auth state changes.
export const useAuth = () => {
  // Initialize isAuthenticated state using useState with lazy initializer function
  // This runs only once on initial mount
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => {
    // Read access token from localStorage
    const token = localStorage.getItem(ACCESS_TOKEN_KEY);
    // Return true if token exists and has length > 0, false otherwise
    return token !== null && token.length > 0;
  });

  // Define logout function using useCallback
  // Empty dependency array since it uses no external values
  const logout = useCallback(() => {
    // Remove ACCESS_TOKEN_KEY from localStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    // Remove REFRESH_TOKEN_KEY from localStorage
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    // Remove TOKEN_EXPIRES_AT_KEY from localStorage
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
    // Call setIsAuthenticated(false) to trigger re-render
    setIsAuthenticated(false);
  }, []);

  // Define storeTokens function using useCallback
  // Empty dependency array since it uses no external values
  const storeTokens = useCallback((tokens: TokenPair) => {
    // Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    // Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    // Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
    // Call setIsAuthenticated(true) to trigger re-render
    setIsAuthenticated(true);
  }, []);

  return {
    isAuthenticated,
    logout,
    storeTokens,
  };
};
