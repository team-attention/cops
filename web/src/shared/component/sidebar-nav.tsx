import { Link, useRouterState } from '@tanstack/react-router'
import { Activity, FolderGit2, MessageSquare, type LucideIcon } from 'lucide-react'
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/gen/shadcn/ui/sidebar'

interface NavItem {
  to: string
  icon: LucideIcon
  label: string
  description: string
}

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
]

export const SidebarNav = () => {
  const router = useRouterState()

  return (
    <SidebarGroup>
      <SidebarGroupLabel className="px-2 font-mono text-[10px] uppercase tracking-[0.15em] text-zinc-600">
        Navigation
      </SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {navItems.map((item) => {
            const isActive = router.location.pathname.startsWith(item.to)
            const Icon = item.icon

            return (
              <SidebarMenuItem key={item.to}>
                <SidebarMenuButton
                  asChild
                  isActive={isActive}
                  tooltip={item.label}
                  className={`
                    group/nav relative transition-all duration-200
                    ${
                      isActive
                        ? 'bg-cyan-500/10 text-cyan-400 hover:bg-cyan-500/15 hover:text-cyan-300'
                        : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200'
                    }
                  `}
                >
                  <Link to={item.to}>
                    {/* Active indicator bar */}
                    {isActive && (
                      <div className="absolute left-0 top-1/2 h-4 w-0.5 -translate-y-1/2 rounded-full bg-cyan-400 shadow-[0_0_8px_rgba(34,211,238,0.5)]" />
                    )}

                    {/* Icon with glow on active */}
                    <div className="relative">
                      <Icon
                        className={`h-4 w-4 transition-all duration-200 ${
                          isActive ? 'text-cyan-400' : 'text-zinc-500 group-hover/nav:text-zinc-300'
                        }`}
                      />
                      {isActive && (
                        <div className="absolute inset-0 animate-pulse blur-md">
                          <Icon className="h-4 w-4 text-cyan-400/50" />
                        </div>
                      )}
                    </div>

                    {/* Label */}
                    <span className="font-medium">{item.label}</span>

                    {/* Active chevron indicator */}
                    {isActive && (
                      <div className="ml-auto font-mono text-[10px] text-cyan-500/70 group-data-[collapsible=icon]:hidden">
                        ›
                      </div>
                    )}
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          })}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
