# Review Result

**Status**: Changes Required

## Request Summary

User reported a critical bug where login fails with error "[unknown] No refresh token available". The console shows:
```
POST http://localhost:8080/auth.v1.AuthService/GoogleAuth 401 (Unauthorized)
```

The authentication interceptor is incorrectly intercepting authentication endpoint responses (like `GoogleAuth`) and attempting token refresh when there is no refresh token available (because the user hasn't logged in yet).

## Acceptance Criteria

- [ ] Auth interceptor skips token refresh for authentication endpoints (`GoogleAuth`, `RefreshToken`, `DeviceCode`, `DevicePoll`, `DeviceCodeApprove`)
- [ ] Auth interceptor checks if refresh token exists before attempting refresh
- [ ] If no refresh token is available, propagate the original 401 error instead of throwing "No refresh token available"
- [ ] Login flow completes successfully without interceptor interference

## Scope

### In Scope
- Fix the auth interceptor to not attempt token refresh for auth endpoints
- Add refresh token existence check before attempting refresh
- Proper error propagation when refresh is not possible

### Out of Scope
- Other authentication features not related to this bug
- Backend changes
- Token refresh success scenarios (working correctly)

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` | 126-136 | Architecture | Interceptor attempts token refresh on ALL 401 errors including auth endpoints that don't require authentication | Add check to skip token refresh for auth endpoints (`auth.v1.AuthService/*`) OR check if refresh token exists before attempting refresh and propagate original error if not |
| `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` | 130-131 | Error Handling | When `refreshTokenWithDeduplication()` fails due to missing refresh token, the error "No refresh token available" obscures the original 401 error from the server | Check if refresh token exists BEFORE calling `refreshTokenWithDeduplication()`. If no refresh token, throw the original error instead of attempting refresh |

## Root Cause Analysis

### The Bug Flow

1. User navigates to `/auth` page and clicks "Sign in with Google"
2. User completes Google OAuth flow and is redirected to `/auth/callback?code=...`
3. Callback page calls `useGoogleAuth().mutateAsync()` which calls `GoogleAuth` RPC
4. `GoogleAuth` RPC uses the shared `transport` from `connect-transport.ts`
5. **BUG**: If the backend returns 401 (e.g., invalid/expired auth code), the interceptor catches this error
6. Interceptor checks `error.code === Code.Unauthenticated` - TRUE
7. Interceptor calls `refreshTokenWithDeduplication()`
8. `performTokenRefresh()` calls `getRefreshToken()` which returns `null` (user not logged in yet)
9. `performTokenRefresh()` throws `Error('No refresh token available')`
10. `refreshTokenWithDeduplication()` catch block calls `useAuthStore.getState().logout()`
11. Original 401 error is lost; user sees "[unknown] No refresh token available"
12. Login flow is broken

### Why This Happens

The auth interceptor was designed to handle token refresh for authenticated API calls, but it doesn't distinguish between:
- **Authenticated endpoints** (require valid JWT, should refresh on 401)
- **Authentication endpoints** (don't require JWT, 401 means auth failed, should NOT refresh)

The interceptor intercepts ALL requests including:
- `auth.v1.AuthService/GoogleAuth` - Login endpoint, no token required
- `auth.v1.AuthService/RefreshToken` - Token refresh endpoint
- `auth.v1.AuthService/DeviceCode` - Device auth flow
- `auth.v1.AuthService/DevicePoll` - Device auth polling
- `auth.v1.AuthService/DeviceCodeApprove` - Device auth approval

### Code Location

```typescript
// /Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts
// Lines 123-140
try {
  const response = await next(req)
  return response
} catch (error) {
  if (
    error instanceof ConnectError &&
    error.code === Code.Unauthenticated
  ) {
    // BUG: This runs for ALL 401 errors, including auth endpoints
    const newTokens = await refreshTokenWithDeduplication()  // <-- Fails when no refresh token
    req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
    return await next(req)
  }
  throw error
}
```

## Recommended Solution

### Option 1: Skip Auth Endpoints (Recommended)

Add a check to skip token refresh for authentication service endpoints:

```typescript
// Define auth endpoints that should NOT trigger token refresh
const AUTH_SERVICE_PATH = '/auth.v1.AuthService/'

const createAuthInterceptor = (): Interceptor => {
  return (next) => async (req) => {
    const token = getAccessToken()
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`)
    }

    try {
      const response = await next(req)
      return response
    } catch (error) {
      if (
        error instanceof ConnectError &&
        error.code === Code.Unauthenticated
      ) {
        // Skip token refresh for auth endpoints - they don't require authentication
        const requestPath = req.url  // or req.service.typeName + '/' + req.method.name
        if (requestPath.includes(AUTH_SERVICE_PATH)) {
          throw error  // Propagate original error for auth endpoints
        }

        // Only attempt refresh for non-auth endpoints
        const newTokens = await refreshTokenWithDeduplication()
        req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        return await next(req)
      }
      throw error
    }
  }
}
```

### Option 2: Check Refresh Token First

Add a refresh token existence check before attempting refresh:

```typescript
const createAuthInterceptor = (): Interceptor => {
  return (next) => async (req) => {
    const token = getAccessToken()
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`)
    }

    try {
      const response = await next(req)
      return response
    } catch (error) {
      if (
        error instanceof ConnectError &&
        error.code === Code.Unauthenticated
      ) {
        // Check if refresh token exists before attempting refresh
        const refreshToken = getRefreshToken()
        if (!refreshToken || refreshToken.length === 0) {
          // No refresh token available - propagate original error
          // This happens during login flow or when tokens have been cleared
          throw error
        }

        // Refresh token exists, attempt refresh
        const newTokens = await refreshTokenWithDeduplication()
        req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        return await next(req)
      }
      throw error
    }
  }
}
```

### Recommended: Combine Both Options

For maximum robustness, combine both checks:

```typescript
// Define auth endpoints that should NOT trigger token refresh
const AUTH_SERVICE_PATH = '/auth.v1.AuthService/'

const createAuthInterceptor = (): Interceptor => {
  return (next) => async (req) => {
    const token = getAccessToken()
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`)
    }

    try {
      const response = await next(req)
      return response
    } catch (error) {
      if (
        error instanceof ConnectError &&
        error.code === Code.Unauthenticated
      ) {
        // Check 1: Skip token refresh for auth endpoints
        const requestUrl = req.url
        if (requestUrl.includes(AUTH_SERVICE_PATH)) {
          throw error
        }

        // Check 2: Skip if no refresh token available
        const refreshToken = getRefreshToken()
        if (!refreshToken || refreshToken.length === 0) {
          throw error
        }

        // Both checks passed, attempt refresh
        const newTokens = await refreshTokenWithDeduplication()
        req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        return await next(req)
      }
      throw error
    }
  }
}
```

## Additional Context

- Requirements document: `.agent/artifacts/20260104-155516/01_clarify.md`
- Initial plan document: `.agent/artifacts/20260104-155516/02_plan.md`
- Previous review (iteration 1): `.agent/artifacts/20260104-155516/04_user_review_iteration1.md`
- Previous plan (iteration 1): `.agent/artifacts/20260104-155516/05_user_plan_iteration1.md`
- This bug was introduced as part of the token refresh feature implementation
- The user feedback indicates login is completely broken

## Files Reviewed

- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` - Auth interceptor with token refresh logic
- `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts` - Auth store with token management
- `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-google-auth.ts` - Google auth mutation hook
- `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx` - OAuth callback page

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](/Users/jayce/team-attention/cops/.agent/rules/common.md) - Code quality and architecture principles
- [`.agent/rules/workflow.md`](/Users/jayce/team-attention/cops/.agent/rules/workflow.md) - Workflow rules
- [`.agent/rules/react/react-web.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md) - React and TypeScript best practices
- [`.agent/rules/react/react-web-src.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md) - Service layer and state management patterns
