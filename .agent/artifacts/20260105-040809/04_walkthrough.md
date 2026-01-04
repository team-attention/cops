# Development Walkthrough

## Summary
Fixed 400 "organization_id is required" errors on all dashboard pages by updating frontend hooks to pass the organization ID from user store and updating all calling components to provide this required parameter.

## Problem Overview

### Symptoms
- Dashboard pages (`/dashboard`, `/projects`, `/sessions`) failing to load with 400 errors
- Error message: "organization_id is required"
- API validation rejecting all dashboard requests at the service layer

### Root Cause
Recent RBAC implementation (commit 1a530c5) added organization-level access control requiring all dashboard API endpoints to validate `organization_id`. The backend service layer (`api/internal/service/dashboard/dashboard_service.go:34-36`) uses a `checkRBAC()` function that validates this field on every request.

However, the frontend hooks were not updated to pass this required parameter:
- Protobuf schemas defined `organization_id` as required fields
- Backend validation was correctly implemented
- Frontend hooks were still calling APIs without the parameter
- User store already tracked `selectedOrganizationId`, but hooks weren't using it

## Code Overview

### Modified Hooks (5 files)

All hooks follow the same pattern: accept `organizationId` parameter, pass it to gRPC function, disable query when not available.

#### `useGetOverview`
- **Location**: `web/src/feature/dashboard/hook/use-get-overview.ts`
- **Purpose**: Fetch dashboard overview statistics
- **Changes**:
  - Added `UseGetOverviewOptions` interface with `organizationId: string | null`
  - Updated hook to accept options object instead of no parameters
  - Pass `organizationId` to `getOverview` RPC call
  - Disable query when `organizationId` is falsy: `enabled: !!organizationId`
  - Use fallback empty string for type safety: `organizationId || ''`

#### `useListProjects`
- **Location**: `web/src/feature/project/hook/use-list-projects.ts`
- **Purpose**: Fetch paginated list of projects
- **Changes**:
  - Added `organizationId: string | null` to existing `UseListProjectsOptions` interface
  - Changed default parameters pattern from `= {}` to required options
  - Updated enabled logic to combine with organizationId check: `enabled && !!organizationId`
  - Pass `organizationId` to `listProjects` RPC alongside existing pagination params
- **Key Methods**:
  - Maintains existing pagination: `page`, `pageSize`
  - Preserves existing `enabled` parameter for conditional fetching

#### `useListSessions`
- **Location**: `web/src/feature/project/hook/use-list-sessions.ts`
- **Purpose**: Fetch paginated list of sessions with filtering and sorting
- **Changes**:
  - Added `organizationId: string | null` to existing `UseListSessionsOptions`
  - Changed default parameters pattern from `= {}` to required options
  - Updated enabled logic: `enabled && !!organizationId`
  - Pass `organizationId` to `listSessions` RPC
- **Key Methods**:
  - Maintains existing features: `projectId` filter, `page`, `pageSize`, `sortBy`, `sortDesc`
  - Preserves conditional query execution via `enabled` parameter

#### `useGetProject`
- **Location**: `web/src/feature/project/hook/use-get-project.ts`
- **Purpose**: Fetch single project details
- **Changes**:
  - Completely refactored from simple parameter to options object pattern
  - Before: `useGetProject(projectId: string)`
  - After: `useGetProject({ organizationId, projectId }: UseGetProjectOptions)`
  - Added query disabling: `enabled: !!organizationId`
  - Pass both `organizationId` and `projectId` to `getProject` RPC

#### `useGetSession`
- **Location**: `web/src/feature/session/hook/use-get-session.ts`
- **Purpose**: Fetch single session details
- **Changes**:
  - Completely refactored from simple parameter to options object pattern
  - Before: `useGetSession(sessionId: string)`
  - After: `useGetSession({ organizationId, sessionId }: UseGetSessionOptions)`
  - Added query disabling: `enabled: !!organizationId`
  - Pass both `organizationId` and `sessionId` to `getSession` RPC

