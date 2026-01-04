import { useAuthStore } from '@/shared/store/auth-store'

// useAuth provides authentication state and management functions
// Returns authentication status and token management utilities from auth store
export const useAuth = () => {
  // Implementation outline:
  // 1. Call useAuthStore() to get the entire store object
  const store = useAuthStore()
  // 2. Destructure: { isAuthenticated, login, logout } from store
  const { isAuthenticated, login, logout } = store
  // 3. Return object with:
  //    - isAuthenticated (from store)
  //    - logout (from store)
  //    - storeTokens (alias for store.login - kept for backward compatibility)
  return {
    isAuthenticated,
    logout,
    storeTokens: login,
  }
}
