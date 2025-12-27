# Research Report

## Mode
General Research

## Request Summary
Implement `/projects` and `/sessions` list pages in the web service. Both pages should display paginated, sortable lists using existing gRPC APIs with ConnectRPC. The sessions page requires a backend modification to support listing all sessions across projects (empty `project_id` parameter).

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File                                                                                   | Reason                                                          |
| -------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx` | Table layout pattern for projects with Link, badges, formatters |
| `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/recent-sessions.tsx` | Table layout pattern for sessions with Link, badges, formatters |
| `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-sessions.ts`   | Hook pattern for paginated API calls                            |
| `/Users/jayce/team-attention/cops/web/src/route/projects/$projectId.tsx`               | Page structure, loading/error states, refresh pattern           |
| `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`                         | Page layout, header design, skeleton pattern                    |
| `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto`           | API definitions for ListProjects, ListSessions                  |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:298-390` | Backend ListSessions implementation to modify |
| `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`                 | Feature structure rules (hook patterns, component organization) |

## Package Candidates

### Problem 1: Pagination UI Component
No existing pagination component in the codebase. Need to add shadcn/ui pagination.

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| shadcn/ui Pagination | (shadcn CLI) | Consistent with existing UI system, already uses Radix primitives, matches project design system |

**Installation Command**: `npx shadcn@latest add pagination`

### Problem 2: Table Sorting
The existing Table component from shadcn/ui is already installed but lacks built-in sorting.

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| Native Implementation | (custom) | Simple sorting state with URL params, minimal dependencies, follows existing patterns |

**Recommendation**: Use native React state with URL search params for sort state persistence. No additional package needed.

### Problem 3: Date/Time Formatting
Already have inline formatters in multiple components.

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| Existing inline formatters | (shared util) | Extract `formatTimestamp` and `formatRelativeTime` to shared utility |

**Recommendation**: Extract duplicated formatters from `project-list.tsx` and `recent-sessions.tsx` to `shared/util/format.ts`.

## Technical Constraints

1. **Backend Modification Required**: The `ListSessions` RPC currently requires a `project_id` parameter. According to requirements, we need to modify it to accept an empty `project_id` to return all sessions across all projects.

2. **Protobuf Already Has Sorting Support**: `ListSessionsRequest` already includes `sort_by` and `sort_desc` fields - these need to be exposed in the UI.

3. **No Sorting for ListProjects**: The `ListProjectsRequest` does not have sorting fields in the protobuf. Either add them or implement client-side sorting.

4. **Pagination Exists in Proto**: Both `ListProjectsRequest` and `ListSessionsRequest` support `PaginationRequest` with `page` and `page_size`.

5. **Icon Library**: Project uses `lucide-react` (v0.562.0) for all icons.

6. **No Select Component Installed**: For project filter dropdown on sessions page, need to install shadcn/ui Select.

## Similar Implementations Found

### Example 1: Dashboard Project Table
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx:43-142`
- **Relevance**: Shows complete table implementation with:
  - Table header with uppercase tracking-widest labels
  - Link wrapping for row navigation
  - Badge component usage for branches
  - `formatTimestamp` utility function
  - Empty state handling
  - Card wrapper with consistent styling

### Example 2: Dashboard Session Table
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/recent-sessions.tsx:41-144`
- **Relevance**: Shows session-specific table with:
  - Session ID truncation pattern (`truncateId`)
  - Violet accent color for sessions (vs cyan for projects)
  - Link to `/sessions/$sessionId`

### Example 3: Project Detail Page with Sessions List
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/project/component/session-list.tsx:57-197`
- **Relevance**: More detailed session table with:
  - Token usage display (`formatTokenCount`)
  - Active session indicator
  - Expandable row pattern potential
  - Tooltip usage for IDs

### Example 4: Hook Pattern for Paginated API
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-sessions.ts:1-15`
- **Relevance**: Shows how to wrap ConnectRPC method with TanStack Query:
  - Interface for options with page/pageSize
  - Default values for pagination
  - Direct use of generated `listSessions` method

### Example 5: Loading States and Page Structure
- **File**: `/Users/jayce/team-attention/cops/web/src/route/projects/$projectId.tsx:14-116`
- **Relevance**: Shows complete page pattern with:
  - `LoadingSkeleton` component pattern
  - Loading/error/data conditional rendering
  - Refresh button with `isFetching` state
  - Footer pattern
  - Breadcrumb navigation

