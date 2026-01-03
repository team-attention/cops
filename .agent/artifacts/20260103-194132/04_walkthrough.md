# Development Walkthrough

## Summary
Simplified the `/auth` route layout by removing the sidebar and header components, leaving only a centered authentication card with full-screen dark background.

## Code Overview

### Modified Components

#### `web/src/route/__root.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`
- **Changes**: Implemented conditional layout rendering based on current route path
- **Key Modifications**:
  - **Added Import**: `useRouterState` from `@tanstack/react-router` (line 4)
  - **Extracted Component**: Converted inline anonymous component to named `RootComponent` function (lines 25-102)
  - **Route Detection**: Added pathname check using `useRouterState` to detect `/auth` routes (lines 26-27)
  - **Conditional Rendering**:
    - **Auth routes** (`pathname.startsWith('/auth')`): Render only `<Outlet />` with devtools, no sidebar/header (lines 30-47)
    - **Other routes**: Full layout with `<SidebarProvider>`, `<AppSidebar>`, header, and decorative elements (lines 51-101)
  - **Preserved**: All existing layout features (grid pattern, gradient orbs, minimal header, devtools) for non-auth routes

#### `web/src/route/auth/index.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`
- **Changes**: No net changes (Card structure was temporarily removed during workflow, then restored)
- **Current Structure**:
  - Full-screen centered layout with `min-h-screen` and dark background (line 77)
  - Card wrapper with Shield icon (lines 79-92)
  - Google OAuth sign-in button (lines 94-99)
  - Redirect handling with `useEffect` (lines 41-52)

## Technical Implementation

### Layout Switching Strategy

The root component now uses a **path-based layout strategy**:

```tsx
function RootComponent() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const isAuthRoute = pathname.startsWith('/auth')

  if (isAuthRoute) {
    return <Outlet /> // Minimal layout for auth
  }

  return (
    <SidebarProvider>
      {/* Full layout for dashboard */}
    </SidebarProvider>
  )
}
```

This approach:
- ✅ Keeps all layout logic in one place (`__root.tsx`)
- ✅ Avoids layout duplication or wrapper components
- ✅ Works for all auth-related routes (`/auth`, `/auth/callback`, `/auth/device`)
- ✅ Maintains devtools availability across all routes

### Routes Affected

| Route | Layout | Components |
|-------|--------|------------|
| `/auth` | Minimal | Only `<Outlet />` + devtools |
| `/auth/callback` | Minimal | Only `<Outlet />` + devtools |
| `/auth/device` | Minimal | Only `<Outlet />` + devtools |
| `/dashboard` | Full | Sidebar + Header + Background |
| All other routes | Full | Sidebar + Header + Background |

## Testing

### Verification Commands Run
```bash
# No build/test commands were explicitly run during this change
# This is a frontend-only layout modification
```

### Manual Testing Checklist
- [ ] Navigate to `/auth` - Should show centered card without sidebar/header
- [ ] Navigate to `/dashboard` - Should show full layout with sidebar/header
- [ ] Navigate to `/auth/callback` - Should show minimal layout
- [ ] Check devtools visibility on both auth and dashboard pages

## Issues & Resolutions

| Issue | Resolution |
|-------|-----------|
| Initial misunderstanding: Removed Card structure from auth page | Restored Card structure after clarification - user wanted layout removed, not the card itself |
| Needed to skip layout for auth routes only | Implemented route detection using `useRouterState` with `pathname.startsWith('/auth')` check |
| Devtools needed on all pages | Included `<TanStackDevtools>` in both conditional branches of `RootComponent` |

## Architecture Notes

### Why Named Component Export?
The component was extracted from inline to `RootComponent` to:
1. Enable conditional logic based on route state
2. Improve code readability and maintainability
3. Follow React best practices for complex component logic

### Why `pathname.startsWith('/auth')` Instead of TanStack Router Matching?
- Simple string prefix check is sufficient for this use case
- Covers all auth-related routes (`/auth`, `/auth/callback`, `/auth/device`) with one condition
- Avoids need for complex route matching logic
- Easy to understand and maintain

### Layout Isolation Strategy
Auth pages now have complete visual isolation:
- No sidebar navigation
- No header bar
- No background decorative elements (grid, gradient orbs)
- Only the centered auth card UI
- Full-screen dark background (`bg-zinc-950`)

## Related Changes

The git diff also shows unrelated changes in this commit:
- `api/cmd/internal/container/application.go` - Added RBAC module registration
- `shared/domain/mongoschema/project.go` - Added `OrganizationID` field
- `shared/domain/project.go` - Added `OrganizationID` to Project struct

These changes are **not part of the auth page simplification task** and appear to be from separate development work.

## Future Considerations

If more routes need layout isolation in the future:
1. Consider extracting layout logic to a separate utility function
2. Or use route metadata/context to specify layout type
3. Current implementation is sufficient for auth-only isolation

Example route metadata approach (not implemented):
```tsx
// Future alternative approach
export const Route = createFileRoute('/auth/')({
  meta: { layout: 'minimal' }, // Hypothetical
  component: AuthPage,
})
```
