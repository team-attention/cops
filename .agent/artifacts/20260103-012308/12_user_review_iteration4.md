# Review Result

**Status**: Changes Required

## Request Summary

The current `/auth/login` page immediately redirects to Google OAuth without showing a login UI, causing errors. The user wants a proper authentication page at `/auth` that displays a "Sign in with Google" button, allowing users to initiate OAuth manually instead of automatic redirect.

## Acceptance Criteria

- [ ] Create `/auth` route (index page) with a "Sign in with Google" button UI
- [ ] Move the current `/auth/login.tsx` functionality to a new file or integrate into `/auth/index.tsx`
- [ ] The `/auth` page should display a login card with a Google sign-in button
- [ ] Button click should initiate the Google OAuth flow (not automatic redirect)
- [ ] Update `device.tsx` to redirect to `/auth` instead of `/auth/login`
- [ ] Update `callback.tsx` error handler to redirect to `/auth` instead of `/auth/login`

## Scope

### In Scope
- Create new `/auth/index.tsx` or `/auth.tsx` with login button UI
- Update redirect paths from `/auth/login` to `/auth`
- Keep Google OAuth initiation logic (but trigger on button click)

### Out of Scope
- Changes to OAuth callback logic
- Changes to token storage mechanism
- Backend authentication changes

## Current Implementation Analysis

### Files Reviewed

| File | Line(s) | Issue |
| ---- | ------- | ----- |
| `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx` | 34-66 | Auto-redirects to Google OAuth in useEffect without user interaction |
| `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx` | 24-27 | Redirects to `/auth/login` when unauthenticated |
| `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx` | 174 | Error "Try Again" button navigates to `/auth/login` |

### Current Behavior

1. **Login Page** (`/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx`):
   - Lines 34-66: `useEffect` immediately redirects to Google OAuth URL
   - Lines 68-85: Only shows a "Redirecting to Google..." loader
   - **Problem**: No user interaction required; causes errors if OAuth config is incorrect

2. **Device Page** (`/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`):
   - Lines 24-27: Uses `redirect({ to: '/auth/login', ... })` when not authenticated
   - **Problem**: Should redirect to `/auth` for the new login page

3. **Callback Page** (`/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`):
   - Line 174: "Try Again" button navigates to `/auth/login`
   - **Problem**: Should navigate to `/auth` for the new login page

## Violations Found

| File | Line | Issue | Suggested Fix |
| ---- | ---- | ----- | ------------- |
| `web/src/route/auth/login.tsx` | 34-66 | Auto-redirect to OAuth without user interaction | Create `/auth/index.tsx` with "Sign in with Google" button that triggers OAuth on click |
| `web/src/route/auth/device.tsx` | 25 | Redirects to `/auth/login` | Change redirect path from `/auth/login` to `/auth` |
| `web/src/route/auth/callback.tsx` | 174 | "Try Again" navigates to `/auth/login` | Change navigation from `/auth/login` to `/auth` |

## Implementation Guidance

### Step 1: Create `/auth/index.tsx` (NEW)

Create a new file `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx` with:

