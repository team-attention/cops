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

  const handleSettingsClick = () => {
    navigate({ to: '/settings' })
  }

  const handleLogoutClick = () => {
    logout()
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
