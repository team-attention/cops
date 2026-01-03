# Implementation Plan: Fix useAuth Hook State Management

## Overview

Fix the `useAuth` hook to use React state management instead of reading directly from localStorage. This will ensure that components using the hook re-render immediately when authentication state changes (e.g., after logout or login), without requiring a manual page refresh.

The current implementation reads from localStorage on every render but doesn't track changes as React state, so when `logout()` clears localStorage, React doesn't know to re-render components. This plan adds `useState` to track authentication state and updates that state whenever tokens are stored or cleared.

## Package Changes

No package changes required. All necessary dependencies (`react`, `useState`, `useCallback`) are already available.

## Implementation Steps

### Step 1: Add React State Management to useAuth Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Understanding React hooks best practices and TypeScript rules
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Current implementation to be modified

#### `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`

**Description**:
Convert the hook from reading localStorage directly to using React state that synchronizes with localStorage. This ensures components re-render when authentication state changes.

```typescript
import { useState, useCallback } from 'react';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// Constants for token storage
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// useAuth provides authentication state and management functions.
// Returns authentication status and token management utilities.
// Uses React state to ensure components re-render on auth state changes.
export const useAuth = () => {
  // Implementation outline:
  // 1. Initialize isAuthenticated state using useState with lazy initializer function
  //    a. Read access token from localStorage
  //    b. Return true if token exists and has length > 0, false otherwise
  //    c. This runs only once on initial mount

  // 2. Define logout function using useCallback
  //    a. Remove ACCESS_TOKEN_KEY from localStorage
  //    b. Remove REFRESH_TOKEN_KEY from localStorage
  //    c. Remove TOKEN_EXPIRES_AT_KEY from localStorage
  //    d. Call setIsAuthenticated(false) to trigger re-render
  //    e. Empty dependency array since it uses no external values

  // 3. Define storeTokens function using useCallback
  //    a. Accept tokens parameter of type TokenPair
  //    b. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
  //    c. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
  //    d. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
  //    e. Call setIsAuthenticated(true) to trigger re-render
  //    f. Empty dependency array since it uses no external values

  // 4. Return object with:
  //    - isAuthenticated (from state)
  //    - logout (memoized function)
  //    - storeTokens (memoized function)
};
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Initial mount when authenticated | localStorage has valid access token | `isAuthenticated: true` | State initialization with existing token |
| Initial mount when not authenticated | localStorage has no access token | `isAuthenticated: false` | State initialization without token |
| Call storeTokens with valid tokens | `{ accessToken: "abc", refreshToken: "xyz", expiresAt: 123 }` | localStorage updated, `isAuthenticated: true`, components re-render | storeTokens happy path |
| Call logout when authenticated | None | localStorage cleared, `isAuthenticated: false`, components re-render | logout happy path |
| Multiple components using useAuth | Multiple components mounted | All components share same auth state and re-render together | State consistency across components |

### Step 2: Verify Integration with Components

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/component/landing-header.tsx`: Component that uses useAuth for conditional rendering (Dashboard button vs Login button)
- `/Users/jayce/team-attention/cops/web/src/route/index.tsx`: Landing page that uses useAuth for CTA button text
- `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`: Authentication callback that may use storeTokens

#### Verification Only (No Code Changes)

**Description**:
Read the components that use `useAuth` to verify they will work correctly with the new state-based implementation. No changes needed to these components since the hook's public API remains the same.

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Logout from landing page header | Click logout in Account dropdown | Header shows Login button, CTA changes to "Get Started", both without refresh | Component re-render on state change |
| Login via auth flow | Complete authentication | Landing page shows Dashboard button and Account dropdown without refresh | Component re-render on token storage |
| Navigate between pages while authenticated | Navigate from landing to dashboard and back | Auth state persists correctly across navigation | State persistence |

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Execute Agent
- [x] All packages are selected (using built-in React hooks, no external packages needed)
- [x] Execution order is clear and dependencies are explicit
- [x] Follows React best practices from `.agent/rules/react/react-web.md`
- [x] Uses `useState` for reactive state management
- [x] Uses `useCallback` to memoize functions
- [x] No breaking changes to the hook's public API
