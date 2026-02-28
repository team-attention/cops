import { useCallback, useRef, useState, useEffect } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useGetFeaturedBoard } from '@/feature/featured/hook/use-get-featured-board'
import { FeaturedBoard } from '@/feature/featured/component/featured-board'

// FeaturedSearchParams defines search params for the featured board route
interface FeaturedSearchParams {
  since?: number
}

// Default since: today at 19:30 local time as Unix timestamp
function getDefaultSince(): bigint {
  const now = new Date()
  const todayAt1930 = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
    19, // 19:30 local time
    30,
    0,
    0,
  )
  if (now.getTime() < todayAt1930.getTime()) {
    todayAt1930.setDate(todayAt1930.getDate() - 1)
  }
  return BigInt(Math.floor(todayAt1930.getTime() / 1000))
}

export const Route = createFileRoute('/featured/$orgSlug')({
  validateSearch: (search: Record<string, unknown>): FeaturedSearchParams => {
    const n = Number(search.since)
    return { since: Number.isFinite(n) ? n : undefined }
  },
  component: FeaturedBoardPage,
})

const LoadingSkeleton = () => (
  <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
    {[...Array(8)].map((_, i) => (
      <Skeleton key={i} className="h-64 bg-zinc-800/50" />
    ))}
  </div>
)

