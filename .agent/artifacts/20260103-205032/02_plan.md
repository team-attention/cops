# Implementation Plan: RBAC for All API Endpoints

## Overview

This implementation applies Role-Based Access Control (RBAC) to all API endpoints (Dashboard and Aggregation services) to ensure users can only access resources belonging to organizations they are members of. The RBAC service has a single responsibility: validate organization membership via `CanAccessOrganization(ctx, userID, organizationID)`. Service layer handles both RBAC validation and resource validation (project belongs to org, session belongs to org, etc.).

Key architectural decisions:
1. **ConnectRPC Interceptor for Authentication**: Create a ConnectRPC unary interceptor that validates JWT from `Authorization` header and adds `userID` to `context.Context` using `context.WithValue`. This replaces reliance on Fiber `c.Locals()`.
2. **RBAC Simplification**: Replace existing `CanAccess` method with `CanAccessOrganization`. DELETE the old `CanAccess` method and remove unused `projectRepo` dependency.
3. **RBAC in Service Layer**: RBAC is business logic, so it belongs in the Service layer. Handlers do NOT inject RBAC service. Services inject RBAC service and validate at the start of each method.
4. **Organization-scoped Queries**: All repository methods receive `organizationID` and filter data at the database level.
5. **LogBatch with OrganizationID**: The LogBatch struct includes OrganizationID field for validation.
6. **Consistent Error Handling**: Missing `organization_id` returns 400 Bad Request; RBAC failures return 403 Forbidden; resource not found returns 404 Not Found.

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | N/A | N/A | All required packages are already available in the project |

## Step 1: Update Protobuf Definitions

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions
- `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`: Current aggregation proto
- `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto`: Current dashboard proto

#### `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`

**Description**:
Add `organization_id` field to `LogBatch` message (NOT SendLogsReq). Each LogBatch represents logs for one Project, and that Project belongs to an Organization. The organization_id is part of the batch metadata.

```protobuf
// LogBatch contains raw JSONL lines for batch sending.
// Each batch represents logs for one Project that belongs to an Organization.
message LogBatch {
  // Organization identifier (required - project belongs to this org)
  string organization_id = 1;

  // Project identifier
  string project_id = 2;

  // Raw JSONL lines
  repeated string jsonl = 3;
}

// SendLogsReq is the request for sending logs.
// NOTE: organization_id is inside LogBatch, NOT at request level
message SendLogsReq {
  // Log batch to send (contains organization_id and project_id)
  LogBatch batch = 1;
}
```

**Field Number Changes**:
- `LogBatch.organization_id` = 1 (NEW)
- `LogBatch.project_id` = 2 (was 2, no change)
- `LogBatch.jsonl` = 3 (was 1, CHANGED - this is a breaking change for existing clients)

#### `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto`

**Description**:
Add `organization_id` field to all request messages.

```protobuf
// GetOverviewReq is the request for GetOverview RPC.
message GetOverviewReq {
  // Organization identifier (required)
  string organization_id = 1;
}

// ListProjectsReq is the request for ListProjects RPC.
message ListProjectsReq {
  // Organization identifier (required)
  string organization_id = 1;

  // Pagination parameters
  PaginationReq pagination = 2;
}

// GetProjectReq is the request for GetProject RPC.
message GetProjectReq {
  // Organization identifier (required)
  string organization_id = 1;

  // Project identifier
  string project_id = 2;
}

// ListSessionsReq is the request for ListSessions RPC.
message ListSessionsReq {
  // Organization identifier (required)
  string organization_id = 1;

  // Project identifier
  string project_id = 2;

  // Pagination parameters
  PaginationReq pagination = 3;

  // Sort field: "started_at", "message_count", "usage"
  string sort_by = 4;

  // Sort in descending order
  bool sort_desc = 5;
}

// GetSessionReq is the request for GetSession RPC.
message GetSessionReq {
  // Organization identifier (required)
  string organization_id = 1;

  // Session identifier
  string session_id = 2;
}
```

After modifying proto files, run:
```bash
cd idl/protobuf && buf generate
```

---

## Step 2: Create ConnectRPC Authentication Interceptor

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform package structure
- `/Users/jayce/team-attention/cops/api/internal/platform/middleware/auth.go`: Existing Fiber auth middleware
- `/Users/jayce/team-attention/cops/api/internal/platform/util/jwtutil/jwtutil.go`: JWT validation utility

#### `/Users/jayce/team-attention/cops/api/internal/platform/interceptor/auth_interceptor.go`

**Description**:
Create a new package `interceptor` under `platform` for ConnectRPC interceptors. This interceptor validates JWT tokens and adds `userID` to the context.

