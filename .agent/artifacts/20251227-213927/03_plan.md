# Implementation Plan

## Overview
Implement `/projects` and `/sessions` list pages with full pagination, sorting, and filtering. This includes refactoring dashboard.proto to use Req/Res naming convention, modifying backend to support listing all sessions, and building frontend components following existing patterns.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| Pagination UI | shadcn/ui pagination | (shadcn CLI) | Consistent with existing UI system, already uses Radix primitives |
| Project Filter Dropdown | shadcn/ui select | (shadcn CLI) | Required for project filter on sessions page |
| Table Sorting | Native implementation | (custom) | Simple state management, no additional dependency needed |
| Date/Time Formatting | Extracted shared utilities | (custom) | Consolidate duplicated formatters from existing components |

## Architecture Decisions

### Decision 1: Protobuf Naming Convention
**Choice**: Rename all `*Request` to `*Req` and all `*Response` to `*Res` in `dashboard.proto`
**Rationale**: Follows the established convention in `aggregation.proto` (see `SendLogsReq`/`SendLogsRes`), aligns with project rules in `.agent/rules/idl/protobuf.md`

### Decision 2: ListSessions API Modification
**Choice**: Modify existing `ListSessions` RPC to treat empty `project_id` as "all projects" rather than adding a new RPC
**Rationale**: Simpler implementation, no breaking changes to existing callers, follows user decision from requirements

### Decision 3: URL State Management
**Choice**: Use TanStack Router search params for pagination/sorting state
**Rationale**: Enables bookmarkable URLs, browser back/forward navigation, follows existing patterns in the codebase

### Decision 4: Shared Utilities Location
**Choice**: Create `shared/util/format.ts` for extracted formatters
**Rationale**: Follows FDD structure from `.agent/rules/react/react-web-src.md`, avoids code duplication

### Decision 5: Component Organization
**Choice**: Create new table components in feature directories (`project/component/projects-table.tsx`, `session/component/sessions-table.tsx`)
**Rationale**: Full-featured tables differ significantly from dashboard's recent items tables, warrants separate components

## Implementation Steps

### Step 1: Protobuf Refactoring

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto` (modify)

**Changes**:
Rename all Request/Response message types:

| Old Name | New Name |
| -------- | -------- |
| `PaginationRequest` | `PaginationReq` |
| `PaginationResponse` | `PaginationRes` |
| `GetOverviewRequest` | `GetOverviewReq` |
| `GetOverviewResponse` | `GetOverviewRes` |
| `ListProjectsRequest` | `ListProjectsReq` |
| `ListProjectsResponse` | `ListProjectsRes` |
| `GetProjectRequest` | `GetProjectReq` |
| `GetProjectResponse` | `GetProjectRes` |
| `ListSessionsRequest` | `ListSessionsReq` |
| `ListSessionsResponse` | `ListSessionsRes` |
| `GetSessionRequest` | `GetSessionReq` |
| `GetSessionResponse` | `GetSessionRes` |

**After Changes - Run**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| buf generate succeeds | Run `buf generate` | No errors, files regenerated in `shared/gen/grpcstub/` | Generation success |
| Go compilation succeeds | Run `go build ./...` from root | Build succeeds after handler updates | Type compatibility |

---

### Step 2: Update API Handler for New Types

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go` (modify)

**Changes**:
Update all method signatures to use new Req/Res types. The generated code will have new type names.

```go
// Before (all methods):
func (h *DashboardGRPCHandler) GetOverview(
    ctx context.Context,
    req *connect.Request[dashboardv1.GetOverviewRequest],
) (*connect.Response[dashboardv1.GetOverviewResponse], error)

// After (all methods):
func (h *DashboardGRPCHandler) GetOverview(
    ctx context.Context,
    req *connect.Request[dashboardv1.GetOverviewReq],
) (*connect.Response[dashboardv1.GetOverviewRes], error)
```

Methods to update:
1. `GetOverview` - lines 36-68
2. `ListProjects` - lines 71-102
3. `GetProject` - lines 105-124
4. `ListSessions` - lines 127-162
5. `GetSession` - lines 165-184

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| API compiles | `go build ./api/...` | No errors | Type compatibility |
| GetOverview works | Call GetOverview RPC | Returns overview data | Handler routing |

---

