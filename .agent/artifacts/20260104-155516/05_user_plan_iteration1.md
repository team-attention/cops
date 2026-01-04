# Implementation Plan: Fix Authentication State Synchronization

## Overview

Fix the infinite redirect loop bug caused by state synchronization issues between localStorage token management and React authentication state. The root cause is that `connect-transport.ts` directly clears localStorage tokens during failed token refresh, but doesn't update the React state in `useAuth`, causing the `/auth` page to incorrectly believe the user is still authenticated.

The solution is to create a centralized authentication store using Zustand (following the existing `user-store.ts` pattern) that provides a single source of truth for authentication state and ensures localStorage and React state are always synchronized.

## Package Changes

No external packages need to be added or removed. The project already has `zustand` installed (used in `user-store.ts`).

## Implementation Steps

### Step 1: Create Centralized Auth Store with Zustand

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`: Example of Zustand store pattern to follow
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Service layer and state management patterns
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript and React best practices

#### `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`

**Description**:
Create a new Zustand store that manages authentication state (tokens and `isAuthenticated` flag) with automatic localStorage synchronization. This store will be the single source of truth for authentication state throughout the application.

```ts
import { create } from 'zustand';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// Token storage key constants
const ACCESS_TOKEN_KEY = 'cops_access_token';
const REFRESH_TOKEN_KEY = 'cops_refresh_token';
const TOKEN_EXPIRES_AT_KEY = 'cops_token_expires_at';

// AuthStoreState defines the authentication state shape
interface AuthStoreState {
  // isAuthenticated indicates if user has valid access token
  isAuthenticated: boolean;
}

// AuthStoreActions defines available authentication actions
interface AuthStoreActions {
  // login stores tokens in localStorage and updates isAuthenticated to true
  login: (tokens: TokenPair) => void;
  // logout removes tokens from localStorage and updates isAuthenticated to false
  logout: () => void;
  // updateTokens updates existing tokens without changing isAuthenticated state (used during refresh)
  updateTokens: (tokens: TokenPair) => void;
}

type AuthStore = AuthStoreState & AuthStoreActions;

// initialState defines the default authentication state
const initialState: AuthStoreState = {
  isAuthenticated: false,
};

// checkInitialAuth checks localStorage for existing tokens on app initialization
const checkInitialAuth = (): boolean => {
  // 1. Read access token from localStorage using ACCESS_TOKEN_KEY
  // 2. Return true if token exists and token.length > 0, false otherwise
};

export const useAuthStore = create<AuthStore>()((set) => ({
  // Initialize isAuthenticated by checking localStorage
  isAuthenticated: checkInitialAuth(),

  login: (tokens) => {
    // Implementation outline:
    // 1. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
    // 2. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
    // 3. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
    // 4. Call set({ isAuthenticated: true }) to update state
  },

  logout: () => {
    // Implementation outline:
    // 1. Remove ACCESS_TOKEN_KEY from localStorage
    // 2. Remove REFRESH_TOKEN_KEY from localStorage
    // 3. Remove TOKEN_EXPIRES_AT_KEY from localStorage
    // 4. Call set({ isAuthenticated: false }) to update state
  },

  updateTokens: (tokens) => {
    // Implementation outline:
    // 1. Store tokens.accessToken to localStorage with ACCESS_TOKEN_KEY
    // 2. Store tokens.refreshToken to localStorage with REFRESH_TOKEN_KEY
    // 3. Store tokens.expiresAt.toString() to localStorage with TOKEN_EXPIRES_AT_KEY
    // Note: Do NOT update isAuthenticated state - this is for silent token refresh
  },
}));

// getAccessToken is a utility function to retrieve access token from localStorage
export const getAccessToken = (): string | null => {
  // 1. Read and return access token from localStorage using ACCESS_TOKEN_KEY
};

// getRefreshToken is a utility function to retrieve refresh token from localStorage
export const getRefreshToken = (): string | null => {
  // 1. Read and return refresh token from localStorage using REFRESH_TOKEN_KEY
};
```

**Test Scenarios**:

| Scenario | Input | Expected Output | State Changes |
|:---------|:------|:----------------|:--------------|
| Initial load with tokens | localStorage has valid tokens | `isAuthenticated: true` | Store initialized with true |
| Initial load without tokens | localStorage empty | `isAuthenticated: false` | Store initialized with false |
| User calls login() | Valid `TokenPair` object | Tokens stored, `isAuthenticated: true` | State updated to authenticated |
| User calls logout() | None | Tokens removed, `isAuthenticated: false` | State updated to unauthenticated |
| Silent token refresh via updateTokens() | Valid `TokenPair` object | Tokens updated, `isAuthenticated` unchanged | Only localStorage updated |

### Step 2: Refactor connect-transport to Use Auth Store

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`: Current implementation to be refactored
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Service layer patterns