```go
package interceptor

import (
	"context"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// userIDContextKey is the context key for storing userID.
type userIDContextKey struct{}

// UserIDFromContext extracts userID from context.
// Returns empty string if not found.
func UserIDFromContext(ctx context.Context) string {
	// Implementation outline:
	// 1. Get value from context using userIDContextKey{}.
	// 2. If value is nil, return empty string.
	// 3. Type assert to string.
	// 4. If assertion fails, return empty string.
	// 5. Return the userID string.
}

// NewAuthInterceptor creates a ConnectRPC unary interceptor for JWT authentication.
// It validates the Authorization header and adds userID to context.
func NewAuthInterceptor(l *slog.Logger, jwtCfg *jwtutil.Config) connect.UnaryInterceptorFunc {
	// Implementation outline:
	// 1. Create logger with name "interceptor.auth".
	// 2. Return UnaryInterceptorFunc that wraps the next handler.
	// 3. Inside the interceptor:
	//    a. Check if request is client-side (req.Spec().IsClient).
	//       - If client-side, call next(ctx, req) directly (no auth needed).
	//    b. Extract Authorization header from req.Header().Get("Authorization").
	//    c. If header is empty:
	//       - Return connect.NewError(connect.CodeUnauthenticated, "missing authorization header").
	//    d. If header does not start with "Bearer ":
	//       - Return connect.NewError(connect.CodeUnauthenticated, "invalid authorization header format").
	//    e. Extract token by trimming "Bearer " prefix.
	//    f. Validate token using jwtutil.ValidateAccessToken(jwtCfg, tokenString).
	//    g. If validation fails:
	//       - Log warning with error.
	//       - Return connect.NewError(connect.CodeUnauthenticated, "invalid or expired token").
	//    h. Create new context with userID: ctx = context.WithValue(ctx, userIDContextKey{}, userID).
	//    i. Call next(ctx, req) with the enriched context.
	//    j. Return the result.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid token | Authorization: Bearer valid_token | userID in context, next called | Happy path |
| Missing header | No Authorization header | CodeUnauthenticated error | Empty header branch |
| Invalid format | Authorization: Basic token | CodeUnauthenticated error | Invalid format branch |
| Invalid token | Authorization: Bearer invalid | CodeUnauthenticated error | JWT validation failure |
| Client request | IsClient = true | next called directly | Client bypass branch |

---

## Step 3: Register Auth Interceptor in Container

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container registration patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`: Current ConnectRPC registration

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`

**Description**:
Modify the ConnectRPC server registration to include the auth interceptor as a handler option.

```go
package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// ConnectHandler interface for ConnectRPC handlers.
type ConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

type connectRPCServerParams struct {
	fx.In

	Lifecycle       fx.Lifecycle
	Logger          *slog.Logger
	Config          *config.Config
	App             *fiber.App
	ConnectHandlers []ConnectHandler `group:"connect_handlers"`
}

func registerConnectRPCServer(params connectRPCServerParams) {
	// Implementation outline:
	// 1. Create logger with name "server.connectrpc".
	// 2. Create JWT config from params.Config.
	// 3. Create auth interceptor using interceptor.NewAuthInterceptor(logger, jwtCfg).
	// 4. Create handler options with the interceptor:
	//    opts := []connect.HandlerOption{connect.WithInterceptors(authInterceptor)}
	// 5. Register ConnectRPC handlers with the interceptor options:
	//    for _, handler := range params.ConnectHandlers {
	//        path, h := handler.GetHandler(opts...)
	//        params.App.All(path+"*", adaptor.HTTPHandler(h))
	//    }
	// 6. Add lifecycle hooks for server start/stop.
}
```

---

## Step 4: Refactor RBAC Service (DELETE CanAccess, ADD CanAccessOrganization)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go`: Current RBAC service
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go`: Member repository interface

#### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go`

**Description**:
Refactor RBAC service:
1. DELETE the existing `CanAccess` method entirely (no backward compatibility)
2. REMOVE `projectRepo` dependency from Service struct (not needed for org membership check)
3. ADD `CanAccessOrganization` method as the ONLY method

