# Implementation Plan: Web App Token Refresh Interceptor

## Overview

This implementation adds automatic token refresh logic to the web application's ConnectRPC interceptor. When any API request receives a 401 Unauthenticated error, the interceptor will automatically:

1. Detect the 401 error from the ConnectRPC response
2. Call the `RefreshToken` RPC endpoint using the stored refresh token
3. Store the new token pair in localStorage
4. Retry the original failed request with the new access token
5. Redirect to `/auth` if the refresh token is also invalid/expired

The implementation uses a queue-based deduplication pattern to handle concurrent requests that fail simultaneously - only one refresh call is made, and all pending requests share the result.

## Package Changes

No package changes required. All necessary packages are already installed:
- `@connectrpc/connect` - Provides `Interceptor`, `ConnectError`, `Code`
- `@connectrpc/connect-web` - Provides `createConnectTransport`
- `@bufbuild/protobuf` - Provides `create` function for message creation

## Implementation Steps

### Step 1: Implement Token Refresh Interceptor with Deduplication

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript rules (no `any`, named exports, named types)
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Project structure and import patterns
- `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`: Reference implementation pattern (detect 401, refresh, retry)
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Token storage key constants
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/auth/v1/auth_pb.ts`: RefreshTokenReqSchema, RefreshTokenRes types

#### `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`

**Description**:
Modify the existing file to add token refresh logic with concurrent request deduplication. The interceptor detects 401 errors, calls RefreshToken RPC, stores new tokens, and retries the original request.

```typescript
import { createConnectTransport } from '@connectrpc/connect-web';
import { createClient, ConnectError, Code } from '@connectrpc/connect';
import type { Interceptor } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { AuthService, RefreshTokenReqSchema } from '@/gen/grpcstub/auth/v1/auth_pb';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// Token storage key constants (must match use-auth.ts)
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// RefreshState represents the current state of token refresh operation
interface RefreshState {
  // promise holds the ongoing refresh promise, null if no refresh in progress
  promise: Promise<TokenPair> | null;
}

// refreshState holds the singleton state for refresh deduplication
const refreshState: RefreshState = {
  promise: null,
};

// getBaseUrl returns the API base URL from environment or default
const getBaseUrl = (): string => {
  // Implementation outline:
  // 1. Return import.meta.env.VITE_API_URL if defined.
  // 2. Otherwise return 'http://localhost:8080'.
};

// createBaseTransport creates a transport without auth interceptor for refresh calls
const createBaseTransport = () => {
  // Implementation outline:
  // 1. Create and return a ConnectTransport with baseUrl only.
  // 2. No interceptors - this avoids infinite loop during refresh.
};

// storeTokens saves the token pair to localStorage
const storeTokens = (tokens: TokenPair): void => {
  // Implementation outline:
  // 1. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY.
  // 2. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY.
  // 3. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY.
};

// clearTokens removes all tokens from localStorage
const clearTokens = (): void => {
  // Implementation outline:
  // 1. Remove ACCESS_TOKEN_KEY from localStorage.
  // 2. Remove REFRESH_TOKEN_KEY from localStorage.
  // 3. Remove TOKEN_EXPIRES_AT_KEY from localStorage.
};

// redirectToAuth navigates browser to auth page
const redirectToAuth = (): void => {
  // Implementation outline:
  // 1. Use window.location.href to redirect to '/auth'.
  // 2. This is a hard redirect, not React Router navigation.
};

// performTokenRefresh executes the actual refresh token RPC call
const performTokenRefresh = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. Get refresh token from localStorage using REFRESH_TOKEN_KEY.
  // 2. If refresh token is null or empty:
  //    a. Throw an error indicating no refresh token available.
  // 3. Create base transport without interceptors (to avoid recursion).
  // 4. Create AuthService client using createClient with base transport.
  // 5. Create RefreshTokenReq message using create(RefreshTokenReqSchema, { refreshToken }).
  // 6. Call client.refreshToken(request).
  // 7. If response.tokens is undefined:
  //    a. Throw an error indicating invalid refresh response.
  // 8. Return response.tokens.
};

// refreshTokenWithDeduplication handles token refresh with request deduplication
const refreshTokenWithDeduplication = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. If refreshState.promise is not null:
  //    a. Return the existing promise (join ongoing refresh).
  // 2. Create new refresh promise:
  //    a. Set refreshState.promise = performTokenRefresh().
  // 3. Try to await the refresh promise.
  // 4. On success:
  //    a. Store the new tokens using storeTokens().
  //    b. Return the tokens.
  // 5. On error:
  //    a. Clear all tokens using clearTokens().
  //    b. Redirect to auth page using redirectToAuth().
  //    c. Re-throw the error.
  // 6. In finally block:
  //    a. Set refreshState.promise = null (clear the state).
};

