# Requirements

## Request Summary

Apply Role-Based Access Control (RBAC) to API endpoints to ensure users can only access projects they have permission to view. All Project and Session resources are organized under Organization, and ALL API endpoints (both Dashboard and Aggregation) operate at the organization level. The RBAC service has a single, focused responsibility: validate that users are members of the requested organization. Once RBAC validation passes, service layer logic handles resource validation (e.g., checking if project belongs to organization). All endpoints (SendLogs, GetProject, GetSession, ListProjects, ListSessions, GetOverview) will require organizationID as a mandatory parameter for consistent access control.

## Acceptance Criteria

- [ ] Protobuf definitions updated to add REQUIRED organizationID parameter to ALL endpoints (SendLogs, GetProject, GetSession, ListProjects, ListSessions, GetOverview)
- [ ] Missing organizationID returns 400 Bad Request for all endpoints
- [ ] RBAC service provides single method: CanAccessOrganization(ctx, userID, organizationID) (bool, error)
- [ ] SendLogs endpoint validates user is member of required organization and returns 403 if not
- [ ] GetProject endpoint validates user is member of required organization and returns 403 if not
- [ ] GetSession endpoint validates user is member of required organization and returns 403 if not
- [ ] ListSessions endpoint validates user is member of required organization and returns 403 if not
- [ ] GetOverview endpoint validates user is member of required organization and returns 403 if not
- [ ] ListProjects endpoint validates user is member of required organization and returns 403 if not
- [ ] Service layer validates resource ownership (project belongs to org, session belongs to project in org)
- [ ] All endpoints filter/validate data by organizationID
- [ ] All RBAC checks extract userID from context using existing auth middleware
- [ ] All RBAC denials are logged for security auditing
- [ ] Error responses consistently return 403 Forbidden for authorization failures and 400 Bad Request for validation failures
- [ ] Database queries filter by organizationID for improved performance
- [ ] Unit tests verify RBAC enforcement and validation for all protected endpoints

## Scope

### In Scope

**Protobuf API Changes:**
- Update `SendLogsReq` (aggregation.proto) to add REQUIRED `organization_id` parameter
- Update `GetProjectReq` (dashboard.proto) to add REQUIRED `organization_id` parameter
- Update `GetSessionReq` (dashboard.proto) to add REQUIRED `organization_id` parameter
- Update `ListProjectsReq` (dashboard.proto) to add REQUIRED `organization_id` parameter
- Update `ListSessionsReq` (dashboard.proto) to add REQUIRED `organization_id` parameter
- Update `GetOverviewReq` (dashboard.proto) to add REQUIRED `organization_id` parameter
- Run `buf generate` to regenerate Go code

**RBAC Service Simplification:**
- RBAC service has ONE responsibility: Check organization membership
- Add `CanAccessOrganization(ctx, userID, organizationID) (bool, error)` method
- Remove dependency on project-level checks from RBAC scope
- Service layer handles resource validation after RBAC check passes

**All Endpoints Follow Consistent Pattern:**

**AggregationService:**
- Apply RBAC to `SendLogs(organization_id, batch)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer validates project belongs to organization (return 404 if not found in org)
  - Process logs

**DashboardService:**

- Apply RBAC to `GetProject(project_id, organization_id)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer validates project belongs to organization (return 404 if not found in org)
  - Return project data

- Apply RBAC to `GetSession(session_id, organization_id)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer validates session's project belongs to organization (return 404 if not found in org)
  - Return session data

- Apply RBAC to `ListSessions(project_id, organization_id, pagination)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer filters sessions by organizationID and project_id

- Apply RBAC to `GetOverview(organization_id)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer filters all data (projects, sessions, counts) by organizationID

- Apply RBAC to `ListProjects(organization_id, pagination)` endpoint
  - Validate `organization_id` is provided (return 400 if missing)
  - Check user is member of organization using `CanAccessOrganization()` (return 403 if not)
  - Service layer filters projects by organizationID

### Out of Scope

- ProjectService `RegisterProject` endpoint (will be addressed in future iteration)
- AuthService endpoints (authentication, not authorization)
- HealthService endpoints (public endpoints)
- Existing `CanAccess(ctx, userID, projectID)` method in RBAC service (keep for backward compatibility, but not used in this implementation)
- Creating additional RBAC rules or permission models beyond organization membership
- Implementing role-based permissions (admin, member, etc.)
- Adding API-key based authentication for daemon

## Constraints

**Technical Constraints:**
- Must use existing RBAC service at `api/internal/service/core/rbac/rbac_service.go`
- Must use existing auth middleware's `GetUserID(c *fiber.Ctx)` helper
- Must integrate with ConnectRPC handler pattern
- RBAC service dependency must be injected via FX lifecycle

**Error Handling Constraints:**
- Missing organizationID parameter must return HTTP 400 Bad Request
- All authorization failures must return HTTP 403 Forbidden
- Error messages must clearly indicate "permission denied" or "access denied"
- Must not leak information about project existence to unauthorized users

**Performance Constraints:**
- All endpoints require organizationID: Single membership check per request (efficient)
- Direct DB filtering by organizationID eliminates need for post-query filtering
- Simplified RBAC: only org membership check, no project lookups in RBAC layer
- No need to query multiple organizations or perform cross-org queries

## Additional Context

**RBAC Service - Simplified Interface:**
```go
type Service struct {
    logger      *slog.Logger
    memberRepo  repository.OrganizationMemberRepositoryPort
}

