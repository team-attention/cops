import { Outlet, createRootRouteWithContext } from '@tanstack/react-router'
import { TanStackRouterDevtoolsPanel } from '@tanstack/react-router-devtools'
import { TanStackDevtools } from '@tanstack/react-devtools'

import {
  SidebarProvider,
  SidebarTrigger,
  SidebarInset,
} from '@/gen/shadcn/ui/sidebar'
import { AppSidebar } from '@/shared/component/app-sidebar'

import TanStackQueryDevtools from '../integration/tanstack-query/devtools'

import type { QueryClient } from '@tanstack/react-query'

interface MyRouterContext {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
  component: () => (
    <SidebarProvider defaultOpen={true}>
      <AppSidebar />
      <SidebarInset>
        {/* Background grid pattern */}
        <div
          className="pointer-events-none fixed inset-0 opacity-[0.02]"
          style={{
            backgroundImage: `
              linear-gradient(rgba(167, 139, 250, 0.5) 1px, transparent 1px),
              linear-gradient(90deg, rgba(167, 139, 250, 0.5) 1px, transparent 1px)
            `,
            backgroundSize: '60px 60px',
          }}
        />

        {/* Gradient orbs */}
        <div className="pointer-events-none fixed left-0 top-0 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-violet-500/5 blur-3xl" />
        <div className="pointer-events-none fixed bottom-0 right-0 h-[400px] w-[400px] translate-x-1/2 translate-y-1/2 rounded-full bg-cyan-500/5 blur-3xl" />
        <div className="pointer-events-none fixed right-1/4 top-1/3 h-[300px] w-[300px] rounded-full bg-amber-500/3 blur-3xl" />

        {/* Minimal header with sidebar trigger */}
        <header className="sticky top-0 z-40 flex h-12 items-center gap-4 border-b border-zinc-800/50 bg-zinc-950/80 px-4 backdrop-blur-xl">
          <SidebarTrigger className="-ml-1 text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200" />
          <div className="h-4 w-px bg-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
            Code Agent Operations
          </span>
        </header>

        {/* Main content area */}
        <main className="relative flex-1">
          <Outlet />
        </main>
      </SidebarInset>

      {/* Devtools */}
      <TanStackDevtools
        config={{
          position: 'bottom-right',
        }}
        plugins={[
          {
            name: 'Tanstack Router',
            render: <TanStackRouterDevtoolsPanel />,
          },
          TanStackQueryDevtools,
        ]}
      />
    </SidebarProvider>
  ),
})
