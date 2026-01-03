# Implementation Plan: Landing Page with Authentication-Aware Header

## Overview

This implementation plan converts the current '/' route from a redirect to '/dashboard' into a proper landing page for C-Ops. The landing page will feature a full-screen hero section explaining C-Ops and an authentication-aware header. Authenticated users see a Dashboard button and Account dropdown menu (with Settings and Logout options). Unauthenticated users see a Login button. Additionally, a placeholder '/settings' route will be created.

The landing page uses a standalone layout (no sidebar) while the settings page uses the existing sidebar layout consistent with other authenticated pages.

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | Need dropdown menu for Account button | `dropdown-menu` (shadcn component) | shadcn/ui DropdownMenu provides accessible, styled dropdown menu. Install via `npx shadcn@latest add dropdown-menu` from `/Users/jayce/team-attention/cops/web` directory. |

## Step 1: Install shadcn DropdownMenu Component

**Command to Execute**:
```bash
cd /Users/jayce/team-attention/cops/web && npx shadcn@latest add dropdown-menu
```

This will generate the file `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/dropdown-menu.tsx` with all necessary DropdownMenu components.

**Expected Exports** (based on shadcn documentation):
- `DropdownMenu`
- `DropdownMenuTrigger`
- `DropdownMenuContent`
- `DropdownMenuItem`
- `DropdownMenuSeparator`
- `DropdownMenuLabel` (optional, may not be needed)

---

## Step 2: Create Landing Header Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript/React component rules
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Directory structure and naming conventions
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Authentication hook API
- `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/button.tsx`: Button component usage
- `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-header.tsx`: Logo/brand styling reference

### `/Users/jayce/team-attention/cops/web/src/shared/component/landing-header.tsx`

**Description**:
Create a header component for the landing page that displays different navigation based on authentication state. Authenticated users see Dashboard button and Account dropdown. Unauthenticated users see Login button.

```tsx
import { Link, useNavigate } from '@tanstack/react-router'
import { Terminal, UserCircle, Settings, LogOut } from 'lucide-react'
import { Button } from '@/gen/shadcn/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import { useAuth } from '@/shared/hook/use-auth'

// LandingHeader displays authentication-aware navigation at top of landing page.
// Authenticated: Dashboard button + Account dropdown (Settings, Logout)
// Unauthenticated: Login button
export const LandingHeader = () => {
  // Implementation outline:
  // 1. Get authentication state from useAuth hook.
  // 2. Get navigate function from useNavigate hook.
  // 3. Define handleLogout function:
  //    a. Call logout() from useAuth.
  //    b. No navigation needed (stay on landing page).
  // 4. Render header container with fixed positioning.
  // 5. Render logo/brand section on the left:
  //    a. Use cyan accent color for icon container.
  //    b. Display "C-OPS" text with styling.
  // 6. Render navigation section on the right:
  //    a. If authenticated:
  //       - Render Dashboard button (Link to '/dashboard').
  //       - Render Account dropdown:
  //         * Trigger: icon button with UserCircle icon.
  //         * Content: DropdownMenuContent aligned to end.
  //         * Items: "Account Settings" (navigates to '/settings'), separator, "Logout" (calls handleLogout).
  //    b. If not authenticated:
  //       - Render Login button (Link to '/auth').
}
```

**Component Structure**:
```
LandingHeader
├── Container (fixed top, z-50, backdrop-blur)
│   ├── Logo Section (left)
│   │   ├── Icon container (cyan accent, Terminal icon)
│   │   └── Brand text "C-OPS"
│   └── Navigation Section (right)
│       ├── [Authenticated]
│       │   ├── Dashboard Button (Link to /dashboard)
│       │   └── Account Dropdown
│       │       ├── Trigger (UserCircle icon button)
│       │       └── Content
│       │           ├── "Account Settings" item (navigates to /settings)
│       │           ├── Separator
│       │           └── "Logout" item (calls logout)
│       └── [Unauthenticated]
│           └── Login Button (Link to /auth)
```