### Example 6: Backend ListSessions with Sorting
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:298-390`
- **Relevance**: Shows current implementation that:
  - Requires `project_id` (line 299-302 validates ObjectID)
  - Supports `sort_by` and `sort_desc` params (lines 321-328)
  - Uses MongoDB aggregation pipeline

## Existing shadcn/ui Components

Already installed (from `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/`):
- `avatar.tsx`
- `badge.tsx`
- `button.tsx`
- `card.tsx`
- `collapsible.tsx`
- `input.tsx`
- `scroll-area.tsx`
- `separator.tsx`
- `sheet.tsx`
- `sidebar.tsx`
- `skeleton.tsx`
- `table.tsx`
- `tabs.tsx`
- `tooltip.tsx`

**Need to install:**
- `pagination` - For page navigation controls
- `select` - For project filter dropdown on sessions page
- `dropdown-menu` - For items-per-page selector (optional, could use select)

## Design System Reference

From existing components:

### Colors
- **Background**: `bg-zinc-950` (page), `bg-zinc-900/80` (cards)
- **Borders**: `border-zinc-800/50` (subtle), `border-zinc-700/50` (hover)
- **Projects accent**: `cyan-400`, `cyan-500/10` (bg), `cyan-500/20` (border)
- **Sessions accent**: `violet-400`, `violet-500/10` (bg), `violet-500/30` (border)
- **Active indicators**: `emerald-400` with pulse animation

### Typography
- **Headers**: `font-mono text-[10px] uppercase tracking-widest text-zinc-600`
- **Page titles**: `text-2xl font-bold tracking-tight text-zinc-100`
- **Mono text**: `font-mono text-sm` for IDs, counts, timestamps

### Spacing
- **Page container**: `mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8`
- **Card content**: `p-0` for tables, `pt-4` for other content
- **Grid gaps**: `gap-4` (stats), `gap-6` (main sections)

## Additional Information for Planning

### Backend Changes Required

The `ListSessions` method in `dashboard_repo.go` needs modification:

**Current behavior** (lines 298-302):
```go
func (r *MongoDashboardRepository) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
    projectOID, err := bson.ObjectIDFromHex(params.Query.ProjectID)
    if err != nil {
        return nil, fmt.Errorf("invalid project ID: %w", err)
    }
```

**Required change**: When `params.Query.ProjectID` is empty, skip the project filter in the aggregation pipeline to return all sessions.

### Frontend Hook Changes

Create new hook `use-list-projects.ts` following the pattern in `use-list-sessions.ts`:
```typescript
// web/src/feature/project/hook/use-list-projects.ts
import { useQuery } from '@connectrpc/connect-query'
import { listProjects } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

interface UseListProjectsOptions {
  page?: number
  pageSize?: number
}

export const useListProjects = ({ page = 1, pageSize = 20 }: UseListProjectsOptions = {}) => {
  return useQuery(listProjects, {
    pagination: { page, pageSize },
  })
}
```

### URL State for Pagination/Sorting

Use TanStack Router search params for state persistence:
- `?page=1` - Current page
- `?pageSize=20` - Items per page
- `?sortBy=started_at` - Sort field
- `?sortDesc=true` - Sort direction
- `?projectId=xxx` - Filter by project (sessions page only)

### Feature Directory Structure

```
web/src/feature/
├── project/
│   ├── component/
│   │   ├── project-header.tsx      (exists)
│   │   ├── session-list.tsx        (exists)
│   │   ├── token-breakdown.tsx     (exists)
│   │   └── projects-table.tsx      (NEW - full table with sorting)
│   └── hook/
│       ├── use-get-project.ts      (exists)
│       ├── use-list-sessions.ts    (exists)
│       └── use-list-projects.ts    (NEW)
├── session/
│   └── component/
│       └── sessions-table.tsx      (NEW - full table with sorting/filtering)
└── dashboard/
    └── ... (existing components)

web/src/shared/
├── component/
│   └── pagination-controls.tsx     (NEW - reusable pagination UI)
└── util/
    └── format.ts                   (NEW - extracted formatters)
```

### Route File Updates

Both route files exist but are placeholder implementations:
- `/Users/jayce/team-attention/cops/web/src/route/projects/index.tsx` - 15 lines placeholder
- `/Users/jayce/team-attention/cops/web/src/route/sessions/index.tsx` - 15 lines placeholder

### Performance Considerations

1. **Default page size**: 20 items (matching backend default in `pagination.go`)
2. **Server-side pagination**: All pagination is handled by the backend
3. **Avoid client-side sorting**: Use `sort_by` and `sort_desc` params in API requests
4. **Debounce filter changes**: If implementing search, debounce input

### API Response Structure

From protobuf:
```protobuf
message ListProjectsResponse {
  repeated ProjectSummary projects = 1;
  PaginationResponse pagination = 2;
}

message ListSessionsResponse {
  repeated SessionSummary sessions = 1;
  PaginationResponse pagination = 2;
}

message PaginationResponse {
  int32 current_page = 1;
  int32 page_size = 2;
  int32 total_pages = 3;
  int64 total_count = 4;
}
```

Both responses include complete pagination metadata for building pagination controls.