### Modified Route Components (5 files)

All route components updated to:
1. Import `useUserStore` from `@/shared/store/user-store`
2. Extract `selectedOrganizationId` via destructuring
3. Pass it to updated hook signatures

#### `/dashboard`
- **Location**: `web/src/route/dashboard.tsx`
- **Changes**:
  - Line 8: Import `useUserStore`
  - Line 39: Destructure `selectedOrganizationId` from store
  - Line 40-42: Updated hook call:
    ```tsx
    const { data, isLoading, isError, refetch, isFetching } = useGetOverview({
      organizationId: selectedOrganizationId,
    })
    ```

#### `/projects/index`
- **Location**: `web/src/route/projects/index.tsx`
- **Changes**:
  - Line 7: Import `useUserStore`
  - Line 31: Destructure `selectedOrganizationId` from store
  - Line 33-37: Updated hook call to pass `organizationId` alongside existing pagination:
    ```tsx
    const { data, isLoading, isError, refetch, isFetching } = useListProjects({
      organizationId: selectedOrganizationId,
      page,
      pageSize,
    })
    ```

#### `/projects/$projectId`
- **Location**: `web/src/route/projects/$projectId.tsx`
- **Changes**:
  - Line 9: Import `useUserStore`
  - Line 37: Destructure `selectedOrganizationId` from store
  - Line 39-45: Updated `useGetProject` to use options object:
    ```tsx
    const { data: projectData, ... } = useGetProject({
      organizationId: selectedOrganizationId,
      projectId
    })
    ```
  - Line 47-52: Updated `useListSessions` to pass `organizationId`:
    ```tsx
    const { data: sessionsData, ... } = useListSessions({
      organizationId: selectedOrganizationId,
      projectId
    })
    ```

#### `/sessions/index`
- **Location**: `web/src/route/sessions/index.tsx`
- **Changes**:
  - Import `useUserStore`
  - Destructure `selectedOrganizationId`
  - Updated both `useListProjects` and `useListSessions` calls to include `organizationId`
  - Both hooks now receive organization context for proper RBAC validation

#### `/sessions/$sessionId`
- **Location**: `web/src/route/sessions/$sessionId.tsx`
- **Changes**:
  - Import `useUserStore`
  - Destructure `selectedOrganizationId`
  - Updated `useGetSession` from simple parameter to options object:
    ```tsx
    const { data } = useGetSession({
      organizationId: selectedOrganizationId,
      sessionId
    })
    ```

## How The Fix Works

### Data Flow
1. **User Store Provides Context**: `useUserStore` maintains `selectedOrganizationId` which is auto-selected when user data loads (see `web/src/shared/store/user-store.ts:62-69`)

2. **Component Accesses Store**: Route components destructure `selectedOrganizationId` from the store

3. **Hook Receives Organization ID**: Updated hooks accept `organizationId` as a required parameter in their options interface

4. **Query Conditional Execution**: Hooks use `enabled: !!organizationId` to prevent API calls when no organization is selected

5. **API Request Includes Organization**: gRPC requests now include `organizationId` field matching protobuf schema requirements

6. **Backend Validation Passes**: Service layer's `checkRBAC()` function receives `organization_id` and validates user access

### Type Safety Pattern

All hooks use a consistent TypeScript pattern:

```typescript
interface UseHookOptions {
  organizationId: string | null  // null allows disabled state
  // ...other params
}

export const useHook = ({ organizationId, ...rest }: UseHookOptions) => {
  return useQuery(
    rpcMethod,
    { organizationId: organizationId || '', ...rest },  // Empty string satisfies protobuf type
    { enabled: !!organizationId }  // Disabled when null
  )
}
```

This pattern ensures:
- TypeScript catches missing parameters at compile time
- Queries don't execute wastefully when organization isn't selected
- Protobuf type requirements (string, not null) are satisfied via fallback