```tsx
import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { Shield } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';
import { Button } from '@/gen/shadcn/ui/button';
import { useAuth } from '@/shared/hook/use-auth';
import { useEffect } from 'react';

// Constants
const GOOGLE_OAUTH_CLIENT_ID = import.meta.env.VITE_GOOGLE_OAUTH_CLIENT_ID;
const GOOGLE_OAUTH_REDIRECT_URI = import.meta.env.VITE_GOOGLE_OAUTH_REDIRECT_URI;
const GOOGLE_OAUTH_SCOPES = 'openid email profile';
const GOOGLE_OAUTH_AUTHORIZE_URL = 'https://accounts.google.com/o/oauth2/v2/auth';

// AuthSearchParams defines search params for auth route
interface AuthSearchParams {
  redirect?: string;
}

// Route configuration
export const Route = createFileRoute('/auth/')({
  component: AuthPage,
  validateSearch: (search: Record<string, unknown>): AuthSearchParams => {
    return {
      redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
    };
  },
});

// AuthPage component displays login options
function AuthPage() {
  const search = useSearch({ from: '/auth/' });
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  // Redirect authenticated users
  useEffect(() => {
    if (isAuthenticated) {
      if (search.redirect) {
        navigate({ to: search.redirect });
        return;
      }
      navigate({ to: '/dashboard' });
    }
  }, [isAuthenticated, search.redirect, navigate]);

  // Handle Google sign-in button click
  const handleGoogleSignIn = () => {
    // Store redirect URL in sessionStorage if it exists
    if (search.redirect) {
      sessionStorage.setItem('cops_oauth_redirect', search.redirect);
    }

    // Build Google OAuth URL
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

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="text-center">
            <div className="flex justify-center mb-4">
              <div className="rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-3">
                <Shield className="h-8 w-8 text-cyan-400" />
              </div>
            </div>
            <CardTitle className="text-xl text-zinc-100">Sign in to C-Ops</CardTitle>
            <CardDescription className="text-zinc-500">
              Continue with your Google account
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={handleGoogleSignIn}
              className="w-full bg-white text-zinc-900 hover:bg-zinc-100"
            >
              {/* Google icon or text */}
              Sign in with Google
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
```

### Step 2: Update `device.tsx` Redirect Path

In `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`, change line 25:

**Before:**
```tsx
throw redirect({
  to: '/auth/login',
  search: { redirect: redirectUrl },
});
```

**After:**
```tsx
throw redirect({
  to: '/auth',
  search: { redirect: redirectUrl },
});
```

### Step 3: Update `callback.tsx` Error Navigation

In `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx`, change line 174:

**Before:**
```tsx
<Button
  onClick={() => navigate({ to: '/auth/login' })}
  className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
>
  Try Again
</Button>
```

**After:**
```tsx
<Button
  onClick={() => navigate({ to: '/auth' })}
  className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
>
  Try Again
</Button>
```

### Step 4: Remove or Deprecate `login.tsx`

Either:
1. **Delete** `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx` (if no longer needed)
2. **Or** keep it as a redirect to `/auth` for backward compatibility

If keeping for backward compatibility:
```tsx
import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/auth/login')({
  beforeLoad: ({ search }) => {
    throw redirect({
      to: '/auth',
      search: search as { redirect?: string },
    });
  },
  component: () => null,
});
```

## Expected User Flow After Changes

1. User visits `/auth/device?code=XXX` without being logged in
2. Route guard detects no token, redirects to `/auth?redirect=/auth/device?code=XXX`
3. User sees "Sign in to C-Ops" page with "Sign in with Google" button
4. User clicks button, initiating Google OAuth flow
5. After OAuth, callback processes token and redirects to `/auth/device?code=XXX`
6. User can now approve the device

## File Checklist

### New Files
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx` - New auth page with login button

### Modified Files
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx` - Update redirect path to `/auth`
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/callback.tsx` - Update "Try Again" navigation to `/auth`
- [ ] `/Users/jayce/team-attention/cops/web/src/route/auth/login.tsx` - Either delete or convert to redirect

## Additional Context

- Requirements document: `.agent/artifacts/20260103-012308/01_requirements.md`
- Plan document: `.agent/artifacts/20260103-012308/02_plan.md`
- Previous iteration plan: `.agent/artifacts/20260103-012308/00_user_plan_iteration3.md`
- User feedback: Auto-redirect to Google OAuth causes errors; need explicit login button

## Rules Applied

- [`.agent/rules/common.md`](.agent/rules/common.md) - General code rules
- [`.agent/rules/workflow.md`](.agent/rules/workflow.md) - Workflow rules
- [`.agent/rules/react/react-web.md`](.agent/rules/react/react-web.md) - TypeScript & React rules (named exports, type definitions)
- [`.agent/rules/react/react-web-src.md`](.agent/rules/react/react-web-src.md) - Feature-driven development structure, file naming conventions