**Styling Requirements**:
- Header: `fixed top-0 left-0 right-0 z-50 h-16 border-b border-zinc-800/50 bg-zinc-950/80 backdrop-blur-xl`
- Inner container: `mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 h-full flex items-center justify-between`
- Logo icon container: `rounded-lg border border-cyan-500/20 bg-zinc-900/80 p-2`
- Terminal icon: `h-5 w-5 text-cyan-400`
- Brand text: `font-mono text-lg font-bold tracking-wider text-zinc-100`
- Dashboard button: Use `variant="ghost"` with `text-zinc-300 hover:text-zinc-100 hover:bg-zinc-800/50`
- Account dropdown trigger: Use `variant="ghost" size="icon"` with `text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800/50`
- UserCircle icon: `h-5 w-5`
- DropdownMenuContent: `w-48 border-zinc-800 bg-zinc-900` with `align="end"`
- DropdownMenuItem: Default styling from shadcn, icons `h-4 w-4 mr-2`
- Logout item: Add `text-red-400 focus:text-red-300 focus:bg-red-500/10` for destructive styling

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Authenticated user | `isAuthenticated = true` | Shows Dashboard button + Account dropdown | Authenticated branch |
| Unauthenticated user | `isAuthenticated = false` | Shows Login button only | Unauthenticated branch |
| Click Dashboard button | User clicks Dashboard | Navigates to /dashboard | Navigation |
| Click Login button | User clicks Login | Navigates to /auth | Navigation |
| Click Account Settings | User clicks menu item | Navigates to /settings | Dropdown navigation |
| Click Logout | User clicks Logout | Calls logout(), stays on page, UI updates to unauthenticated state | Logout handler |

---

## Step 3: Create Landing Page Route

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Component rules
- `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`: Page layout and styling patterns
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`: Authentication state handling
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/overview-stats.tsx`: Glassmorphic card styling
- `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`: Background decorations (grid, orbs)

### `/Users/jayce/team-attention/cops/web/src/route/index.tsx`

**Description**:
Replace the current redirect with a full landing page component featuring the LandingHeader and a hero section with headline, description, and CTA button that adapts based on authentication state.

**IMPORTANT**: The landing page must NOT use the sidebar layout from `__root.tsx`. The root layout wraps all routes with SidebarProvider and AppSidebar. To create a standalone landing page without sidebar, the index route needs to render its own full-screen layout that visually overrides/ignores the sidebar context, OR the `__root.tsx` needs to be modified to conditionally render the sidebar based on the current route.

**Recommended Approach**: Modify the landing page to render a full-screen overlay that covers the sidebar, using `fixed inset-0 z-50` positioning. This avoids modifying the root layout.

```tsx
import { createFileRoute, Link } from '@tanstack/react-router'
import { Terminal, ArrowRight } from 'lucide-react'
import { Button } from '@/gen/shadcn/ui/button'
import { LandingHeader } from '@/shared/component/landing-header'
import { useAuth } from '@/shared/hook/use-auth'

export const Route = createFileRoute('/')({
  component: LandingPage,
})

// LandingPage displays the C-Ops landing page with hero section.
// Adapts CTA button text and destination based on authentication state.
function LandingPage() {
  // Implementation outline:
  // 1. Get isAuthenticated from useAuth hook.
  // 2. Render fixed full-screen container to overlay sidebar (fixed inset-0 z-50).
  // 3. Set background to bg-zinc-950 to cover underlying content.
  // 4. Add background decorations:
  //    a. Grid pattern overlay (violet accent, low opacity).
  //    b. Gradient orbs (violet top-left, cyan bottom-right, amber center-right).
  // 5. Render LandingHeader component.
  // 6. Render main content area with flex centering (account for header height with pt-16).
  // 7. Render hero section centered:
  //    a. Icon container with Terminal icon (cyan glow effect).
  //    b. Headline: "Track Your Claude Code Sessions" (large, bold).
  //    c. Subtitle: "C-OPS // Code Agent Operations" (mono, zinc-600).
  //    d. Description paragraph explaining C-Ops (max-w-2xl, text-zinc-400).
  //    e. CTA Button:
  //       - If authenticated: "Go to Dashboard" -> Link to '/dashboard'
  //       - If not authenticated: "Get Started" -> Link to '/auth'
  //       - Include ArrowRight icon.
  // 8. Render footer with version info at bottom.
}
```

