import { createFileRoute } from '@tanstack/react-router'
import { Activity, RefreshCw } from 'lucide-react'
import { useGetOverview } from '@/feature/dashboard/hook/use-get-overview'
import { OverviewStats } from '@/feature/dashboard/component/overview-stats'
import { ProjectList } from '@/feature/dashboard/component/project-list'
import { RecentSessions } from '@/feature/dashboard/component/recent-sessions'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
})

const LoadingSkeleton = () => (
  <div className="space-y-6">
    {/* Stats skeleton */}
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {[...Array(4)].map((_, i) => (
        <Skeleton key={i} className="h-32 bg-zinc-800/50" />
      ))}
    </div>
    {/* Tables skeleton */}
    <div className="grid gap-6 lg:grid-cols-2">
      <Skeleton className="h-80 bg-zinc-800/50" />
      <Skeleton className="h-80 bg-zinc-800/50" />
    </div>
  </div>
)

function DashboardPage() {
  const { data, isLoading, isError, refetch, isFetching } = useGetOverview()

  return (
    <div className="relative">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-xl bg-cyan-500/20 blur-xl" />
              <div className="relative rounded-xl border border-cyan-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm">
                <Activity className="h-6 w-6 text-cyan-400" />
              </div>
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-zinc-100">
                Dashboard
              </h1>
              <p className="mt-0.5 font-mono text-xs text-zinc-600">
                C-Ops // Code Agent Analytics
              </p>
            </div>
          </div>

          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="group flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-2 text-sm text-zinc-400 transition-all hover:border-zinc-700 hover:bg-zinc-800/50 hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <RefreshCw
              className={`h-4 w-4 transition-transform ${isFetching ? 'animate-spin' : 'group-hover:rotate-90'}`}
            />
            <span className="font-mono text-xs">Refresh</span>
          </button>
        </div>

        {/* Content */}
        {isLoading ? (
          <LoadingSkeleton />
        ) : isError ? (
          <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
            <Activity className="mb-4 h-12 w-12 opacity-50" />
            <p className="font-mono text-sm">Failed to load dashboard data</p>
            <button
              onClick={() => refetch()}
              className="mt-4 rounded-lg bg-zinc-800 px-4 py-2 font-mono text-xs text-zinc-300 transition-colors hover:bg-zinc-700"
            >
              Try again
            </button>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Overview Stats */}
            <OverviewStats
              totalUsage={data?.totalUsage}
              projectCount={data?.projectCount ?? 0}
              sessionCount={data?.sessionCount ?? 0}
            />

            {/* Two-column layout for tables */}
            <div className="grid gap-6 lg:grid-cols-2">
              <ProjectList projects={data?.recentProjects ?? []} />
              <RecentSessions sessions={data?.recentSessions ?? []} />
            </div>
          </div>
        )}

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
