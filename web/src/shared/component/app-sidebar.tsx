import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarRail,
  SidebarSeparator,
} from '@/gen/shadcn/ui/sidebar'
import { SidebarHeader } from './sidebar-header'
import { SidebarNav } from './sidebar-nav'
import { SidebarUser } from './sidebar-user'

export const AppSidebar = () => {
  return (
    <Sidebar
      collapsible="icon"
      className="border-sidebar"
    >
      {/* Subtle background texture */}
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.015]"
        style={{
          backgroundImage: `
            radial-gradient(circle at 20% 50%, rgba(34, 211, 238, 0.15) 0%, transparent 50%),
            radial-gradient(circle at 80% 80%, rgba(167, 139, 250, 0.1) 0%, transparent 50%)
          `,
        }}
      />

      {/* Scanline effect overlay */}
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.02]"
        style={{
          backgroundImage:
            'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(255,255,255,0.03) 2px, rgba(255,255,255,0.03) 4px)',
        }}
      />

      {/* Header */}
      <SidebarHeader />

      {/* Main navigation content */}
      <SidebarContent className="relative">
        <SidebarNav />
      </SidebarContent>

      {/* Footer with user profile */}
      <SidebarSeparator className="bg-zinc-800/50" />
      <SidebarFooter>
        <SidebarUser />
      </SidebarFooter>

      {/* Collapse rail */}
      <SidebarRail className="after:bg-zinc-700/50 hover:after:bg-cyan-500/30" />
    </Sidebar>
  )
}
