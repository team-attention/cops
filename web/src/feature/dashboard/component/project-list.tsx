import { Link } from '@tanstack/react-router'
import { ChevronRight, Clock, FolderGit2 } from 'lucide-react'
import type { ProjectSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { formatRelativeTime, truncatePath } from '@/shared/util/format'

interface ProjectListProps {
  projects: Array<ProjectSummary>
}

export const ProjectList = ({ projects }: ProjectListProps) => {
  return (
    <Card className="border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
      <CardHeader className="border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-md bg-cyan-500/10 p-2">
              <FolderGit2 className="h-4 w-4 text-cyan-400" />
            </div>
            <CardTitle className="text-sm font-medium uppercase tracking-widest text-zinc-400">
              Recent Projects
            </CardTitle>
          </div>
          <Link
            to="/projects"
            search={{} as never}
            className="group flex items-center gap-1 text-xs text-zinc-500 transition-colors hover:text-cyan-400"
          >
            View all
            <ChevronRight className="h-3 w-3 transition-transform group-hover:translate-x-0.5" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {projects.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-zinc-600">
            <FolderGit2 className="mb-3 h-8 w-8" />
            <p className="font-mono text-sm">No projects tracked yet</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="border-zinc-800/50 hover:bg-transparent">
                <TableHead className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Project
                </TableHead>
                <TableHead className="text-right font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Sessions
                </TableHead>
                <TableHead className="text-right font-mono text-[10px] uppercase tracking-widest text-zinc-600">
                  Activity
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {projects.map((project, index) => (
                <TableRow
                  key={project.id}
                  className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
                  style={{ animationDelay: `${index * 50}ms` }}
                >
                  <TableCell className="py-3">
                    <Link
                      to="/projects/$projectId"
                      params={{ projectId: project.id }}
                      className="block"
                    >
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium text-zinc-200 transition-colors group-hover:text-cyan-400">
                          {project.name}
                        </span>
                        <span className="font-mono text-[10px] text-zinc-600">
                          {truncatePath(project.path)}
                        </span>
                      </div>
                    </Link>
                  </TableCell>
                  <TableCell className="py-3 text-right">
                    <span className="font-mono text-sm text-zinc-300">
                      {project.sessionCount}
                    </span>
                  </TableCell>
                  <TableCell className="py-3 text-right">
                    <div className="flex items-center justify-end gap-1 text-zinc-500">
                      <Clock className="h-3 w-3" />
                      <span className="font-mono text-xs">
                        {formatRelativeTime(project.lastActivity)}
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
