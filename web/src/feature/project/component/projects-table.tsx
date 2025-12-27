import { Link, useNavigate } from '@tanstack/react-router'
import { FolderGit2, Clock, ChevronUp, ChevronDown, Zap } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/gen/shadcn/ui/tooltip'
import type { ProjectSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { formatRelativeTime, formatTokenCount, truncatePath } from '@/shared/util/format'

type SortField = 'name' | 'session_count' | 'last_activity' | 'usage'

interface ProjectsTableProps {
  projects: ProjectSummary[]
  sortBy: SortField
  sortDesc: boolean
  onSortChange: (field: SortField) => void
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
      <div className={`flex items-center gap-1 ${align === 'right' ? 'justify-end' : ''}`}>
        {children}
        {isActive && (sortDesc ? <ChevronDown className="h-3 w-3" /> : <ChevronUp className="h-3 w-3" />)}
      </div>
    </TableHead>
  )
}

export const ProjectsTable = ({ projects, sortBy, sortDesc, onSortChange }: ProjectsTableProps) => {
  const navigate = useNavigate()

  if (projects.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-zinc-600">
        <FolderGit2 className="mb-3 h-10 w-10" />
        <p className="font-mono text-sm">No projects found</p>
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow className="border-zinc-800/50 hover:bg-transparent">
          <SortableHeader field="name" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange}>
            Project
          </SortableHeader>
          <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
            Path
          </TableHead>
          <SortableHeader field="session_count" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
            Sessions
          </SortableHeader>
          <SortableHeader field="usage" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
            Tokens
          </SortableHeader>
          <SortableHeader field="last_activity" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
            Activity
          </SortableHeader>
        </TableRow>
      </TableHeader>
      <TableBody>
        {projects.map((project) => {
          const inputTokens = project.usage?.totalInputTokens ?? 0n
          const outputTokens = project.usage?.totalOutputTokens ?? 0n
          const totalTokens = inputTokens + outputTokens

          return (
            <TableRow
              key={project.id}
              className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
              onClick={() => navigate({ to: '/projects/$projectId', params: { projectId: project.id } })}
            >
              <TableCell>
                <Link
                  to="/projects/$projectId"
                  params={{ projectId: project.id }}
                  className="block"
                >
                  <span className="font-medium text-zinc-200 transition-colors group-hover:text-cyan-400">
                    {project.name}
                  </span>
                </Link>
              </TableCell>
              <TableCell>
                <span className="font-mono text-[10px] text-zinc-600">
                  {truncatePath(project.path)}
                </span>
              </TableCell>
              <TableCell className="text-right">
                <span className="font-mono text-sm text-zinc-300">
                  {project.sessionCount}
                </span>
              </TableCell>
              <TableCell className="text-right">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="flex items-center justify-end gap-1.5">
                      <Zap className="h-3 w-3 text-cyan-500/70" />
                      <span className="font-mono text-sm text-cyan-400">
                        {formatTokenCount(totalTokens)}
                      </span>
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="border-zinc-700 bg-zinc-900">
                    <div className="space-y-1 font-mono text-xs">
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Input:</span>
                        <span className="text-zinc-300">{formatTokenCount(inputTokens)}</span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Output:</span>
                        <span className="text-zinc-300">{formatTokenCount(outputTokens)}</span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Cache Read:</span>
                        <span className="text-zinc-300">{formatTokenCount(project.usage?.totalCacheReadTokens)}</span>
                      </div>
                    </div>
                  </TooltipContent>
                </Tooltip>
              </TableCell>
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-1 text-zinc-500">
                  <Clock className="h-3 w-3" />
                  <span className="font-mono text-xs">
                    {formatRelativeTime(project.lastActivity)}
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
