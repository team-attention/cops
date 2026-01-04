# Review Result

**Status**: Pass

All changes follow project rules correctly and meet the acceptance criteria from the plan.

## Summary

The auth interceptor fix has been correctly implemented with both guard conditions as specified in the iteration 2 plan. The implementation addresses the critical bug where login was failing with "[unknown] No refresh token available" error.

## Files Reviewed

- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`
- `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`

## Verification Results

### Guard 1: AUTH_SERVICE_PREFIX Check

**Requirement**: Skip token refresh for auth endpoints (GoogleAuth, RefreshToken, etc.)

**Implementation** (lines 27-28, 133-138):
```typescript
// AUTH_SERVICE_PREFIX is the URL path prefix for auth service endpoints that should not trigger token refresh
const AUTH_SERVICE_PREFIX = '/auth.v1.AuthService/'

// ...

// Guard 1: Skip token refresh for auth service endpoints
// Auth endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) don't require
// authentication and should propagate their 401 errors directly
if (req.url.includes(AUTH_SERVICE_PREFIX)) {
  throw error
}
```

**Verdict**: PASS - The constant is properly defined and the guard correctly checks if the request URL includes the auth service prefix, throwing the original error when it matches.

### Guard 2: Refresh Token Existence Check

**Requirement**: Skip token refresh if no refresh token exists

**Implementation** (lines 140-145):
```typescript
// Guard 2: Skip token refresh if no refresh token is available
// This handles the case where tokens have been cleared or user never logged in
const refreshToken = getRefreshToken()
if (!refreshToken || refreshToken.length === 0) {
  throw error
}
```

**Verdict**: PASS - The guard correctly retrieves the refresh token using `getRefreshToken()` from auth-store and throws the original error if it's null or empty.

### Original 401 Error Propagation

**Requirement**: Propagate the original 401 error when guards fail (not a new error)

**Implementation**: Both guards use `throw error` which re-throws the original `ConnectError` with `Code.Unauthenticated` that was caught.

**Verdict**: PASS - The original error is preserved and propagated, not replaced with a new error.

### Login Flow Correctness

**Analysis**:
1. User clicks "Sign in with Google" on `/auth` page
2. User is redirected to Google OAuth
3. After Google OAuth, user lands on callback page with auth code
4. Callback calls `GoogleAuth` RPC endpoint
5. If `GoogleAuth` returns 401 (invalid code):
   - Interceptor catches the error
   - Guard 1 checks: `req.url.includes('/auth.v1.AuthService/')` = TRUE
   - Original 401 error is thrown immediately
   - Login page can display proper error message
6. If `GoogleAuth` succeeds:
   - Tokens are returned and stored via `storeTokens()`
   - User is redirected to dashboard

**Verdict**: PASS - Login flow will work correctly because auth endpoint 401 errors are now propagated directly without attempting token refresh.

## Code Quality Checks

| Check | Status | Notes |
|-------|--------|-------|
| Named exports used | PASS | All exports are named (`transport`, `getAccessToken`, `getRefreshToken`) |
| No `any` types | PASS | All types are properly defined |
| Named interfaces defined | PASS | `RefreshState`, `AuthStoreState`, `AuthStoreActions` are properly named |
| Comments in English | PASS | All comments are in English |
| Proper error handling | PASS | Errors are properly caught and propagated |
| Service layer pattern followed | PASS | Transport service follows `shared/service/` pattern |

## Test Scenarios Coverage

Based on the plan's test scenarios:

| Scenario | Expected Behavior | Implementation Coverage |
|----------|-------------------|-------------------------|
| Auth endpoint (GoogleAuth) returns 401 | Original 401 error thrown | Guard 1 covers this |
| Auth endpoint (RefreshToken) returns 401 | Original 401 error thrown | Guard 1 covers this |
| Auth endpoint (DeviceCode) returns 401 | Original 401 error thrown | Guard 1 covers this |
| Auth endpoint (DevicePoll) returns 401 | Original 401 error thrown | Guard 1 covers this |
| Protected endpoint 401, no refresh token | Original 401 error thrown | Guard 2 covers this |
| Protected endpoint 401, empty refresh token | Original 401 error thrown | Guard 2 covers this |
| Protected endpoint 401, valid refresh token | Token refresh attempted | Happy path after guards |
| Protected endpoint returns 200 | Response returned directly | Try block success path |
| Protected endpoint returns 500 | 500 error thrown | Catch block non-auth error path |

## Implementation Checklist Verification

From the plan:

- [x] Add `AUTH_SERVICE_PREFIX` constant after `refreshState` declaration (line 28)
- [x] Add Guard 1: Check if `req.url.includes(AUTH_SERVICE_PREFIX)` and throw original error if true (lines 133-138)
- [x] Add Guard 2: Check if refresh token exists using `getRefreshToken()` and throw original error if null/empty (lines 140-145)
- [x] Ensure existing token refresh logic only runs when both guards pass (lines 147-153)
- [x] Verify login flow completes successfully without interceptor interference

## Related Files Review

### `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`

The `getRefreshToken()` function is correctly implemented:
```typescript
export const getRefreshToken = (): string | null => {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}
```

This returns `null` when no token exists, which the guard correctly handles.

### `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`

Simple hook that correctly wraps auth store. No issues.

### `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`

Auth page correctly uses `useAuth()` hook and handles redirect flow. The fix in the interceptor will allow proper error propagation from the OAuth callback.

## Rules Applied

- [`.agent/rules/common.md`](/Users/jayce/team-attention/cops/.agent/rules/common.md) - Comments in English, no unnecessary code
- [`.agent/rules/workflow.md`](/Users/jayce/team-attention/cops/.agent/rules/workflow.md) - Pre-action context loading
- [`.agent/rules/react/react-web.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md) - Named exports, proper types, no `any`
- [`.agent/rules/react/react-web-src.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md) - Service layer patterns, gRPC integration

## Conclusion

The implementation correctly addresses the critical login bug by adding two guard conditions before attempting token refresh:

1. **Guard 1** checks if the request is to an auth service endpoint (which should never trigger token refresh since they are authentication endpoints, not authenticated endpoints)

2. **Guard 2** checks if a refresh token exists before attempting refresh (preventing the "No refresh token available" error from obscuring the original 401)

Both guards correctly propagate the original error rather than creating a new one, preserving the server's error message for proper handling by the calling code.
