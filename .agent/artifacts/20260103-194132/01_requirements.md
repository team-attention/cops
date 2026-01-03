# Requirements

## Request Summary

Simplify the `/auth` page by removing the Card component wrapper and decorative elements (Shield icon, title, description). The simplified page should only display a centered "Sign in with Google" button on a dark background, while maintaining all existing authentication functionality including redirect parameter handling and automatic navigation for authenticated users.

## Acceptance Criteria

- [ ] Remove Card, CardHeader, CardContent, CardTitle, and CardDescription components
- [ ] Remove Shield icon and all decorative elements from the header
- [ ] Keep the full-screen centered layout with dark background (`bg-zinc-950`)
- [ ] Display only the "Sign in with Google" button centered on the page
- [ ] Preserve all existing authentication logic (Google OAuth flow, redirect handling, sessionStorage)
- [ ] Maintain automatic redirect behavior for already-authenticated users
- [ ] Keep the same button styling (white background, full width not needed anymore)
- [ ] Remove unused imports after simplification

## Scope

### In Scope
- Simplifying the UI by removing Card wrapper and decorative elements
- Adjusting button styling to work without the Card container
- Cleaning up unused component imports

### Out of Scope
- Changing the authentication logic or OAuth flow
- Modifying redirect parameter handling
- Changing the background color or overall layout structure
- Adding new authentication methods or features

## Constraints
- Must maintain all existing authentication functionality
- Should not break the OAuth flow or redirect behavior
- File location remains: `web/src/route/auth/index.tsx`

## Additional Context
- The auth page currently uses shadcn/ui Card components which are being removed
- The page already operates standalone without the dashboard sidebar layout
- The simplification focuses on visual presentation only, not functionality

## Questions Resolved

| Question                                                                 | Answer                                                                                                    |
| ------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| Should we keep the max-width container or make the button truly centered? | Remove the max-width container div as well - just center the button directly in the full-screen container |
| Should the button still be full-width or have auto width?               | Change to auto width (no `w-full` class) so it only takes up necessary space                             |
| Should we keep the white button styling?                                | Yes, keep the white background with dark text for good contrast against the dark background              |
