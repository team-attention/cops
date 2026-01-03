# Requirements

## Request Summary

Add a context menu to the account button at the bottom of the left sidebar in the `/dashboard` layout. Currently, the sidebar user component displays account information but has no interactive functionality. The context menu should provide quick access to Settings page navigation and logout functionality, improving the user experience by making these common actions easily accessible from any page with the sidebar.

## Acceptance Criteria

- [ ] Clicking the sidebar user button opens a context menu (dropdown)
- [ ] Context menu appears above the button (positioned appropriately for bottom placement)
- [ ] Context menu contains two menu items: "Settings" and "Logout"
- [ ] Each menu item displays an appropriate icon (Settings icon, Logout/LogOut icon)
- [ ] Clicking "Settings" navigates to `/settings` route
- [ ] Clicking "Logout" immediately logs out the user without confirmation dialog
- [ ] Logout clears authentication state/tokens and redirects appropriately
- [ ] Context menu closes when clicking outside or selecting an item
- [ ] Context menu follows existing dark theme styling consistent with the app
- [ ] Implementation uses shadcn/ui DropdownMenu component
- [ ] Visual design maintains consistency with the sidebar's terminal-style aesthetic

## Scope

### In Scope
- Modify `sidebar-user.tsx` to add DropdownMenu functionality
- Implement context menu with Settings and Logout menu items
- Add appropriate icons from lucide-react for each menu item
- Navigation to `/settings` page when Settings is clicked
- Logout functionality that clears auth state and redirects user
- Context menu positioning above the button
- Styling to match existing dark theme and sidebar design

### Out of Scope
- Backend API changes for session invalidation (use existing auth mechanisms)
- Confirmation dialog for logout action
- Additional menu items beyond Settings and Logout
- Profile editing functionality
- Settings page implementation (already exists as placeholder)
- Mobile-specific menu behavior (follow shadcn/ui defaults)

## Constraints
- Must use shadcn/ui DropdownMenu component for consistency
- Must maintain existing visual design language (dark theme, cyan/violet accents, terminal aesthetic)
- Must work with existing Google OAuth authentication system
- Must preserve existing sidebar collapse/expand functionality
- Menu must remain accessible when sidebar is in icon-only collapsed state

## Additional Context
- Existing file: `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-user.tsx`
- Settings page route: `/settings` (already implemented)
- Auth routes exist: `/auth`, `/auth/device`, `/auth/callback`
- Project uses TanStack Router for navigation
- Google OAuth integration already implemented in `feature/auth/hook/use-google-auth.ts`
- shadcn/ui components located in `gen/shadcn/ui/`
- Icons from lucide-react library

## Questions Resolved

| Question | Answer |
| --- | --- |
| Should the context menu appear on click only or support right-click? | Click only (standard dropdown behavior) |
| Should logout show a confirmation dialog? | No, logout immediately without confirmation |
| Where should the context menu appear? | Above the button (since user button is at bottom of sidebar) |
| Should menu items include icons? | Yes, include Settings and Logout icons from lucide-react |
| Which component library to use for the menu? | shadcn/ui DropdownMenu component |
| Should this include full logout implementation? | Yes, implement complete logout with auth state clearing and redirect |
| Where should logout redirect to? | Landing page (`/`) or auth page (`/auth`) - follow existing auth flow pattern |
| Should menu close when clicking outside? | Yes (default DropdownMenu behavior) |
| Should menu close after selecting an item? | Yes (default DropdownMenu behavior) |
