import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
import { FolderGit2, RefreshCw, ChevronRight, Home } from 'lucide-react'
import { useListProjects } from '@/feature/project/hook/use-list-projects'
import { ProjectsTable } from '@/feature/project/component/projects-table'
import { PaginationControls } from '@/shared/component/pagination-controls'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'

export const Route = createFileRoute('/projects/')({
  component: ProjectsListPage,
  validateSearch: (search: Record<string, unknown>) => ({
    page: Number(search.page) || 1,
    pageSize: Number(search.pageSize) || 20,
    sortBy: String(search.sortBy || 'last_activity'),
    sortDesc: search.sortDesc !== 'false' && search.sortDesc !== false,
  }),
})

const LoadingSkeleton = () => (
  <div className="space-y-4">
    <Skeleton className="h-12 w-full bg-zinc-800/50" />
    {[...Array(5)].map((_, i) => (
      <Skeleton key={i} className="h-16 w-full bg-zinc-800/50" />
    ))}
  </div>
)

function ProjectsListPage() {
  const navigate = useNavigate({ from: '/projects' })
  const { page, pageSize, sortBy, sortDesc } = Route.useSearch()

  const { data, isLoading, isError, refetch, isFetching } = useListProjects({
    page,
    pageSize,
  })

  const handlePageChange = (newPage: number) => {
    navigate({ search: (prev) => ({ ...prev, page: newPage }) })
  }

  const handlePageSizeChange = (newSize: number) => {
    navigate({ search: (prev) => ({ ...prev, pageSize: newSize, page: 1 }) })
  }

  const handleSortChange = (field: string) => {
    const newSortDesc = field === sortBy ? !sortDesc : true
    navigate({ search: (prev) => ({ ...prev, sortBy: field, sortDesc: newSortDesc }) })
  }

  const projects = data?.projects ?? []
  const pagination = data?.pagination

  return (
    <div className="relative min-h-screen bg-zinc-950">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Breadcrumb */}
        <nav className="mb-6 flex items-center gap-2 font-mono text-xs text-zinc-500">
          <Link to="/dashboard" className="flex items-center gap-1 hover:text-zinc-300">
            <Home className="h-3 w-3" />
            Dashboard
          </Link>
          <ChevronRight className="h-3 w-3" />
          <span className="text-zinc-300">Projects</span>
        </nav>

        {/* Header */}
        <div className="mb-8 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-xl bg-cyan-500/20 blur-xl" />
              <div className="relative rounded-xl border border-cyan-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm">
                <FolderGit2 className="h-6 w-6 text-cyan-400" />
              </div>
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Projects</h1>
              <p className="mt-0.5 font-mono text-xs text-zinc-600">
                {pagination ? `${pagination.totalCount} total` : 'Loading...'}
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
            <FolderGit2 className="mb-4 h-12 w-12 opacity-50" />
            <p className="font-mono text-sm">Failed to load projects</p>
            <button
              onClick={() => refetch()}
              className="mt-4 rounded-lg bg-zinc-800 px-4 py-2 font-mono text-xs text-zinc-300 transition-colors hover:bg-zinc-700"
            >
              Try again
            </button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
            <ProjectsTable
              projects={projects}
              sortBy={sortBy as 'name' | 'session_count' | 'last_activity' | 'usage'}
              sortDesc={sortDesc}
              onSortChange={handleSortChange}
            />
            {pagination && (
              <PaginationControls
                currentPage={pagination.currentPage}
                totalPages={pagination.totalPages}
                pageSize={pagination.pageSize}
                totalCount={Number(pagination.totalCount)}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
              />
            )}
          </div>
        )}

        {/* Footer */}
        <div className="mt-12 flex items-center justify-center gap-2 text-zinc-700">
          <div className="h-px flex-1 bg-gradient-to-r from-transparent to-zinc-800" />
          <span className="font-mono text-[10px] uppercase tracking-widest">C-Ops v0.1.0</span>
          <div className="h-px flex-1 bg-gradient-to-l from-transparent to-zinc-800" />
        </div>
      </div>
    </div>
  )
}
