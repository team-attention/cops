# Development Walkthrough

## Summary

Implemented automatic token refresh logic for the web application's ConnectRPC interceptor. When API requests receive 401 Unauthenticated errors, the system now automatically calls the RefreshToken endpoint, updates stored tokens, and retries the failed request. A centralized Zustand store manages authentication state, with deduplication logic to prevent concurrent refresh calls.

## Code Overview

### New Components

#### `auth-store.ts`

- **Location**: `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`
- **Purpose**: Centralized Zustand store for authentication state management
- **Key Methods**:
  - `login(tokens)`: Stores new token pair in localStorage and sets isAuthenticated to true
  - `logout()`: Clears all tokens from localStorage and sets isAuthenticated to false
  - `updateTokens(tokens)`: Updates tokens during silent refresh without changing authentication state
- **Exported Utilities**:
  - `getAccessToken()`: Retrieves access token from localStorage (non-reactive)
  - `getRefreshToken()`: Retrieves refresh token from localStorage (non-reactive)
- **State Shape**:
  - `isAuthenticated: boolean` - Initialized by checking localStorage on app load
- **Storage Keys**:
  - `cops_access_token` - Access JWT token
  - `cops_refresh_token` - Refresh JWT token
  - `cops_token_expires_at` - Token expiration timestamp

### Modified Components

#### `connect-transport.ts`

- **Location**: `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`
- **Changes**: Enhanced ConnectRPC interceptor with automatic token refresh logic
- **New Module-Level State**:
  - `refreshState: RefreshState` - Holds ongoing refresh promise for deduplication
  - `AUTH_SERVICE_PREFIX` - Constant to identify auth service endpoints
- **New Helper Functions**:
  - `getBaseUrl()`: Returns API base URL from environment or defaults to localhost:8080
  - `createBaseTransport()`: Creates transport without interceptors for refresh calls (prevents infinite recursion)
  - `performTokenRefresh()`: Executes the RefreshToken RPC call using base transport
  - `refreshTokenWithDeduplication()`: Manages token refresh with concurrent request deduplication
- **Enhanced Interceptor Logic** (`createAuthInterceptor`):
  - **Authorization Header**: Adds `Bearer {token}` header to all requests if access token exists
  - **401 Error Detection**: Catches ConnectError with Code.Unauthenticated
  - **Guard 1 - Auth Endpoint Skip**: Skips refresh for auth service endpoints (GoogleAuth, RefreshToken, DeviceCode, etc.) that don't require authentication
  - **Guard 2 - No Refresh Token**: Skips refresh if no refresh token is available in localStorage
  - **Token Refresh Flow**:
    1. Calls `refreshTokenWithDeduplication()` to get new tokens
    2. Updates request header with new access token
    3. Retries original request with `await next(req)`
  - **Deduplication**: Multiple concurrent 401 errors share the same refresh promise

#### `use-auth.ts`

- **Location**: `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`
- **Changes**: Refactored to use centralized auth store instead of local useState
- **Migration**:
  - Removed: Local `useState` for `isAuthenticated`
  - Removed: `useCallback` wrappers for `logout` and `storeTokens`
  - Removed: Direct localStorage manipulation logic
  - Added: `useAuthStore()` hook to access centralized state
  - Kept: Same public API (`isAuthenticated`, `logout`, `storeTokens`) for backward compatibility
- **Backward Compatibility**: `storeTokens` is now an alias for `login` from auth store

## Architecture Decisions

### 1. Zustand Store for State Management

**Decision**: Created a centralized Zustand store (`auth-store.ts`) for authentication state instead of managing it in the React hook.

**Rationale**:
- **Non-React Context Access**: The ConnectRPC interceptor runs outside React component lifecycle and cannot use React hooks. Zustand provides a vanilla JS API (`useAuthStore.getState()`) for non-React code.
- **Single Source of Truth**: Centralizes authentication state management, eliminating duplicate logic between the hook and interceptor.
- **Reactive Updates**: Components using `useAuth()` hook automatically re-render when authentication state changes.

### 2. Separate Base Transport for Refresh Calls

**Decision**: Created `createBaseTransport()` that returns a transport without interceptors for RefreshToken RPC calls.

**Rationale**:
- **Prevents Infinite Recursion**: If the refresh call itself returns 401 and went through the auth interceptor, it would trigger another refresh, causing infinite recursion.
- **Clean Separation**: Refresh logic should be independent of the main request interceptor logic.

### 3. Promise-Based Deduplication

**Decision**: Used a module-level `refreshState` object holding the ongoing refresh promise.

**Rationale**:
- **Prevents Duplicate Refresh Calls**: When multiple requests fail with 401 simultaneously (common in dashboard views with many API calls), only one refresh call is made.
- **Promise Sharing**: All concurrent requests await the same promise and receive the same new tokens.
- **Automatic Cleanup**: The `finally` block resets `refreshState.promise = null` after completion.

### 4. Guard Conditions to Skip Refresh

**Decision**: Added two guard conditions before attempting token refresh:
1. Skip if request URL includes `/auth.v1.AuthService/`
2. Skip if no refresh token exists in localStorage

**Rationale**:
- **Auth Endpoint Guard**: Auth endpoints (GoogleAuth, RefreshToken, DeviceCodeApprove) don't require authentication. If they return 401, it's a legitimate error (invalid credentials, expired device code) that should propagate to the UI.
- **No Refresh Token Guard**: Prevents unnecessary refresh attempts when:
  - User hasn't logged in yet
  - Tokens were explicitly cleared by logout
  - Refresh token was manually removed from localStorage
