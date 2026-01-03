# Implementation Plan: Sidebar User Context Menu

## Overview

Add a context menu (dropdown) to the sidebar user button that provides quick access to Settings navigation and Logout functionality. The dropdown will use shadcn/ui DropdownMenu component and appear above the button (since user button is at bottom of sidebar). When clicked, "Settings" navigates to `/settings` route, and "Logout" immediately clears authentication state and redirects to landing page.

## Package Changes

No package changes required. All necessary components already exist:
- `@/gen/shadcn/ui/dropdown-menu` - DropdownMenu component already installed
- `lucide-react` - Already installed, provides Settings and LogOut icons
- `@tanstack/react-router` - Already installed, provides useNavigate hook

## Step 1: Update SidebarUser Component with DropdownMenu

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: React component rules (named exports, no any types)
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Feature structure and import rules
- `/Users/jayce/team-attention/cops/web/src/shared/component/landing-header.tsx`: Reference implementation of DropdownMenu with Settings and Logout items
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Auth hook providing logout function

### `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-user.tsx`

**Description**:
Wrap the existing SidebarMenuButton with DropdownMenu components. Add Settings and Logout menu items with icons. Use useNavigate for navigation and useAuth for logout functionality. Position dropdown above button using `side="top"` prop.

```tsx
import { useNavigate } from '@tanstack/react-router'
import { Settings, LogOut } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/gen/shadcn/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/gen/shadcn/ui/sidebar'
import { useAuth } from '@/shared/hook/use-auth'

// SidebarUser displays user info with dropdown menu for Settings and Logout.
// Dropdown appears above the button since user button is at bottom of sidebar.
export const SidebarUser = () => {
  // Implementation outline:
  // 1. Get navigate function from useNavigate hook
  // 2. Get logout function from useAuth hook
  // 3. Define handleSettingsClick function:
  //    a. Navigate to '/settings' route
  // 4. Define handleLogoutClick function:
  //    a. Call logout() to clear auth state
  //    b. Navigate to '/' (landing page)
  // 5. Render DropdownMenu wrapping SidebarMenu:
  //    a. DropdownMenuTrigger wraps SidebarMenuButton (use asChild prop)
  //    b. DropdownMenuContent with side="top" and align="start" props
  //       - Custom styling: border-zinc-800, bg-zinc-900 for dark theme
  //       - sideOffset={8} for spacing from trigger
  //    c. First DropdownMenuItem for Settings:
  //       - Settings icon (h-4 w-4)
  //       - "Settings" text
  //       - onClick calls handleSettingsClick
  //    d. DropdownMenuSeparator for visual separation
  //    e. Second DropdownMenuItem for Logout:
  //       - LogOut icon (h-4 w-4)
  //       - "Logout" text
  //       - onClick calls handleLogoutClick
  //       - Destructive styling: text-red-400 focus:text-red-300 focus:bg-red-500/10
  // 6. Preserve all existing SidebarMenuButton content (avatar, user info, decorations)
}
```

**Complete Implementation Structure**:

