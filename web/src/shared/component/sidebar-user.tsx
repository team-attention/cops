import { useNavigate } from '@tanstack/react-router'
import {
  AlertCircle,
  Building2,
  ChevronDown,
  LogOut,
  Plus,
  RefreshCw,
  Settings,
} from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/gen/shadcn/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/gen/shadcn/ui/sidebar'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useAuth } from '@/shared/hook/use-auth'
import { useUser } from '@/shared/hook/use-user'

// getInitials extracts initials from name or email for avatar fallback.
const getInitials = (
  name: string | undefined,
  email: string | undefined,
): string => {
  // 1. If name exists and has length > 0:
  if (name && name.length > 0) {
    // a. Split name by spaces.
    const parts = name.split(' ')
    // b. Take first character of first word and first character of last word.
    // c. Return uppercase initials (max 2 characters).
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    }
    return parts[0].slice(0, 2).toUpperCase()
  }
  // 2. If email exists and has length > 0:
  if (email && email.length > 0) {
    // a. Take first 2 characters of email before @ symbol.
    // b. Return uppercase.
    const atIndex = email.indexOf('@')
    const prefix = atIndex > 0 ? email.substring(0, atIndex) : email
    return prefix.slice(0, 2).toUpperCase()
  }
  // 3. Return "U" as default.
  return 'U'
}

// getDisplayName returns the best available display name.
const getDisplayName = (
  name: string | undefined,
  email: string | undefined,
): string => {
  // 1. If name exists and has length > 0, return name.
  if (name && name.length > 0) {
    return name
  }
  // 2. If email exists and has length > 0, return email.
  if (email && email.length > 0) {
    return email
  }
  // 3. Return "User" as default.
  return 'User'
}

// SidebarUser displays user info with dropdown menu for Settings and Logout.
// Dropdown appears above the button since user button is at bottom of sidebar.
export const SidebarUser = () => {
  const navigate = useNavigate()
  const { logout } = useAuth()
  const {
    user,
    organizations,
    selectedOrganization,
    selectedOrganizationId,
    isLoading,
    error,
    setSelectedOrganizationId,
    refetch,
    reset,
  } = useUser()

  const handleSettingsClick = () => {
    navigate({ to: '/settings' })
  }

  const handleLogoutClick = () => {
    reset()
    logout()
    navigate({ to: '/' })
  }

  const handleRetryClick = () => {
    refetch()
  }

  const handleOrganizationChange = (orgId: string) => {
    setSelectedOrganizationId(orgId)
  }

  // handleCreateOrganizationClick navigates to the organization creation page.
  const handleCreateOrganizationClick = () => {
    navigate({ to: '/organizations/new' })
  }

  // Render loading state
  if (isLoading) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton size="lg" className="cursor-default">
            <Skeleton className="h-8 w-8 rounded-lg" />
            <div className="flex min-w-0 flex-col gap-1 group-data-[collapsible=icon]:hidden">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-16" />
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    )
  }

  // Render error state
  if (error) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            size="lg"
            onClick={handleRetryClick}
            className="group/user relative overflow-hidden transition-all duration-200 hover:bg-zinc-800/50"
          >
            <div className="relative">
              <Avatar className="h-8 w-8 rounded-lg border border-red-700/50 bg-red-900/20">
                <AvatarFallback className="rounded-lg bg-red-900/30 text-red-400">
                  <AlertCircle className="h-4 w-4" />
                </AvatarFallback>
              </Avatar>
            </div>
            <div className="flex min-w-0 flex-col group-data-[collapsible=icon]:hidden">
              <span className="truncate text-sm font-medium text-red-400">
                Failed to load
              </span>
              <span className="flex items-center gap-1.5 text-xs text-red-500">
                <RefreshCw className="h-3 w-3" />
                Click to retry
              </span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    )
  }

  // Render normal state with user data
  const displayName = getDisplayName(user?.name, user?.email)
  const initials = getInitials(user?.name, user?.email)
  const organizationName = selectedOrganization?.name || 'No organization'

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
                  {user?.avatarUrl && (
                    <AvatarImage src={user.avatarUrl} alt={displayName} />
                  )}
                  <AvatarFallback className="rounded-lg bg-gradient-to-br from-zinc-700 to-zinc-800 font-mono text-xs font-bold text-zinc-300">
                    {initials}
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
                  {displayName}
                </span>
                <span className="flex items-center gap-1.5 font-mono text-[10px] text-zinc-500">
                  <Building2 className="h-3 w-3" />
                  {organizationName}
                </span>
              </div>

              {/* Chevron indicator */}
              <ChevronDown className="ml-auto h-4 w-4 text-zinc-600 group-data-[collapsible=icon]:hidden" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            side="top"
            align="start"
            sideOffset={8}
            className="w-56 border-zinc-800 bg-zinc-900"
          >
            {/* User email display */}
            {user?.email && (
              <>
                <div className="px-2 py-1.5">
                  <p className="truncate text-xs text-zinc-500">{user.email}</p>
                </div>
                <DropdownMenuSeparator />
              </>
            )}

            {/* Organization switcher submenu */}
            {organizations.length > 1 && (
              <>
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger>
                    <Building2 className="mr-2 h-4 w-4" />
                    Switch Organization
                  </DropdownMenuSubTrigger>
                  <DropdownMenuSubContent className="border-zinc-800 bg-zinc-900">
                    <DropdownMenuRadioGroup
                      value={selectedOrganizationId || ''}
                      onValueChange={handleOrganizationChange}
                    >
                      {organizations.map((org) => (
                        <DropdownMenuRadioItem key={org.id} value={org.id}>
                          {org.name}
                        </DropdownMenuRadioItem>
                      ))}
                    </DropdownMenuRadioGroup>
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
                <DropdownMenuSeparator />
              </>
            )}

            {/* Create Organization menu item */}
            <DropdownMenuItem onClick={handleCreateOrganizationClick}>
              <Plus className="mr-2 h-4 w-4" />
              Create Organization
            </DropdownMenuItem>
            <DropdownMenuSeparator />

            {/* Settings menu item */}
            <DropdownMenuItem onClick={handleSettingsClick}>
              <Settings className="mr-2 h-4 w-4" />
              Settings
            </DropdownMenuItem>
            <DropdownMenuSeparator />

            {/* Logout menu item */}
            <DropdownMenuItem
              onClick={handleLogoutClick}
              className="text-red-400 focus:bg-red-500/10 focus:text-red-300"
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
