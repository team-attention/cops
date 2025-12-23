import { createFileRoute } from '@tanstack/react-router'
import { FolderGit2, RefreshCw } from 'lucide-react'
import { useGetProject } from '@/feature/project/hook/use-get-project'
import { useListSessions } from '@/feature/project/hook/use-list-sessions'
import { ProjectHeader } from '@/feature/project/component/project-header'
import { TokenBreakdown } from '@/feature/project/component/token-breakdown'
import { SessionList } from '@/feature/project/component/session-list'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'

export const Route = createFileRoute('/projects/$projectId')({
  component: ProjectDetailPage,
})

const LoadingSkeleton = () => (
  <div className="space-y-6">
    {/* Breadcrumb skeleton */}
    <Skeleton className="h-4 w-48 bg-zinc-800/50" />

    {/* Header skeleton */}
    <Skeleton className="h-40 bg-zinc-800/50" />

    {/* Content skeleton */}
    <div className="grid gap-6 lg:grid-cols-5">
      <div className="lg:col-span-2">
        <Skeleton className="h-96 bg-zinc-800/50" />
      </div>
      <div className="lg:col-span-3">
        <Skeleton className="h-96 bg-zinc-800/50" />
      </div>
    </div>
  </div>
)

function ProjectDetailPage() {
  const { projectId } = Route.useParams()

  const {
    data: projectData,
    isLoading: isProjectLoading,
    isError: isProjectError,
    refetch: refetchProject,
    isFetching: isProjectFetching,
  } = useGetProject(projectId)

  const {
    data: sessionsData,
    isLoading: isSessionsLoading,
    refetch: refetchSessions,
    isFetching: isSessionsFetching,
  } = useListSessions({ projectId })

  const isLoading = isProjectLoading || isSessionsLoading
  const isFetching = isProjectFetching || isSessionsFetching
  const project = projectData?.project

  const handleRefresh = () => {
    refetchProject()
    refetchSessions()
  }

  return (
    <div className="relative">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {isLoading ? (
          <LoadingSkeleton />
        ) : isProjectError || !project ? (
          <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
            <FolderGit2 className="mb-4 h-12 w-12 opacity-50" />
            <p className="font-mono text-sm">Project not found</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              The project may have been deleted or doesn't exist
            </p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Refresh button - fixed position */}
            <div className="flex justify-end">
              <button
                onClick={handleRefresh}
                disabled={isFetching}
                className="group flex items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-900/50 px-4 py-2 text-sm text-zinc-400 transition-all hover:border-zinc-700 hover:bg-zinc-800/50 hover:text-zinc-200 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <RefreshCw
                  className={`h-4 w-4 transition-transform ${isFetching ? 'animate-spin' : 'group-hover:rotate-90'}`}
                />
                <span className="font-mono text-xs">Refresh</span>
              </button>
            </div>

            {/* Project Header */}
            <ProjectHeader project={project} />

            {/* Two-column layout: Token Breakdown | Session List */}
            <div className="grid gap-6 lg:grid-cols-5">
              <div className="lg:col-span-2">
                <TokenBreakdown usage={project.usage} />
              </div>
              <div className="lg:col-span-3">
                <SessionList sessions={sessionsData?.sessions ?? []} />
              </div>
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