```tsx
import { useNavigate } from '@tanstack/react-router'
import { Settings, LogOut } from 'lucide-react'
import { Avatar, AvatarFallback } from '@/gen/shadcn/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/gen/shadcn/ui/sidebar'
import { useAuth } from '@/shared/hook/use-auth'

// SidebarUser displays user info with dropdown menu for Settings and Logout.
// Dropdown appears above the button since user button is at bottom of sidebar.
export const SidebarUser = () => {
  const navigate = useNavigate()
  const { logout } = useAuth()

  // handleSettingsClick navigates to settings page
  const handleSettingsClick = () => {
    // Navigate to /settings route
    navigate({ to: '/settings' })
  }

  // handleLogoutClick clears auth and redirects to landing
  const handleLogoutClick = () => {
    // Clear auth tokens from localStorage
    logout()
    // Redirect to landing page
    navigate({ to: '/' })
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="group/user relative overflow-hidden transition-all duration-200 hover:bg-zinc-800/50"
            >
              {/* Subtle gradient overlay on hover */}
              <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-cyan-500/0 via-cyan-500/5 to-cyan-500/0 opacity-0 transition-opacity duration-300 group-hover/user:opacity-100" />

              {/* Avatar */}
              <div className="relative">
                <Avatar className="h-8 w-8 rounded-lg border border-zinc-700/50 bg-zinc-800">
                  <AvatarFallback className="rounded-lg bg-gradient-to-br from-zinc-700 to-zinc-800 font-mono text-xs font-bold text-zinc-300">
                    CO
                  </AvatarFallback>
                </Avatar>
                {/* Online status indicator */}
                <div className="absolute -bottom-0.5 -right-0.5 flex items-center justify-center">
                  <div className="h-2.5 w-2.5 rounded-full border-2 border-zinc-900 bg-emerald-400">
                    <div className="h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                  </div>
                </div>
              </div>

              {/* User info - hidden when collapsed */}
              <div className="flex min-w-0 flex-col group-data-[collapsible=icon]:hidden">
                <span className="truncate text-sm font-medium text-zinc-200">
                  Code Operator
                </span>
                <span className="flex items-center gap-1.5 font-mono text-[10px] text-zinc-500">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                  Online
                </span>
              </div>

              {/* Terminal-style decoration */}
              <div className="ml-auto font-mono text-[10px] text-zinc-600 group-data-[collapsible=icon]:hidden">
                [~]
              </div>
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            side="top"
            align="start"
            sideOffset={8}
            className="w-48 border-zinc-800 bg-zinc-900"
          >
            <DropdownMenuItem onClick={handleSettingsClick}>
              <Settings className="mr-2 h-4 w-4" />
              Settings
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={handleLogoutClick}
              className="text-red-400 focus:text-red-300 focus:bg-red-500/10"
            >
              <LogOut className="mr-2 h-4 w-4" />
              Logout
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Component renders | Mount component | SidebarUser displays with user info and avatar | Happy path |
| Dropdown opens on click | Click SidebarMenuButton | DropdownMenuContent appears above button | Trigger interaction |
| Settings navigation | Click "Settings" menu item | Navigate called with { to: '/settings' } | Settings handler |
| Logout action | Click "Logout" menu item | logout() called, navigate to '/' | Logout handler |
| Dropdown closes after selection | Click any menu item | DropdownMenuContent closes (default behavior) | Auto-close |
| Dropdown closes on outside click | Click outside dropdown | DropdownMenuContent closes (default behavior) | Auto-close |
| Sidebar collapsed state | Sidebar in icon-only mode | Dropdown still accessible via avatar click | Collapsed UI |

## Implementation Notes

### Key Design Decisions

1. **Dropdown Position**: Using `side="top"` and `align="start"` because the user button is at the bottom of the sidebar. This ensures the menu appears above the button and doesn't get cut off.

2. **Styling Consistency**: Following the existing pattern from `landing-header.tsx`:
   - Dark theme: `border-zinc-800 bg-zinc-900`
   - Destructive logout style: `text-red-400 focus:text-red-300 focus:bg-red-500/10`
   - Icon sizing: `h-4 w-4` with `mr-2` spacing

3. **Navigation After Logout**: Redirecting to `/` (landing page) rather than `/auth` because:
   - Landing page has authentication-aware header
   - Provides better UX showing the public landing page
   - Follows the pattern established in the existing auth flow

4. **Using `asChild` on DropdownMenuTrigger**: This allows the SidebarMenuButton to serve as the trigger while maintaining its styling and sidebar integration (including tooltip support in collapsed mode).

5. **sideOffset**: Using `8` pixels to provide visual breathing room between the button and dropdown.

### Accessibility

- DropdownMenu from shadcn/ui (Radix UI) provides built-in keyboard navigation
- Menu items are focusable and selectable via keyboard
- Escape key closes dropdown
- Focus is managed correctly when opening/closing