**Layout Structure**:
```
LandingPage (fixed inset-0 z-50, bg-zinc-950)
├── Background Decorations
│   ├── Grid pattern (absolute inset-0, opacity-[0.02])
│   └── Gradient orbs (absolute, blur-3xl)
│       ├── Violet orb (top-left)
│       ├── Cyan orb (bottom-right)
│       └── Amber orb (right-center)
├── LandingHeader (fixed top via its own styling)
└── Main Content (flex min-h-screen items-center justify-center pt-16)
    └── Hero Section (text-center, px-4)
        ├── Icon Container (relative, mb-8)
        │   ├── Glow effect (absolute, animate-pulse, blur-xl)
        │   └── Icon box (relative, rounded-2xl, border, bg, p-4)
        │       └── Terminal icon (h-8 w-8, text-cyan-400)
        ├── Headline (text-4xl sm:text-5xl lg:text-6xl, mb-4)
        ├── Subtitle (font-mono, text-xs, uppercase, mb-6)
        ├── Description (text-lg, text-zinc-400, max-w-2xl, mb-8)
        └── CTA Button (conditional text/link)
            └── Button with ArrowRight icon
```

**Content**:
- Headline: `"Track Your Claude Code Sessions"`
- Subtitle: `"C-OPS // Code Agent Operations"`
- Description: `"Monitor and analyze your AI coding sessions across all your projects. Get insights into token usage, session history, and agent interactions in one unified dashboard."`
- CTA (authenticated): `"Go to Dashboard"`
- CTA (unauthenticated): `"Get Started"`

**Styling Requirements**:
- Outer container: `fixed inset-0 z-50 bg-zinc-950 overflow-auto`
- Grid pattern:
  ```tsx
  className="pointer-events-none absolute inset-0 opacity-[0.02]"
  style={{
    backgroundImage: `
      linear-gradient(rgba(167, 139, 250, 0.5) 1px, transparent 1px),
      linear-gradient(90deg, rgba(167, 139, 250, 0.5) 1px, transparent 1px)
    `,
    backgroundSize: '60px 60px',
  }}
  ```
- Gradient orbs:
  - Violet (top-left): `absolute left-0 top-0 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-violet-500/5 blur-3xl`
  - Cyan (bottom-right): `absolute bottom-0 right-0 h-[400px] w-[400px] translate-x-1/2 translate-y-1/2 rounded-full bg-cyan-500/5 blur-3xl`
  - Amber (right-center): `absolute right-1/4 top-1/3 h-[300px] w-[300px] rounded-full bg-amber-500/3 blur-3xl`
- Main content: `relative flex min-h-screen flex-col items-center justify-center px-4 pt-16`
- Hero icon glow: `absolute inset-0 animate-pulse rounded-2xl bg-cyan-500/20 blur-xl`
- Hero icon container: `relative rounded-2xl border border-cyan-500/20 bg-zinc-900/80 p-4 backdrop-blur-sm`
- Terminal icon: `h-8 w-8 text-cyan-400`
- Headline: `text-4xl font-bold tracking-tight text-zinc-100 sm:text-5xl lg:text-6xl`
- Subtitle: `font-mono text-xs uppercase tracking-[0.2em] text-zinc-600`
- Description: `mx-auto max-w-2xl text-lg leading-relaxed text-zinc-400`
- CTA Button:
  ```tsx
  className="group inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-cyan-500 to-cyan-400 px-6 py-3 font-medium text-zinc-900 transition-all hover:from-cyan-400 hover:to-cyan-300 hover:shadow-lg hover:shadow-cyan-500/25"
  ```
