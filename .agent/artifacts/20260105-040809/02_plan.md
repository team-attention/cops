# Implementation Plan: Pass Organization ID to Dashboard API Hooks

## Overview

The web application's dashboard pages are failing with 400 errors because recent RBAC changes require all dashboard API endpoints to include an `organizationId` parameter. The frontend hooks (`useGetOverview`, `useListProjects`, `useListSessions`, `useGetProject`, `useGetSession`) currently do not pass this required parameter.

This plan updates all five dashboard hooks to:
1. Accept `organizationId` as a required parameter
2. Pass it to the generated gRPC service functions
3. Disable queries when `organizationId` is not available

## Package Changes

None required. All necessary packages are already installed.

## Implementation Steps

### Step 1: Update `useGetOverview` Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook implementation patterns
- `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-get-organization-members.ts`: Reference for hook pattern with organizationId
- `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`: To understand how to access selectedOrganizationId

#### `/Users/jayce/team-attention/cops/web/src/feature/dashboard/hook/use-get-overview.ts`

**Description**:
Update the hook to accept organizationId and pass it to the getOverview RPC. Disable the query when organizationId is not provided.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { getOverview } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetOverviewOptions defines the input parameters for the hook.
interface UseGetOverviewOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
}

// useGetOverview provides a query hook for fetching dashboard overview data.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null or empty.
export const useGetOverview = ({ organizationId }: UseGetOverviewOptions) => {
  // Implementation outline:
  // 1. Call useQuery with getOverview RPC function
  // 2. Pass organizationId in the request object (use empty string if null to satisfy type)
  // 3. Set enabled option to true only when organizationId is truthy
  // 4. Return the query result
  return useQuery(
    getOverview,
    { organizationId: organizationId || '' },
    { enabled: !!organizationId },
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid organizationId | `{ organizationId: "org-123" }` | Query enabled, API called with organizationId | Happy path |
| Null organizationId | `{ organizationId: null }` | Query disabled, no API call made | Disabled path |
| Empty string organizationId | `{ organizationId: "" }` | Query disabled, no API call made | Disabled path (empty string is falsy) |

---

### Step 2: Update `useListProjects` Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook implementation patterns
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/hook/use-get-overview.ts`: After Step 1, for consistency

#### `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-projects.ts`

**Description**:
Update the hook to require organizationId. Combine the existing enabled logic with organizationId validation.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { listProjects } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseListProjectsOptions defines the input parameters for the hook.
interface UseListProjectsOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // page is the pagination page number (defaults to 1)
  page?: number
  // pageSize is the number of items per page (defaults to 20)
  pageSize?: number
  // enabled controls whether the query should run (in addition to organizationId check)
  enabled?: boolean
}

// useListProjects provides a query hook for fetching projects list.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty or enabled is false.
export const useListProjects = ({
  organizationId,
  page = 1,
  pageSize = 20,
  enabled = true,
}: UseListProjectsOptions) => {
  // Implementation outline:
  // 1. Call useQuery with listProjects RPC function
  // 2. Pass request object with:
  //    - organizationId (use empty string if null to satisfy type)
  //    - pagination object with page and pageSize
  // 3. Set enabled option to: enabled && !!organizationId
  // 4. Return the query result
  return useQuery(
    listProjects,
    {
      organizationId: organizationId || '',
      pagination: { page, pageSize },
    },
    { enabled: enabled && !!organizationId },
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid organizationId, enabled | `{ organizationId: "org-123" }` | Query enabled, API called | Happy path |
| Valid organizationId, explicitly disabled | `{ organizationId: "org-123", enabled: false }` | Query disabled | Explicit disable |
| Null organizationId | `{ organizationId: null }` | Query disabled | Missing org |
| Custom pagination | `{ organizationId: "org-123", page: 2, pageSize: 10 }` | Query with custom pagination | Pagination params |

---

### Step 3: Update `useListSessions` Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook implementation patterns

#### `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-sessions.ts`

**Description**:
Update the hook to require organizationId. Preserve existing pagination and sorting options.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { listSessions } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseListSessionsOptions defines the input parameters for the hook.
interface UseListSessionsOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // projectId filters sessions by project (optional)
  projectId?: string
  // page is the pagination page number (defaults to 1)
  page?: number
  // pageSize is the number of items per page (defaults to 20)
  pageSize?: number
  // sortBy is the field to sort by (defaults to "started_at")
  sortBy?: string
  // sortDesc indicates descending sort order (defaults to true)
  sortDesc?: boolean
  // enabled controls whether the query should run (in addition to organizationId check)
  enabled?: boolean
}

// useListSessions provides a query hook for fetching sessions list.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty or enabled is false.
export const useListSessions = ({
  organizationId,
  projectId,
  page = 1,
  pageSize = 20,
  sortBy = 'started_at',
  sortDesc = true,
  enabled = true,
}: UseListSessionsOptions) => {
  // Implementation outline:
  // 1. Call useQuery with listSessions RPC function
  // 2. Pass request object with:
  //    - organizationId (use empty string if null to satisfy type)
  //    - projectId (use empty string if undefined)
  //    - pagination object with page and pageSize
  //    - sortBy field
  //    - sortDesc boolean
  // 3. Set enabled option to: enabled && !!organizationId
  // 4. Return the query result
  return useQuery(
    listSessions,
    {
      organizationId: organizationId || '',
      projectId: projectId || '',
      pagination: { page, pageSize },
      sortBy,
      sortDesc,
    },
    { enabled: enabled && !!organizationId },
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid organizationId | `{ organizationId: "org-123" }` | Query enabled with defaults | Happy path |
| With projectId filter | `{ organizationId: "org-123", projectId: "proj-456" }` | Query with projectId filter | Project filter |
| Custom sort | `{ organizationId: "org-123", sortBy: "message_count", sortDesc: false }` | Query with custom sort | Sort options |
| Null organizationId | `{ organizationId: null }` | Query disabled | Missing org |
| Explicitly disabled | `{ organizationId: "org-123", enabled: false }` | Query disabled | Explicit disable |

---

### Step 4: Update `useGetProject` Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook implementation patterns

#### `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-get-project.ts`

**Description**:
Update the hook to require organizationId alongside projectId. Disable query when either is missing.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { getProject } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetProjectOptions defines the input parameters for the hook.
interface UseGetProjectOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // projectId is the project's unique identifier (required for API call)
  projectId: string
}

// useGetProject provides a query hook for fetching a single project's details.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetProject = ({ organizationId, projectId }: UseGetProjectOptions) => {
  // Implementation outline:
  // 1. Call useQuery with getProject RPC function
  // 2. Pass request object with:
  //    - organizationId (use empty string if null to satisfy type)
  //    - projectId
  // 3. Set enabled option to: !!organizationId
  // 4. Return the query result
  return useQuery(
    getProject,
    {
      organizationId: organizationId || '',
      projectId,
    },
    { enabled: !!organizationId },
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid inputs | `{ organizationId: "org-123", projectId: "proj-456" }` | Query enabled, API called | Happy path |
| Null organizationId | `{ organizationId: null, projectId: "proj-456" }` | Query disabled | Missing org |
| Empty organizationId | `{ organizationId: "", projectId: "proj-456" }` | Query disabled | Empty org string |

---

### Step 5: Update `useGetSession` Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook implementation patterns

#### `/Users/jayce/team-attention/cops/web/src/feature/session/hook/use-get-session.ts`

**Description**:
Update the hook to require organizationId alongside sessionId. Disable query when organizationId is missing.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { getSession } from '@/gen/grpcstub/dashboard/v1/dashboard-DashboardService_connectquery'

// UseGetSessionOptions defines the input parameters for the hook.
interface UseGetSessionOptions {
  // organizationId is the selected organization's ID (required for API call)
  organizationId: string | null
  // sessionId is the session's unique identifier (required for API call)
  sessionId: string
}

// useGetSession provides a query hook for fetching a single session's details.
// Returns a TanStack Query object with data, isLoading, error states.
// Query is disabled when organizationId is null/empty.
export const useGetSession = ({ organizationId, sessionId }: UseGetSessionOptions) => {
  // Implementation outline:
  // 1. Call useQuery with getSession RPC function
  // 2. Pass request object with:
  //    - organizationId (use empty string if null to satisfy type)
  //    - sessionId
  // 3. Set enabled option to: !!organizationId
  // 4. Return the query result
  return useQuery(
    getSession,
    {
      organizationId: organizationId || '',
      sessionId,
    },
    { enabled: !!organizationId },
  )
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid inputs | `{ organizationId: "org-123", sessionId: "sess-789" }` | Query enabled, API called | Happy path |
| Null organizationId | `{ organizationId: null, sessionId: "sess-789" }` | Query disabled | Missing org |
| Empty organizationId | `{ organizationId: "", sessionId: "sess-789" }` | Query disabled | Empty org string |

---

### Step 6: Update Hook Call Sites

After updating the hooks, all call sites must be updated to pass `organizationId`. The implementation agent must search for usages of each hook and update them.

**Search Commands**:
Use Grep to find all usages of each hook in `/Users/jayce/team-attention/cops/web/src`:
- `useGetOverview`
- `useListProjects`
- `useListSessions`
- `useGetProject`
- `useGetSession`

**Pattern for Call Site Updates**:

Each component using these hooks must:
1. Import `useUserStore` from `@/shared/store/user-store`
2. Get `selectedOrganizationId` from the store using destructuring
3. Pass it to the hook as part of the options object

**Call Site Update Pattern**:

```typescript
// Add import at top of file (if not already present):
import { useUserStore } from '@/shared/store/user-store'

// Inside component function, add store access:
const { selectedOrganizationId } = useUserStore()

// Update hook calls:

// For useGetOverview:
// Before: useGetOverview()
// After:
const overviewQuery = useGetOverview({ organizationId: selectedOrganizationId })

// For useListProjects:
// Before: useListProjects({ page, pageSize, enabled })
// After:
const projectsQuery = useListProjects({ organizationId: selectedOrganizationId, page, pageSize, enabled })

// For useListSessions:
// Before: useListSessions({ projectId, page, pageSize, sortBy, sortDesc, enabled })
// After:
const sessionsQuery = useListSessions({ organizationId: selectedOrganizationId, projectId, page, pageSize, sortBy, sortDesc, enabled })

// For useGetProject:
// Before: useGetProject(projectId)
// After:
const projectQuery = useGetProject({ organizationId: selectedOrganizationId, projectId })

// For useGetSession:
// Before: useGetSession(sessionId)
// After:
const sessionQuery = useGetSession({ organizationId: selectedOrganizationId, sessionId })
```

---

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation included as comments in every function body
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Execute Agent
- [x] All packages are selected (no candidates)
- [x] Execution order is clear and dependencies are explicit
