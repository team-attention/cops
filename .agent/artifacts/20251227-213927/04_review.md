# Pre-PR Code Review

## Review Summary
- **Status**: FAIL
- **Files Reviewed**: 15
- **Issues Found**: 8 (Critical: 4, Major: 1, Minor: 3)

## Implementation Verification

### Plan Completion Checklist

| Step | Description | Status | Notes |
|------|-------------|--------|-------|
| 1 | Protobuf Refactoring (Req/Res naming) | DONE | All `*Request`/`*Response` renamed to `*Req`/`*Res` |
| 2 | Update API Handler for New Types | DONE | All 5 RPC methods updated |
| 3 | Modify Backend ListSessions for Empty ProjectID | DONE | Conditional project filter implemented |
| 4 | Install shadcn/ui Components | DONE | pagination.tsx and select.tsx installed |
| 5 | Create Shared Formatting Utilities | DONE | format.ts created with 4 utility functions |
| 6 | Create Pagination Controls Component | DONE | pagination-controls.tsx created |
| 7 | Create useListProjects Hook | DONE | use-list-projects.ts created |
| 8 | Update useListSessions Hook | DONE | projectId now optional, sortBy/sortDesc added |
| 9 | Create Projects Table Component | DONE | projects-table.tsx created |
| 10 | Create Sessions Table Component | DONE | sessions-table.tsx created |
| 11 | Create Project Filter Component | DONE | project-filter.tsx created |
| 12 | Implement /projects Route Page | DONE | Route with pagination, sorting, validation |
| 13 | Implement /sessions Route Page | DONE | Route with filter, pagination, sorting |
| 14 | Create Session Feature Directory | DONE | Directory already existed, components added |
| 15 | Update Existing Components to Use Shared Utilities | DONE | 3 files updated to use shared utils |

### Files Changed Summary

**Backend (Go)**:
- `idl/protobuf/dashboard/v1/dashboard.proto` - Req/Res naming convention
- `shared/gen/grpcstub/dashboard/v1/dashboard.pb.go` - Regenerated
- `shared/gen/grpcstub/dashboard/v1/dashboardv1connect/dashboard.connect.go` - Regenerated
- `api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go` - Updated types
- `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` - Updated types
- `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` - Optional projectID filter

**Frontend (TypeScript/React)**:
- `web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts` - Regenerated
- `web/src/gen/shadcn/ui/pagination.tsx` - New
- `web/src/gen/shadcn/ui/select.tsx` - New
- `web/src/shared/util/format.ts` - New
- `web/src/shared/component/pagination-controls.tsx` - New
- `web/src/feature/project/hook/use-list-projects.ts` - New
- `web/src/feature/project/hook/use-list-sessions.ts` - Modified
- `web/src/feature/project/component/projects-table.tsx` - New
- `web/src/feature/session/component/sessions-table.tsx` - New
- `web/src/feature/session/component/project-filter.tsx` - New
- `web/src/route/projects/index.tsx` - Replaced
- `web/src/route/sessions/index.tsx` - Replaced
- `web/src/feature/dashboard/component/project-list.tsx` - Updated to use shared utils
- `web/src/feature/dashboard/component/recent-sessions.tsx` - Updated to use shared utils
- `web/src/feature/project/component/session-list.tsx` - Updated to use shared utils

---

## Issues Found

### Critical Issues

#### Issue 1: Link components missing required `search` prop
- **Files**:
  - `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx:33`
  - `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/recent-sessions.tsx:33`
  - `/Users/jayce/team-attention/cops/web/src/feature/project/component/project-header.tsx:59`
  - `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx:68`
- **Severity**: Critical
- **Description**: After adding `validateSearch` to `/projects` and `/sessions` routes, TanStack Router now requires the `search` prop to be provided for Links to these routes. The Links in existing components are now failing TypeScript validation.
- **Rule Violated**: TypeScript type safety
- **Required Fix**:
  ```tsx
  // Before
  <Link to="/projects" className="...">View all</Link>

  // After
  <Link to="/projects" search={{}} className="...">View all</Link>
  ```

**For project-list.tsx line 33:**
```tsx
<Link to="/projects" search={{}} className="group flex items-center gap-1 font-mono text-xs text-cyan-500/70 transition-colors hover:text-cyan-400">
```

**For recent-sessions.tsx line 33:**
```tsx
<Link to="/sessions" search={{}} className="group flex items-center gap-1 font-mono text-xs text-violet-500/70 transition-colors hover:text-violet-400">
```

**For project-header.tsx line 59:**
```tsx
<Link to="/projects" search={{}} className="text-zinc-400 transition-colors hover:text-zinc-100">
```

**For session-header.tsx line 68:**
```tsx
<Link to="/projects" search={{}} className="text-zinc-400 transition-colors hover:text-zinc-100">
```

#### Issue 2: Invalid property access `gitBranch` on ProjectSummary
- **Files**:
  - `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx:95`
  - `/Users/jayce/team-attention/cops/web/src/feature/project/component/project-header.tsx:107`
- **Severity**: Critical
- **Description**: These files are accessing a `gitBranch` property that does not exist on `ProjectSummary` or `ProjectDetail` types. This appears to be pre-existing code that was not modified by this implementation, but is now failing TypeScript checks.
- **Note**: This is a pre-existing issue, not caused by the current changes, but must be fixed before commit.