- ArrowRight icon in CTA: `h-4 w-4 transition-transform group-hover:translate-x-1`
- Footer: Same pattern as dashboard.tsx
  ```tsx
  <div className="absolute bottom-8 left-0 right-0 flex items-center justify-center gap-2 text-zinc-700">
    <div className="h-px flex-1 max-w-[100px] bg-gradient-to-r from-transparent to-zinc-800" />
    <span className="font-mono text-[10px] uppercase tracking-widest">C-Ops v0.1.0</span>
    <div className="h-px flex-1 max-w-[100px] bg-gradient-to-l from-transparent to-zinc-800" />
  </div>
  ```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Authenticated user visits | `isAuthenticated = true` | CTA shows "Go to Dashboard", links to /dashboard | Authenticated CTA |
| Unauthenticated user visits | `isAuthenticated = false` | CTA shows "Get Started", links to /auth | Unauthenticated CTA |
| Click CTA (authenticated) | User clicks button | Navigates to /dashboard | Navigation |
| Click CTA (unauthenticated) | User clicks button | Navigates to /auth | Navigation |
| Page renders full-screen | Route loads | Covers sidebar, shows full landing page | Layout override |

---

## Step 4: Create Settings Placeholder Page

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Component rules
- `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`: Page layout with sidebar

### `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`

**Description**:
Create a placeholder settings page that displays "Coming soon" message. Uses the existing sidebar layout (renders inside SidebarInset from `__root.tsx`).

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { Settings } from 'lucide-react'

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
})

// SettingsPage displays a placeholder for future account settings.
function SettingsPage() {
  // Implementation outline:
  // 1. Render container with max-w-7xl and padding (same as dashboard).
  // 2. Render header section:
  //    a. Icon container with Settings icon (violet accent).
  //    b. Title: "Account Settings"
  //    c. Subtitle: "Manage your preferences" (mono, text-xs).
  // 3. Render "Coming soon" card:
  //    a. Use glassmorphic card styling (border-zinc-800/50, bg-zinc-900/80, backdrop-blur).
  //    b. Center content with padding.
  //    c. Display Settings icon (large, muted).
  //    d. Display "Coming soon" text.
  //    e. Display description about future features.
  // 4. Render footer with version info.
}
```

**Layout Structure**:
```
SettingsPage (relative)
├── Container (mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8)
│   ├── Header (mb-8 flex items-center gap-4)
│   │   ├── Icon Container (relative)
│   │   │   ├── Glow effect (absolute, animate-pulse, blur-xl, violet)
│   │   │   └── Icon box (relative, rounded-xl, border-violet, bg, p-3)
│   │   │       └── Settings icon (h-6 w-6, text-violet-400)
│   │   └── Text Container
│   │       ├── Title "Account Settings" (text-2xl, font-bold)
│   │       └── Subtitle "Manage your preferences" (font-mono, text-xs)
│   ├── Coming Soon Card (rounded-xl, border, bg, backdrop-blur, p-16, text-center)
│   │   ├── Large Settings Icon (h-16 w-16, text-zinc-700, mx-auto, mb-6)
│   │   ├── "Coming soon" heading (text-xl, font-semibold, text-zinc-300, mb-2)
│   │   └── Description (text-sm, text-zinc-500, max-w-md, mx-auto)
│   └── Footer (mt-12, same as dashboard)
```

**Content**:
- Title: `"Account Settings"`
- Subtitle: `"Manage your preferences"`
- Coming soon heading: `"Coming soon"`
- Description: `"Account settings and preferences will be available in a future update. Stay tuned!"`

**Styling Requirements**:
- Outer container: `relative` (no fixed positioning, uses sidebar layout)
- Inner container: `mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8`
- Header icon glow: `absolute inset-0 animate-pulse rounded-xl bg-violet-500/20 blur-xl`
- Header icon container: `relative rounded-xl border border-violet-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm`
- Settings icon (header): `h-6 w-6 text-violet-400`
- Title: `text-2xl font-bold tracking-tight text-zinc-100`
- Subtitle: `mt-0.5 font-mono text-xs text-zinc-600`
- Coming soon card: `rounded-xl border border-zinc-800/50 bg-zinc-900/80 p-16 text-center backdrop-blur-sm`
- Large Settings icon: `mx-auto mb-6 h-16 w-16 text-zinc-700`
- Coming soon text: `mb-2 text-xl font-semibold text-zinc-300`
- Description: `mx-auto max-w-md text-sm text-zinc-500`
- Footer: Same pattern as dashboard.tsx

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Visit /settings | Navigate to route | Displays placeholder page with "Coming soon" | Happy path |
| Sidebar visible | Page loads | Sidebar is visible (from root layout) | Layout integration |

---

## Step 5: Add Settings to Sidebar Navigation

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-nav.tsx`: Current navigation items