#### `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`

**Description**:
Refactor the transport service to use the new auth store for token management. Remove direct localStorage manipulation and state management responsibilities, delegating them to the auth store. The transport will only read tokens and trigger logout through the store.

```ts
import { Code, ConnectError, createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import type { Interceptor } from '@connectrpc/connect';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';
import { AuthService, RefreshTokenReqSchema } from '@/gen/grpcstub/auth/v1/auth_pb';
import { useAuthStore, getAccessToken, getRefreshToken } from '@/shared/store/auth-store';

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
  // 1. Return import.meta.env.VITE_API_URL if defined
  // 2. Otherwise return 'http://localhost:8080'
};

// createBaseTransport creates a transport without auth interceptor for refresh calls
const createBaseTransport = () => {
  // Implementation outline:
  // 1. Call createConnectTransport with baseUrl from getBaseUrl()
  // 2. Return the transport instance
};

// performTokenRefresh executes the actual refresh token RPC call
const performTokenRefresh = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. Get refresh token using getRefreshToken() from auth-store
  // 2. If refreshToken is null or empty string:
  //    a. Throw new Error('No refresh token available')
  // 3. Create baseTransport using createBaseTransport()
  // 4. Create AuthService client with baseTransport
  // 5. Create RefreshTokenReq using create(RefreshTokenReqSchema, { refreshToken })
  // 6. Call client.refreshToken(request) and await response
  // 7. If response.tokens is undefined or null:
  //    a. Throw new Error('Invalid refresh response: no tokens returned')
  // 8. Return response.tokens
};

// refreshTokenWithDeduplication handles token refresh with request deduplication
const refreshTokenWithDeduplication = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. If refreshState.promise is not null:
  //    a. Return refreshState.promise (deduplicate concurrent refresh requests)
  // 2. Set refreshState.promise = performTokenRefresh()
  // 3. Try block:
  //    a. Await tokens from refreshState.promise
  //    b. Call useAuthStore.getState().updateTokens(tokens) to update tokens
  //    c. Return tokens
  // 4. Catch block (error):
  //    a. Call useAuthStore.getState().logout() to clear state and tokens
  //    b. Throw error (let caller handle navigation)
  // 5. Finally block:
  //    a. Set refreshState.promise = null (reset deduplication state)
};

// createAuthInterceptor creates an interceptor that adds JWT and handles token refresh
const createAuthInterceptor = (): Interceptor => {
  // Implementation outline:
  // 1. Return interceptor function: (next) => async (req) => { ... }
  // 2. Inside interceptor:
  //    a. Get access token using getAccessToken() from auth-store
  //    b. If token exists and token.length > 0:
  //       i. Set Authorization header: req.header.set('Authorization', `Bearer ${token}`)
  //    c. Try block:
  //       i. Await response from next(req)
  //       ii. Return response
  //    d. Catch block (error):
  //       i. If error is ConnectError and error.code === Code.Unauthenticated:
  //          - Await newTokens from refreshTokenWithDeduplication()
  //          - Set Authorization header with new token: req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
  //          - Await and return response from next(req) (retry request)
  //       ii. Otherwise:
  //          - Throw error (not an auth error)
};

export const transport = createConnectTransport({
  baseUrl: getBaseUrl(),
  interceptors: [createAuthInterceptor()],
});
```

**Key Changes**:
1. **Remove** `storeTokens()` function - replaced by `useAuthStore.login()` and `useAuthStore.updateTokens()`
2. **Remove** `clearTokens()` function - replaced by `useAuthStore.logout()`
3. **Remove** `redirectToAuth()` function - auth page will handle navigation via route guards
4. **Import** `useAuthStore`, `getAccessToken`, `getRefreshToken` from auth-store
5. **Update** `refreshTokenWithDeduplication()`:
   - Use `useAuthStore.getState().updateTokens()` on success
   - Use `useAuthStore.getState().logout()` on error
   - Remove `redirectToAuth()` call - let React Router handle navigation
6. **Update** `createAuthInterceptor()`:
   - Use `getAccessToken()` instead of direct localStorage access

**Test Scenarios**:

