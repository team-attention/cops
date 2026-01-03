# Review Result

**Status**: Changes Required

## Request Summary

The `/auth/device?code=XXX` page currently shows "You must be logged in to approve devices" message when the user is not logged in. This is a poor user experience because the user must manually navigate to a login page and then return to the device approval URL. The expected behavior is automatic redirect to login with return URL preservation.

## Acceptance Criteria

- [ ] Check authentication state on page load in `/auth/device` route or `DeviceApproval` component
- [ ] If user is not authenticated, redirect to login page (e.g., Google OAuth flow)
- [ ] Preserve the current URL (`/auth/device?code=XXX`) as the redirect destination after login
- [ ] After successful login, automatically return to `/auth/device?code=XXX`
- [ ] Then allow the user to click the "Approve Device" button

## Scope

### In Scope
- Implement authentication state check in the device approval flow
- Implement redirect to login with return URL parameter
- Handle post-login redirect back to device approval page
- Create authentication hook if one does not exist

### Out of Scope
- Changes to the backend authentication logic
- Changes to the device approval API endpoint
- Implementing a full login page (use existing OAuth flow)

## Current Implementation Analysis

### Files Reviewed

| File | Status | Notes |
| ---- | ------ | ----- |
| `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx` | Issue Found | No auth check or redirect logic |
| `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx` | Issue Found | Shows error message instead of redirecting |
| `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` | Reference | Shows auth token stored in `localStorage` as `cops_access_token` |
| `/Users/jayce/team-attention/cops/web/src/route/__root.tsx` | Reference | Root layout - no auth context provider found |
| `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-user.tsx` | Reference | Static user display - no actual auth state |

### Current Behavior

1. **Device Approval Page** (`/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`):
   - Line 13-20: Route defined with `createFileRoute`, validates `code` search param
   - Line 22-71: Component renders `DeviceApproval` when code is present
   - **Issue**: No authentication check before rendering

2. **DeviceApproval Component** (`/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx`):
   - Line 43-44: Catches `Code.Unauthenticated` error after API call
   - Line 83: Maps `UNAUTHORIZED` error to message "You must be logged in to approve devices"
   - **Issue**: Error is only caught AFTER user clicks "Approve Device" button, not on page load

3. **Authentication Storage** (`/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`):
   - Line 7: Token read from `localStorage.getItem('cops_access_token')`
   - Token is attached to requests via interceptor
   - **Note**: Auth state can be determined by checking if token exists

### Missing Components

1. **No Auth Context/Hook**: The web app lacks a shared authentication hook (e.g., `useAuth`) to check login status
2. **No Login Route**: No login page or route exists (`/login` or `/auth/login`)
3. **No Redirect Logic**: No mechanism to preserve return URL and redirect after login
4. **No OAuth Callback Route**: If Google OAuth is used, a callback route may be needed

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
| ---- | ---- | ---- | ----- | ------------- |
| `web/src/route/auth/device.tsx` | 13-20 | UX Best Practice | Route renders protected content without authentication check | Add `beforeLoad` guard to check auth and redirect if not authenticated |
| `web/src/feature/auth/component/device-approval.tsx` | 78-103 | UX Best Practice | Shows error message for unauthenticated users instead of redirecting | Remove static error display; handle at route level with redirect |

## Implementation Guidance

### Option 1: Route-Level Auth Guard (Recommended)

TanStack Router supports `beforeLoad` for route guards:

```tsx
// web/src/route/auth/device.tsx
export const Route = createFileRoute('/auth/device')({
  beforeLoad: ({ location }) => {
    const token = localStorage.getItem('cops_access_token');
    if (!token) {
      // Redirect to login with return URL
      throw redirect({
        to: '/auth/login',
        search: {
          redirect: location.href,
        },
      });
    }
  },
  component: DeviceApprovalPage,
  validateSearch: (search: Record<string, unknown>): DeviceSearchParams => {
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
    };
  },
});
```

### Option 2: Component-Level Check

If route guards are not preferred, check auth in the component:

```tsx
// web/src/route/auth/device.tsx
import { useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect } from 'react';

function DeviceApprovalPage() {
  const navigate = useNavigate();
  const search = useSearch({ from: '/auth/device' });
  const code = search.code;

  useEffect(() => {
    const token = localStorage.getItem('cops_access_token');
    if (!token) {
      // Redirect to login with return URL
      navigate({
        to: '/auth/login',
        search: {
          redirect: `/auth/device?code=${code}`,
        },
      });
    }
  }, [navigate, code]);

  // ... rest of component
}
```

### Required New Files

1. **Auth Hook** (`web/src/shared/hook/use-auth.ts`):
   - Provide `isAuthenticated` state
   - Provide `login()` and `logout()` functions
   - Read/write `cops_access_token` from localStorage

2. **Login Route** (`web/src/route/auth/login.tsx`):
   - Handle `redirect` search param
   - Initiate Google OAuth flow
   - After successful auth, redirect to stored return URL

3. **OAuth Callback Route** (`web/src/route/auth/callback.tsx`):
   - Handle Google OAuth callback
   - Exchange code for tokens via `googleAuth` RPC
   - Store tokens in localStorage
   - Redirect to original destination

## Architecture Alignment

Per `.agent/rules/react/react-web-src.md`:
- Auth hook should be placed in `shared/hook/use-auth.ts`
- Auth routes should be in `route/auth/` directory
- Use TanStack Router's built-in redirect functionality

## Additional Context

- Requirements document: `.agent/artifacts/20260103-012308/01_requirements.md`
- Plan document: `.agent/artifacts/20260103-012308/02_plan.md`
- User feedback indicates critical UX issue blocking CLI device flow authentication

## Rules Applied

- [`.agent/rules/common.md`](.agent/rules/common.md) - General code rules
- [`.agent/rules/workflow.md`](.agent/rules/workflow.md) - Workflow rules
- [`.agent/rules/react/react-web.md`](.agent/rules/react/react-web.md) - TypeScript & React rules
- [`.agent/rules/react/react-web-src.md`](.agent/rules/react/react-web-src.md) - Feature-driven development structure

## Summary

The device approval page needs to implement proper authentication flow:

1. **Current**: User visits `/auth/device?code=XXX` -> Clicks "Approve" -> Sees error message
2. **Expected**: User visits `/auth/device?code=XXX` -> Redirected to login -> Logs in -> Returns to `/auth/device?code=XXX` -> Clicks "Approve" -> Success

This requires:
1. Creating an auth hook to check authentication state
2. Creating a login route that initiates OAuth flow
3. Creating an OAuth callback route to handle token exchange
4. Adding auth guards to the device approval route
5. Preserving the return URL through the OAuth flow