### `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-nav.tsx`

**Description**:
Add Settings to the sidebar navigation so users can access it from the sidebar as well as the header dropdown.

**Modification**:
1. Add `Settings` to the lucide-react imports.
2. Add a new navigation item to the `navItems` array.

```tsx
// Add to imports:
import { Activity, FolderGit2, MessageSquare, Settings, type LucideIcon } from 'lucide-react'

// Add to navItems array (at the end):
const navItems: NavItem[] = [
  {
    to: '/dashboard',
    icon: Activity,
    label: 'Dashboard',
    description: 'Overview & metrics',
  },
  {
    to: '/projects',
    icon: FolderGit2,
    label: 'Projects',
    description: 'Monitored repos',
  },
  {
    to: '/sessions',
    icon: MessageSquare,
    label: 'Sessions',
    description: 'Agent interactions',
  },
  {
    to: '/settings',
    icon: Settings,
    label: 'Settings',
    description: 'Account settings',
  },
]
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Sidebar renders | Page loads | Settings item visible in navigation | Rendering |
| Click Settings | User clicks item | Navigates to /settings | Navigation |
| Active state | On /settings route | Settings item highlighted with cyan accent | Active state |

---

## Implementation Order

1. **Step 1**: Install shadcn DropdownMenu component
2. **Step 2**: Create `landing-header.tsx` component
3. **Step 3**: Replace `index.tsx` with landing page
4. **Step 4**: Create `settings.tsx` placeholder page
5. **Step 5**: Update sidebar navigation

## File Summary

| Action | File Path |
| :----- | :-------- |
| Install (CLI) | `cd /Users/jayce/team-attention/cops/web && npx shadcn@latest add dropdown-menu` |
| Create | `/Users/jayce/team-attention/cops/web/src/shared/component/landing-header.tsx` |
| Replace | `/Users/jayce/team-attention/cops/web/src/route/index.tsx` |
| Create | `/Users/jayce/team-attention/cops/web/src/route/settings.tsx` |
| Modify | `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-nav.tsx` |

## Design System Reference

### Colors
- Background: `zinc-950`
- Card backgrounds: `zinc-900/80` with `backdrop-blur-sm`
- Borders: `zinc-800/50`, `zinc-800`
- Text primary: `zinc-100`
- Text secondary: `zinc-400`
- Text muted: `zinc-500`, `zinc-600`
- Accent cyan: `cyan-400`, `cyan-500`
- Accent violet: `violet-400`, `violet-500`
- Destructive: `red-400`, `red-500`

### Effects
- Glassmorphic: `bg-zinc-900/80 backdrop-blur-sm border-zinc-800/50`
- Glow: `animate-pulse blur-xl` with accent color at 20% opacity
- Grid pattern: `linear-gradient` with 60px grid, violet accent at 0.02 opacity
- Gradient orbs: `blur-3xl rounded-full` at 5% opacity

### Typography
- Mono labels: `font-mono text-xs uppercase tracking-widest`
- Headlines: `font-bold tracking-tight`
- Brand text: `font-mono font-bold tracking-wider`

### Component Patterns
- Icon containers: `rounded-lg/xl border border-{accent}-500/20 bg-zinc-900/80 p-2/3`
- Buttons: Use shadcn Button with `variant="ghost"` for secondary actions
- Dropdown content: `border-zinc-800 bg-zinc-900`