```go
package rbac

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// Service implements RBAC business logic.
// ONLY checks organization membership - no project-level checks.
type Service struct {
	logger     *slog.Logger
	memberRepo repository.OrganizationMemberRepositoryPort
	// NOTE: projectRepo REMOVED - not needed for org membership check
}

// NewService creates a new RBAC service.
func NewService(
	l *slog.Logger,
	memberRepo repository.OrganizationMemberRepositoryPort,
) *Service {
	// Implementation outline:
	// 1. Return Service with logger and memberRepo only.
	// 2. Logger bound with name "rbac.service".
	// NOTE: projectRepo parameter REMOVED from constructor.
}

// CanAccessOrganization checks if a user is a member of an organization.
// Returns true if user is a member (any role: admin or member).
// Returns false, nil if user is not a member.
// Returns false, error if a system error occurs.
func (s *Service) CanAccessOrganization(ctx context.Context, userID, organizationID string) (bool, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	//    - If empty, log warning and return false, fmt.Errorf("userID is required").
	// 2. Validate organizationID is not empty.
	//    - If empty, log warning and return false, fmt.Errorf("organizationID is required").
	// 3. Call s.memberRepo.IsMember(ctx, userID, organizationID).
	// 4. If error:
	//    - Log error with userID, organizationID, and error.
	//    - Return false, error.
	// 5. If not a member:
	//    - Log info "access denied: user is not member of organization" with userID and organizationID.
	// 6. If is a member:
	//    - Log debug "access granted" with userID and organizationID.
	// 7. Return isMember, nil.
}

// NOTE: CanAccess method DELETED - no backward compatibility needed
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid member | userID="user1", orgID="org1" (member exists) | true, nil | Happy path |
| Not a member | userID="user1", orgID="org2" (no membership) | false, nil | Not member branch |
| Empty userID | userID="", orgID="org1" | false, error | Validation branch |
| Empty orgID | userID="user1", orgID="" | false, error | Validation branch |
| DB error | userID="user1", orgID="org1" (DB fails) | false, error | Error handling branch |

---

## Step 5: Update RBAC Module Registration (Remove projectRepo)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_rbac.go`: RBAC module registration

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_rbac.go`

**Description**:
Remove ProjectRepository provider since RBAC service no longer needs it.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb"
)

func newRBACModule() fx.Option {
	return fx.Module("rbac",
		// NOTE: Project repository REMOVED - no longer needed

		// Organization member repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationMemberRepository,
				fx.As(new(repository.OrganizationMemberRepositoryPort)),
			),
		),

		// RBAC Service (now only requires memberRepo)
		fx.Provide(rbac.NewService),
	)
}
```

---

## Step 6: Update Dashboard Handler (NO RBAC - delegate to Service)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go`: Current handler
- `/Users/jayce/team-attention/cops/api/internal/platform/util/errutil/errutil.go`: Error utilities

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go`

**Description**:
Handler does NOT inject RBAC service. It only extracts userID from context and passes it to service methods. RBAC validation happens in the Service layer.

```go
package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	dashboardservice "github.com/team-attention/cops/api/internal/service/dashboard"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1/dashboardv1connect"
)

// DashboardGRPCHandler handles dashboard gRPC endpoints.
// NOTE: NO rbacSvc field - RBAC is handled in Service layer
type DashboardGRPCHandler struct {
	svc    *dashboardservice.Service
	logger *slog.Logger
}

// NewDashboardGRPCHandler creates a new dashboard gRPC handler.
// NOTE: NO rbac.Service parameter - RBAC is handled in Service layer
func NewDashboardGRPCHandler(l *slog.Logger, svc *dashboardservice.Service) *DashboardGRPCHandler {
	// Implementation outline:
	// 1. Return new handler with service and logger.
	// 2. Logger bound with name "dashboard.grpc.connectrpc".
}