| Scenario | Input | Expected Behavior | Auth Store State |
|:---------|:------|:------------------|:-----------------|
| Request with valid token | API request | Token added to header, request succeeds | No change |
| Request with expired token (refresh succeeds) | API request → 401 → refresh succeeds | New tokens stored via `updateTokens()`, request retried | `isAuthenticated` stays true |
| Request with expired token (refresh fails) | API request → 401 → refresh fails | `logout()` called, error thrown | `isAuthenticated` set to false |
| Concurrent requests trigger refresh | Multiple API requests → 401s | Only one refresh call made (deduplication) | Tokens updated once |

### Step 3: Refactor useAuth Hook to Use Auth Store

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Current implementation to be refactored
- `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`: New auth store to integrate

#### `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`

**Description**:
Simplify the `useAuth` hook to be a thin wrapper around the auth store. Remove internal state management and delegate all authentication operations to the centralized store.

```ts
import { useAuthStore } from '@/shared/store/auth-store';
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb';

// useAuth provides authentication state and management functions
// Returns authentication status and token management utilities from auth store
export const useAuth = () => {
  // Implementation outline:
  // 1. Call useAuthStore() to get the entire store object
  // 2. Destructure: { isAuthenticated, login, logout } from store
  // 3. Return object with:
  //    - isAuthenticated (from store)
  //    - logout (from store)
  //    - storeTokens (alias for store.login - kept for backward compatibility)
};
```

**Key Changes**:
1. **Remove** `useState` and local state management
2. **Remove** `useCallback` hooks - functions come from store
3. **Replace** with direct calls to `useAuthStore`
4. **Keep** `storeTokens` as alias for `login` to maintain backward compatibility with existing code (OAuth callback uses it)

**Test Scenarios**:

| Scenario | Input | Expected Output | Integration Test |
|:---------|:------|:----------------|:-----------------|
| Component uses useAuth | None | Returns current store state | Hook subscribes to store updates |
| Component calls logout() | None | Store logout called, components re-render | All components using useAuth receive update |
| Component calls storeTokens() | Valid TokenPair | Store login called, components re-render | All components using useAuth receive update |

### Step 4: Update Auth Page to Handle Unauthenticated Redirect

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`: Current auth page implementation
- `/Users/jayce/team-attention/cops/web/src/shared/store/auth-store.ts`: Auth store for state access

#### `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`

**Description**:
Update the auth page redirect logic to handle the case where users land on `/auth` after token refresh failure. Add error handling for the redirect navigation to prevent infinite loops.

```ts
import { useEffect } from 'react';
import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router';
import { Shield } from 'lucide-react';
import { Button } from '@/gen/shadcn/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card';
import { useAuth } from '@/shared/hook/use-auth';

// Constants (no changes)
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// AuthSearchParams defines search params for auth route (no changes)
interface AuthSearchParams {
  redirect?: string;
}

// Route configuration (no changes)
export const Route = createFileRoute('/auth/')({
  component: AuthPage,
  validateSearch: (search: Record<string, unknown>): AuthSearchParams => {
    return {
      redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
    };
  },
});

// AuthPage displays the auth landing page with Google sign-in button
function AuthPage() {
  // Existing code (no changes to variable declarations)
  const search = useSearch({ from: '/auth/' });
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    // Implementation outline:
    // 1. If isAuthenticated is false:
    //    a. Return early (user needs to login, show the page)
    // 2. If isAuthenticated is true:
    //    a. If search.redirect exists and is not empty:
    //       i. Try-catch block:
    //          - Try: Call navigate({ to: search.redirect })
    //          - Catch: If navigation fails, call navigate({ to: '/dashboard' }) as fallback
    //       ii. Return after navigation
    //    b. Otherwise (no redirect param):
    //       i. Call navigate({ to: '/dashboard' })
  }, [isAuthenticated, search.redirect, navigate]);

  // handleGoogleSignIn function (no changes)
  const handleGoogleSignIn = () => {
    // Existing implementation remains unchanged
    if (search.redirect) {
      sessionStorage.setItem('cops_oauth_redirect', search.redirect);
    }

    const params = new URLSearchParams({
      response_type: 'code',
      client_id: GOOGLE_OAUTH_CLIENT_ID,
      redirect_uri: GOOGLE_OAUTH_REDIRECT_URI,
      scope: GOOGLE_OAUTH_SCOPES,
      access_type: 'offline',
      prompt: 'consent',
    });

    const oauthUrl = `${GOOGLE_OAUTH_AUTHORIZE_URL}?${params.toString()}`;
    window.location.href = oauthUrl;
  };

  // JSX return (no changes)
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="text-center">
            <div className="mb-4 flex justify-center">
              <div className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-3">
                <Shield className="h-8 w-8 text-cyan-400" />
              </div>
            </div>
            <CardTitle className="text-xl text-zinc-100">
              Sign in to C-Ops
            </CardTitle>
            <CardDescription className="text-zinc-500">
              Continue with your Google account
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleGoogleSignIn}
              className="w-full bg-white text-zinc-900 hover:bg-zinc-100"
            >
              Sign in with Google
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
```

**Key Changes**:
1. **Update** `useEffect` to check `isAuthenticated === false` first and return early
2. **Add** try-catch block around navigation to handle potential navigation errors
3. **Keep** all other logic unchanged

**Test Scenarios**:

| Scenario | Auth State | Redirect Param | Expected Behavior |
|:---------|:-----------|:---------------|:------------------|
| User not authenticated | `false` | Any | Show login page (no redirect) |
| User authenticated, no redirect | `true` | `undefined` | Navigate to `/dashboard` |
| User authenticated, with redirect | `true` | `/projects` | Navigate to `/projects` |
| User authenticated, invalid redirect | `true` | `/invalid` | Try redirect, fallback to `/dashboard` on error |
| Token refresh failed (logout called) | `false` | Any | Show login page (state updated by store) |

### Step 5: Verify OAuth Callback Works with New Store

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`: OAuth callback handler

