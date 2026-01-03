# Requirements

## Request Summary

Create a core RBAC (Role-Based Access Control) service in `api/internal/service/core/rbac/` that verifies if authenticated users have permission to access specific resources. Projects are owned by organizations, and access is granted to organization members. The service provides a simple binary access check API (CanAccess) that accepts userID and projectID as parameters. This service will be injectable into other services (like aggregation service) to enforce authorization before resource operations.

## Acceptance Criteria

- [ ] OrganizationID field added to Project domain model and MongoDB schema
- [ ] RBAC service created under `api/internal/service/core/rbac/` following hexagonal architecture
- [ ] RBAC service can be injected into other services via fx dependency injection
- [ ] Service provides `CanAccess(ctx context.Context, userID, projectID string) (bool, error)` method
- [ ] Permission logic: User can access project if they are a member of the project's organization
- [ ] Service accepts userID and projectID as parameters (not from context)
- [ ] Service assumes userID is already authenticated (no anonymous access handling)
- [ ] Outbound port defined for querying organization membership
- [ ] Service follows project logging conventions with structured slog logger
- [ ] Unit tests verify permission logic for different scenarios (member, non-member, invalid IDs)

## Scope

### In Scope
- Add OrganizationID field to `shared/domain/project.go` and `shared/domain/mongoschema/project.go`
- Create RBAC service structure under `api/internal/service/core/rbac/`
- Implement `CanAccess(ctx, userID, projectID)` method with simple binary access check
- Define outbound port for querying project's organization and user's membership
- Register RBAC service in fx container for injection into other services
- Permission logic: Grant access if user is a member of the project's organization (any role)
- Unit tests for permission scenarios (member has access, non-member denied, error cases)

### Out of Scope
- Applying RBAC to existing endpoints (will be done separately)
- Differentiating between admin/member roles (both get same access for now)
- Resource-level permissions beyond projects (sessions, logs, etc.)
- Permission caching or performance optimization
- Audit logging of authorization decisions
- ConnectRPC middleware for automatic RBAC enforcement
- Handling anonymous/unauthenticated users (authentication happens at middleware layer)
- Daemon authentication mechanism (separate concern)

## Constraints

- Must follow hexagonal architecture pattern (see `.agent/rules/go/go-hexagonal-layout.md`)
- Must be placed under `internal/service/core/` as it's a cross-cutting service
- Must use `go.uber.org/fx` for dependency injection
- Must follow existing logging conventions (`*slog.Logger` as first parameter)
- Must integrate with existing domain models (`shared/domain/user.go`, `shared/domain/organization.go`, `shared/domain/project.go`)
- Cannot import other services directly (only platform and domain packages)

## Additional Context

### Existing Authentication System
- JWT-based authentication is implemented (`api/internal/platform/middleware/auth.go`)
- AuthMiddleware extracts userID and stores in Fiber context
- JWT utilities available in `api/internal/platform/util/jwtutil/`

### Current Authorization Gap
- The aggregation service's `SendLogs` endpoint has no authentication
- No project ownership validation before saving logs
- Daemon sends logs without authentication credentials

### Domain Models
- `domain.User` - User entity with email, name, accounts
- `domain.Organization` - Organization entity with name, slug
- `domain.OrganizationMember` - Membership with roles (admin, member)
- `domain.MemberRole` - Enum: admin, member
- `domain.Project` - Project entity with ID, name, path (will add organizationID field)

### Service Architecture
- Services follow pattern: `{domain}/{domain}_service.go`
- Constructor: `NewService(l *slog.Logger, ...dependencies) *Service`
- Logger binding: `l.With(slog.String("name", "{domain}.service"))`
- Outbound ports defined in `{domain}/outbound/{category}/{name}_port.go`
- Outbound adapters in `{domain}/outbound/{category}/{implementation}/`

### Implementation Details

**Access Check Logic:**
```
CanAccess(ctx, userID, projectID) -> bool, error
  1. Validate userID and projectID are valid ObjectIDs
  2. Query project by projectID to get organizationID
  3. Query organization_members to check if userID is member of organizationID
  4. Return true if membership exists, false otherwise
```

**Required Outbound Ports:**
- `ProjectRepositoryPort.GetByID(ctx, projectID) -> Project, error`
- `OrganizationMemberRepositoryPort.IsMember(ctx, userID, organizationID) -> bool, error`

**Error Handling:**
- Return `(false, error)` for system errors (DB failures, etc.)
- Return `(false, nil)` for valid requests where access is denied
- Log authorization decisions at appropriate level (denied = INFO, errors = ERROR)

## Questions Resolved

| Question | Answer |
| --- | --- |
| Should RBAC service check project ownership only, or also organization membership? | Organization membership - Projects will have an organizationID field, and access is granted to organization members. |
| What is the relationship between projects and organizations? Does each project belong to an organization? | Yes, each project belongs to exactly one organization. Need to add organizationID field to Project model. |
| For the aggregation service's SendLogs endpoint, should we verify the daemon is authorized to write logs for a specific project? | Out of scope for this iteration - daemon authentication is a separate concern to be addressed later. |
| Should we support different permission levels (read-only, read-write, admin), or just binary access (can access / cannot access)? | Simple binary access (can access / cannot access). Both admin and member roles get the same access for now. |
| When checking permissions, should we retrieve the userID from context (already authenticated), or pass it as a parameter? | Pass as parameter - `CanAccess(ctx, userID, projectID)`. This makes the service more flexible and testable. |
| Should the RBAC service handle anonymous/unauthenticated access, or assume authentication happens at middleware layer? | Assume authentication happens at middleware layer. RBAC service only works with authenticated users (valid userID). Always deny access if userID is invalid. |