// NEW: Primary method for all endpoints
func (s *Service) CanAccessOrganization(ctx context.Context, userID, organizationID string) (bool, error)

// EXISTING: Keep for backward compatibility (not used in this implementation)
func (s *Service) CanAccess(ctx context.Context, userID, projectID string) (bool, error)
```

**RBAC Responsibility:**
- RBAC service ONLY checks organization membership
- Returns (true, nil) if user is member of organization
- Returns (false, nil) if user is not a member
- Returns (false, error) only for system errors
- Does NOT check project ownership, session ownership, or any resource-level validation
- Service layer handles all resource validation after RBAC check passes

**Authentication Context:**
- Auth middleware extracts JWT token from `Authorization: Bearer <token>` header
- Sets userID in Fiber context using `middleware.UserIDContextKey`
- Use `middleware.GetUserID(c)` to retrieve userID in handlers

**ConnectRPC Context:**
- ConnectRPC handlers receive `context.Context` not `fiber.Ctx`
- Need to investigate how to pass userID from Fiber middleware to ConnectRPC handler context
- May need to use ConnectRPC interceptors or modify handler registration

**RBAC Service Enhancement:**
- Add `CanAccessOrganization(ctx, userID, organizationID)` method
- Simplify RBAC service to ONLY check organization membership
- Remove `projectRepo` dependency from RBAC service (not needed for org membership check)
- Keep existing `CanAccess()` method for backward compatibility (not used in this implementation)

**Resource Model & Client Context:**
- All Projects belong to an Organization
- All Sessions belong to a Project (which belongs to an Organization)
- All log batches are sent to a Project (which belongs to an Organization)
- Web client maintains "currently selected Organization" state
- All API calls (both Dashboard and Aggregation) include organizationID
- Daemon includes organizationID when sending logs
- Missing organizationID in any request is a validation error (400)

**Separation of Concerns:**
- **RBAC Layer**: Only validates organization membership (403 if not member)
- **Service Layer**: Validates resource belongs to organization (404 if not found in org)
- **Repository Layer**: Filters queries by organizationID

**Files to Modify:**
- `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto` - Add organizationID to SendLogsReq
- `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto` - Add organizationID to all request messages
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go` - Add CanAccessOrganization method, simplify to only org checks
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` - Add RBAC check
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go` - Add resource validation
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go` - Add RBAC checks
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/dashboard_service.go` - Add resource validation and organizationID filtering
- Repository layers - Add organizationID filtering to queries

**Dependencies:**
- Handlers need RBAC service injected
- May need to update FX module providers
- Run `buf generate` after protobuf changes

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should SendLogs require organization_id? | YES - Need to know which organization's project the log belongs to |
| Should ALL endpoints require organization_id? | YES - SendLogs, GetProject, GetSession, ListProjects, ListSessions, GetOverview |
| Is organizationID optional or required? | REQUIRED for all endpoints - Missing organizationID = 400 Bad Request |
| How does web client provide organizationID? | Web client maintains "currently selected Organization" state and includes it in all API calls |
| How does daemon provide organizationID? | Daemon knows project's organizationID and includes it when sending logs |
| What is RBAC service's responsibility? | ONLY check organization membership - nothing else |
| What is service layer's responsibility? | Validate resources belong to organization (project in org, session in org, etc.) |
| Should RBAC check project ownership? | NO - RBAC only checks org membership. Service layer validates project belongs to org |
| For SendLogs: Should we reject batch if RBAC fails? | YES - Reject entire batch and return 403 error |
| Pattern for all endpoints? | Validate org_id provided → RBAC checks org membership → Service validates resources → Process request |
| RegisterProject - Check organization membership? | OUT OF SCOPE for this iteration |
| Error behavior on RBAC failure | Return 403 Forbidden (not member of organization) |
| Error behavior on missing organizationID | Return 400 Bad Request (validation error) |
| Error behavior when resource not in org | Return 404 Not Found (project/session not found in organization) |
| Should we log RBAC denials? | YES - For security auditing purposes |
| Need new RBAC method? | YES - Add CanAccessOrganization(ctx, userID, organizationID) as primary method |
| Keep existing CanAccess(projectID) method? | YES - Keep for backward compatibility but don't use in this implementation |
