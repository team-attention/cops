# Implementation Plan: Create Auth Landing Page with Google Sign-In Button

## Overview

Replace the auto-redirect login flow (`/auth/login`) with a user-initiated login flow at `/auth`. The current implementation redirects users to Google OAuth immediately without user interaction, causing errors. The new implementation will display a "Sign in with Google" button that users must click to initiate the OAuth flow.

## Implementation Steps

### Step 1: Create Auth Landing Page (`/auth/index.tsx`)

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx`: Contains OAuth URL building logic and environment variable references to reuse
- `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`: Reference for page layout structure and card styling
- `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx`: Reference for button styling patterns
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Reference for authentication hook usage
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript and React component naming rules
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Feature-driven development structure and file naming conventions

#### `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`

**Description**:
Create a new auth landing page that displays a "Sign in with Google" button. When clicked, the button triggers OAuth flow. The page follows the same visual style as other auth pages (dark theme with cyan accents).

```tsx
// Constants for OAuth configuration
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// AuthSearchParams defines search params for auth route
interface AuthSearchParams {
  redirect?: string;
}

// Route configuration
// Route path: '/auth/'
// Component: AuthPage
// validateSearch: Returns AuthSearchParams with optional redirect param

// AuthPage displays the auth landing page with Google sign-in button
function AuthPage() {
  // 1. Get search params using useSearch({ from: '/auth/' })
  // 2. Get navigate function using useNavigate()
  // 3. Get isAuthenticated from useAuth()

  // 4. useEffect to redirect authenticated users:
  //    a. If isAuthenticated is true:
  //       - If search.redirect exists, navigate to search.redirect
  //       - Otherwise, navigate to '/dashboard'

  // 5. Define handleGoogleSignIn function:
  //    a. If search.redirect exists:
  //       - Store it in sessionStorage with key 'cops_oauth_redirect'
  //    b. Build URLSearchParams with:
  //       - response_type: 'code'
  //       - client_id: GOOGLE_OAUTH_CLIENT_ID
  //       - redirect_uri: GOOGLE_OAUTH_REDIRECT_URI
  //       - scope: GOOGLE_OAUTH_SCOPES
  //       - access_type: 'offline'
  //       - prompt: 'consent'
  //    c. Build OAuth URL: `${GOOGLE_OAUTH_AUTHORIZE_URL}?${params.toString()}`
  //    d. Redirect browser: window.location.href = oauthUrl

  // 6. Return JSX:
  //    - Outer div: className="flex min-h-screen items-center justify-center bg-zinc-950"
  //    - Inner div: className="w-full max-w-md px-4"
  //    - Card: className="border-zinc-800 bg-zinc-900"
  //      - CardHeader: className="text-center"
  //        - Icon container div: className="flex justify-center mb-4"
  //          - Icon wrapper div: className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-3"
  //            - Shield icon: className="h-8 w-8 text-cyan-400"
  //        - CardTitle: className="text-xl text-zinc-100", text="Sign in to C-Ops"
  //        - CardDescription: className="text-zinc-500", text="Continue with your Google account"
  //      - CardContent:
  //        - Button:
  //          - onClick={handleGoogleSignIn}
  //          - className="w-full bg-white text-zinc-900 hover:bg-zinc-100"
  //          - text="Sign in with Google"
}
```

**Imports Required**:
```tsx
import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { Shield } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';
import { Button } from '@/gen/shadcn/ui/button';
import { useAuth } from '@/shared/hook/use-auth';
import { useEffect } from 'react';
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unauthenticated user visits `/auth` | No token in localStorage | Display sign-in button | Happy path |
| Authenticated user visits `/auth` | Valid token in localStorage | Redirect to `/dashboard` | Authenticated redirect (no redirect param) |
| Authenticated user with redirect param | Valid token, `redirect=/auth/device?code=ABC` | Redirect to `/auth/device?code=ABC` | Authenticated redirect (with redirect param) |
| User clicks "Sign in with Google" | Button click | Redirect to Google OAuth URL | OAuth initiation |
| User clicks button with redirect param | Button click, `redirect=/auth/device?code=ABC` | Store redirect in sessionStorage, then redirect to Google | OAuth with stored redirect |

