# Implementation Plan: Fix Auth Interceptor Token Refresh on Login

## Overview

This plan fixes a critical bug where the authentication interceptor incorrectly attempts token refresh for authentication endpoints (like `GoogleAuth`). When a user tries to log in, the backend may return a 401 error (e.g., invalid auth code), and the interceptor catches this error and attempts to refresh tokens. However, since the user is not logged in yet, there is no refresh token available, causing the error "No refresh token available" to obscure the actual authentication failure.

The fix adds two guard conditions before attempting token refresh:
1. Skip token refresh for auth service endpoints (they don't require authentication)
2. Check if a refresh token exists before attempting refresh

## Package Changes

No package changes required.

## Step 1: Update Auth Interceptor with Guard Conditions

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Service layer patterns and code organization rules
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript and React coding standards
- `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`: Understand `getRefreshToken()` function signature

### `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`

**Description**:
Modify the `createAuthInterceptor` function to add two guard conditions before attempting token refresh on 401 errors. The interceptor will skip token refresh for auth service endpoints and when no refresh token is available.

```typescript
// AUTH_SERVICE_PREFIX is the URL path prefix for auth service endpoints that should not trigger token refresh
const AUTH_SERVICE_PREFIX = '/auth.v1.AuthService/'

// createAuthInterceptor creates an interceptor that adds JWT and handles token refresh
const createAuthInterceptor = (): Interceptor => {
  // Implementation outline:
  // 1. Return interceptor function: (next) => async (req) => { ... }
  return (next) => async (req) => {
    // 2. Inside interceptor:
    //    a. Get access token using getAccessToken() from auth-store
    const token = getAccessToken()
    //    b. If token exists and token.length > 0:
    //       i. Set Authorization header: req.header.set('Authorization', `Bearer ${token}`)
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`)
    }

    //    c. Try block:
    try {
      //       i. Await response from next(req)
      const response = await next(req)
      //       ii. Return response
      return response
    } catch (error) {
      //    d. Catch block (error):
      //       i. If error is ConnectError and error.code === Code.Unauthenticated:
      if (
        error instanceof ConnectError &&
        error.code === Code.Unauthenticated
      ) {
        //          - Guard 1: Check if request URL includes AUTH_SERVICE_PREFIX
        //            - If true: throw error (auth endpoints should not trigger refresh)
        if (req.url.includes(AUTH_SERVICE_PREFIX)) {
          throw error
        }

        //          - Guard 2: Get refresh token using getRefreshToken() from auth-store
        //            - If refreshToken is null or empty: throw error (no token to refresh with)
        const refreshToken = getRefreshToken()
        if (!refreshToken || refreshToken.length === 0) {
          throw error
        }

        //          - Both guards passed: Await newTokens from refreshTokenWithDeduplication()
        const newTokens = await refreshTokenWithDeduplication()
        //          - Set Authorization header with new token
        req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        //          - Await and return response from next(req) (retry request)
        return await next(req)
      }
      //       ii. Otherwise:
      //          - Throw error (not an auth error)
      throw error
    }
  }
}
```

**Specific Changes**:

1. **Add constant `AUTH_SERVICE_PREFIX`** at module level (after `refreshState` declaration, around line 25):
   ```typescript
   // AUTH_SERVICE_PREFIX is the URL path prefix for auth service endpoints that should not trigger token refresh
   const AUTH_SERVICE_PREFIX = '/auth.v1.AuthService/'
   ```

2. **Modify the catch block** in `createAuthInterceptor` (lines 126-136):

   **Current code** (lines 126-136):
   ```typescript
   if (
     error instanceof ConnectError &&
     error.code === Code.Unauthenticated
   ) {
     const newTokens = await refreshTokenWithDeduplication()
     req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
     return await next(req)
   }
   ```

   **New code**:
   ```typescript
   if (
     error instanceof ConnectError &&
     error.code === Code.Unauthenticated
   ) {
     // Guard 1: Skip token refresh for auth service endpoints
     // Auth endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) don't require
     // authentication and should propagate their 401 errors directly
     if (req.url.includes(AUTH_SERVICE_PREFIX)) {
       throw error
     }

     // Guard 2: Skip token refresh if no refresh token is available
     // This handles the case where tokens have been cleared or user never logged in
     const refreshToken = getRefreshToken()
     if (!refreshToken || refreshToken.length === 0) {
       throw error
     }

     // Both guards passed - attempt token refresh
     const newTokens = await refreshTokenWithDeduplication()
     req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
     return await next(req)
   }
   ```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Auth endpoint (GoogleAuth) returns 401 | `req.url = 'http://localhost:8080/auth.v1.AuthService/GoogleAuth'`, 401 error | Original 401 error thrown | Guard 1: Auth endpoint skip |
| Auth endpoint (RefreshToken) returns 401 | `req.url = 'http://localhost:8080/auth.v1.AuthService/RefreshToken'`, 401 error | Original 401 error thrown | Guard 1: Auth endpoint skip |
| Auth endpoint (DeviceCode) returns 401 | `req.url = 'http://localhost:8080/auth.v1.AuthService/DeviceCode'`, 401 error | Original 401 error thrown | Guard 1: Auth endpoint skip |
| Auth endpoint (DevicePoll) returns 401 | `req.url = 'http://localhost:8080/auth.v1.AuthService/DevicePoll'`, 401 error | Original 401 error thrown | Guard 1: Auth endpoint skip |
| Protected endpoint returns 401, no refresh token | `req.url = 'http://localhost:8080/project.v1.ProjectService/GetProject'`, 401 error, `getRefreshToken() = null` | Original 401 error thrown | Guard 2: No refresh token |
| Protected endpoint returns 401, empty refresh token | `req.url = 'http://localhost:8080/project.v1.ProjectService/GetProject'`, 401 error, `getRefreshToken() = ''` | Original 401 error thrown | Guard 2: Empty refresh token |
| Protected endpoint returns 401, valid refresh token | `req.url = 'http://localhost:8080/project.v1.ProjectService/GetProject'`, 401 error, `getRefreshToken() = 'valid-token'` | Token refresh attempted, request retried | Happy path: Token refresh |
| Protected endpoint returns 200 | `req.url = 'http://localhost:8080/project.v1.ProjectService/GetProject'`, 200 response | Response returned directly | Happy path: No error |
| Protected endpoint returns 500 | `req.url = 'http://localhost:8080/project.v1.ProjectService/GetProject'`, 500 error | 500 error thrown | Non-auth error passthrough |

## Implementation Checklist

- [ ] Add `AUTH_SERVICE_PREFIX` constant after `refreshState` declaration (line 25)
- [ ] Add Guard 1: Check if `req.url.includes(AUTH_SERVICE_PREFIX)` and throw original error if true
- [ ] Add Guard 2: Check if refresh token exists using `getRefreshToken()` and throw original error if null/empty
- [ ] Ensure existing token refresh logic only runs when both guards pass
- [ ] Verify login flow completes successfully without interceptor interference
