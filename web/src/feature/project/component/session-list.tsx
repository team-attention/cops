import { Link } from '@tanstack/react-router'
import { MessageSquare, GitBranch, Zap, ExternalLink, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { Badge } from '@/gen/shadcn/ui/badge'
import type { SessionSummary, TokenUsageSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { formatRelativeTime, formatTokenCount } from '@/shared/util/format'

interface SessionListProps {
  sessions: SessionSummary[]
}

const getTotalTokens = (usage: TokenUsageSummary | undefined): bigint => {
  if (!usage) return 0n
  return (
    (usage.totalInputTokens ?? 0n) +
    (usage.totalOutputTokens ?? 0n)
  )
}

const truncateId = (id: string): string => {
  if (id.length <= 12) return id
  return `${id.slice(0, 6)}...${id.slice(-4)}`
}

const SessionRow = ({ session }: { session: SessionSummary }) => {
  const isActive = !session.endedAt
  const totalTokens = getTotalTokens(session.usage)

  return (
    <TableRow className="group border-zinc-800/50 transition-colors hover:bg-zinc-800/30">
      <TableCell>
        <Link
          to="/sessions/$sessionId"
          params={{ sessionId: session.id }}
          className="group/link flex items-center gap-2 text-zinc-300 transition-colors hover:text-cyan-400"
        >
          <span className="font-mono text-sm" title={session.id}>
            {truncateId(session.id)}
          </span>
          <ExternalLink className="h-3 w-3 opacity-0 transition-opacity group-hover/link:opacity-100" />
        </Link>
      </TableCell>

      <TableCell>
        <Badge
          variant="outline"
          className="border-violet-500/30 bg-violet-500/10 font-mono text-[10px] text-violet-300"
        >
          <GitBranch className="mr-1 h-2.5 w-2.5" />
          {session.gitBranch || 'main'}
        </Badge>
      </TableCell>

      <TableCell>
        <div className="flex items-center gap-1.5 text-zinc-400">
          <MessageSquare className="h-3 w-3 text-zinc-600" />
          <span className="font-mono text-sm">{session.messageCount}</span>
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center gap-1.5">
          <Zap className="h-3 w-3 text-cyan-500/70" />
          <span className="font-mono text-sm text-cyan-400">
            {formatTokenCount(totalTokens)}
          </span>
        </div>
      </TableCell>

      <TableCell>
        <div className="flex items-center gap-2">
          <Clock className="h-3 w-3 text-zinc-600" />
          <span className="font-mono text-xs text-zinc-500">
            {formatRelativeTime(session.startedAt)}
          </span>
          {isActive && (
            <div className="flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5">
              <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
              <span className="font-mono text-[9px] uppercase tracking-wider text-emerald-400">
                Active
              </span>
            </div>
          )}
        </div>
      </TableCell>
    </TableRow>
  )
}

export const SessionList = ({ sessions }: SessionListProps) => {
  return (
    <Card className="group relative flex flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient glow */}
      <div className="pointer-events-none absolute -left-16 -top-16 h-32 w-32 rounded-full bg-emerald-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-emerald-500/10" />

      <CardHeader className="border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-2">
              <MessageSquare className="h-4 w-4 text-emerald-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Sessions
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                All project sessions
              </p>
            </div>
          </div>
          <Badge
            variant="outline"
            className="border-zinc-700 bg-zinc-800/50 font-mono text-xs text-zinc-400"
          >
            {sessions.length} sessions
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="flex-1 p-0">
        {sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-zinc-500">
            <MessageSquare className="mb-3 h-10 w-10 opacity-30" />
            <p className="font-mono text-sm">No sessions yet</p>
            <p className="mt-1 font-mono text-xs text-zinc-600">
              Sessions will appear here when created
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-zinc-800/50 hover:bg-transparent">
                  <TableHead className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
                    Session ID
                  </TableHead>
                  <TableHead className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
                    Branch
                  </TableHead>
                  <TableHead className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
                    Messages
                  </TableHead>
                  <TableHead className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
                    Tokens
                  </TableHead>
                  <TableHead className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
                    Started
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <SessionRow key={session.id} session={session} />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-emerald-500 to-cyan-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
