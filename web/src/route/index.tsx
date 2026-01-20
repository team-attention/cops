import { Link, createFileRoute } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { Button } from '@/gen/shadcn/ui/button'
import { LandingHeader } from '@/shared/component/landing-header'
import { useAuth } from '@/shared/hook/use-auth'
import { APP_VERSION } from '@/shared/config/version'

export const Route = createFileRoute('/')({
  component: LandingPage,
})

// LandingPage displays the C-Ops landing page with hero section.
// Adapts CTA button text and destination based on authentication state.
function LandingPage() {
  const { isAuthenticated } = useAuth()

  return (
    <div className="fixed inset-0 z-50 bg-zinc-950 overflow-auto">
      {/* Background decorations */}
      {/* Grid pattern */}
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.02]"
        style={{
          backgroundImage: `
            linear-gradient(rgba(167, 139, 250, 0.5) 1px, transparent 1px),
            linear-gradient(90deg, rgba(167, 139, 250, 0.5) 1px, transparent 1px)
          `,
          backgroundSize: '60px 60px',
        }}
      />

      {/* Gradient orbs */}
      <div className="absolute left-0 top-0 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-violet-500/5 blur-3xl" />
      <div className="absolute bottom-0 right-0 h-[400px] w-[400px] translate-x-1/2 translate-y-1/2 rounded-full bg-cyan-500/5 blur-3xl" />
      <div className="absolute right-1/4 top-1/3 h-[300px] w-[300px] rounded-full bg-amber-500/3 blur-3xl" />

      {/* Header */}
      <LandingHeader />

      {/* Main content */}
      <main className="relative flex min-h-screen flex-col items-center justify-center px-4 pt-16">
        {/* Hero section */}
        <div className="text-center">
          {/* Icon container with glow */}
          <div className="relative mb-8 inline-block">
            <div className="absolute inset-0 animate-pulse rounded-2xl bg-cyan-500/20 blur-xl" />
            <div className="relative rounded-2xl border border-cyan-500/20 bg-zinc-900/80 p-4 backdrop-blur-sm">
              <img src="/logo192.png" alt="C-Ops" className="h-10 w-10" />
            </div>
          </div>

          {/* Headline */}
          <h1 className="text-4xl font-bold tracking-tight text-zinc-100 sm:text-5xl lg:text-6xl mb-4">
            Track Your Claude Code Sessions
          </h1>

          {/* Subtitle */}
          <p className="font-mono text-xs uppercase tracking-[0.2em] text-zinc-600 mb-6">
            C-OPS // Code Agent Operations
          </p>

          {/* Description */}
          <p className="mx-auto max-w-2xl text-lg leading-relaxed text-zinc-400 mb-8">
            Monitor and analyze your AI coding sessions across all your
            projects. Get insights into token usage, session history, and agent
            interactions in one unified dashboard.
          </p>

          {/* CTA Button */}
          <Button
            asChild
            className="group inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-cyan-500 to-cyan-400 px-6 py-3 font-medium text-zinc-900 transition-all hover:from-cyan-400 hover:to-cyan-300 hover:shadow-lg hover:shadow-cyan-500/25"
          >
            <Link to={isAuthenticated ? '/dashboard' : '/auth'}>
              {isAuthenticated ? 'Go to Dashboard' : 'Get Started'}
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </Link>
          </Button>
        </div>

        {/* Footer */}
        <div className="absolute bottom-8 left-0 right-0 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 max-w-[100px] bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">
            C-Ops v{APP_VERSION}
          </span>
          <div className="h-px flex-1 max-w-[100px] bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </main>
    </div>
  )
}