### Step 3: Modify Backend ListSessions for Empty ProjectID

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` (modify)

**Changes**:
Modify `ListSessions` method (lines 298-390) to conditionally apply project filter:

```go
func (r *MongoDashboardRepository) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
    // Build aggregation pipeline
    pipeline := bson.A{}

    // Only filter by project if projectID is provided
    if params.Query.ProjectID != "" {
        projectOID, err := bson.ObjectIDFromHex(params.Query.ProjectID)
        if err != nil {
            return nil, fmt.Errorf("invalid project ID: %w", err)
        }
        pipeline = append(pipeline, bson.M{"$match": bson.M{mongoschema.SessionRecordProjectIDField: projectOID}})
    }

    // Add group stage
    pipeline = append(pipeline, bson.M{"$group": bson.M{
        "_id":                                   "$" + mongoschema.SessionRecordSessionIDField,
        "messageCount":                          bson.M{"$sum": 1},
        "startedAt":                             bson.M{"$min": "$" + mongoschema.SessionRecordTimestampField},
        "endedAt":                               bson.M{"$max": "$" + mongoschema.SessionRecordTimestampField},
        mongoschema.SessionRecordProjectIDField: bson.M{"$first": "$" + mongoschema.SessionRecordProjectIDField},
        mongoschema.SessionRecordGitBranchField: bson.M{"$first": "$" + mongoschema.SessionRecordGitBranchField},
        mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
        mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
        mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
    }})

    // Add sort stage
    sortField := "startedAt"
    if params.Query.SortBy != "" {
        sortField = params.Query.SortBy
    }
    sortOrder := -1
    if !params.Query.SortDesc {
        sortOrder = 1
    }
    pipeline = append(pipeline, bson.M{"$sort": bson.M{sortField: sortOrder}})

    // ... rest unchanged (count, pagination, cursor iteration)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Empty projectID returns all | `projectId: ""` | All sessions across projects | No filter branch |
| Valid projectID filters | `projectId: "validHex"` | Only sessions for that project | With filter branch |
| Invalid projectID returns error | `projectId: "invalid"` | Error: "invalid project ID" | Validation error branch |

---

### Step 4: Install shadcn/ui Components

**Commands to Run**:
```bash
cd /Users/jayce/team-attention/cops/web && npx shadcn@latest add pagination
cd /Users/jayce/team-attention/cops/web && npx shadcn@latest add select
```

**Files Created** (by shadcn CLI):
- `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/pagination.tsx`
- `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/select.tsx`

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Components installed | Check file existence | Files present in gen/shadcn/ui/ | Installation success |
| Imports work | `import { Pagination } from '@/gen/shadcn/ui/pagination'` | No TypeScript errors | Import resolution |

---

### Step 5: Create Shared Formatting Utilities

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/shared/util/format.ts` (create)

**Functions**:

```typescript
import type { Timestamp } from '@bufbuild/protobuf/wkt'

/**
 * Formats a protobuf Timestamp to relative time string (e.g., "5m ago", "2d ago")
 */
