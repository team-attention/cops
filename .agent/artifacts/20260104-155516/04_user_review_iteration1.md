# Review Result

**Status**: Changes Required

## Request Summary

User reported a critical authentication bug: after successful login, users are redirected back to `/auth` page in an infinite loop. Investigation revealed a state synchronization issue between the token refresh mechanism and React authentication state management.

The token refresh implementation in `connect-transport.ts` clears localStorage tokens when refresh fails, but the `useAuth` hook's React state (`isAuthenticated`) is not updated, causing the `/auth` page to incorrectly believe the user is still authenticated and redirect them back to protected routes.

## Acceptance Criteria

- [ ] Fix state synchronization between token storage and React authentication state
- [ ] Ensure `isAuthenticated` state is updated when tokens are cleared during failed refresh
- [ ] Prevent infinite redirect loop between `/auth` and protected routes
- [ ] Maintain proper authentication flow for successful login and logout

## Scope

### In Scope
- Fix token refresh error handling to properly update React state
- Synchronize localStorage token changes with React state
- Test the complete authentication flow including token refresh failure

### Out of Scope
- Any other authentication features not related to this bug
- Token refresh success scenarios (working correctly)
- OAuth flow implementation (working correctly)

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` | 43-48 | Architecture | State synchronization violation: `clearTokens()` modifies localStorage but doesn't update React state, breaking the single source of truth principle | Remove `clearTokens()` and `redirectToAuth()` functions from transport service. Use the `logout()` function from `useAuth` hook instead. The transport should not directly manage authentication state. |
| `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` | 86-92 | Architecture | Token refresh error handler calls `clearTokens()` and `redirectToAuth()` which creates state inconsistency | Replace direct token clearing with a callback mechanism or event that triggers the `useAuth.logout()` function to properly update both localStorage and React state |

## Root Cause Analysis

### The Bug Flow

1. User successfully logs in via OAuth callback
2. Tokens stored in localStorage, `isAuthenticated` state set to `true`
3. User redirected to `/dashboard`
4. Dashboard makes API call
5. If access token is expired/invalid, auth interceptor attempts refresh
6. **If refresh token is also expired**, refresh fails
7. `refreshTokenWithDeduplication()` catch block executes (line 86-92):
   - Calls `clearTokens()` → removes tokens from localStorage
   - Calls `redirectToAuth()` → navigates to `/auth`
8. User lands on `/auth` page
9. **BUG**: `/auth` page's `useEffect` checks `isAuthenticated` (line 42)
10. `isAuthenticated` is still `true` because React state was never updated
11. `useEffect` triggers redirect back to `/dashboard` (line 50)
12. **Infinite loop** starts: `/dashboard` → token refresh fails → `/auth` → `/dashboard` → ...

### Why This Happens

The `connect-transport.ts` service layer directly manipulates localStorage via `clearTokens()` but has no way to update the React state in the `useAuth` hook. The `useAuth` hook maintains its own state that becomes stale when localStorage is modified externally.

**State Management Violation**: Two separate systems manage the same authentication state:
- `connect-transport.ts`: Manages localStorage tokens
- `useAuth` hook: Manages React state (`isAuthenticated`)

When one updates without the other, they become out of sync.

## Recommended Solution

### Option 1: Use Callback/Event Pattern (Recommended)

Create a callback mechanism that allows `connect-transport.ts` to trigger `useAuth.logout()`:

1. Create a global auth event emitter or callback registry
2. Register `useAuth.logout()` as the handler
3. In token refresh error handler, emit logout event instead of calling `clearTokens()`

### Option 2: Remove State from Transport

Move token refresh logic out of the transport interceptor into a React context/hook where it has access to `setIsAuthenticated`:

1. Create `AuthProvider` context that wraps the app
2. Move token refresh logic into this context
3. Provide both tokens and refresh function to components
4. Transport interceptor just uses tokens, doesn't manage them

### Option 3: Sync localStorage with React State

Add a storage event listener in `useAuth` to detect external localStorage changes:

1. Add `storage` event listener in `useAuth` hook
2. When localStorage tokens are cleared, update `isAuthenticated` state
3. This keeps React state in sync with localStorage changes

## Additional Context

- User feedback: "There's a bug where after login, it keeps redirecting back to /auth page. Even after successful login, the user is redirected back to /auth."
- This is a regression likely introduced in commit `d448cca` which added automatic token refresh
- The bug only manifests when refresh tokens expire, which explains why it wasn't caught immediately
- The OAuth callback flow and initial login work correctly - the bug only occurs on subsequent page loads when token refresh is needed

## Files Reviewed

- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` - Token refresh implementation
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts` - Authentication state management
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx` - Auth page with redirect logic
- `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx` - OAuth callback handler
- `/Users/jayce/team-attention/cops/web/src/route/__root.tsx` - Root layout
- `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx` - Protected dashboard route

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](/Users/jayce/team-attention/cops/.agent/rules/common.md) - Code quality and architecture principles
- [`.agent/rules/react/react-web.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md) - React and TypeScript best practices
- [`.agent/rules/react/react-web-src.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md) - Service layer and state management patterns