// GetOverview returns dashboard summary statistics.
func (h *DashboardGRPCHandler) GetOverview(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetOverviewReq],
) (*connect.Response[dashboardv1.GetOverviewRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Call service: h.svc.GetOverview(ctx, userID, req.Msg.GetOrganizationId()).
	//    - Service handles RBAC validation internally.
	// 3. If error, return nil, error (service returns appropriate connect.Error).
	// 4. Convert result to protobuf response.
	// 5. Return response.
}

// ListProjects returns a paginated list of projects.
func (h *DashboardGRPCHandler) ListProjects(
	ctx context.Context,
	req *connect.Request[dashboardv1.ListProjectsReq],
) (*connect.Response[dashboardv1.ListProjectsRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Build params with OrganizationID from request.
	// 3. Call service: h.svc.ListProjects(ctx, userID, params).
	//    - Service handles RBAC validation internally.
	// 4. If error, return nil, error.
	// 5. Convert result to protobuf response.
	// 6. Return response.
}

// GetProject returns detailed project information.
func (h *DashboardGRPCHandler) GetProject(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetProjectReq],
) (*connect.Response[dashboardv1.GetProjectRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Call service: h.svc.GetProject(ctx, userID, req.Msg.GetOrganizationId(), req.Msg.GetProjectId()).
	//    - Service handles RBAC validation and project ownership.
	// 3. If error, return nil, error.
	// 4. Convert result to protobuf response.
	// 5. Return response.
}

// ListSessions returns sessions for a project.
func (h *DashboardGRPCHandler) ListSessions(
	ctx context.Context,
	req *connect.Request[dashboardv1.ListSessionsReq],
) (*connect.Response[dashboardv1.ListSessionsRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Build params with OrganizationID and ProjectID from request.
	// 3. Call service: h.svc.ListSessions(ctx, userID, params).
	//    - Service handles RBAC validation and project ownership.
	// 4. If error, return nil, error.
	// 5. Convert result to protobuf response.
	// 6. Return response.
}

// GetSession returns detailed session information with records.
func (h *DashboardGRPCHandler) GetSession(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetSessionReq],
) (*connect.Response[dashboardv1.GetSessionRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Call service: h.svc.GetSession(ctx, userID, req.Msg.GetOrganizationId(), req.Msg.GetSessionId()).
	//    - Service handles RBAC validation and session ownership.
	// 3. If error, return nil, error.
	// 4. Convert result to protobuf response.
	// 5. Return response.
}
```

---

## Step 7: Update Dashboard Service with RBAC Injection

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/dashboard_service.go`: Current service
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go`: Repository interface

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go`

**Description**:
Update all repository interface methods to accept `organizationID` parameter.

```go
package repository

// DashboardRepositoryPort defines the interface for dashboard data access.
type DashboardRepositoryPort interface {
	// GetOverviewStats retrieves dashboard overview statistics for an organization.
	GetOverviewStats(ctx context.Context, organizationID string) (*OverviewStats, error)

	// ListProjects retrieves a paginated list of projects for an organization.
	ListProjects(ctx context.Context, params ListProjectsParams) (*PaginatedProjects, error)

	// GetProject retrieves detailed project information.
	// Returns nil, errutil.NotFound if project not found or does not belong to organization.
	GetProject(ctx context.Context, organizationID, projectID string) (*ProjectDetail, error)

	// ListSessions retrieves paginated sessions for a project in an organization.
	// Returns empty result if project does not belong to organization.
	ListSessions(ctx context.Context, params ListSessionsParams) (*PaginatedSessions, error)

	// GetSession retrieves detailed session information with all records.
	// Returns nil, errutil.NotFound if session not found or its project does not belong to organization.
	GetSession(ctx context.Context, organizationID, sessionID string) (*SessionDetail, error)
}

// ListProjectsQuery contains filter conditions for listing projects.
type ListProjectsQuery struct {
	OrganizationID string
}

// ListSessionsQuery contains filter conditions for listing sessions.
type ListSessionsQuery struct {
	OrganizationID string
	ProjectID      string
	SortBy         string
	SortDesc       bool
}
```

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/dashboard_service.go`

**Description**:
Update service to inject RBAC service and validate RBAC at the start of each method.

```go
package dashboard

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
)

// Service implements dashboard business logic.
// Injects RBAC service for authorization checks.
type Service struct {
	logger  *slog.Logger
	repo    repository.DashboardRepositoryPort
	rbacSvc *rbac.Service
}

// NewService creates a new dashboard service.
func NewService(l *slog.Logger, repo repository.DashboardRepositoryPort, rbacSvc *rbac.Service) *Service {
	// Implementation outline:
	// 1. Return Service with logger, repo, and rbacSvc.
	// 2. Logger bound with name "dashboard.service".
}

// checkRBAC validates organization access for the user.
// Returns connect.Error if validation fails.
func (s *Service) checkRBAC(ctx context.Context, userID, organizationID string) error {
	// Implementation outline:
	// 1. If organizationID is empty:
	//    - Return connect.NewError(connect.CodeInvalidArgument, "organization_id is required").
	// 2. If userID is empty:
	//    - Return connect.NewError(connect.CodeUnauthenticated, "user not authenticated").
	// 3. Call s.rbacSvc.CanAccessOrganization(ctx, userID, organizationID).
	// 4. If error:
	//    - Log error.
	//    - Return connect.NewError(connect.CodeInternal, "failed to check access").
	// 5. If not authorized:
	//    - Log info for security audit with userID and organizationID.
	//    - Return connect.NewError(connect.CodePermissionDenied, "access denied to organization").
	// 6. Return nil (access granted).
}

// GetOverview retrieves dashboard overview statistics for an organization.
func (s *Service) GetOverview(ctx context.Context, userID, organizationID string) (*repository.OverviewStats, error) {
	// Implementation outline:
	// 1. Check RBAC using s.checkRBAC(ctx, userID, organizationID).
	//    - If error, return nil, error.
	// 2. Call s.repo.GetOverviewStats(ctx, organizationID).
	// 3. If error, log and return error.
	// 4. Return stats.
}

// ListProjects retrieves a paginated list of projects for an organization.
func (s *Service) ListProjects(ctx context.Context, userID string, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	// Implementation outline:
	// 1. Check RBAC using s.checkRBAC(ctx, userID, params.Query.OrganizationID).
	//    - If error, return nil, error.
	// 2. Apply defaults to params.
	// 3. Call s.repo.ListProjects(ctx, params).
	// 4. If error, log and return error.
	// 5. Return projects.
}

// GetProject retrieves detailed project information.
// Validates RBAC and project belongs to organization.
func (s *Service) GetProject(ctx context.Context, userID, organizationID, projectID string) (*repository.ProjectDetail, error) {
	// Implementation outline:
	// 1. Check RBAC using s.checkRBAC(ctx, userID, organizationID).
	//    - If error, return nil, error.
	// 2. Call s.repo.GetProject(ctx, organizationID, projectID).
	//    - Repository validates project belongs to organization.
	//    - Returns errutil.NotFound if not found or wrong org.
	// 3. If error, log and return error.
	// 4. Return project.
}

// ListSessions retrieves paginated sessions for a project.
// Validates RBAC and project belongs to organization.
func (s *Service) ListSessions(ctx context.Context, userID string, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	// Implementation outline:
	// 1. Check RBAC using s.checkRBAC(ctx, userID, params.Query.OrganizationID).
	//    - If error, return nil, error.
	// 2. Apply defaults to params.
	// 3. Call s.repo.ListSessions(ctx, params).
	//    - Repository validates project belongs to organization.
	// 4. If error, log and return error.
	// 5. Return sessions.
}

// GetSession retrieves detailed session information with all records.
// Validates RBAC and session's project belongs to organization.
func (s *Service) GetSession(ctx context.Context, userID, organizationID, sessionID string) (*repository.SessionDetail, error) {
	// Implementation outline:
	// 1. Check RBAC using s.checkRBAC(ctx, userID, organizationID).
	//    - If error, return nil, error.
	// 2. Call s.repo.GetSession(ctx, organizationID, sessionID).
	//    - Repository validates session's project belongs to organization.
	//    - Returns errutil.NotFound if not found or wrong org.
	// 3. If error, log and return error.
	// 4. Return session.
}
```

**Test Scenarios for checkRBAC**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid access | orgID="org1", userID="user1" (member) | nil | Happy path |
| Missing orgID | orgID="", userID="user1" | CodeInvalidArgument | Empty orgID branch |
| Missing userID | orgID="org1", userID="" | CodeUnauthenticated | Empty userID branch |
| Not member | orgID="org1", userID="user1" (not member) | CodePermissionDenied | Not authorized branch |
| RBAC error | orgID="org1", RBAC service fails | CodeInternal | Error handling branch |

**Test Scenarios for GetProject**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Project exists in org | userID, orgID="org1", projectID="p1" (member, belongs to org1) | ProjectDetail, nil | Happy path |
| User not member | userID (not member), orgID="org1", projectID="p1" | nil, PermissionDenied | RBAC failure |
| Project in different org | userID (member), orgID="org1", projectID="p2" (belongs to org2) | nil, NotFoundError | Wrong org branch |
| Project not found | userID (member), orgID="org1", projectID="nonexistent" | nil, NotFoundError | Not found branch |
| Invalid projectID | userID (member), orgID="org1", projectID="invalid" | nil, BadRequestError | Invalid ID branch |

---

## Step 8: Update Dashboard Repository with Organization Filtering

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Project schema with organizationId field

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Update all repository methods to filter by `organizationID`.

```go
package mongodb

// GetOverviewStats retrieves dashboard overview statistics for an organization.
func (r *MongoDashboardRepository) GetOverviewStats(ctx context.Context, organizationID string) (*repository.OverviewStats, error) {
	// Implementation outline:
	// 1. Convert organizationID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. Create organization filter: bson.M{mongoschema.ProjectOrganizationIDField: orgOID}.
	// 3. Get total usage from session records:
	//    a. First, get project IDs for this organization.
	//    b. Then aggregate usage from session_records where projectId is in those project IDs.
	//    - Alternatively, use $lookup from projects to records.
	// 4. Get project count with organization filter.
	// 5. Get session count (distinct session_ids for projects in this org).
	// 6. Get recent projects (top 5) with organization filter.
	// 7. Get recent sessions (top 5) for projects in this org.
	// 8. Return stats.
}

// ListProjects retrieves a paginated list of projects for an organization.
func (r *MongoDashboardRepository) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	// Implementation outline:
	// 1. Convert params.Query.OrganizationID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. Add $match stage at the beginning of pipeline:
	//    bson.M{"$match": bson.M{mongoschema.ProjectOrganizationIDField: orgOID}}.
	// 3. Continue with existing aggregation (lookup, project, facet).
	// 4. Return paginated result.
}

// GetProject retrieves detailed project information.
// Returns errutil.NotFound if project not found or does not belong to organization.
func (r *MongoDashboardRepository) GetProject(ctx context.Context, organizationID, projectID string) (*repository.ProjectDetail, error) {
	// Implementation outline:
	// 1. Convert organizationID and projectID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. Query with both conditions:
	//    bson.M{"_id": projectOID, mongoschema.ProjectOrganizationIDField: orgOID}.
	// 3. If not found (mongo.ErrNoDocuments):
	//    - Return nil, errutil.NotFound("project not found").
	// 4. Get session stats for this project.
	// 5. Return ProjectDetail.
}

// ListSessions retrieves paginated sessions for a project in an organization.
func (r *MongoDashboardRepository) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	// Implementation outline:
	// 1. Convert params.Query.OrganizationID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. If projectID is provided:
	//    a. Verify project belongs to organization by querying projects collection.
	//    b. If project not found or wrong org, return empty result (not error).
	// 3. Build aggregation pipeline filtering by projectId(s) belonging to org.
	// 4. Continue with existing aggregation (group, sort, paginate).
	// 5. Return paginated result.
}

