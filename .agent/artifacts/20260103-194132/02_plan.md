# Implementation Plan

## Overview

This plan simplifies the `/auth` page by removing the Card component wrapper and all decorative elements (Shield icon, title, description). The resulting page will display only a centered "Sign in with Google" button on a dark background (`bg-zinc-950`), while preserving all existing authentication functionality including Google OAuth flow, redirect parameter handling, and automatic navigation for authenticated users.

## Package Changes

None. This is a UI simplification task that removes unused components without requiring new dependencies.

## Implementation Steps

### Step 1: Simplify AuthPage Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Component export conventions and TypeScript rules
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Feature structure and import conventions

#### `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`

**Description**:
Remove Card-related imports and Shield icon, simplify the JSX to render only the centered button, and update import statements to remove unused dependencies.

**Changes Required**:

1. **Remove unused imports** (lines 3-4):
   - Remove: `import { Shield } from 'lucide-react';`
   - Remove: `import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';`

2. **Keep required imports** (lines 1-2, 5-6):
   - Keep: `import { useEffect } from 'react';`
   - Keep: `import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';`
   - Keep: `import { Button } from '@/gen/shadcn/ui/button';`
   - Keep: `import { useAuth } from '@/shared/hook/use-auth';`

3. **Simplify the return JSX** (lines 70-96):
   - Keep the outer `<div>` with `flex min-h-screen items-center justify-center bg-zinc-950` classes
   - Remove the inner `<div className="w-full max-w-md px-4">` wrapper
   - Remove the entire `<Card>` component and its children (`CardHeader`, `CardContent`, `CardTitle`, `CardDescription`)
   - Remove the Shield icon and its container
   - Keep only the `<Button>` component, placed directly inside the outer container
   - Update Button: Remove `w-full` class, keep `bg-white text-zinc-900 hover:bg-zinc-100`

**Final Implementation Structure**:

```tsx
import { useEffect } from 'react';
import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { Button } from '@/gen/shadcn/ui/button';
import { useAuth } from '@/shared/hook/use-auth';

// Constants (unchanged)
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// AuthSearchParams defines search params for auth route (unchanged)
interface AuthSearchParams {
  redirect?: string;
}

// Route configuration (unchanged)
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
  const search = useSearch({ from: '/auth/' });
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    // Implementation: Check authentication and redirect
    // 1. If user is already authenticated:
    //    a. If redirect param exists, navigate to redirect URL
    //    b. Otherwise, navigate to dashboard
  }, [isAuthenticated, search.redirect, navigate]);

  const handleGoogleSignIn = () => {
    // Implementation: Initiate Google OAuth flow
    // 1. Store redirect URL in sessionStorage if it exists
    // 2. Build Google OAuth URL with required parameters
    // 3. Redirect browser to Google OAuth URL
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <Button
        onClick={handleGoogleSignIn}
        className="bg-white text-zinc-900 hover:bg-zinc-100"
      >
        Sign in with Google
      </Button>
    </div>
  );
}
```

**Preserved Logic** (no changes required):
- All constants (`GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_REDIRECT_URI`, `GOOGLE_OAUTH_SCOPES`, `GOOGLE_OAUTH_AUTHORIZE_URL`)
- `AuthSearchParams` interface
- `Route` configuration with `validateSearch`
- `useEffect` hook for authenticated user redirect
- `handleGoogleSignIn` function with sessionStorage and OAuth URL construction

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unauthenticated user visits page | `isAuthenticated = false`, no redirect param | Displays centered "Sign in with Google" button on dark background | Default render |
| Unauthenticated user clicks sign-in | Click button, no redirect param | Redirects to Google OAuth URL, no sessionStorage entry | OAuth flow without redirect |
| Unauthenticated user clicks sign-in with redirect | Click button, `?redirect=/dashboard/sessions` | Stores redirect in sessionStorage, redirects to Google OAuth URL | OAuth flow with redirect |
| Authenticated user visits page | `isAuthenticated = true`, no redirect param | Automatically redirects to `/dashboard` | useEffect redirect to dashboard |
| Authenticated user visits page with redirect | `isAuthenticated = true`, `?redirect=/custom-page` | Automatically redirects to `/custom-page` | useEffect redirect to custom URL |
| Button styling | Visual inspection | White background, dark text, auto-width (not full-width) | CSS styling |
| Page layout | Visual inspection | Full-screen dark background, button centered horizontally and vertically | CSS layout |

## Summary of Changes

| Line Range | Action | Description |
| :--------- | :----- | :---------- |
| 3 | Remove | `import { Shield } from 'lucide-react';` |
| 4 | Remove | `import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';` |
| 72 | Remove | `<div className="w-full max-w-md px-4">` wrapper |
| 73-84 | Remove | Entire `<Card>` component with `CardHeader`, Shield icon, `CardTitle`, `CardDescription` |
| 85-92 | Modify | Keep `<Button>` but remove `w-full` class and move directly under outer div |
| 93-95 | Remove | Closing tags for `CardContent`, `Card`, and inner div |

## Quality Checklist

- [x] Every function has a concrete signature
- [x] Algorithm explanation included as comments in function bodies
- [x] Test scenarios cover all branches (authenticated/unauthenticated, with/without redirect)
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All decisions are final (no package selection needed)
- [x] Execution order is clear (single file modification)
