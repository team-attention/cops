# Requirements

## Request Summary

The web application's dashboard and projects pages are failing to load data with a 400 error: "organization_id is required". Recent commits added organization-based RBAC to the API, requiring all dashboard endpoints to include an organization_id parameter. However, the frontend hooks (useGetOverview, useListProjects, useListSessions, etc.) are not passing this required parameter, causing API calls to fail validation at the service layer.

## Acceptance Criteria

- [ ] All dashboard API hooks (useGetOverview, useListProjects, useListSessions, useGetProject, useGetSession) must pass the selected organization ID from the user store
- [ ] The organization ID must be retrieved from useUserStore's selectedOrganizationId
- [ ] API calls should only be made when a valid organization ID exists (enabled: !!selectedOrganizationId)
- [ ] The /dashboard, /projects, /sessions pages successfully load data without 400 errors
- [ ] The organization ID is correctly sent in all dashboard API requests matching the protobuf schema
- [ ] Edge case: Hooks should be disabled when no organization is selected

## Scope

### In Scope
- Update frontend hooks in `web/src/feature/` to pass organization_id:
  - `dashboard/hook/use-get-overview.ts`
  - `project/hook/use-list-projects.ts`
  - `project/hook/use-list-sessions.ts`
  - `project/hook/use-get-project.ts`
  - `session/hook/use-get-session.ts`
- All hooks should access `useUserStore` to get `selectedOrganizationId`
- Disable queries when organization ID is not available

### Out of Scope
- Backend API changes (validation is working as intended)
- Protobuf schema changes (organization_id fields already exist)
- User store modifications (selectedOrganizationId already exists and is managed)
- Organization selection UI (already implemented)

## Constraints

- Must follow existing hook patterns established in the codebase
- Must use the ConnectRPC/TanStack Query pattern already in place
- Frontend hook files should follow the naming convention: `use-{rpc-method}.ts`
- Must not modify generated protobuf stub code in `web/src/gen/grpcstub/`

## Additional Context

### Root Cause Analysis

1. **Recent RBAC Implementation** (commit 1a530c5 "feat(rbac): apply organization-level RBAC to all API endpoints"):
   - Added organization_id requirement to all dashboard API endpoints
   - Updated protobuf schemas to include organization_id as required field
   - Added validation in service layer (dashboard_service.go line 34-36)

2. **Frontend State Management**:
   - User store (`shared/store/user-store.ts`) already tracks selectedOrganizationId
   - Organization selection is auto-set when user data loads (line 62-69 of user-store.ts)
   - useUser hook provides access to selectedOrganizationId

3. **API Request Flow**:
   - Frontend hooks call protobuf-generated functions
   - API handler receives request, calls service layer
   - Service validates organization_id via checkRBAC() which throws error if missing

### Affected Protobuf Messages

All dashboard requests require organization_id:
- GetOverviewReq (line 152 in dashboard.proto)
- ListProjectsReq (line 176)
- GetProjectReq (line 194)
- ListSessionsReq (line 209)
- GetSessionReq (line 236)

### Related Files

Backend validation:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/dashboard_service.go` (checkRBAC function)

Frontend hooks needing updates:
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/hook/use-get-overview.ts`
- `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-projects.ts`
- `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-list-sessions.ts`
- `/Users/jayce/team-attention/cops/web/src/feature/project/hook/use-get-project.ts`
- `/Users/jayce/team-attention/cops/web/src/feature/session/hook/use-get-session.ts`

## Questions Resolved

| Question | Answer |
|----------|--------|
| Where does the organization_id validation happen? | In the API service layer at `dashboard_service.go:34-36` via the checkRBAC function |
| Where should the frontend get the organization ID? | From useUserStore's selectedOrganizationId, which is auto-selected when user data loads |
| Should we modify the protobuf schema? | No, the schema is correct and already has organization_id as a required field |
| What happens when no organization is selected? | The hooks should be disabled (enabled: false) to prevent API calls, though the dashboard page already redirects users without organizations to /organizations/new |
| Are there other endpoints affected? | Yes, all dashboard endpoints (GetOverview, ListProjects, GetProject, ListSessions, GetSession) require the fix |
