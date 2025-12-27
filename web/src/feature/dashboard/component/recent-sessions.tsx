import { Link } from '@tanstack/react-router'
import { MessageSquare, GitBranch, ChevronRight, Clock, Hash } from 'lucide-react'
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
import type { SessionSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { formatRelativeTime, truncateId } from '@/shared/util/format'

interface RecentSessionsProps {
  sessions: SessionSummary[]
}

export const RecentSessions = ({ sessions }: RecentSessionsProps) => {
  return (
    <Card className="border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
      <CardHeader className="border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-md bg-violet-500/10 p-2">
              <MessageSquare className="h-4 w-4 text-violet-400" />
            </div>
            <CardTitle className="text-sm font-medium uppercase tracking-widest text-zinc-400">
              Recent Sessions
            </CardTitle>
          </div>
          <Link
            to="/sessions"
            search={{} as never}
            className="group flex items-center gap-1 text-xs text-zinc-500 transition-colors hover:text-violet-400"
          >
            View all
            <ChevronRight className="h-3 w-3 transition-transform group-hover:translate-x-0.5" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-zinc-600">
            <MessageSquare className="mb-3 h-8 w-8" />
            <p className="font-mono text-sm">No sessions recorded yet</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-zinc-800/50 hover:bg-transparent">
                <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Session
                </TableHead>
                <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Branch
                </TableHead>
                <TableHead className="text-right font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Messages
                </TableHead>
                <TableHead className="text-right font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Started
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sessions.map((session, index) => (
                <TableRow
                  key={session.id}
                  className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
                  style={{ animationDelay: `${index * 50}ms` }}
                >
                  <TableCell className="py-3">
                    <Link
                      to="/sessions/$sessionId"
                      params={{ sessionId: session.id }}
                      className="block"
                    >
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="outline"
                          className="border-violet-500/30 bg-violet-500/10 font-mono text-[10px] text-violet-400 transition-colors group-hover:border-violet-500/50 group-hover:bg-violet-500/20"
                        >
                          <Hash className="mr-0.5 h-2.5 w-2.5" />
                          {truncateId(session.id)}
                        </Badge>
                      </div>
                    </Link>
                  </TableCell>
                  <TableCell className="py-3">
                    <Badge
                      variant="outline"
                      className="border-zinc-700/50 bg-zinc-800/50 font-mono text-[10px] text-zinc-400"
                    >
                      <GitBranch className="mr-1 h-3 w-3" />
                      {session.gitBranch || 'main'}
                    </Badge>
                  </TableCell>
                  <TableCell className="py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <span className="font-mono text-sm text-zinc-300">
                        {session.messageCount}
                      </span>
                      <MessageSquare className="h-3 w-3 text-zinc-600" />
                    </div>
                  </TableCell>
                  <TableCell className="py-3 text-right">
                    <div className="flex items-center justify-end gap-1 text-zinc-500">
                      <Clock className="h-3 w-3" />
                      <span className="font-mono text-xs">
                        {formatRelativeTime(session.startedAt)}
                      </span>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
