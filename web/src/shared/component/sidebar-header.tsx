import { SidebarHeader as ShadcnSidebarHeader } from '@/gen/shadcn/ui/sidebar'
import { APP_VERSION } from '@/shared/config/version'

export const SidebarHeader = () => {
  return (
    <ShadcnSidebarHeader className="border-b border-zinc-800/50 px-2 py-4">
      <div className="flex items-center gap-3">
        {/* Logo container with glow effect */}
        <div className="relative">
          {/* Ambient glow */}
          <div className="absolute inset-0 animate-pulse rounded-lg bg-cyan-500/20 blur-xl" />
          {/* Outer ring */}
          <div className="relative rounded-lg border border-cyan-500/30 bg-gradient-to-br from-zinc-900 to-zinc-950 p-2 shadow-lg shadow-cyan-500/10">
            {/* Inner icon with scanline effect */}
            <div className="relative overflow-hidden">
              <img src="/logo192.png" alt="C-Ops" className="relative z-10 h-6 w-6" />
              {/* Scanline overlay */}
              <div
                className="pointer-events-none absolute inset-0 opacity-20"
                style={{
                  backgroundImage:
                    'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(34, 211, 238, 0.1) 2px, rgba(34, 211, 238, 0.1) 4px)',
                }}
              />
            </div>
          </div>
        </div>

        {/* Brand text - hidden when collapsed */}
        <div className="flex flex-col group-data-[collapsible=icon]:hidden">
          <div className="flex items-baseline gap-1.5">
            <span className="bg-gradient-to-r from-zinc-100 to-zinc-300 bg-clip-text text-lg font-bold tracking-tight text-transparent">
              C-Ops
            </span>
            <span className="rounded border border-cyan-500/20 bg-cyan-500/5 px-1.5 py-0.5 font-mono text-[9px] font-medium text-cyan-400/70">
              v{APP_VERSION}
            </span>
          </div>
          <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-zinc-600">
            Agent Monitor
          </span>
        </div>
      </div>

      {/* Decorative line */}
      <div className="absolute bottom-0 left-0 h-px w-full bg-gradient-to-r from-transparent via-cyan-500/20 to-transparent" />
    </ShadcnSidebarHeader>
  )
}
