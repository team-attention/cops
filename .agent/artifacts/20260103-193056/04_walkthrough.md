# Development Walkthrough: RBAC Service Implementation

## Summary

Implemented a Role-Based Access Control (RBAC) service that verifies whether authenticated users have permission to access specific projects. Projects are now owned by organizations, and access is granted to organization members through a simple binary permission check (CanAccess). The service follows hexagonal architecture and is fully integrated with the fx dependency injection container, making it available for other services to inject and use.

## Code Overview

### Domain Model Changes

#### `Project` Domain Model
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/project.go`
- **Changes**: Added `OrganizationID ID` field to the Project struct
- **Rationale**: Projects now belong to organizations, enabling organization-based access control
- **Type Decision**: Used value type (not pointer) because OrganizationID is a required field for all projects

#### `Project` MongoDB Schema
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`
- **Changes**:
  - Added `ProjectOrganizationIDField` constant for BSON field name
  - Added `OrganizationID bson.ObjectID` field to MongoDB schema
  - Updated `FromDomain()` and `ToDomain()` methods to convert between domain ID and BSON ObjectID
- **Rationale**: Ensures proper serialization/deserialization of organization ownership in MongoDB

### New Components

#### RBAC Service
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go`
- **Purpose**: Core business logic for authorization checks
- **Key Methods**:
  - `NewService(l *slog.Logger, projectRepo, memberRepo) *Service`: Constructor that binds logger with "rbac.service" name
  - `CanAccess(ctx, userID, projectID) (bool, error)`: Checks if a user can access a project
- **Logic Flow**:
  1. Validates userID and projectID are not empty (returns false, error if empty)
  2. Queries project by ID to get organizationID (returns false, nil if project not found)
  3. Checks if user is member of project's organization (returns false, nil if not member)
  4. Returns true only if user is confirmed member of project's organization
- **Error Handling**: Returns (false, error) for system errors, (false, nil) for valid denials
- **Logging**:
  - WARN for validation failures
  - INFO for project not found and access denied
  - DEBUG for access granted
  - ERROR for database/system errors

#### Repository Port Interfaces

##### `ProjectRepositoryPort`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/project_repo_port.go`
- **Purpose**: Define the contract for querying project data needed by RBAC
- **Methods**:
  - `GetByID(ctx, projectID) (*domain.Project, error)`: Returns nil, nil if not found; nil, error on DB error

##### `OrganizationMemberRepositoryPort`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go`
- **Purpose**: Define the contract for querying organization membership
- **Methods**:
  - `IsMember(ctx, userID, organizationID) (bool, error)`: Returns true if membership exists, false if not, error on DB error

#### MongoDB Repository Adapters

##### `MongoProjectRepository`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/project_repo.go`
- **Purpose**: Implements ProjectRepositoryPort for MongoDB
- **Key Implementation Details**:
  - Converts projectID string to bson.ObjectID before querying
  - Returns nil, nil when project not found (mongo.ErrNoDocuments)
  - Uses mongoschema.Project for BSON marshaling and domain conversion
  - Logger bound with "rbac.repository.mongodb.project"

##### `MongoOrganizationMemberRepository`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`
- **Purpose**: Implements OrganizationMemberRepositoryPort for MongoDB
- **Key Implementation Details**:
  - Converts both userID and organizationID to bson.ObjectID before querying
  - Uses CountDocuments with limit 1 for efficient membership check
  - Returns count > 0 as boolean membership result
  - Logger bound with "rbac.repository.mongodb.organization_member"

#### Mock Repository Implementations

##### `ProjectRepository` Mock
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/project_repo_mock.go`
- **Purpose**: Test double for ProjectRepositoryPort
- **Features**:
  - `GetByIDFunc` for injecting test behavior
  - `CallCount` for verifying number of calls
  - `ProjectIDs` slice for tracking all queried project IDs

##### `OrganizationMemberRepository` Mock
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/organization_member_repo_mock.go`
- **Purpose**: Test double for OrganizationMemberRepositoryPort
- **Features**:
  - `IsMemberFunc` for injecting test behavior
  - `CallCount` for verifying number of calls
  - `Queries` slice of `MembershipQuery` structs for tracking all membership checks

#### fx Module Registration

##### RBAC Module
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_rbac.go`
- **Purpose**: Registers RBAC service and its dependencies in the fx container
- **Providers**:
  1. `MongoProjectRepository` with `fx.As(new(repository.ProjectRepositoryPort))`
  2. `MongoOrganizationMemberRepository` with `fx.As(new(repository.OrganizationMemberRepositoryPort))`
  3. `rbac.NewService` (receives port interfaces, not concrete implementations)
- **Module Name**: "rbac"

##### Application Composition
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`
- **Changes**: Added `newRBACModule()` to fx.New() call
- **Integration**: RBAC service is now available for injection into other services

## Testing