export const formatRelativeTime = (timestamp: Timestamp | undefined): string => {
  if (!timestamp) return '-'
  const date = new Date(Number(timestamp.seconds) * 1000)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

/**
 * Formats a token count to abbreviated form (e.g., "1.2M", "45K")
 */
export const formatTokenCount = (value: bigint | number | undefined): string => {
  if (value === undefined || value === null) return '0'
  const num = typeof value === 'bigint' ? Number(value) : value
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(0)}K`
  return num.toLocaleString()
}

/**
 * Truncates a string ID with ellipsis (e.g., "abc123...xyz9")
 */
export const truncateId = (id: string, maxLength = 12): string => {
  if (id.length <= maxLength) return id
  const half = Math.floor((maxLength - 3) / 2)
  return `${id.slice(0, half)}...${id.slice(-half)}`
}

/**
 * Truncates a file path, keeping the last N segments
 */
export const truncatePath = (path: string, maxLength = 40): string => {
  if (path.length <= maxLength) return path
  const parts = path.split('/')
  if (parts.length <= 3) return path
  return `.../${parts.slice(-2).join('/')}`
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Recent time | timestamp 30 seconds ago | "just now" | < 1 minute |
| Minutes ago | timestamp 5 minutes ago | "5m ago" | 1-59 minutes |
| Hours ago | timestamp 3 hours ago | "3h ago" | 1-23 hours |
| Days ago | timestamp 2 days ago | "2d ago" | 1-6 days |
| Older date | timestamp 10 days ago | "12/17/2025" | >= 7 days |
| Null timestamp | undefined | "-" | Null handling |
| Token millions | 1500000 | "1.5M" | >= 1M |
| Token thousands | 45000 | "45K" | >= 1K |
| Token small | 500 | "500" | < 1K |

---

### Step 6: Create Shared Pagination Controls Component

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/shared/component/pagination-controls.tsx` (create)

**Functions**:

```typescript
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/gen/shadcn/ui/pagination'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/gen/shadcn/ui/select'

interface PaginationControlsProps {
  currentPage: number
  totalPages: number
  pageSize: number
  totalCount: number
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
  pageSizeOptions?: number[]
}

export const PaginationControls = ({
  currentPage,
  totalPages,
  pageSize,
  totalCount,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50],
}: PaginationControlsProps) => {
  // Implementation:
  // 1. Render page size selector (Select component)
  // 2. Show item range (e.g., "Showing 1-20 of 100")
  // 3. Render pagination links with ellipsis for large page counts
  // 4. Handle prev/next navigation with disabled states

  const startItem = (currentPage - 1) * pageSize + 1
  const endItem = Math.min(currentPage * pageSize, totalCount)

  // Generate visible page numbers with ellipsis
  const getVisiblePages = (): (number | 'ellipsis')[] => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1)
    }

    if (currentPage <= 3) {
      return [1, 2, 3, 4, 5, 'ellipsis', totalPages]
    }

    if (currentPage >= totalPages - 2) {
      return [1, 'ellipsis', totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages]
    }

    return [1, 'ellipsis', currentPage - 1, currentPage, currentPage + 1, 'ellipsis', totalPages]
  }

  return (
    <div className="flex items-center justify-between border-t border-zinc-800/50 px-4 py-3">
      {/* Left: Page size selector */}
      <div className="flex items-center gap-2">
        <span className="font-mono text-xs text-zinc-500">Show</span>
        <Select
          value={String(pageSize)}
          onValueChange={(v) => onPageSizeChange(Number(v))}
        >
          <SelectTrigger className="h-8 w-[70px] border-zinc-700 bg-zinc-800/50 font-mono text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent className="border-zinc-700 bg-zinc-900">
            {pageSizeOptions.map((size) => (
              <SelectItem key={size} value={String(size)} className="font-mono text-xs">
                {size}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="font-mono text-xs text-zinc-500">per page</span>
      </div>

      {/* Center: Item range */}
      <span className="font-mono text-xs text-zinc-500">
        {startItem}-{endItem} of {totalCount.toLocaleString()}
      </span>

      {/* Right: Page navigation */}
      <Pagination>
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious
              onClick={() => currentPage > 1 && onPageChange(currentPage - 1)}
              className={currentPage <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
            />
          </PaginationItem>

          {getVisiblePages().map((page, idx) => (
            <PaginationItem key={idx}>
              {page === 'ellipsis' ? (
                <PaginationEllipsis />
              ) : (
                <PaginationLink
                  onClick={() => onPageChange(page)}
                  isActive={page === currentPage}
                  className="cursor-pointer font-mono"
                >
                  {page}
                </PaginationLink>
              )}
            </PaginationItem>
          ))}

          <PaginationItem>
            <PaginationNext
              onClick={() => currentPage < totalPages && onPageChange(currentPage + 1)}
              className={currentPage >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
            />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| First page | currentPage=1 | Previous disabled, page 1 active | First page state |
| Last page | currentPage=totalPages | Next disabled, last page active | Last page state |
| Middle page | currentPage=5, totalPages=10 | Both nav enabled, ellipsis shown | Middle state |
| Few pages | totalPages=3 | All pages shown, no ellipsis | Small page count |
| Page size change | Select 50 | onPageSizeChange(50) called | Size change handler |

---

### Step 7: Create useListProjects Hook

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-projects.ts` (create)

**Functions**:

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { listProjects } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListProjectsOptions {
  page?: number
  pageSize?: number
  enabled?: boolean
}

export const useListProjects = ({
  page = 1,
  pageSize = 20,
  enabled = true,
}: UseListProjectsOptions = {}) => {
  return useQuery(
    listProjects,
    {
      pagination: { page, pageSize },
    },
    { enabled }
  )
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Default options | {} | Query with page=1, pageSize=20 | Default values |
| Custom pagination | { page: 2, pageSize: 50 } | Query with specified params | Custom values |
| Disabled query | { enabled: false } | No network request | Disabled state |

---

### Step 8: Update useListSessions Hook for Optional ProjectID

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-sessions.ts` (modify)

**Changes**:

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { listSessions } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListSessionsOptions {
  projectId?: string  // Changed from required to optional
  page?: number
  pageSize?: number
  sortBy?: string
  sortDesc?: boolean
  enabled?: boolean
}

export const useListSessions = ({
  projectId = '',  // Empty string means all projects
  page = 1,
  pageSize = 20,
  sortBy = 'started_at',
  sortDesc = true,
  enabled = true,
}: UseListSessionsOptions = {}) => {
  return useQuery(
    listSessions,
    {
      projectId,
      pagination: { page, pageSize },
      sortBy,
      sortDesc,
    },
    { enabled }
  )
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| No projectId | {} | Query with projectId='' (all sessions) | All sessions |
| With projectId | { projectId: 'abc123' } | Query filtered by project | Project filter |
| Custom sort | { sortBy: 'message_count', sortDesc: false } | Query with custom sort | Sort options |

---

### Step 9: Create Projects Table Component

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx` (create)

**Functions**:

```typescript
import { Link } from '@tanstack/react-router'
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Empty list | projects=[] | Empty state with icon | Empty state |
| With projects | projects=[...] | Table rows rendered | Data rendering |
| Sort by name | Click name header | onSortChange('name') called | Sort handler |
| Token tooltip | Hover token cell | Breakdown popup shown | Tooltip display |
| Row click | Click project row | Navigate to /projects/$projectId | Navigation |

---

### Step 10: Create Sessions Table Component

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/session/component/sessions-table.tsx` (create)

**Functions**:

```typescript
import { Link } from '@tanstack/react-router'
import { MessageSquare, GitBranch, Clock, ChevronUp, ChevronDown, Zap, Hash, FolderGit2 } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/gen/shadcn/ui/table'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/gen/shadcn/ui/tooltip'
import type { SessionSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { formatRelativeTime, formatTokenCount, truncateId } from '@/shared/util/format'

type SortField = 'started_at' | 'message_count' | 'usage'

interface SessionsTableProps {
  sessions: SessionSummary[]
  sortBy: SortField
  sortDesc: boolean
  onSortChange: (field: SortField) => void
  showProjectColumn?: boolean  // true when showing all sessions
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
          <SortableHeader field="message_count" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
            Messages
          </SortableHeader>
          <SortableHeader field="usage" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
            Tokens
          </SortableHeader>
          <SortableHeader field="started_at" currentSort={sortBy} sortDesc={sortDesc} onSort={onSortChange} align="right">
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
                    <span className="font-mono text-xs">{truncateId(session.projectId, 8)}</span>
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
                  <span className="font-mono text-sm text-zinc-300">{session.messageCount}</span>
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
                        <span className="text-zinc-300">{formatTokenCount(inputTokens)}</span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span className="text-zinc-500">Output:</span>
                        <span className="text-zinc-300">{formatTokenCount(outputTokens)}</span>
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Empty list | sessions=[] | Empty state | Empty state |
| With project column | showProjectColumn=true | Project column visible | All sessions view |
| Without project column | showProjectColumn=false | No project column | Project-specific view |
| Active session | session.endedAt=null | Green "Active" badge | Active indicator |
| Completed session | session.endedAt=timestamp | No active badge | Completed state |

---

### Step 11: Create Project Filter Component

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/session/component/project-filter.tsx` (create)

**Functions**:

```typescript
import { FolderGit2 } from 'lucide-react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/gen/shadcn/ui/select'
import type { ProjectSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface ProjectFilterProps {
  projects: ProjectSummary[]
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
            <SelectItem key={project.id} value={project.id} className="font-mono text-sm">
              {project.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| All projects selected | selectedProjectId=null | "All Projects" displayed | Default state |
| Project selected | selectedProjectId="abc" | Project name displayed | Selected state |
| Change to all | Select "All Projects" | onProjectChange(null) called | Clear filter |
| Change to project | Select specific project | onProjectChange(projectId) called | Set filter |

---

### Step 12: Implement /projects Route Page

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/route/projects/index.tsx` (modify - replace placeholder)

**Functions**:

```typescript
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Initial load | Navigate to /projects | Shows loading skeleton, then data | Loading state |
| Error state | API error | Error message with retry button | Error handling |
| Empty state | No projects | Empty state message | Empty list |
| Pagination | Click page 2 | URL updates, new data loads | Pagination |
| Sort change | Click column header | URL updates with new sort | Sorting |
| Refresh | Click refresh button | Data reloads with spinner | Refresh |

---

### Step 13: Implement /sessions Route Page

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/route/sessions/index.tsx` (modify - replace placeholder)

**Functions**:

```typescript
import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
import { MessageSquare, RefreshCw, ChevronRight, Home } from 'lucide-react'
import { useListSessions } from '@/feature/project/hook/use-list-sessions'
import { useListProjects } from '@/feature/project/hook/use-list-projects'
import { SessionsTable } from '@/feature/session/component/sessions-table'
import { ProjectFilter } from '@/feature/session/component/project-filter'
import { PaginationControls } from '@/shared/component/pagination-controls'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'

export const Route = createFileRoute('/sessions/')({
  component: SessionsListPage,
  validateSearch: (search: Record<string, unknown>) => ({
    page: Number(search.page) || 1,
    pageSize: Number(search.pageSize) || 20,
    sortBy: String(search.sortBy || 'started_at'),
    sortDesc: search.sortDesc !== 'false' && search.sortDesc !== false,
    projectId: search.projectId ? String(search.projectId) : null,
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

function SessionsListPage() {
  const navigate = useNavigate({ from: '/sessions' })
  const { page, pageSize, sortBy, sortDesc, projectId } = Route.useSearch()

  // Fetch projects for filter dropdown
  const { data: projectsData, isLoading: isProjectsLoading } = useListProjects({
    page: 1,
    pageSize: 100, // Get all projects for dropdown
  })

  // Fetch sessions with optional project filter
  const { data, isLoading, isError, refetch, isFetching } = useListSessions({
    projectId: projectId ?? '',
    page,
    pageSize,
    sortBy,
    sortDesc,
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

  const handleProjectChange = (newProjectId: string | null) => {
    navigate({ search: (prev) => ({ ...prev, projectId: newProjectId, page: 1 }) })
  }

  const sessions = data?.sessions ?? []
  const pagination = data?.pagination
  const projects = projectsData?.projects ?? []

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
          <span className="text-zinc-300">Sessions</span>
        </nav>

        {/* Header */}
        <div className="mb-8 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="relative">
              <div className="absolute inset-0 animate-pulse rounded-xl bg-violet-500/20 blur-xl" />
              <div className="relative rounded-xl border border-violet-500/20 bg-zinc-900/80 p-3 backdrop-blur-sm">
                <MessageSquare className="h-6 w-6 text-violet-400" />
              </div>
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-zinc-100">Sessions</h1>
              <p className="mt-0.5 font-mono text-xs text-zinc-600">
                {pagination ? `${pagination.totalCount} total` : 'Loading...'}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <ProjectFilter
              projects={projects}
              selectedProjectId={projectId}
              onProjectChange={handleProjectChange}
              isLoading={isProjectsLoading}
            />

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
        </div>

        {/* Content */}
        {isLoading ? (
          <LoadingSkeleton />
        ) : isError ? (
          <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
            <MessageSquare className="mb-4 h-12 w-12 opacity-50" />
            <p className="font-mono text-sm">Failed to load sessions</p>
            <button
              onClick={() => refetch()}
              className="mt-4 rounded-lg bg-zinc-800 px-4 py-2 font-mono text-xs text-zinc-300 transition-colors hover:bg-zinc-700"
            >
              Try again
            </button>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm">
            <SessionsTable
              sessions={sessions}
              sortBy={sortBy as 'started_at' | 'message_count' | 'usage'}
              sortDesc={sortDesc}
              onSortChange={handleSortChange}
              showProjectColumn={!projectId}
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| All sessions | No projectId | All sessions shown, project column visible | All sessions view |
| Filtered by project | projectId="abc" | Only project sessions, no project column | Project filter |
| Filter change | Select different project | URL updates, data reloads, page resets to 1 | Filter change |
| Clear filter | Select "All Projects" | URL clears projectId, shows all | Clear filter |

---

### Step 14: Create Session Feature Directory

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/session/` (create directory structure)

**Directory Structure**:
```
web/src/feature/session/
├── component/
│   ├── sessions-table.tsx    (from Step 10)
│   └── project-filter.tsx    (from Step 11)
```

This step is organizational - ensure the feature directory exists before creating components.

---

### Step 15: Update Existing Components to Use Shared Utilities

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx` (modify)
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/recent-sessions.tsx` (modify)
- `/Users/jayce/team-attention/cops/web/src/feature/project/component/session-list.tsx` (modify)

**Changes**:
Replace inline `formatTimestamp`, `formatTokenCount`, `truncateId`, `truncatePath` functions with imports from shared utilities:

```typescript
// Before (in each file):
const formatTimestamp = (timestamp: Timestamp | undefined): string => {
  // ... duplicated code
}

// After (in each file):
import { formatRelativeTime, formatTokenCount, truncateId, truncatePath } from '@/shared/util/format'

// Replace usage:
// formatTimestamp(project.lastActivity) -> formatRelativeTime(project.lastActivity)
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Dashboard project list | Navigate to /dashboard | Projects display with formatted times | Shared util integration |
| Dashboard recent sessions | Navigate to /dashboard | Sessions display with formatted times | Shared util integration |
| Project detail sessions | Navigate to /projects/$id | Sessions display correctly | Shared util integration |

---

## Execution Order

1. **Step 1: Protobuf Refactoring** (no dependencies)
2. **Step 2: Update API Handler** (depends on Step 1 - needs regenerated types)
3. **Step 3: Backend ListSessions Modification** (no dependencies, can parallel with Step 2)
4. **Step 4: Install shadcn/ui Components** (no dependencies, can parallel)
5. **Step 5: Create Shared Formatting Utilities** (no dependencies, can parallel)
6. **Step 6: Create Pagination Controls** (depends on Step 4 - needs pagination component)
7. **Step 7: Create useListProjects Hook** (depends on Step 1 - needs regenerated types)
8. **Step 8: Update useListSessions Hook** (depends on Step 1 - needs regenerated types)
9. **Step 9: Create Projects Table** (depends on Steps 5, 7)
10. **Step 10: Create Sessions Table** (depends on Steps 5, 8)
11. **Step 11: Create Project Filter** (depends on Step 4 - needs select component)
12. **Step 12: Implement /projects Route** (depends on Steps 6, 9)
13. **Step 13: Implement /sessions Route** (depends on Steps 6, 10, 11)
14. **Step 14: Create Session Feature Directory** (do before Steps 10, 11)
15. **Step 15: Update Existing Components** (depends on Step 5)

**Suggested Parallel Execution Groups**:
- Group A (Backend): Steps 1 -> 2+3 (parallel)
- Group B (Frontend Setup): Steps 4+5 (parallel) -> 6
- Group C (Hooks): Steps 7+8 (after Step 1)
- Group D (Components): Steps 14 -> 9+10+11 (parallel) -> 12+13 (parallel)
- Group E (Cleanup): Step 15 (after Step 5)

## Notes for Execute Agent

1. **Protobuf First**: Always regenerate protobuf before updating Go or TypeScript files that depend on generated types

2. **Go Build Verification**: After Steps 1-3, run `go build ./api/... ./daemon/...` to verify no compilation errors

3. **TypeScript Type Checking**: After updating hooks and components, run `npm run typecheck` in the web directory

4. **Backend Testing**: After Step 3, manually test the ListSessions endpoint with empty projectId using curl or a gRPC client

5. **Feature Directory Creation**: Create the `session` feature directory before creating its components to avoid path errors

6. **Search Params Validation**: The TanStack Router validateSearch function must handle all edge cases (undefined, wrong types, empty strings)

7. **Sorting Note**: The protobuf `ListProjectsRequest` does not have sort fields. Client-side sorting may be needed, or consider adding sort fields to the proto in a future iteration

8. **BigInt Handling**: Token values are `bigint` from protobuf. Ensure formatTokenCount handles both `bigint` and `number` inputs

9. **Active Session Detection**: A session is "active" when `endedAt` is undefined/null. The protobuf uses `google.protobuf.Timestamp` which will be undefined for unset values