// createAuthInterceptor creates an interceptor that adds JWT and handles token refresh
const createAuthInterceptor = (): Interceptor => {
  // Implementation outline:
  // 1. Return interceptor function (next) => async (req) => { ... }.
  // 2. Get access token from localStorage using ACCESS_TOKEN_KEY.
  // 3. If token exists and has length > 0:
  //    a. Set Authorization header: req.header.set('Authorization', `Bearer ${token}`).
  // 4. Try to execute request: const response = await next(req).
  // 5. Return response on success.
  // 6. Catch error:
  //    a. Check if error is ConnectError and code is Code.Unauthenticated.
  //    b. If not a 401 error, re-throw the original error.
  //    c. If 401 error:
  //       i. Call refreshTokenWithDeduplication() to get new tokens.
  //       ii. Update request header with new access token.
  //       iii. Retry the request: return await next(req).
  //       iv. If refresh fails (caught in refreshTokenWithDeduplication), error propagates.
};

export const transport = createConnectTransport({
  baseUrl: getBaseUrl(),
  interceptors: [createAuthInterceptor()],
});
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Request succeeds without auth | No token in localStorage | Request passes through, no Authorization header | No token path |
| Request succeeds with valid token | Valid token in localStorage | Request passes with Authorization header | Happy path with token |
| 401 error, refresh succeeds | Expired access token, valid refresh token | Refresh called, new tokens stored, request retried successfully | Token refresh success path |
| 401 error, refresh fails | Expired access token, expired refresh token | Tokens cleared, redirect to /auth, error thrown | Token refresh failure path |
| Non-401 error | Valid token, server returns 500 | Original error propagated | Non-401 error path |
| Concurrent 401 errors | Multiple requests fail with 401 simultaneously | Only one refresh call made, all requests retry with same new token | Deduplication path |
| Refresh returns no tokens | Valid refresh token, server returns empty tokens | Error thrown, tokens cleared, redirect to /auth | Invalid refresh response path |
| No refresh token available | 401 error, no refresh token in localStorage | Error thrown, redirect to /auth | Missing refresh token path |

## Summary of Changes

1. **New Imports**:
   - `createClient`, `ConnectError`, `Code` from `@connectrpc/connect`
   - `create` from `@bufbuild/protobuf`
   - `AuthService`, `RefreshTokenReqSchema` from `@/gen/grpcstub/auth/v1/auth_pb`
   - `TokenPair` type from `@/gen/grpcstub/auth/v1/auth_pb`

2. **New Constants**:
   - `ACCESS_TOKEN_KEY`, `REFRESH_TOKEN_KEY`, `TOKEN_EXPIRES_AT_KEY` - Token storage keys

3. **New Types**:
   - `RefreshState` - Interface for refresh deduplication state

4. **New Module-Level State**:
   - `refreshState` - Singleton object holding current refresh promise

5. **New Functions**:
   - `getBaseUrl()` - Returns API base URL
   - `createBaseTransport()` - Creates transport without interceptors for refresh calls
   - `storeTokens()` - Saves tokens to localStorage
   - `clearTokens()` - Removes tokens from localStorage
   - `redirectToAuth()` - Hard redirect to /auth page
   - `performTokenRefresh()` - Executes the RefreshToken RPC call
   - `refreshTokenWithDeduplication()` - Handles refresh with concurrent request deduplication

6. **Modified Function**:
   - `createAuthInterceptor()` - Enhanced to detect 401 errors, trigger refresh, and retry requests

## Architecture Decisions

1. **Self-contained in interceptor**: All logic is contained within `connect-transport.ts`. No modifications to `use-auth.ts` hook are required. This avoids React hook dependency issues in non-React code.

2. **Separate base transport for refresh**: The `createBaseTransport()` function creates a transport without interceptors to avoid infinite recursion when the refresh call itself returns 401.

3. **Promise-based deduplication**: Uses a module-level `refreshState` object to hold the ongoing refresh promise. Concurrent requests that encounter 401 will await the same promise instead of making duplicate refresh calls.

4. **Hard redirect for auth failure**: Uses `window.location.href` instead of React Router navigation because the interceptor runs outside React component context. This ensures clean state reset on auth failure.

5. **Error propagation after refresh failure**: When refresh fails, the interceptor clears tokens and redirects, but also re-throws the error so the original request promise rejects properly.
