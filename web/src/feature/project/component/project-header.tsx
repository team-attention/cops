import { Link } from '@tanstack/react-router'
import { ChevronRight, Clock, FolderGit2, Terminal } from 'lucide-react'
import type {
  ProjectDetail,
  TokenUsageSummary,
} from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import type { Timestamp } from '@bufbuild/protobuf/wkt'

interface ProjectHeaderProps {
  project: ProjectDetail
}

const formatTokenCount = (value: bigint | undefined): string => {
  if (!value) return '0'
  const num = Number(value)
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toLocaleString()
}

const formatRelativeTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return 'Never'
  const date = new Date(Number(timestamp.seconds) * 1000)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

const getTotalTokens = (usage: TokenUsageSummary | undefined): bigint => {
  if (!usage) return 0n
  return (
    (usage.totalInputTokens ?? 0n) +
    (usage.totalOutputTokens ?? 0n) +
    (usage.totalCacheCreationTokens ?? 0n) +
    (usage.totalCacheReadTokens ?? 0n)
  )
}

export const ProjectHeader = ({ project }: ProjectHeaderProps) => {
  const totalTokens = formatTokenCount(getTotalTokens(project.usage))

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-2 font-mono text-xs text-zinc-500">
        <Link to="/dashboard" className="transition-colors hover:text-cyan-400">
          Dashboard
        </Link>
        <ChevronRight className="h-3 w-3" />
        <Link
          to="/projects"
          search={{} as never}
          className="transition-colors hover:text-cyan-400"
        >
          Projects
        </Link>
        <ChevronRight className="h-3 w-3" />
        <span className="text-zinc-300">{project.name}</span>
      </nav>

      {/* Main Header Card */}
      <div className="group relative overflow-hidden rounded-xl border border-zinc-800/50 bg-zinc-900/80 p-6 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
        {/* Ambient glow */}
        <div className="pointer-events-none absolute -right-20 -top-20 h-40 w-40 rounded-full bg-cyan-500/10 blur-3xl transition-all duration-500 group-hover:bg-cyan-500/15" />
        <div className="pointer-events-none absolute -bottom-10 -left-10 h-32 w-32 rounded-full bg-violet-500/10 blur-3xl transition-all duration-500 group-hover:bg-violet-500/15" />

        {/* Scanline overlay */}
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.05)_50%)] bg-[length:100%_4px] opacity-0 transition-opacity duration-300 group-hover:opacity-30" />

        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
          {/* Project Info */}
          <div className="flex items-start gap-4">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-xl bg-cyan-500/20 blur-xl" />
              <div className="relative rounded-xl border border-cyan-500/20 bg-zinc-950/80 p-4">
                <FolderGit2 className="h-8 w-8 text-cyan-400" />
              </div>
            </div>

            <div className="space-y-2">
              <h1 className="text-2xl font-bold tracking-tight text-zinc-100">
                {project.name}
              </h1>

              <div className="flex items-center gap-2 font-mono text-xs text-zinc-500">
                <Terminal className="h-3 w-3" />
                <span
                  className="max-w-xs truncate lg:max-w-md"
                  title={project.path}
                >
                  {project.path}
                </span>
              </div>

              <div className="flex flex-wrap items-center gap-3 pt-1">
                {/* Last Activity */}
                <div className="flex items-center gap-1.5 text-xs text-zinc-500">
                  <Clock className="h-3 w-3" />
                  <span>Active {formatRelativeTime(project.lastActivity)}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Stats Badges */}
          <div className="flex flex-wrap gap-3 lg:flex-col lg:items-end">
            <div className="flex items-center gap-3 rounded-lg border border-zinc-800 bg-zinc-950/50 px-4 py-2">
              <div className="text-right">
                <p className="font-mono text-xs text-zinc-500">Total Tokens</p>
                <p className="font-mono text-lg font-bold text-cyan-400">
                  {totalTokens}
                </p>
              </div>
              <div className="h-8 w-px bg-zinc-800" />
              <div className="text-right">
                <p className="font-mono text-xs text-zinc-500">Sessions</p>
                <p className="font-mono text-lg font-bold text-emerald-400">
                  {project.sessionCount}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 rounded-full border border-zinc-800 bg-zinc-950/50 px-3 py-1">
              <div className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
              <span className="font-mono text-[10px] uppercase tracking-wider text-zinc-500">
                Active
              </span>
            </div>
          </div>
        </div>

        {/* Bottom accent line */}
        <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-cyan-500 to-violet-500 transition-all duration-700 group-hover:w-full" />
      </div>
    </div>
  )
}