### Edge Case Handling

**No Organization Selected**: When `selectedOrganizationId` is `null`:
- Hooks receive `null` value
- `enabled: !!organizationId` evaluates to `false`
- TanStack Query doesn't execute, preventing invalid API calls
- UI shows loading state (hooks return `isLoading: true` for disabled queries)
- Dashboard page already has route guard redirecting users without organizations to `/organizations/new` (see `dashboard.tsx:11-18`)

## Testing

### Manual Verification Commands

```bash
# Start development server
cd web && npm run dev

# Navigate to affected pages and verify:
# 1. /dashboard - Overview stats load without errors
# 2. /projects - Projects list displays
# 3. /projects/{id} - Project details and sessions load
# 4. /sessions - Sessions list displays
# 5. /sessions/{id} - Session details load
```

### Verification Checklist

- [ ] Dashboard overview loads successfully (no 400 errors in browser console)
- [ ] Projects list page displays projects with pagination
- [ ] Project detail page shows project stats and session list
- [ ] Sessions list page displays all sessions across projects
- [ ] Session detail page shows individual session data
- [ ] Network tab shows `organization_id` field in all dashboard API requests
- [ ] No errors when switching between organizations (if user has multiple)
- [ ] Queries don't execute when no organization is selected

### Browser DevTools Inspection

1. Open browser DevTools → Network tab
2. Filter by "dashboard" to see gRPC requests
3. Inspect request payload - should include:
   ```json
   {
     "organizationId": "org-xxxxx",
     // ...other params
   }
   ```
4. Response should be 200 OK with data, not 400 Bad Request

### TypeScript Compilation

```bash
cd web && npm run build  # Should complete without type errors
```

## Related Files

### Backend (Not Modified - Already Correct)
- `api/internal/service/dashboard/dashboard_service.go` - RBAC validation logic (lines 34-36)
- `idl/protobuf/dashboard/v1/dashboard.proto` - Protobuf schema with `organization_id` fields

### Frontend State Management (Not Modified - Already Correct)
- `web/src/shared/store/user-store.ts` - Provides `selectedOrganizationId` (lines 62-69 handle auto-selection)

### Modified Frontend Files
**Hooks** (5 files):
- `web/src/feature/dashboard/hook/use-get-overview.ts`
- `web/src/feature/project/hook/use-list-projects.ts`
- `web/src/feature/project/hook/use-list-sessions.ts`
- `web/src/feature/project/hook/use-get-project.ts`
- `web/src/feature/session/hook/use-get-session.ts`

**Route Components** (5 files):
- `web/src/route/dashboard.tsx`
- `web/src/route/projects/index.tsx`
- `web/src/route/projects/$projectId.tsx`
- `web/src/route/sessions/index.tsx`
- `web/src/route/sessions/$sessionId.tsx`

## Implementation Notes

### Why Not Automatically Inject Organization ID?

We considered auto-injecting `organizationId` inside hooks by accessing `useUserStore` directly, but chose explicit passing because:
1. **Testability**: Easier to test hooks with explicit parameters
2. **Flexibility**: Components can override if needed (e.g., viewing different org's data)
3. **Clarity**: Call sites explicitly show what data is being used
4. **React Rules**: Hooks shouldn't call other hooks conditionally or in non-standard ways

### Consistency with Existing Patterns

The implementation follows established project patterns from `web/src/feature/organization/hook/use-get-organization-members.ts`:
- Uses named interface for hook options
- Exports named const with typed parameters
- Passes transport explicitly to Connect Query
- Includes JSDoc comments describing purpose and behavior

### Generated Code Formatting

The `git diff` shows formatting changes in `web/src/gen/grpcstub/` files. These are auto-generated files that were reformatted by Prettier during development. While typically we don't edit generated code, these are cosmetic changes only (import ordering, line breaks) and don't affect functionality. The actual generated API signatures remain unchanged.