#### `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`

**Description**:
Review the OAuth callback to ensure it works correctly with the new auth store. The callback currently uses `storeTokens` from `useAuth`, which we've kept as an alias for `login`, so it should work without changes. This step is for verification only.

**Expected Code Pattern**:
```ts
// No modifications needed - this is verification only
// The callback should call storeTokens(tokens) after successful OAuth exchange
// storeTokens is now an alias for authStore.login()
// Example pattern to verify:
const { storeTokens } = useAuth();
// ... OAuth exchange logic ...
storeTokens(response.tokens); // This now calls authStore.login() under the hood
```

**Test Scenarios**:

| Scenario | OAuth Result | Expected Behavior | Store State Change |
|:---------|:-------------|:------------------|:-------------------|
| Successful OAuth callback | Valid tokens received | `storeTokens()` called → store.login() → redirect to dashboard | `isAuthenticated: true`, tokens stored |
| Failed OAuth callback | No tokens | Error shown, no store update | No change |

## Quality Checklist

- [x] Every function has concrete signature (no "something like X")
- [x] Detailed algorithm explanations are included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Execute Agent
- [x] All architectural decisions are made (using Zustand store pattern)
- [x] Execution order is clear:
  1. Create auth store (centralized state)
  2. Refactor transport to use store
  3. Refactor useAuth to use store
  4. Update auth page redirect logic
  5. Verify OAuth callback compatibility
- [x] Dependencies are explicit (uses existing Zustand, no new packages)
- [x] Solution addresses root cause (single source of truth for auth state)
- [x] Backward compatibility maintained (storeTokens alias)
- [x] Error handling specified (try-catch in navigation, error propagation in refresh)

## Expected Outcomes

After implementation:

1. **No more infinite redirect loops**: When token refresh fails, `logout()` is called which updates both localStorage and React state atomically via the store
2. **Consistent auth state**: All components using `useAuth` will receive synchronized state updates from the Zustand store
3. **Single source of truth**: The auth store is the only place managing authentication state and token storage
4. **Proper separation of concerns**: Transport service only handles token refresh logic, auth store handles state management
5. **Backward compatibility**: Existing code using `storeTokens` continues to work through the alias

## Testing Strategy

Manual testing steps after implementation:

1. **Initial login flow**:
   - Navigate to `/auth`
   - Click "Sign in with Google"
   - Complete OAuth flow
   - Verify redirect to dashboard
   - Verify `isAuthenticated` is true

2. **Token refresh success**:
   - Make API call with valid but soon-to-expire token
   - Verify token is refreshed silently
   - Verify `isAuthenticated` stays true
   - Verify UI doesn't flash or redirect

3. **Token refresh failure (the bug scenario)**:
   - Manually expire both access and refresh tokens in localStorage
   - Make API call or reload page
   - Verify token refresh is attempted
   - Verify refresh fails
   - Verify `logout()` is called
   - Verify redirect to `/auth`
   - **Verify no infinite redirect loop**
   - Verify `/auth` page shows login button
   - Verify `isAuthenticated` is false

4. **Logout flow**:
   - While authenticated, call `logout()`
   - Verify tokens are cleared from localStorage
   - Verify `isAuthenticated` becomes false
   - Verify redirect to `/auth` (if on protected route)

5. **Concurrent API calls during token refresh**:
   - Make multiple API calls simultaneously with expired token
   - Verify only one token refresh call is made
   - Verify all calls receive new token
   - Verify all calls complete successfully