// GetSession retrieves detailed session information with all records.
// Returns errutil.NotFound if session not found or its project does not belong to organization.
func (r *MongoDashboardRepository) GetSession(ctx context.Context, organizationID, sessionID string) (*repository.SessionDetail, error) {
	// Implementation outline:
	// 1. Convert organizationID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. Find session records by sessionID.
	// 3. If no records found:
	//    - Return nil, errutil.NotFound("session not found").
	// 4. From first record, get projectId.
	// 5. Verify project belongs to organization:
	//    a. Query projects collection with {_id: projectOID, organizationId: orgOID}.
	//    b. If not found:
	//       - Return nil, errutil.NotFound("session not found").
	// 6. Continue building SessionDetail from records.
	// 7. Return SessionDetail.
}

// Helper: getProjectIDsForOrganization returns all project IDs for an organization.
func (r *MongoDashboardRepository) getProjectIDsForOrganization(ctx context.Context, orgOID bson.ObjectID) ([]bson.ObjectID, error) {
	// Implementation outline:
	// 1. Query projects collection with {organizationId: orgOID}.
	// 2. Project only _id field.
	// 3. Collect all project IDs into slice.
	// 4. Return slice.
}
```

**Test Scenarios for GetProject**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid project in org | orgID="org1", projectID="p1" (org matches) | ProjectDetail, nil | Happy path |
| Project in different org | orgID="org1", projectID="p2" (org2's project) | nil, NotFound | Wrong org |
| Project not found | orgID="org1", projectID="nonexistent" | nil, NotFound | Not found |
| Invalid orgID format | orgID="invalid", projectID="p1" | nil, BadRequest | Invalid ID |
| Invalid projectID format | orgID="org1", projectID="invalid" | nil, BadRequest | Invalid ID |

---

## Step 9: Update Aggregation Handler (NO RBAC - delegate to Service)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`: Current handler

#### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`

**Description**:
Handler does NOT inject RBAC service. It extracts userID from context, passes organization_id to LogBatch, and delegates to service. RBAC validation happens in the Service layer.

```go
package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/bytedance/sonic"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	aggregationservice "github.com/team-attention/cops/api/internal/service/aggregation"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// AggregationGRPCHandler handles aggregation service gRPC endpoints.
// NOTE: NO rbacSvc field - RBAC is handled in Service layer
type AggregationGRPCHandler struct {
	svc    *aggregationservice.Service
	logger *slog.Logger
}

// NewAggregationGRPCHandler creates a new aggregation gRPC handler.
// NOTE: NO rbac.Service parameter - RBAC is handled in Service layer
func NewAggregationGRPCHandler(l *slog.Logger, svc *aggregationservice.Service) *AggregationGRPCHandler {
	// Implementation outline:
	// 1. Return new handler with service and logger.
	// 2. Logger bound with name "aggregation.grpc.connectrpc".
}

// SendLogs implements aggregationv1connect.AggregationServiceHandler.
func (h *AggregationGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. Get protobuf batch from request: pbBatch := req.Msg.GetBatch().
	// 3. Validate batch is provided:
	//    - If pbBatch is nil, return error response with success=false, error_message="batch is required".
	// 4. Parse JSONL lines into records.
	// 5. Create LogBatch with OrganizationID from protobuf batch (NOT from request):
	//    batch := &repository.LogBatch{
	//        Records:        records,
	//        ProjectID:      pbBatch.GetProjectId(),
	//        OrganizationID: pbBatch.GetOrganizationId(),  // From LogBatch message
	//    }
	// 6. Call service: h.svc.CollectLogs(ctx, userID, batch).
	//    - Service handles RBAC validation internally.
	//    - Service validates project belongs to organization.
	// 7. If error (from service returning connect.Error), return nil, error.
	// 8. Build and return response from result.
}
```

---

## Step 10: Update Aggregation Repository Port with OrganizationID

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go`: Repository interface

#### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go`

**Description**:
Update LogBatch to include OrganizationID field.

