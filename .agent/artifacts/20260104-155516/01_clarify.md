# Requirements

## Request Summary

The web application currently lacks automatic token refresh logic when Access Tokens expire. When users interact with the app after their Access Token expires, API requests fail with authentication errors. The backend API already provides a `RefreshToken` RPC endpoint that exchanges a refresh token for a new token pair. The implementation needs to add client-side interceptor logic to detect 401 errors, automatically call the refresh endpoint using the stored refresh token, and retry failed requests with the new access token.

## Current State Analysis

### Existing Components

**Backend (API Server):**
- `/api/internal/service/auth/auth_service.go`: Contains `RefreshToken()` method that validates refresh tokens and generates new token pairs
- gRPC endpoint: `auth.v1.AuthService.RefreshToken` (available in generated stubs)

**Frontend (Web App):**
- `/web/src/shared/hook/use-auth.ts`: Manages authentication state and stores tokens in localStorage
  - Stores: `cops_access_token`, `cops_refresh_token`, `cops_token_expires_at`
  - Provides: `isAuthenticated`, `logout()`, `storeTokens()`
- `/web/src/shared/service/connect-transport.ts`: Creates ConnectRPC transport with auth interceptor
  - Current interceptor only adds `Authorization: Bearer {token}` header
  - Does NOT handle token refresh or retry on 401 errors

**Reference Implementation (Daemon):**
- `/daemon/internal/platform/interceptor/auth_interceptor.go`: Shows token refresh pattern
  - Detects 401 Unauthenticated errors
  - Calls `RefreshAccessToken()` to get new token
  - Retries request with refreshed token

### Missing Functionality

The web app's `connect-transport.ts` interceptor lacks:
1. Detection of 401/Unauthenticated errors from API responses
2. Automatic refresh token call to backend
3. Token storage update after successful refresh
4. Request retry logic with new access token
5. Fallback handling when refresh fails (e.g., redirect to login)

## Acceptance Criteria

- [ ] Web app interceptor detects 401 Unauthenticated errors from ConnectRPC responses
- [ ] On 401 error, interceptor automatically calls `RefreshToken` RPC using stored refresh token
- [ ] New tokens from refresh response are stored in localStorage (via `useAuth` hook or direct storage)
- [ ] Original failed request is retried with the new access token
- [ ] If refresh fails (refresh token expired/invalid), user is redirected to login page or logout is triggered
- [ ] Token refresh only happens once per concurrent batch of requests (avoid duplicate refresh calls)
- [ ] User experience is seamless - no visible errors during automatic token refresh
- [ ] Implementation follows existing ConnectRPC interceptor patterns from daemon reference

## Scope

### In Scope
- Creating or modifying ConnectRPC interceptor in `/web/src/shared/service/connect-transport.ts`
- Implementing 401 error detection logic
- Calling `refreshToken` RPC method (import from generated stubs)
- Storing refreshed tokens to localStorage
- Retrying failed requests with new tokens
- Handling refresh failure (logout or redirect)
- Preventing concurrent refresh requests (mutex/flag pattern)

### Out of Scope
- Modifying backend authentication logic (already working)
- Changing token storage mechanism (keep using localStorage)
- Implementing proactive token refresh before expiration (only reactive on 401)
- Adding token refresh UI indicators or loading states
- Modifying `use-auth.ts` hook beyond potentially adding a `refreshToken()` helper function

## Constraints

**Technical Constraints:**
- Must use ConnectRPC interceptor pattern (`@connectrpc/connect` package)
- Must maintain compatibility with existing `useAuth` hook
- Must follow project rules in `/Users/jayce/team-attention/cops/.agent/rules/react/`
- Generated gRPC stubs are read-only - cannot modify them
- Must handle race conditions (multiple requests failing simultaneously)

**Implementation Requirements:**
- Follow TypeScript strict typing rules (no `any` types)
- Use named exports (no default exports)
- Import from `@/gen/grpcstub/auth/v1/` for auth service stubs
- Error handling must be graceful - no unhandled promise rejections

## Additional Context

### Related Files
- Backend auth service: `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go` (line 267-314)
- Generated auth stubs: `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/auth/v1/auth-AuthService_connectquery.ts`
- Daemon reference implementation: `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go` (line 56-74)

### Token Storage Keys
- Access Token: `cops_access_token`
- Refresh Token: `cops_refresh_token`
- Expires At: `cops_token_expires_at`

### ConnectRPC Error Codes
- Use `connect.CodeOf(error)` to check error codes
- 401 errors map to `connect.CodeUnauthenticated`

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should the implementation follow the daemon's Go interceptor pattern (detect 401, refresh, retry)? | Yes, detect 401 errors, call refresh endpoint, and retry original request |
| What should happen when refresh token is also expired/invalid? | Redirect to `/auth` page to force re-authentication |
| Should we add a helper method to `use-auth.ts` for refresh, or call the RPC directly in the interceptor? | Self-contained in the ConnectRPC interceptor (no modifications to `use-auth.ts` hook) |
| Should the implementation prevent concurrent refresh calls (e.g., when multiple requests fail at once)? | Yes, deduplicate refresh calls: queue pending requests, make single refresh call, share result with all queued requests |
| Refresh timing | Reactive only - refresh only after receiving 401 error (not proactive based on expiry time) |