#### Issue 3: SessionRecord type mismatch
- **File**: `/Users/jayce/team-attention/cops/web/src/route/sessions/$sessionId.tsx:52,100`
- **Severity**: Critical
- **Description**: Type mismatch between `aggregation.v1.SessionRecord` and `collector.v1.SessionRecord`. The session detail page is using the wrong SessionRecord type.
- **Note**: This is a pre-existing issue, not caused by the current changes, but must be fixed before commit.

#### Issue 4: TypeScript build fails
- **Severity**: Critical
- **Description**: The TypeScript compilation fails due to the above type errors. All 8 TypeScript errors must be resolved before the code can be committed.

### Major Issues

#### Issue 5: Sorting not implemented on backend for ListProjects
- **Location**: Backend API and Projects page
- **Severity**: Major
- **Description**: The plan notes that `ListProjectsRequest` does not have sort fields in the protobuf. The frontend UI has sorting controls but the sorting is purely UI state - it does not actually re-sort data from the backend. The frontend should implement client-side sorting or the backend should be updated.
- **Recommendation**: Since `ListProjects` returns paginated data, client-side sorting would only work within a single page. Consider:
  1. Adding sort fields to `ListProjectsReq` in a follow-up PR
  2. Or removing the sort UI from the projects table (since it doesn't actually work)
  3. Or implementing client-side sorting with a note that it only sorts the current page

### Minor Issues

#### Issue 6: Empty search object pattern could be improved
- **Files**: Links to `/projects` and `/sessions`
- **Severity**: Minor
- **Description**: Passing `search={{}}` means default values from `validateSearch` will be used. This is correct but could be more explicit with default values.

#### Issue 7: Project column shows project ID instead of name
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/session/component/sessions-table.tsx:140`
- **Severity**: Minor
- **Description**: The sessions table shows a truncated project ID in the Project column. It would be more user-friendly to show the project name. This would require either:
  - Fetching project details for each session (expensive)
  - Including project name in SessionSummary (protobuf change)
  - Creating a lookup map from the projects list

#### Issue 8: TypeScript loose type assertion for sortBy
- **Files**:
  - `/Users/jayce/team-attention/cops/web/src/route/projects/index.tsx:112`
  - `/Users/jayce/team-attention/cops/web/src/route/sessions/index.tsx:139`
- **Severity**: Minor
- **Description**: Type assertion `sortBy as 'name' | 'session_count' | ...` is used. This could be improved with proper validation in `validateSearch` to ensure the value is one of the allowed values.

---

## Execution Plan for Execute Agent

**To pass this review, Execute Agent must fix the following Critical issues:**

### Fix 1: Add `search={{}}` to Link components

**File**: `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx`
- Line 33: Add `search={{}}` prop to Link

**File**: `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/recent-sessions.tsx`
- Line 33: Add `search={{}}` prop to Link

**File**: `/Users/jayce/team-attention/cops/web/src/feature/project/component/project-header.tsx`
- Line 59: Add `search={{}}` prop to Link

**File**: `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx`
- Line 68: Add `search={{}}` prop to Link

### Fix 2: Remove or fix gitBranch property access

**File**: `/Users/jayce/team-attention/cops/web/src/feature/dashboard/component/project-list.tsx`
- Line 95: Remove or fix the gitBranch property access (this appears to be in a section that may need to be removed entirely)

**File**: `/Users/jayce/team-attention/cops/web/src/feature/project/component/project-header.tsx`
- Line 107: Remove or fix the gitBranch property access

### Fix 3: Fix SessionRecord type import

**File**: `/Users/jayce/team-attention/cops/web/src/route/sessions/$sessionId.tsx`
- Lines 52 and 100: Update the import or type reference to use the correct `aggregation.v1.SessionRecord` type instead of `collector.v1.SessionRecord`

---

## Test Verification

- [x] Go build succeeds: `go build ./api/... ./daemon/... ./shared/...`
- [ ] TypeScript types pass: `npx tsc --noEmit` - **FAILS with 8 errors**
- [ ] All tests pass: Not verified (no test command run)

---

## Code Quality Assessment

### Positive Observations

1. **Protobuf naming convention**: Correctly follows `.agent/rules/idl/protobuf.md` with Req/Res naming
2. **Backend modification**: Clean implementation of optional projectID filtering in MongoDB repository
3. **Shared utilities**: Well-organized format utilities with proper TypeScript types
4. **Component structure**: Follows Feature Driven Development structure from `.agent/rules/react/react-web-src.md`
5. **UI consistency**: Maintains dark theme with cyan/violet accent colors
6. **Error handling**: Proper loading, error, and empty states in route pages

### Areas for Improvement

1. Link components need search props due to route validation changes
2. Pre-existing type issues should be addressed
3. Sorting on projects page is UI-only (backend doesn't support it)

---

## Sign-off Statement

**This review cannot be approved** until the 4 Critical issues are resolved. After fixes are applied, the implementation will be ready for commit. The Minor issues can be addressed in follow-up PRs.

---

*Review completed: 2025-12-27*
*Reviewer: Review Agent*
