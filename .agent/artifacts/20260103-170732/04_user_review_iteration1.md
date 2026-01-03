# Review Result

**Status**: Changes Required

## Request Summary

User reported a bug on the '/' landing page: clicking the logout button doesn't update the Header UI - it only changes after manually refreshing the page. This is a state management issue where the `useAuth` hook reads from localStorage without React state, so components don't re-render when authentication state changes.

## Acceptance Criteria

- [ ] After clicking logout, the Header immediately updates from authenticated state (Dashboard button + Account dropdown) to unauthenticated state (Login button)
- [ ] After clicking logout, the landing page CTA immediately updates from "Go to Dashboard" to "Get Started"
- [ ] No page refresh required to see authentication state changes
- [ ] Logout functionality still clears all tokens from localStorage
- [ ] Other authentication-dependent components also update reactively

## Scope

### In Scope
- Fix the `useAuth` hook to use React state that triggers re-renders on authentication changes
- Ensure the `logout()` function updates both localStorage AND React state
- Test that all components using `useAuth` re-render correctly after logout

### Out of Scope
- Changes to the logout logic itself (clearing tokens)
- Changes to the UI/styling of the Header or landing page
- Adding new features to authentication flow
- Modifying authentication persistence strategy (localStorage is acceptable)

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts` | 10-15 | `.agent/rules/react/react-web.md` | Hook reads from localStorage directly without React state, preventing components from re-rendering when auth state changes | Add `useState` and `useEffect` to track authentication state. Update state in `logout()` and `storeTokens()` functions to trigger re-renders. |

## Root Cause Analysis

The `useAuth` hook currently works like this:

```typescript
export const useAuth = () => {
  // Reads from localStorage on every hook call
  const token = localStorage.getItem(ACCESS_TOKEN_KEY);
  const isAuthenticated = token !== null && token.length > 0;

  const logout = () => {
    // Clears localStorage but doesn't trigger React re-render
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
  };

  return { isAuthenticated, logout, storeTokens };
};
```

**Problem**:
1. `isAuthenticated` is computed from `localStorage.getItem()` on each render
2. When `logout()` is called, it updates localStorage
3. BUT there's no React state change, so React doesn't know to re-render components
4. The value of `isAuthenticated` only updates when the component re-renders for other reasons (like navigation or manual refresh)

**Why this is a React anti-pattern**:
- React components re-render when state or props change
- localStorage changes are NOT tracked by React
- Reading from localStorage directly gives you the current value, but doesn't subscribe to changes

## Recommended Solution

Convert `useAuth` to use React state with localStorage synchronization:

```typescript
import { useState, useEffect, useCallback } from 'react';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

export const useAuth = () => {
  // React state that triggers re-renders
  const [isAuthenticated, setIsAuthenticated] = useState(() => {
    const token = localStorage.getItem(ACCESS_TOKEN_KEY);
    return token !== null && token.length > 0;
  });

  // Logout updates both localStorage AND state
  const logout = useCallback(() => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
    setIsAuthenticated(false); // ← Triggers re-render
  }, []);

  // storeTokens updates both localStorage AND state
  const storeTokens = useCallback((tokens: TokenPair) => {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
    setIsAuthenticated(true); // ← Triggers re-render
  }, []);

  return {
    isAuthenticated,
    logout,
    storeTokens,
  };
};
```

**Why this works**:
1. `isAuthenticated` is now a React state variable
2. When `logout()` or `storeTokens()` is called, it updates state via `setIsAuthenticated()`
3. React detects the state change and re-renders all components using this hook
4. The Header and landing page CTA immediately update to reflect the new authentication state

## Alternative Solutions (Not Recommended)

### Option 2: Context + Provider Pattern
Create an `AuthContext` with a provider that wraps the app. More complex but allows global state management.

**Pros**:
- Centralized auth state
- Can add more complex auth logic later

**Cons**:
- Requires modifying the app structure to add provider
- More boilerplate code
- Overkill for this simple use case

### Option 3: External State Management (Zustand, Jotai, etc.)
Use a third-party state management library.

**Pros**:
- More features
- Better for complex state

**Cons**:
- Additional dependency
- Unnecessary complexity for this simple auth state
- Against the "common.md" rule to minimize dependencies

**Recommendation**: Use Option 1 (useState + useCallback) as it's the simplest solution that follows React best practices and doesn't require additional dependencies or architectural changes.

## Testing Checklist

After implementing the fix, verify:

1. **Logout on Landing Page**:
   - [ ] Navigate to '/' while authenticated
   - [ ] Click Account dropdown → Logout
   - [ ] Header immediately shows "Login" button (no refresh needed)
   - [ ] CTA button immediately changes to "Get Started" (no refresh needed)

2. **Login Flow**:
   - [ ] Navigate to '/auth' while unauthenticated
   - [ ] Complete login
   - [ ] Navigate back to '/'
   - [ ] Header immediately shows Dashboard button + Account dropdown
   - [ ] CTA button immediately shows "Go to Dashboard"

3. **Cross-Component Consistency**:
   - [ ] All components using `useAuth` update simultaneously
   - [ ] No stale authentication state in any component

4. **Persistence**:
   - [ ] Logout clears all tokens from localStorage
   - [ ] Login stores all tokens to localStorage
   - [ ] Refreshing page shows correct authentication state

## Additional Context

- Requirements document: `.agent/artifacts/20260103-170732/01_requirements.md`
- Plan document: `.agent/artifacts/20260103-170732/02_plan.md`
- Initial review: `.agent/artifacts/20260103-170732/03_review.md`
- This is user feedback on the implemented landing page

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](/Users/jayce/team-attention/cops/.agent/rules/common.md) - General coding standards
- [`.agent/rules/workflow.md`](/Users/jayce/team-attention/cops/.agent/rules/workflow.md) - Development workflow
- [`.agent/rules/react/react-web.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md) - React and TypeScript best practices
- [`.agent/rules/react/react-web-src.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md) - Project structure and naming conventions

## Files to Modify

| File | Changes Required |
|------|------------------|
| `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts` | Add `useState` for `isAuthenticated`, update `logout()` and `storeTokens()` to modify state, wrap functions in `useCallback` |

## Implementation Priority

**High Priority** - This is a user-facing bug that affects core authentication functionality and creates a confusing user experience.