### Unit Tests
- **Test Suite**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_suite_test.go`
- **Test Implementation**: `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service_test.go`

### Test Scenarios (7 scenarios, all passing)

| Scenario | Setup | Expected Result | Purpose |
|----------|-------|-----------------|---------|
| User is member | Project exists, user is org member | (true, nil) | Verify access granted for valid members |
| User is not member | Project exists, user not org member | (false, nil) | Verify access denied for non-members |
| Project not found | Project does not exist | (false, nil) | Verify graceful handling of missing projects |
| Empty userID | userID = "" | (false, error) | Verify input validation |
| Empty projectID | projectID = "" | (false, error) | Verify input validation |
| Project query fails | DB error on project lookup | (false, error) | Verify error propagation |
| Membership query fails | DB error on membership check | (false, error) | Verify error propagation |

### Test Framework
- **Framework**: Ginkgo v2 with Gomega matchers
- **Mock Pattern**: Function injection pattern from daemon service examples
- **Logger**: Real slog.Logger with text handler to stdout

### Verification Commands Run

```bash
go test ./internal/service/core/rbac/... -v  # Result: PASS (7/7 tests)
go build ./api/... ./shared/...              # Result: SUCCESS
```

## Architecture Decisions

### 1. Core Service Placement

**Decision**: Placed RBAC under `internal/service/core/rbac/`

**Rationale**:
- RBAC is a cross-cutting service used by multiple domains
- Core services are allowed to be injected into other services per project rules
- Follows the pattern of other shared services in the codebase

### 2. Simple Binary Access Check

**Decision**: Implemented CanAccess as boolean (true/false) rather than role-based permissions

**Rationale**:
- Requirements specified simple binary access check
- Both admin and member roles get same access for now
- Easier to extend later with role-based logic if needed
- Meets current use case for project access verification

### 3. Parameters vs Context

**Decision**: Pass userID and projectID as method parameters, not extracted from context

**Rationale**:
- Makes service more flexible and testable
- Service doesn't need to know about HTTP middleware or context structure
- Caller controls what user and project to check
- Explicit parameter passing is clearer than implicit context extraction

### 4. Error Handling Strategy

**Decision**: Return (false, nil) for access denied, (false, error) for system errors

**Rationale**:
- Distinguishes between "permission denied" (business logic) and "system failure" (technical error)
- Allows callers to handle authorization failures differently from system errors
- Project not found is treated as access denied (false, nil) rather than error
- Follows Go convention of using error for unexpected failures only

### 5. Repository Port Separation

**Decision**: Created separate port interfaces for ProjectRepository and OrganizationMemberRepository

**Rationale**:
- Each port has focused, single responsibility
- Allows different implementations for different data sources
- Easier to mock individual dependencies in tests
- Follows Interface Segregation Principle (ISP)

### 6. Struct Field Type: OrganizationID as Value Type

**Decision**: Used `OrganizationID ID` (value type) instead of `*ID` (pointer type)

**Rationale**:
- OrganizationID is a required field per requirements
- Follows project rule from `.agent/rules/go/go-struct.md`: required fields use value types
- Cannot be nil/absent in JSON or database
- Enforces that all projects must belong to an organization

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Need to distinguish between project queries for different services | Created RBAC-specific repository ports instead of reusing existing project repository to avoid coupling |
| MongoDB ObjectID vs domain.ID conversion | Used existing mongoschema pattern with FromDomain/ToDomain methods for consistent conversion |
| Test isolation without real database | Implemented mock repositories with function injection pattern following daemon service examples |
| Determining error vs denied scenarios | Established convention: (false, nil) for valid denials, (false, error) for system failures |

## Integration Points

### How Other Services Use RBAC

Other services can now inject the RBAC service to verify permissions:

```go
type AggregationService struct {
    logger  *slog.Logger
    rbacSvc *rbac.Service  // Inject RBAC service
    // other dependencies...
}

func (s *AggregationService) SendLogs(ctx context.Context, userID, projectID string, logs []Log) error {
    // Check access first
    canAccess, err := s.rbacSvc.CanAccess(ctx, userID, projectID)
    if err != nil {
        return fmt.Errorf("permission check failed: %w", err)
    }
    if !canAccess {
        return fmt.Errorf("access denied to project %s", projectID)
    }

    // Proceed with business logic...
}
```

### fx Dependency Injection

The RBAC service is automatically available through fx container:

1. `newRBACModule()` registers all RBAC components
2. Services declare `*rbac.Service` in their constructor parameters
3. fx automatically injects the RBAC service instance
4. No manual wiring needed

## Next Steps (Out of Scope for This Implementation)

The following were explicitly scoped out but documented for future work:

1. **Apply RBAC to existing endpoints**: Update aggregation service SendLogs to use RBAC
2. **ConnectRPC middleware**: Automatic RBAC enforcement for all gRPC endpoints
3. **Role-based permissions**: Distinguish between admin and member access levels
4. **Daemon authentication**: Separate mechanism for daemon to authenticate when sending logs
5. **Permission caching**: Performance optimization for frequently checked permissions
6. **Audit logging**: Track authorization decisions for security monitoring

## Related Tickets

This implementation establishes the foundation for project-level access control in the C-Ops system, enabling secure multi-tenant operation where users can only access projects within their organizations.