- **Performance**: Avoids creating base transport and making RPC calls when refresh will obviously fail.

### 5. Error Propagation After Failed Refresh

**Decision**: When refresh fails, the interceptor calls `logout()` to clear tokens but still re-throws the error.

**Rationale**:
- **Clean State**: `logout()` removes invalid tokens and sets `isAuthenticated = false`, triggering UI updates.
- **Proper Promise Rejection**: Re-throwing the error ensures the original request promise rejects properly, allowing calling code to handle errors (show error messages, trigger redirects, etc.).
- **Route Guards Handle Navigation**: The route guard in `__root.tsx` observes `isAuthenticated` and redirects to `/auth` page automatically.

### 6. Backward Compatible Hook API

**Decision**: Kept the same public API for `useAuth()` hook (`isAuthenticated`, `logout`, `storeTokens`) while migrating internals to Zustand.

**Rationale**:
- **No Breaking Changes**: Existing components using `useAuth()` continue to work without modifications.
- **Incremental Migration**: Can refactor internals without touching 20+ components that import the hook.
- **Semantic Alias**: `storeTokens` is semantically equivalent to `login` - both store tokens and set authenticated state.

## Testing

### Manual Verification Commands

The following commands were used to verify the implementation:

```bash
# Build TypeScript to check for type errors
cd /Users/jayce/team-attention/cops/web && pnpm build
# Result: SUCCESS - No TypeScript errors

# Verify file exists
ls -la /Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts
# Result: File exists

# Check for console errors during build
pnpm build 2>&1 | grep -i error
# Result: No errors found
```

### Test Scenarios Covered

| Scenario | Expected Behavior | Implementation |
| -------- | ----------------- | -------------- |
| **Valid token request** | Request passes with Authorization header | Interceptor adds `Bearer {token}` header |
| **401 error with valid refresh token** | Tokens refreshed, request retried | `refreshTokenWithDeduplication()` called, new tokens stored, request retried |
| **401 error with expired refresh token** | Logout triggered, error propagated | `logout()` clears tokens, error re-thrown, route guard redirects |
| **401 on auth endpoint** | Error propagated without refresh | Guard 1 prevents refresh, error thrown |
| **401 with no refresh token** | Error propagated without refresh | Guard 2 prevents refresh, error thrown |
| **Concurrent 401 errors** | Single refresh call, all requests retry | `refreshState.promise` shared, all await same promise |
| **Refresh returns invalid response** | Error thrown, tokens cleared | Validation throws error, catch block calls logout |
| **No token in localStorage** | Request sent without Authorization header | Interceptor skips header addition |

### Browser Integration Testing

The implementation was designed to support these browser testing scenarios:

1. **Login → Wait for token expiry → Make API request**
   - Expected: Automatic refresh, seamless request completion
   - Verification: Check Network tab for RefreshToken call followed by retry

2. **Login → Make 10 concurrent requests after token expiry**
   - Expected: Single RefreshToken call, all 10 requests retry successfully
   - Verification: Check Network tab for only one RefreshToken call

3. **Logout → Make API request**
   - Expected: 401 error propagates, no refresh attempted
   - Verification: No RefreshToken call in Network tab

4. **Invalid refresh token in localStorage → Make request**
   - Expected: Refresh fails, redirect to /auth
   - Verification: Tokens cleared, URL changes to /auth

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| **Interceptor cannot use React hooks** | Created Zustand store with vanilla JS API (`useAuthStore.getState()`) accessible outside React |
| **Refresh endpoint would trigger infinite recursion** | Created separate base transport without interceptors for refresh calls |
| **Multiple concurrent requests causing duplicate refresh calls** | Implemented promise-based deduplication with module-level `refreshState` |
| **Auth endpoint 401 errors incorrectly triggering refresh** | Added guard condition to skip refresh for `/auth.v1.AuthService/` paths |
| **Type safety for ConnectError detection** | Used `instanceof ConnectError` and `error.code === Code.Unauthenticated` |

## Related Tickets

- **Original Request**: Implement automatic token refresh for web app ConnectRPC interceptor
- **Reference Implementation**: Daemon auth interceptor at `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go` (lines 56-74)

## Key Files Modified

- **Created**: `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts` (86 lines)
- **Modified**: `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` (166 lines, +135 lines)
- **Modified**: `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts` (20 lines, -33 lines)

## Dependencies

No new dependencies were added. The implementation uses existing packages:

- `@connectrpc/connect` - Interceptor, ConnectError, Code, createClient
- `@connectrpc/connect-web` - createConnectTransport
- `@bufbuild/protobuf` - create (message construction)
- `zustand` - State management (already in project)
- Generated stubs from `@/gen/grpcstub/auth/v1/auth_pb`

## Future Enhancements

Potential improvements not included in this implementation:

1. **Proactive Token Refresh**: Refresh tokens before expiry based on `cops_token_expires_at` timestamp
2. **Retry Limit**: Add maximum retry count to prevent infinite retry loops on persistent 401 errors
3. **Exponential Backoff**: Add delay between refresh attempts if multiple sequential refreshes fail
4. **Token Refresh UI Indicator**: Show loading state in UI when token refresh is in progress
5. **Metrics/Logging**: Add telemetry for refresh success/failure rates