```go
package repository

import (
	"context"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of records from a daemon.
type LogBatch struct {
	// Records contains the parsed Record instances (all types).
	Records []shareddomain.Record
	// ProjectID is the project identifier for this batch.
	ProjectID string
	// OrganizationID is the organization identifier for RBAC validation.
	// Each batch is for one Project that belongs to this Organization.
	OrganizationID string
}

// SessionRecordRepositoryPort defines the interface for record persistence.
type SessionRecordRepositoryPort interface {
	// SaveBatch saves a batch of records to storage.
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	SaveBatch(ctx context.Context, batch *LogBatch) error
}
```

---

## Step 11: Update Aggregation Service with RBAC Injection

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go`: Current service

#### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go`

**Description**:
Update service to inject RBAC service and validate RBAC at the start of CollectLogs.

```go
package aggregation

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
)

// Service handles log collection operations.
// Injects RBAC service for authorization checks.
type Service struct {
	logger  *slog.Logger
	repo    repository.SessionRecordRepositoryPort
	rbacSvc *rbac.Service
}

// NewService creates a new aggregation service.
func NewService(l *slog.Logger, repo repository.SessionRecordRepositoryPort, rbacSvc *rbac.Service) *Service {
	// Implementation outline:
	// 1. Return Service with logger, repo, and rbacSvc.
	// 2. Logger bound with name "aggregation.service".
}

// CollectLogsResult contains the result of log collection.
type CollectLogsResult struct {
	Success        bool
	ProcessedCount int32
	ErrorMessage   string
}

// CollectLogs processes a batch of session records and saves them to storage.
// Validates RBAC at the start, then validates project belongs to organization.
func (s *Service) CollectLogs(ctx context.Context, userID string, batch *repository.LogBatch) (*CollectLogsResult, error) {
	// Implementation outline:
	// 1. Validate organizationID is provided:
	//    - If batch.OrganizationID is empty:
	//      - Return nil, connect.NewError(connect.CodeInvalidArgument, "organization_id is required").
	// 2. Validate userID is provided:
	//    - If userID is empty:
	//      - Return nil, connect.NewError(connect.CodeUnauthenticated, "user not authenticated").
	// 3. Check RBAC: Call s.rbacSvc.CanAccessOrganization(ctx, userID, batch.OrganizationID).
	// 4. If error:
	//    - Log error.
	//    - Return nil, connect.NewError(connect.CodeInternal, "failed to check access").
	// 5. If not authorized:
	//    - Log info for security audit with userID and organizationID.
	//    - Return nil, connect.NewError(connect.CodePermissionDenied, "access denied to organization").
	// 6. If no records, return success result with 0 processed.
	// 7. Log "collecting log batch" with projectId, organizationId, and recordCount.
	// 8. Call s.repo.SaveBatch(ctx, batch).
	//    - Repository validates project belongs to organization.
	//    - Returns errutil.NotFound if project not in org.
	// 9. If error:
	//    a. If errutil.IsNotFound(err):
	//       - Log warning "project not found in organization".
	//       - Return &CollectLogsResult{Success: false, ErrorMessage: "project not found in organization"}, nil.
	//    b. Else:
	//       - Log error "failed to save log batch".
	//       - Return &CollectLogsResult{Success: false, ErrorMessage: err.Error()}, nil.
	// 10. Return success result with processed count.
}
```

**Test Scenarios for CollectLogs**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid request | userID, orgID, batch (member, project in org) | Success result | Happy path |
| Missing orgID | userID, orgID="", batch | nil, CodeInvalidArgument | Validation branch |
| Missing userID | userID="", orgID, batch | nil, CodeUnauthenticated | Auth branch |
| User not member | userID (not member), orgID, batch | nil, CodePermissionDenied | RBAC denied branch |
| Project not in org | userID (member), orgID, batch (wrong project) | Result{Success: false, ErrorMessage: "project not found..."} | Project validation |
| Empty batch | userID (member), orgID, batch with 0 records | Success result with 0 processed | Empty records |

---

## Step 12: Update Aggregation Repository with Organization Validation

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`

**Description**:
Update SaveBatch to validate project belongs to organization before saving.

```go
package mongodb

// MongoSessionRecordRepository implements SessionRecordRepositoryPort using MongoDB.
type MongoSessionRecordRepository struct {
	logger       *slog.Logger
	recordsColl  *mongo.Collection
	projectsColl *mongo.Collection
}

// NewMongoSessionRecordRepository creates a new MongoDB session record repository adapter.
func NewMongoSessionRecordRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRecordRepository {
	// Implementation outline:
	// 1. Return repository with logger, records collection, and projects collection.
	// 2. Add projects collection for validation.
}

