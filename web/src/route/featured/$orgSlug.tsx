import { createFileRoute } from '@tanstack/react-router'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useGetFeaturedBoard } from '@/feature/featured/hook/use-get-featured-board'
import { FeaturedBoard } from '@/feature/featured/component/featured-board'

// FeaturedSearchParams defines search params for the featured board route
interface FeaturedSearchParams {
  since?: number
}

// Default since: today at 20:00 (8PM) local time as Unix timestamp
function getDefaultSince(): bigint {
  const now = new Date()
  const todayAt8PM = new Date(
    now.getFullYear(),
    now.getMonth(),
    now.getDate(),
    20, // 20:00 local time
    0,
    0,
    0,
  )
  if (now.getTime() < todayAt8PM.getTime()) {
    todayAt8PM.setDate(todayAt8PM.getDate() - 1)
  }
  return BigInt(Math.floor(todayAt8PM.getTime() / 1000))
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

  const sinceUnix = since !== undefined ? BigInt(since) : getDefaultSince()

  const { data, isLoading, isError } = useGetFeaturedBoard({
    orgSlug,
    sinceUnix,
  })

  return (
    <div className="min-h-screen bg-zinc-950">
      {/* CSS keyframes for inactive card blink animation (1Hz, WCAG compliant < 3Hz) */}
      <style>{`
        @keyframes member-blink {
          0%, 100% { background-color: rgb(23 7 7 / 0.1); }
          50% { background-color: rgb(127 29 29 / 0.2); }
        }
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
        <div className="mb-8">
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
          <FeaturedBoard members={data?.members ?? []} />
        )}
      </div>
    </div>
  )
}
