# Requirements

## Request Summary
Implement `/projects` and `/sessions` list pages in the web service. Both pages should follow the existing dashboard design patterns and display comprehensive data from the API. The data sources (`ListProjects` and `ListSessions` RPCs) are already available through the gRPC Dashboard service.

## Current State Analysis

### Available API Endpoints
Based on the protobuf definitions, we have:

**For `/projects` page:**
- `ListProjects(ListProjectsRequest)` - Returns paginated list of projects
- Available data per project:
  - Project ID, name, path
  - Session count
  - Token usage summary (input, output, cache creation, cache read)
  - Last activity timestamp

**For `/sessions` page:**
- The current protobuf only has `ListSessions` which requires a `project_id`
- This means we need to either:
  1. Add a new RPC `ListAllSessions` that returns sessions across all projects
  2. Or use a special sentinel value (e.g., empty string or "*") to mean "all projects"

### Existing Patterns
- Dashboard shows recent projects (limited to 5) and recent sessions (limited to 5)
- Both use table layouts with columns for key information
- Design uses dark theme with cyan accent for projects, violet accent for sessions
- Pagination is supported in the API but not yet implemented in the UI

## Acceptance Criteria

- [ ] **Projects List Page (`/projects/index.tsx`)**
  - [ ] Displays paginated list of all projects using `ListProjects` RPC
  - [ ] Shows project name, path, session count, token usage, and last activity
  - [ ] Implements pagination controls (page navigation, items per page)
  - [ ] Supports sorting by different columns (name, session count, last activity, token usage)
  - [ ] Each project row is clickable and navigates to `/projects/$projectId`
  - [ ] Shows empty state when no projects exist
  - [ ] Includes loading skeleton while fetching data
  - [ ] Includes refresh button to reload data

- [ ] **Sessions List Page (`/sessions/index.tsx`)**
  - [ ] Displays paginated list of all sessions across projects
  - [ ] Shows session ID, project name, git branch, message count, token usage, and timestamps
  - [ ] Implements pagination controls (page navigation, items per page)
  - [ ] Supports filtering by project (optional dropdown)
  - [ ] Supports sorting by different columns (started_at, message_count, token usage)
  - [ ] Each session row is clickable and navigates to `/sessions/$sessionId`
  - [ ] Shows empty state when no sessions exist
  - [ ] Includes loading skeleton while fetching data
  - [ ] Includes refresh button to reload data

- [ ] **API Hooks**
  - [ ] Create `use-list-projects.ts` hook wrapping `ListProjects` RPC
  - [ ] Create or modify hook for listing all sessions (may require backend changes)
  - [ ] Hooks support pagination parameters
  - [ ] Hooks support sorting parameters

- [ ] **Shared Components** (if needed)
  - [ ] Pagination component for navigating pages
  - [ ] Table sorting controls (reusable across both pages)

## Scope

### In Scope
- **Protobuf Refactoring**: Unify naming convention to Req/Res across dashboard.proto
  - Rename all `*Request` → `*Req` and `*Response` → `*Res` in dashboard.proto
  - Regenerate protobuf code with `buf generate`
  - Update all usage in API, daemon, and web
- **Backend API Modifications**:
  - Modify `ListSessions` to support empty `project_id` for listing all sessions
  - Update all dashboard service handlers for new Req/Res types
- **Frontend Implementation**:
  - Implementing `/projects` list page with full table view
  - Implementing `/sessions` list page with full table view and project filter
  - Creating necessary API hooks for pagination
  - Adding pagination UI controls
  - Adding sorting capabilities
  - Maintaining consistent design with existing dashboard
  - Error handling and loading states
  - Empty states for both pages

### Out of Scope
- Filtering capabilities beyond basic project filter for sessions
- Search functionality
- Bulk operations (delete, export, etc.)
- Column customization (show/hide columns)
- Export to CSV or other formats
- Advanced analytics or charts
- Modifying existing project/session detail pages

