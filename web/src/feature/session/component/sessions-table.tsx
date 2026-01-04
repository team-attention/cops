import { Link } from '@tanstack/react-router'
import {
  ChevronDown,
  ChevronUp,
  Clock,
  FolderGit2,
  GitBranch,
  Hash,
  MessageSquare,
  Zap,
} from 'lucide-react'
import type { SessionSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { Badge } from '@/gen/shadcn/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/gen/shadcn/ui/tooltip'
import {
  formatRelativeTime,
  formatTokenCount,
  truncateId,
} from '@/shared/util/format'

type SortField = 'started_at' | 'message_count' | 'usage'

interface SessionsTableProps {
  sessions: Array<SessionSummary>
  sortBy: SortField
  sortDesc: boolean
  onSortChange: (field: SortField) => void
  showProjectColumn?: boolean
}

const SortableHeader = ({
  field,
  currentSort,
  sortDesc,
  onSort,
  children,
  align = 'left',
}: {
  field: SortField
  currentSort: SortField
  sortDesc: boolean
  onSort: (field: SortField) => void
  children: React.ReactNode
  align?: 'left' | 'right'
}) => {
  const isActive = field === currentSort
  return (
    <TableHead
      className={`cursor-pointer font-mono text-[10px] uppercase tracking-widest text-zinc-600 hover:text-zinc-400 ${align === 'right' ? 'text-right' : ''}`}
      onClick={() => onSort(field)}
    >
      <div
        className={`flex items-center gap-1 ${align === 'right' ? 'justify-end' : ''}`}
      >
        {children}
        {isActive &&
          (sortDesc ? (
            <ChevronDown className="h-3 w-3" />
          ) : (
            <ChevronUp className="h-3 w-3" />
          ))}
      </div>
    </TableHead>
  )
}

export const SessionsTable = ({
  sessions,
  sortBy,
  sortDesc,
  onSortChange,
  showProjectColumn = true,
}: SessionsTableProps) => {
  if (sessions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-600">
        <MessageSquare className="mb-3 h-10 w-10" />
        <p className="font-mono text-sm">No sessions found</p>
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow className="border-zinc-800/50 hover:bg-transparent">
          <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
            Session
          </TableHead>
          {showProjectColumn && (
            <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
              Project
            </TableHead>
          )}
          <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
            Branch
          </TableHead>
          <SortableHeader
            field="message_count"
            currentSort={sortBy}
            sortDesc={sortDesc}
            onSort={onSortChange}
            align="right"
          >
            Messages
          </SortableHeader>
          <SortableHeader
            field="usage"
            currentSort={sortBy}
            sortDesc={sortDesc}
            onSort={onSortChange}
            align="right"
          >
            Tokens
          </SortableHeader>
          <SortableHeader
            field="started_at"
            currentSort={sortBy}
            sortDesc={sortDesc}
            onSort={onSortChange}
            align="right"
          >
            Started
          </SortableHeader>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sessions.map((session) => {
          const isActive = !session.endedAt
          const inputTokens = session.usage?.totalInputTokens ?? 0n
          const outputTokens = session.usage?.totalOutputTokens ?? 0n
          const totalTokens = inputTokens + outputTokens

          return (
            <TableRow
              key={session.id}
              className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
            >
              <TableCell>
                <Link
                  to="/sessions/$sessionId"
                  params={{ sessionId: session.id }}
                  className="flex items-center gap-2"
                >
                  <Badge
                    variant="outline"
                    className="border-violet-500/30 bg-violet-500/10 font-mono text-[10px] text-violet-400 transition-colors group-hover:border-violet-500/50"
                  >
                    <Hash className="mr-0.5 h-2.5 w-2.5" />
                    {truncateId(session.id)}
                  </Badge>
                  {isActive && (
                    <div className="flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0.5">
                      <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-400" />
                      <span className="font-mono text-[9px] uppercase tracking-wider text-emerald-400">
                        Active
                      </span>
                    </div>
                  )}
                </Link>
              </TableCell>
              {showProjectColumn && (
                <TableCell>
                  <Link
                    to="/projects/$projectId"
                    params={{ projectId: session.projectId }}
                    className="flex items-center gap-1.5 text-zinc-400 transition-colors hover:text-cyan-400"
                  >
                    <FolderGit2 className="h-3 w-3" />
                    <span className="font-mono text-xs">
                      {truncateId(session.projectId, 8)}
                    </span>
                  </Link>
                </TableCell>
              )}
              <TableCell>
                <Badge
                  variant="outline"
                  className="border-zinc-700/50 bg-zinc-800/50 font-mono text-[10px] text-zinc-400"
                >
                  <GitBranch className="mr-1 h-3 w-3" />
                  {session.gitBranch || 'main'}
                </Badge>
              </TableCell>
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-1">
                  <span className="font-mono text-sm text-zinc-300">
                    {session.messageCount}
                  </span>
                  <MessageSquare className="h-3 w-3 text-zinc-600" />
                </div>
              </TableCell>
              <TableCell className="text-right">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="flex items-center justify-end gap-1.5">
                      <Zap className="h-3 w-3 text-violet-500/70" />
                      <span className="font-mono text-sm text-violet-400">
                        {formatTokenCount(totalTokens)}
                      </span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="border-zinc-700 bg-zinc-900">
                    <div className="space-y-1 font-mono text-xs">
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Input:</span>
                        <span className="text-zinc-300">
                          {formatTokenCount(inputTokens)}
                        </span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Output:</span>
                        <span className="text-zinc-300">
                          {formatTokenCount(outputTokens)}
                        </span>
                      </div>
                    </div>
                  </TooltipContent>
                </Tooltip>
              </TableCell>
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-1 text-zinc-500">
                  <Clock className="h-3 w-3" />
                  <span className="font-mono text-xs">
                    {formatRelativeTime(session.startedAt)}
                  </span>
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