// SaveBatch saves a batch of records to storage.
// Validates project belongs to organization before saving.
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
	// Implementation outline:
	// 1. Convert batch.ProjectID and batch.OrganizationID to bson.ObjectID.
	//    - If invalid, return errutil.BadRequest.
	// 2. Validate project belongs to organization:
	//    a. Query projects collection: {_id: projectOID, organizationId: orgOID}.
	//    b. If not found (mongo.ErrNoDocuments):
	//       - Return errutil.NotFound("project not found in organization").
	// 3. Prepare documents for insertion (existing logic).
	// 4. Insert records into collection (existing logic).
	// 5. Return nil on success.
}
```

**Test Scenarios for SaveBatch**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid batch | project in org, valid records | nil | Happy path |
| Project not in org | project belongs to different org | NotFoundError | Wrong org |
| Project not found | projectID doesn't exist | NotFoundError | Not found |
| Invalid projectID | invalid ObjectID format | BadRequestError | Invalid ID |
| Empty batch | no records | nil (or skip) | Empty records |

---

## Step 13: Update FX Module Registrations

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`: Dashboard module
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`: Aggregation module

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`

**Description**:
Service constructor now requires RBAC service. Handler does NOT require RBAC service. FX will automatically inject RBAC into Service.

```go
package container

func newDashboardModule() fx.Option {
	return fx.Module("dashboard",
		// MongoDB Repository Adapter
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoDashboardRepository,
				fx.As(new(repository.DashboardRepositoryPort)),
			),
		),

		// Service (now requires *rbac.Service, injected by fx)
		fx.Provide(dashboard.NewService),

		// gRPC Handler (does NOT require rbac.Service)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewDashboardGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
```

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`

**Description**:
Service constructor now requires RBAC service. Handler does NOT require RBAC service. FX will automatically inject RBAC into Service.

```go
package container

func newAggregationModule() fx.Option {
	return fx.Module("aggregation",
		// Repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoSessionRecordRepository,
				fx.As(new(repository.SessionRecordRepositoryPort)),
			),
		),

		// Service (now requires *rbac.Service, injected by fx)
		fx.Provide(aggregationservice.NewService),

		// gRPC Handler (does NOT require rbac.Service)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAggregationGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
```

---

## Summary of Files to Modify

| File | Action | Description |
| :--- | :----- | :---------- |
| `idl/protobuf/aggregation/v1/aggregation.proto` | Modify | Add `organization_id` to `LogBatch` message (NOT SendLogsReq) |
| `idl/protobuf/dashboard/v1/dashboard.proto` | Modify | Add `organization_id` to all request messages |
| `api/internal/platform/interceptor/auth_interceptor.go` | Create | New auth interceptor for ConnectRPC |
| `api/cmd/internal/container/register_connectrpc.go` | Modify | Register auth interceptor |
| `api/internal/service/core/rbac/rbac_service.go` | Modify | DELETE `CanAccess`, REMOVE `projectRepo`, ADD `CanAccessOrganization` |
| `api/cmd/internal/container/module_rbac.go` | Modify | Remove ProjectRepository provider |
| `api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go` | Modify | Remove RBAC, pass userID to service |
| `api/internal/service/dashboard/dashboard_service.go` | Modify | Inject RBAC, validate at method start, update signatures |
| `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go` | Modify | Add organizationID to interface methods |
| `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` | Modify | Add organization filtering to all queries |
| `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` | Modify | Remove RBAC, pass userID and orgID to service |
| `api/internal/service/aggregation/aggregation_service.go` | Modify | Inject RBAC, validate at method start, update signature |
| `api/internal/service/aggregation/outbound/repository/port.go` | Modify | Add OrganizationID to LogBatch |
| `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | Modify | Add project-org validation |
| `api/cmd/internal/container/module_dashboard.go` | Verify | No changes needed (fx auto-injects RBAC into Service) |
| `api/cmd/internal/container/module_aggregation.go` | Verify | No changes needed (fx auto-injects RBAC into Service) |

---

## Execution Order

1. **Step 1**: Update protobuf definitions and regenerate code
2. **Step 2**: Create auth interceptor
3. **Step 3**: Register auth interceptor in container
4. **Step 4**: Refactor RBAC service (DELETE CanAccess, ADD CanAccessOrganization, REMOVE projectRepo)
5. **Step 5**: Update RBAC module registration (remove projectRepo provider)
6. **Step 6**: Update Dashboard handler (remove RBAC, pass userID to service)
7. **Step 7**: Update Dashboard service (inject RBAC, validate at method start)
8. **Step 8**: Update Dashboard repository (add organization filtering)
9. **Step 9**: Update Aggregation handler (remove RBAC, pass userID and orgID to service)
10. **Step 10**: Update Aggregation repository port (add OrganizationID to LogBatch)
11. **Step 11**: Update Aggregation service (inject RBAC, validate at method start)
12. **Step 12**: Update Aggregation repository (add project-org validation)
13. **Step 13**: Verify FX module registrations