## Constraints
- Must use ConnectRPC with TanStack Query pattern (established in codebase)
- Must follow Feature Driven Development structure (`feature/` directory)
- Must use shadcn/ui components for UI elements
- Must maintain dark theme with consistent color scheme (cyan for projects, violet for sessions)
- Backend API changes may be required for listing all sessions

## Questions for Clarification

### 1. Sessions List Page - API Endpoint
The current `ListSessions` RPC requires a `project_id` parameter, but we want to show ALL sessions across all projects. What approach should we take?

**Option A:** Add a new RPC method `ListAllSessions` to the backend
**Option B:** Modify `ListSessions` to accept an optional/empty `project_id` to mean "all projects"
**Option C:** Fetch all projects first, then aggregate sessions from each project (not recommended - performance issue)

**Recommended:** Option A - Add new `ListAllSessions` RPC with pagination and sorting

### 2. Pagination Configuration
What should be the default page size for these list pages?
- Dashboard shows 5 items (recent items)
- Suggested options: 10, 20, 25, 50 items per page
- Should users be able to change page size?

### 3. Default Sorting
What should be the default sort order for each page?

**Projects page:**
- Last activity (most recent first) - **Recommended**
- Name (alphabetical)
- Session count (most active first)

**Sessions page:**
- Started at (most recent first) - **Recommended**
- Message count (most active first)
- Token usage (highest first)

### 4. Column Display Priority
For responsive design, which columns are most important and should remain visible on smaller screens?

**Projects page:**
- Essential: Project name, Session count, Last activity
- Optional: Path, Token usage details

**Sessions page:**
- Essential: Session ID, Project name, Started at
- Optional: Git branch, Message count, Token usage

### 5. Token Usage Display
Token usage has 4 metrics (input, output, cache creation, cache read). How should we display this in the table?

**Option A:** Show total tokens only (sum of all 4)
**Option B:** Show input + output prominently, cache in tooltip
**Option C:** Show all 4 metrics in compact format
**Option D:** Add expandable row for detailed breakdown

**Recommended:** Option B - Show main metrics, details on hover/tooltip

### 6. Project Filter on Sessions Page
Should the sessions list page include a filter dropdown to show sessions from a specific project?
- This would allow narrowing down the "all sessions" view
- Would use the existing `ListSessions` RPC with selected project ID

### 7. Direct Navigation
Currently there are "View all" links from dashboard to `/projects` and `/sessions`. Should we also add:
- Link from sessions page to filter by specific project?
- Breadcrumbs for navigation context?

## Additional Context

### Existing Components to Reference
- `/web/src/feature/dashboard/component/project-list.tsx` - Table layout for projects
- `/web/src/feature/dashboard/component/recent-sessions.tsx` - Table layout for sessions
- `/web/src/route/projects/$projectId.tsx` - Pattern for loading/error states and pagination

### Design System
- Dark theme: `bg-zinc-950` base, `bg-zinc-900/80` cards
- Accent colors: Cyan (`cyan-400/500`) for projects, Violet (`violet-400/500`) for sessions
- Typography: Sans-serif for UI, Mono for IDs/timestamps
- Border colors: `zinc-800/50` for subtle borders

### API Transport
- All RPC calls use shared transport from `@/shared/service/connect-transport`
- Generated stubs are in `@/gen/grpcstub/dashboard/v1/`
- Follow the hook pattern established in existing feature hooks

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| 1. Sessions API Endpoint | **Option B**: Modify `ListSessions` to accept optional/empty `project_id` to mean "all projects" |
| 2. Pagination Configuration | **20 items per page** (default), users can change page size |
| 3. Default Sorting | **Projects**: Last activity (most recent first)<br>**Sessions**: Started at (most recent first) |
| 4. Column Display Priority | Will follow responsive design best practices, hiding less critical columns on mobile |
| 5. Token Usage Display | **Input/Output prominently** with cache details in tooltip on hover |
| 6. Project Filter on Sessions | **Yes**, include dropdown filter to show sessions from specific project |
| 7. Direct Navigation | Will add breadcrumbs for navigation context |