function FeaturedBoardPage() {
  const { orgSlug } = Route.useParams()
  const { since } = Route.useSearch()
  const containerRef = useRef<HTMLDivElement>(null)
  const [isFullscreen, setIsFullscreen] = useState(false)

  const sinceUnix = since !== undefined ? BigInt(since) : getDefaultSince()

  const { data, isLoading, isError } = useGetFeaturedBoard({
    orgSlug,
    sinceUnix,
  })

  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      containerRef.current?.requestFullscreen()
    } else {
      document.exitFullscreen()
    }
  }, [])

  useEffect(() => {
    const onChange = () => setIsFullscreen(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [])

  return (
    <div ref={containerRef} className="min-h-screen bg-zinc-950">
      {/* CSS keyframes for character animations */}
      <style>{`
        @keyframes member-blink {
          0%, 100% { background-color: rgb(23 7 7 / 0.1); }
          50% { background-color: rgb(127 29 29 / 0.2); }
        }

        /* Shared keyframes */
        @keyframes leg-stride {
          0%, 100% { transform: rotate(0deg); }
          25% { transform: rotate(-22deg); }
          75% { transform: rotate(22deg); }
        }
        @keyframes arm-swing {
          0%, 100% { transform: rotate(0deg); }
          50% { transform: rotate(18deg); }
        }
        @keyframes arm-swing-alt {
          0%, 100% { transform: rotate(0deg); }
          50% { transform: rotate(-18deg); }
        }

        /* Homer */
        .homer-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: 19px 38px; }
        .homer-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: 25px 38px; }
        .homer-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: 18px 28px; }
        .homer-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: 26px 26px; }

        /* Bart */
        .bart-leg-back  { animation: leg-stride 0.4s ease-in-out infinite; transform-origin: 19px 48px; }
        .bart-leg-front { animation: leg-stride 0.4s ease-in-out infinite reverse; transform-origin: 29px 48px; }
        .bart-arm-back  { animation: arm-swing 0.4s ease-in-out infinite; transform-origin: 15px 36px; }
        .bart-arm-front { animation: arm-swing-alt 0.4s ease-in-out infinite; transform-origin: 33px 36px; }

        /* Lisa */
        .lisa-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: 19px 48px; }
        .lisa-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: 26px 48px; }
        .lisa-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: 19px 34px; }
        .lisa-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: 30px 34px; }

        /* Marge */
        .marge-leg-back  { animation: leg-stride 0.5s ease-in-out infinite; transform-origin: 17px 46px; }
        .marge-leg-front { animation: leg-stride 0.5s ease-in-out infinite reverse; transform-origin: 25px 46px; }
        .marge-arm-back  { animation: arm-swing 0.5s ease-in-out infinite; transform-origin: 17px 37px; }
        .marge-arm-front { animation: arm-swing-alt 0.5s ease-in-out infinite; transform-origin: 28px 37px; }

        /* Maggie */
        .maggie-leg-back  { animation: leg-stride 0.35s ease-in-out infinite; transform-origin: 13px 38px; }
        .maggie-leg-front { animation: leg-stride 0.35s ease-in-out infinite reverse; transform-origin: 21px 38px; }
        .maggie-arm-back  { animation: arm-swing 0.35s ease-in-out infinite; transform-origin: 12px 30px; }
        .maggie-arm-front { animation: arm-swing-alt 0.35s ease-in-out infinite; transform-origin: 26px 29px; }

        /* Ned */
        .ned-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: 15px 38px; }
        .ned-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: 23px 38px; }
        .ned-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: 8px 24px; }
        .ned-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: 31px 23px; }

        /* Burns */
        .burns-leg-back  { animation: leg-stride 0.55s ease-in-out infinite; transform-origin: 19px 38px; }
        .burns-leg-front { animation: leg-stride 0.55s ease-in-out infinite reverse; transform-origin: 23px 37px; }
        .burns-arm-back  { animation: arm-swing 0.55s ease-in-out infinite; transform-origin: 16px 26px; }
        .burns-arm-front { animation: arm-swing-alt 0.55s ease-in-out infinite; transform-origin: 27px 26px; }

        /* Krusty */
        .krusty-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: 12px 36px; }
        .krusty-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: 22px 36px; }
        .krusty-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: 11px 24px; }
        .krusty-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: 26px 23px; }

        /* Ralph */
        .ralph-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: center 40px; }
        .ralph-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: center 42px; }
        .ralph-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: 22px 28px; }
        .ralph-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: 30px 28px; }

        /* Milhouse */
        .milhouse-leg-back  { animation: leg-stride 0.45s ease-in-out infinite; transform-origin: center 36px; }
        .milhouse-leg-front { animation: leg-stride 0.45s ease-in-out infinite reverse; transform-origin: center 36px; }
        .milhouse-arm-back  { animation: arm-swing 0.45s ease-in-out infinite; transform-origin: center 22px; }
        .milhouse-arm-front { animation: arm-swing-alt 0.45s ease-in-out infinite; transform-origin: center 21px; }
      `}</style>

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

      <div className="relative mx-auto max-w-screen-2xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8 flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3">
              <div className="h-2 w-2 animate-pulse rounded-full bg-cyan-400" />
              <h1 className="font-mono text-lg font-bold uppercase tracking-widest text-zinc-100">
                {data?.organizationName || orgSlug}
              </h1>
              <span className="font-mono text-xs text-zinc-600">
                // Featured Board
              </span>
            </div>
            <p className="mt-1 font-mono text-[10px] uppercase tracking-widest text-zinc-700">
              C-Ops // Live Agent Activity
            </p>
          </div>
          <button
            onClick={toggleFullscreen}
            className="shrink-0 rounded-lg border border-zinc-800 bg-zinc-900/80 px-3 py-1.5 font-mono text-[10px] uppercase tracking-widest text-zinc-400 transition-colors hover:border-zinc-600 hover:text-zinc-200"
          >
            {isFullscreen ? '[ Exit ]' : '[ Fullscreen ]'}
          </button>
        </div>

        {/* Content */}
        {isLoading ? (
          <LoadingSkeleton />
        ) : isError ? (
          <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
            <p className="font-mono text-sm">Organization not found</p>
            <p className="mt-2 font-mono text-xs text-zinc-600">
              Could not load featured board for &quot;{orgSlug}&quot;
            </p>
          </div>
        ) : (
          <FeaturedBoard members={data?.members ?? []} sinceUnix={sinceUnix} />
        )}
      </div>
    </div>
  )
}
