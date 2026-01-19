import { FolderGit2 } from 'lucide-react'
import type { ProjectSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/gen/shadcn/ui/select'

interface ProjectFilterProps {
  projects: Array<ProjectSummary>
  selectedProjectId: string | null
  onProjectChange: (projectId: string | null) => void
  isLoading?: boolean
}

export const ProjectFilter = ({
  projects,
  selectedProjectId,
  onProjectChange,
  isLoading = false,
}: ProjectFilterProps) => {
  return (
    <div className="flex items-center gap-2">
      <FolderGit2 className="h-4 w-4 text-zinc-500" />
      <Select
        value={selectedProjectId ?? 'all'}
        onValueChange={(v) => onProjectChange(v === 'all' ? null : v)}
        disabled={isLoading}
      >
        <SelectTrigger className="w-[200px] border-zinc-700 bg-zinc-800/50 font-mono text-sm">
          <SelectValue placeholder="All Projects" />
        </SelectTrigger>
        <SelectContent className="border-zinc-700 bg-zinc-900">
          <SelectItem value="all" className="font-mono text-sm">
            All Projects
          </SelectItem>
          {projects.map((project) => (
            <SelectItem
              key={project.id}
              value={project.id}
              className="font-mono text-sm"
            >
              {project.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
