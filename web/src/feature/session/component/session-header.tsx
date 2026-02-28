import { Link } from '@tanstack/react-router'
import {
  ArrowDownRight,
  ChevronRight,
  Clock,
  GitBranch,
  MessageSquare,
  Timer,
  Zap,
} from 'lucide-react'
import type { SessionDetail } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import { Badge } from '@/gen/shadcn/ui/badge'

interface SessionHeaderProps {
  session: SessionDetail
  totalMessageCount?: number
}

const formatTokenCount = (value: bigint | number | undefined): string => {
  if (!value) return '0'
  const num = Number(value)
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toLocaleString()
}

const formatDuration = (
  startedAt: Timestamp | undefined,
  endedAt: Timestamp | undefined,
): string => {
  if (!startedAt) return '-'
  const start = new Date(Number(startedAt.seconds) * 1000)
  const end = endedAt ? new Date(Number(endedAt.seconds) * 1000) : new Date()
  const diffMs = end.getTime() - start.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)

  if (diffMins < 1) return '<1m'
  if (diffMins < 60) return `${diffMins}m`
  if (diffHours < 24) return `${diffHours}h ${diffMins % 60}m`
  return `${Math.floor(diffHours / 24)}d ${diffHours % 24}h`
}

const formatTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return '-'
  const date = new Date(Number(timestamp.seconds) * 1000)
  return date.toLocaleString()
}

const truncateId = (id: string): string => {
  if (id.length <= 12) return id
  return `${id.slice(0, 6)}...${id.slice(-4)}`
}

export const SessionHeader = ({
  session,
  totalMessageCount,
}: SessionHeaderProps) => {
  const isActive = !session.endedAt
  const messageCount = totalMessageCount ?? session.sessions?.length ?? 0
  const inputTokens = formatTokenCount(session.usage?.totalInputTokens)
  const outputTokens = formatTokenCount(session.usage?.totalOutputTokens)
  const duration = formatDuration(session.startedAt, session.endedAt)

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
        <Link
          to="/projects/$projectId"
          params={{ projectId: session.projectId }}
          className="transition-colors hover:text-cyan-400"
        >
          Project
        </Link>
        <ChevronRight className="h-3 w-3" />
        <span className="text-zinc-300">{truncateId(session.id)}</span>
      </nav>

      {/* Main Header Card */}
      <div className="group relative overflow-hidden rounded-xl border border-zinc-800/50 bg-zinc-900/80 p-6 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
        {/* Ambient glow */}
        <div className="pointer-events-none absolute -right-20 -top-20 h-40 w-40 rounded-full bg-violet-500/10 blur-3xl transition-all duration-500 group-hover:bg-violet-500/15" />
        <div className="pointer-events-none absolute -bottom-10 -left-10 h-32 w-32 rounded-full bg-cyan-500/10 blur-3xl transition-all duration-500 group-hover:bg-cyan-500/15" />

        {/* Scanline overlay */}
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.05)_50%)] bg-[length:100%_4px] opacity-0 transition-opacity duration-300 group-hover:opacity-30" />

        <div className="relative flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
          {/* Session Info */}
          <div className="flex items-start gap-4">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-xl bg-violet-500/20 blur-xl" />
              <div className="relative rounded-xl border border-violet-500/20 bg-zinc-950/80 p-4">
                <MessageSquare className="h-8 w-8 text-violet-400" />
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <h1 className="font-mono text-xl font-bold tracking-tight text-zinc-100">
                  Session {truncateId(session.id)}
                </h1>
                {isActive && (
                  <div className="flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5">
                    <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
                    <span className="font-mono text-[9px] uppercase tracking-wider text-emerald-400">
                      Active
                    </span>
                  </div>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-3 pt-1">
                {/* Git Branch Badge */}
                <Badge
                  variant="outline"
                  className="border-violet-500/30 bg-violet-500/10 text-violet-300"
                >
                  <GitBranch className="mr-1 h-3 w-3" />
                  {session.gitBranch || 'main'}
                </Badge>

                {/* Version */}
                <Badge
                  variant="outline"
                  className="border-zinc-700 bg-zinc-800/50 font-mono text-zinc-400"
                >
                  v{session.version || '?'}
                </Badge>

                {/* Started At */}
                <div className="flex items-center gap-1.5 text-xs text-zinc-500">
                  <Clock className="h-3 w-3" />
                  <span>{formatTime(session.startedAt)}</span>
                </div>
              </div>
            </div>
          </div>

          {/* Stats Grid */}
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            {/* Messages */}
            <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-4 py-3 text-center">
              <div className="flex items-center justify-center gap-1.5 text-zinc-500">
                <MessageSquare className="h-3 w-3" />
                <span className="font-mono text-[10px] uppercase tracking-wider">
                  Messages
                </span>
              </div>
              <p className="mt-1 font-mono text-xl font-bold text-zinc-100">
                {messageCount}
              </p>
            </div>

            {/* Input Tokens */}
            <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-4 py-3 text-center">
              <div className="flex items-center justify-center gap-1.5 text-zinc-500">
                <ArrowDownRight className="h-3 w-3" />
                <span className="font-mono text-[10px] uppercase tracking-wider">
                  Input
                </span>
              </div>
              <p className="mt-1 font-mono text-xl font-bold text-cyan-400">
                {inputTokens}
              </p>
            </div>

            {/* Output Tokens */}
            <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-4 py-3 text-center">
              <div className="flex items-center justify-center gap-1.5 text-zinc-500">
                <Zap className="h-3 w-3" />
                <span className="font-mono text-[10px] uppercase tracking-wider">
                  Output
                </span>
              </div>
              <p className="mt-1 font-mono text-xl font-bold text-violet-400">
                {outputTokens}
              </p>
            </div>

            {/* Duration */}
            <div className="rounded-lg border border-zinc-800 bg-zinc-950/50 px-4 py-3 text-center">
              <div className="flex items-center justify-center gap-1.5 text-zinc-500">
                <Timer className="h-3 w-3" />
                <span className="font-mono text-[10px] uppercase tracking-wider">
                  Duration
                </span>
              </div>
              <p className="mt-1 font-mono text-xl font-bold text-emerald-400">
                {duration}
              </p>
            </div>
          </div>
        </div>

        {/* Bottom accent line */}
        <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-violet-500 to-cyan-500 transition-all duration-700 group-hover:w-full" />
      </div>
    </div>
  )
}
