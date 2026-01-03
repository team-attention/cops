# Requirements

## Request Summary

Convert the current '/' route (which immediately redirects to '/dashboard') into a proper landing page for the C-Ops service. The landing page should explain what C-Ops is and provide authentication-aware navigation in the header. For authenticated users, the header displays a Dashboard button and Account icon button (with context menu for settings/logout). For unauthenticated users, the header displays a Login button that navigates to '/auth'.

## Acceptance Criteria

### Landing Page
- [ ] The '/' route displays a landing page (not a redirect to '/dashboard')
- [ ] Landing page uses a single full-screen hero section layout
- [ ] Hero section includes:
  - Headline describing C-Ops
  - Description text explaining the service
  - Call-to-action (CTA) button
- [ ] CTA button text and destination adapts to authentication state:
  - Authenticated: "Go to Dashboard" → '/dashboard'
  - Unauthenticated: "Get Started" or "Sign In" → '/auth'
- [ ] Landing page is accessible to both authenticated and unauthenticated users
- [ ] Landing page follows the existing design system (zinc color palette, glassmorphic effects, gradients, grid patterns)

### Header Component
- [ ] Header is positioned at the top of the landing page
- [ ] Header displays different content based on authentication state
- [ ] **Authenticated users** see:
  - Dashboard button that navigates to '/dashboard'
  - Account icon button (user circle icon) that opens a dropdown menu
  - Dropdown menu contains: "Account Settings" option and "Logout" option
- [ ] **Unauthenticated users** see:
  - Login button that navigates to '/auth'
- [ ] Header uses consistent styling with the design system

### Authentication Features
- [ ] Logout button in dropdown menu calls `useAuth().logout()` to clear tokens
- [ ] After logout, user remains on the landing page ('/')
- [ ] Landing page UI updates immediately after logout (shows unauthenticated state)
- [ ] Account Settings menu item navigates to '/settings'

### Settings Page
- [ ] Create a '/settings' route with a placeholder page
- [ ] Settings page displays "Coming soon" message
- [ ] Settings page uses the same layout as other authenticated pages (with sidebar)
- [ ] Settings page follows the existing design system

## Scope

### In Scope
- Creating a new landing page component at '/' route with single full-screen hero section
- Building a header component with authentication-aware navigation
- Implementing dropdown menu (using shadcn DropdownMenu) for authenticated user's Account button
- Adding logout functionality to the dropdown menu
- Writing landing page content that describes C-Ops (headline, description, CTA)
- Creating a '/settings' placeholder page with "Coming soon" message
- Ensuring all components follow the existing design system (dark theme, glassmorphic effects, gradients)
- Making the landing page accessible to both authenticated and unauthenticated users

### Out of Scope
- Full account settings page implementation (only creating placeholder)
- Modifying the existing '/dashboard' page
- Changes to the authentication flow ('/auth' routes)
- Changes to the sidebar layout used in other pages
- Multi-section landing page (features section, pricing, etc.)
- Adding new UI components beyond what's needed for header/landing page

## Constraints

- Must use existing authentication hook (`useAuth`) from `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`
- Must use TanStack Router for navigation (`useNavigate`, `Link`)
- Must use shadcn/ui DropdownMenu component for the Account button menu
- Must use shadcn/ui Button component for all buttons
- Landing page header should NOT include the sidebar - this is a standalone layout
- Settings page MUST use the existing sidebar layout (AppSidebar + SidebarProvider)
- Must maintain consistency with existing design patterns:
  - Dark theme (zinc-950 backgrounds)
  - Cyan/violet accent colors
  - Glassmorphic effects (backdrop-blur, semi-transparent backgrounds)
  - Grid patterns and gradient orbs for visual depth
  - Font mono for labels and technical text
- Landing page should be full-screen (min-h-screen)
- After logout, must stay on landing page ('/') without redirect

## Additional Context

**Current Implementation:**
- The '/' route currently uses `beforeLoad` to redirect to '/dashboard' (file: `/Users/jayce/team-attention/cops/web/src/route/index.tsx`)
- Authentication state is managed via localStorage tokens
- The existing `/auth` page already handles Google OAuth login
- Dashboard and other pages use a sidebar layout with `AppSidebar` component

**Required Dependencies:**
- shadcn/ui DropdownMenu component is NOT currently installed
- Will need to install DropdownMenu component using shadcn CLI before implementation
- Command: `npx shadcn@latest add dropdown-menu` (from `/Users/jayce/team-attention/cops/web` directory)

**Authentication Hook:**
- `useAuth()` returns: `{ isAuthenticated, logout, storeTokens }`
- `isAuthenticated` is `true` when access token exists in localStorage
- `logout()` clears all tokens from localStorage

**Design System:**
- Color palette: zinc (backgrounds), cyan (primary accent), violet (secondary accent)
- Typography: System fonts for UI, mono fonts for technical/label text
- Effects: Gradient orbs, grid patterns, glassmorphic cards with backdrop-blur
- Components: shadcn/ui with custom styling

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| What content should appear on the landing page? Should it explain what C-Ops is, its features, or include screenshots/demos? | Simple hero section with headline, description, and CTA button only |
| Should the landing page be a single full-screen hero section, or should it have multiple sections (hero, features, footer)? | Single full-screen hero section |
| For the authenticated user's Account button context menu, what should "Account Settings" link to? Should we create a placeholder route like '/settings'? | Create '/settings' placeholder route with "Coming soon" message |
| When a user logs out, where should they be redirected? Back to the landing page ('/') or to the auth page ('/auth')? | Stay on landing page ('/') - no redirect needed |
| Should the header be sticky (fixed to top) or scroll with the page? | Not explicitly specified - will use fixed/sticky header for better UX |
| Should the landing page be accessible to authenticated users, or should authenticated users automatically redirect to '/dashboard' (current behavior)? | Yes, landing page accessible to both authenticated and unauthenticated users |
| For the context menu component, should we use shadcn's DropdownMenu component? | Yes, use shadcn/ui DropdownMenu |
| What icon should we use for the Account button? A user circle icon, avatar, or something else? | User circle icon (will use lucide-react's User or UserCircle icon) |

## Implementation Notes

**Files to Create:**
1. `/Users/jayce/team-attention/cops/web/src/route/index.tsx` - Replace redirect with landing page component
2. `/Users/jayce/team-attention/cops/web/src/route/settings.tsx` - New settings placeholder page
3. `/Users/jayce/team-attention/cops/web/src/shared/component/landing-header.tsx` - New header component for landing page
4. `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/dropdown-menu.tsx` - Install via shadcn CLI

**Files to Reference for Design Patterns:**
- `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx` - For layout, styling, and gradient/grid patterns
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx` - For authentication state handling
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/overview-stats.tsx` - For card styling with glassmorphic effects

**Component Structure:**
```
Landing Page (index.tsx)
├── LandingHeader (authentication-aware)
│   ├── Logo/Brand
│   └── Navigation (conditional)
│       ├── Authenticated: Dashboard Button + Account Dropdown
│       └── Unauthenticated: Login Button
└── Hero Section
    ├── Headline
    ├── Description
    └── CTA Button (conditional text/destination)
```
