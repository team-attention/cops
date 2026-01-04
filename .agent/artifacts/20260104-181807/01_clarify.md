# Requirements

## Request Summary

Fix a bug in the Device Login Flow where the redirect URL is malformed due to incorrect string concatenation. After completing device authentication, users are redirected to `http://localhost:3000/auth/device[object%20Object]` instead of the correct URL with query parameters. The root cause is that `location.search` (a SearchParams object) is being concatenated directly to a string, resulting in `[object Object]` being added to the URL.

## Acceptance Criteria

- [ ] The redirect URL in `/web/src/route/auth/device.tsx` correctly includes query parameters as a string
- [ ] After device approval, users are redirected back to the device page with the correct query parameters intact
- [ ] The beforeLoad logic properly constructs redirect URLs with both pathname and search params
- [ ] No `[object Object]` appears in any redirect URLs during the device login flow

## Scope

### In Scope
- Fix the redirect URL construction in `/web/src/route/auth/device.tsx` beforeLoad hook
- Ensure `location.search` is properly converted to a string before concatenation

### Out of Scope
- Changes to other authentication flows (Google OAuth, callback handling)
- Modifications to the device approval component or logic
- Changes to the API or backend device code validation

## Constraints

- Must maintain compatibility with TanStack Router's location object structure
- Must preserve existing authentication flow behavior
- Must not break redirect functionality for authenticated users

## Additional Context

### Bug Location
File: `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`
Lines: 27-32

### Current Code (Buggy)
```typescript
const redirectUrl = location.pathname + location.search

throw redirect({
  to: '/auth',
  search: { redirect: redirectUrl },
})
```

### Issue Explanation
In TanStack Router, `location.search` is a `SearchParams` object, not a string. When concatenating an object to a string using the `+` operator, JavaScript calls the object's `toString()` method, which for most objects returns `[object Object]`.

### Expected Behavior
The redirect URL should be constructed as a proper URL string with query parameters, e.g., `/auth/device?code=ABC123`.

### Similar Patterns in Codebase
The `/web/src/route/auth/index.tsx` file shows the correct pattern for handling redirects with search params:
- Line 72-73: `sessionStorage.setItem('cops_oauth_redirect', search.redirect)` stores redirect as string
- The auth flow properly handles redirect strings throughout

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Is the bug confirmed in the device.tsx beforeLoad hook? | Yes, line 27 concatenates `location.search` object to string causing `[object Object]` |
| Should we fix only this specific occurrence? | Yes, this is the only occurrence causing the reported bug in the device login flow |
| Should the fix preserve query parameters? | Yes, the full URL with query parameters must be preserved for the redirect to work correctly |
| Are there any other similar bugs in the auth flow? | No, other routes use proper string handling for redirects |
