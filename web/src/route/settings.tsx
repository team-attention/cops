import { createFileRoute } from '@tanstack/react-router'
import { Settings } from 'lucide-react'

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
})

// SettingsPage displays a placeholder for future account settings.
function SettingsPage() {
  return (
    <div className="relative">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8 flex items-center gap-4">
          <div className="relative">
            <div className="absolute inset-0 animate-pulse rounded-xl bg-violet-500/20 blur-xl" />
            <div className="relative rounded-xl border border-violet-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm">
              <Settings className="h-6 w-6 text-violet-400" />
            </div>
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-zinc-100">
              Account Settings
            </h1>
            <p className="mt-0.5 font-mono text-xs text-zinc-600">
              Manage your preferences
            </p>
          </div>
        </div>

        {/* Coming soon card */}
        <div className="rounded-xl border border-zinc-800/50 bg-zinc-900/80 p-16 text-center backdrop-blur-sm">
          <Settings className="mx-auto mb-6 h-16 w-16 text-zinc-700" />
          <h2 className="mb-2 text-xl font-semibold text-zinc-300">
            Coming soon
          </h2>
          <p className="mx-auto max-w-md text-sm text-zinc-500">
            Account settings and preferences will be available in a future update. Stay tuned!
          </p>
        </div>

        {/* Footer */}
        <div className="mt-12 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">
            C-Ops v0.1.0
          </span>
          <div className="h-px flex-1 bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </div>
    </div>
  )
}