---

### Step 2: Update Device Page Redirect Path

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`: Current implementation to modify

#### `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`

**Description**:
Change the redirect path from `/auth/login` to `/auth` when user is not authenticated.

**Changes**:
- Line 25: Change redirect `to` value from `/auth/login` to `/auth`

**Before**:
```tsx
throw redirect({
  to: '/auth/login',
  search: { redirect: redirectUrl },
});
```

**After**:
```tsx
throw redirect({
  to: '/auth',
  search: { redirect: redirectUrl },
});
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unauthenticated user visits `/auth/device?code=ABC` | No token in localStorage | Redirect to `/auth?redirect=/auth/device?code=ABC` | Unauthenticated redirect |
| Authenticated user visits `/auth/device?code=ABC` | Valid token in localStorage | Display device approval page | Authenticated access |

---

### Step 3: Update Callback Page Error Navigation

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`: Current implementation to modify

#### `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`

**Description**:
Change the "Try Again" button navigation from `/auth/login` to `/auth` when OAuth callback fails.

**Changes**:
- Line 174: Change navigate `to` value from `/auth/login` to `/auth`

**Before**:
```tsx
<Button
  onClick={() => navigate({ to: '/auth/login' })}
  className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
>
  Try Again
</Button>
```

**After**:
```tsx
<Button
  onClick={() => navigate({ to: '/auth' })}
  className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
>
  Try Again
</Button>
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| OAuth callback fails | Error from Google OAuth | Display error card with "Try Again" button | Error state |
| User clicks "Try Again" | Button click | Navigate to `/auth` | Error recovery navigation |
| OAuth callback succeeds | Valid authorization code | Exchange code for tokens, redirect to stored redirect or dashboard | Success state |

---

### Step 4: Delete Old Login Page

**Files to Delete**:
- `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx`: No longer needed, replaced by `/auth/index.tsx`

**Description**:
Remove the old login page that auto-redirected to Google OAuth. The functionality is now handled by the new `/auth/index.tsx` with explicit user interaction.

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| User visits `/auth/login` after deletion | Direct navigation | TanStack Router 404 or error (expected behavior) | Old route removed |
| User visits `/auth` | Direct navigation | Display new auth landing page | New route active |

---

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation is included as comments in function bodies
- [x] Every function/component has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (using existing shadcn/ui components and TanStack Router)
- [x] Execution order is clear:
  1. Create `/auth/index.tsx` (new page)
  2. Update `device.tsx` redirect path
  3. Update `callback.tsx` navigation
  4. Delete `login.tsx`

## Expected User Flow After Implementation

1. User visits `/auth/device?code=XXX` without being authenticated
2. `device.tsx` beforeLoad guard redirects to `/auth?redirect=/auth/device?code=XXX`
3. User sees "Sign in to C-Ops" page with "Sign in with Google" button
4. User clicks button → redirect param stored in sessionStorage → browser redirects to Google OAuth
5. After OAuth approval, Google redirects to `/auth/callback?code=YYY`
6. Callback page exchanges code for tokens, stores in localStorage
7. Callback page retrieves stored redirect from sessionStorage and navigates to `/auth/device?code=XXX`
8. User can now approve the device

## Visual Design Consistency

The new auth page follows the existing design patterns:

- **Layout**: Centered card on dark background (matches `device.tsx`)
- **Color scheme**:
  - Background: `bg-zinc-950`
  - Card: `border-zinc-800 bg-zinc-900`
  - Accent color: Cyan (`text-cyan-400`, `border-cyan-500/20`)
  - Text: `text-zinc-100` (titles), `text-zinc-500` (descriptions)
- **Button style**: White background with dark text (Google branding convention)
- **Icons**: Shield icon from lucide-react (consistent with other auth pages)
