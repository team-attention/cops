import { create } from 'zustand'
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb'

// Token storage key constants
const ACCESS_TOKEN_KEY = 'cops_access_token'
const REFRESH_TOKEN_KEY = 'cops_refresh_token'
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at'

// AuthStoreState defines the authentication state shape
interface AuthStoreState {
  // isAuthenticated indicates if user has valid access token
  isAuthenticated: boolean
}

// AuthStoreActions defines available authentication actions
interface AuthStoreActions {
  // login stores tokens in localStorage and updates isAuthenticated to true
  login: (tokens: TokenPair) => void
  // logout removes tokens from localStorage and updates isAuthenticated to false
  logout: () => void
  // updateTokens updates existing tokens without changing isAuthenticated state (used during refresh)
  updateTokens: (tokens: TokenPair) => void
}

type AuthStore = AuthStoreState & AuthStoreActions

// checkInitialAuth checks localStorage for existing tokens on app initialization
const checkInitialAuth = (): boolean => {
  // 1. Read access token from localStorage using ACCESS_TOKEN_KEY
  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  // 2. Return true if token exists and token.length > 0, false otherwise
  return token !== null && token.length > 0
}

export const useAuthStore = create<AuthStore>()((set) => ({
  // Initialize isAuthenticated by checking localStorage
  isAuthenticated: checkInitialAuth(),

  login: (tokens) => {
    // Implementation outline:
    // 1. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken)
    // 2. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken)
    // 3. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString())
    // 4. Call set({ isAuthenticated: true }) to update state
    set({ isAuthenticated: true })
  },

  logout: () => {
    // Implementation outline:
    // 1. Remove ACCESS_TOKEN_KEY from localStorage
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    // 2. Remove REFRESH_TOKEN_KEY from localStorage
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    // 3. Remove TOKEN_EXPIRES_AT_KEY from localStorage
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)
    // 4. Call set({ isAuthenticated: false }) to update state
    set({ isAuthenticated: false })
  },

  updateTokens: (tokens) => {
    // Implementation outline:
    // 1. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken)
    // 2. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken)
    // 3. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString())
    // Note: Do NOT update isAuthenticated state - this is for silent token refresh
  },
}))

// getAccessToken is a utility function to retrieve access token from localStorage
export const getAccessToken = (): string | null => {
  // 1. Read and return access token from localStorage using ACCESS_TOKEN_KEY
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

// getRefreshToken is a utility function to retrieve refresh token from localStorage
export const getRefreshToken = (): string | null => {
  // 1. Read and return refresh token from localStorage using REFRESH_TOKEN_KEY
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}
