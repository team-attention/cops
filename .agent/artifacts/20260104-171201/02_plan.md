# Implementation Plan: Fix Google OAuth Authentication Bug

## Overview

The API server's global authentication interceptor currently blocks ALL ConnectRPC endpoints, including authentication endpoints that should be publicly accessible. This creates a chicken-and-egg problem where users cannot call `GoogleAuth` to obtain tokens because the interceptor requires tokens to access any endpoint.

The fix involves splitting the handler registration into two distinct interfaces:
- `PublicConnectHandler` - for handlers that do not require authentication (registered WITHOUT auth interceptor)
- `PrivateConnectHandler` - for handlers that require authentication (registered WITH auth interceptor)

This approach is cleaner than a whitelist because:
1. Authentication policy is explicit at the handler registration level
2. No need for string-based procedure matching
3. The Auth handler can be split: public methods in one handler, private methods in another

## Package Changes

None required. The implementation uses only existing packages already imported.

## Implementation Steps

### Step 1: Define New Handler Interfaces in register_connectrpc.go

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container registration patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md`: Handler interface patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`

**Description**:
Replace the single `ConnectHandler` interface with two interfaces: `PublicConnectHandler` and `PrivateConnectHandler`. Update the registration function to collect handlers from two separate groups and register them with different interceptor options.

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

// PublicConnectHandler interface for ConnectRPC handlers that do not require authentication.
type PublicConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

// PrivateConnectHandler interface for ConnectRPC handlers that require authentication.
type PrivateConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

type connectRPCServerParams struct {
	fx.In

	Lifecycle       fx.Lifecycle
	Logger          *slog.Logger
	Config          *config.Config
	App             *fiber.App
	PublicHandlers  []PublicConnectHandler  `group:"public_connect_handlers"`
	PrivateHandlers []PrivateConnectHandler `group:"private_connect_handlers"`
}

func registerConnectRPCServer(params connectRPCServerParams) {
	// Implementation outline:
	// 1. Create child logger with component name "server.connectrpc".
	//
	// 2. Create JWT config from params.Config.JWT.
	//
	// 3. Create auth interceptor using interceptor.NewAuthInterceptor().
	//
	// 4. Create handler options with interceptor for private handlers.
	//    - privateOpts := []connect.HandlerOption{connect.WithInterceptors(authInterceptor)}
	//
	// 5. Register public handlers WITHOUT any interceptor options.
	//    - for _, handler := range params.PublicHandlers
	//    - path, h := handler.GetHandler() // No opts passed
	//    - params.App.All(path+"*", adaptor.HTTPHandler(h))
	//
	// 6. Register private handlers WITH auth interceptor options.
	//    - for _, handler := range params.PrivateHandlers
	//    - path, h := handler.GetHandler(privateOpts...)
	//    - params.App.All(path+"*", adaptor.HTTPHandler(h))
	//
	// 7. Lifecycle hooks for server start/stop (unchanged from current implementation).
	//    - OnStart: Start HTTP server on configured port
	//    - OnStop: Graceful shutdown with context
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Public handler registration | PublicConnectHandler | Registered without auth interceptor | Public handler loop |
| Private handler registration | PrivateConnectHandler | Registered with auth interceptor | Private handler loop |
| Mixed handlers | Both public and private handlers | Both registered correctly with appropriate options | Both loops |
| Empty public handlers | No public handlers | No error, only private handlers registered | Empty slice handling |
| Empty private handlers | No private handlers | No error, only public handlers registered | Empty slice handling |

---

### Step 2: Split Auth Handler into Public and Private Handlers

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md`: Handler file structure
- `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: Current implementation

**Description**:
The current `AuthGRPCHandler` handles both public methods (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) and a private method (DeviceCodeApprove). We need to split this into two handlers:
1. `AuthPublicGRPCHandler` - Handles public authentication methods
2. `AuthPrivateGRPCHandler` - Handles DeviceCodeApprove (requires authentication)

#### `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**:
Modify the existing handler file to define two handler structs. Both handlers share the same service and configuration dependencies but implement different subsets of the AuthService methods. The private handler will use `interceptor.UserIDFromContext()` to get the authenticated user ID.

```go
package connectrpc

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/auth"
	"github.com/team-attention/cops/shared/domain"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

// AuthPublicGRPCHandler handles public auth gRPC endpoints (no authentication required).
type AuthPublicGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
	cfg    *config.Config
}

// NewAuthPublicGRPCHandler creates a new public auth gRPC handler.
func NewAuthPublicGRPCHandler(l *slog.Logger, svc *auth.Service, cfg *config.Config) *AuthPublicGRPCHandler {
	// Implementation outline:
	// 1. Create handler struct with service and config.
	// 2. Bind logger with name "auth.grpc.connectrpc.public".
	// 3. Return handler pointer.
}

// GetHandler implements PublicConnectHandler interface.
func (h *AuthPublicGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// Implementation outline:
	// 1. Return authv1connect.NewAuthServiceHandler(h, opts...)
}

// GoogleAuth handles Google OAuth authentication.
func (h *AuthPublicGRPCHandler) GoogleAuth(
	ctx context.Context,
	req *connect.Request[authv1.GoogleAuthReq],
) (*connect.Response[authv1.GoogleAuthRes], error) {
	// Implementation outline:
	// 1. Parse request parameters into auth.GoogleAuthParams.
	// 2. Call h.svc.GoogleAuth(ctx, params).
	// 3. If error, log and return connect.CodeInternal error.
	// 4. Convert result to authv1.GoogleAuthRes with TokenPair.
	// 5. Return connect.NewResponse(res).
}

// RefreshToken handles token refresh.
func (h *AuthPublicGRPCHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[authv1.RefreshTokenReq],
) (*connect.Response[authv1.RefreshTokenRes], error) {
	// Implementation outline:
	// 1. Extract refresh token from request.
	// 2. Call h.svc.RefreshToken(ctx, refreshToken).
	// 3. If error, log and return connect.CodeUnauthenticated error.
	// 4. Convert result to authv1.RefreshTokenRes with TokenPair.
	// 5. Return connect.NewResponse(res).
}

// DeviceCode initiates device code flow.
func (h *AuthPublicGRPCHandler) DeviceCode(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeReq],
) (*connect.Response[authv1.DeviceCodeRes], error) {
	// Implementation outline:
	// 1. Call h.svc.DeviceCode(ctx).
	// 2. If error, log and return connect.CodeInternal error.
	// 3. Convert result to authv1.DeviceCodeRes.
	// 4. Return connect.NewResponse(res).
}

// DevicePoll polls for device code approval status.
func (h *AuthPublicGRPCHandler) DevicePoll(
	ctx context.Context,
	req *connect.Request[authv1.DevicePollReq],
) (*connect.Response[authv1.DevicePollRes], error) {
	// Implementation outline:
	// 1. Extract device code from request.
	// 2. Call h.svc.DevicePoll(ctx, deviceCode).
	// 3. If error, log and return connect.CodeInternal error.
	// 4. Build response with pending status and tokens if available.
	// 5. Return connect.NewResponse(res).
}

// DeviceCodeApprove returns unimplemented for public handler.
// This method is handled by AuthPrivateGRPCHandler.
func (h *AuthPublicGRPCHandler) DeviceCodeApprove(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
	// Implementation outline:
	// 1. Return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("use authenticated endpoint")).
	//    This ensures that if someone calls the public path, they get a clear error.
}

// AuthPrivateGRPCHandler handles private auth gRPC endpoints (authentication required).
type AuthPrivateGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
}

// NewAuthPrivateGRPCHandler creates a new private auth gRPC handler.
func NewAuthPrivateGRPCHandler(l *slog.Logger, svc *auth.Service) *AuthPrivateGRPCHandler {
	// Implementation outline:
	// 1. Create handler struct with service.
	// 2. Bind logger with name "auth.grpc.connectrpc.private".
	// 3. Return handler pointer.
}

// GetHandler implements PrivateConnectHandler interface.
func (h *AuthPrivateGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// Implementation outline:
	// 1. Return authv1connect.NewAuthServiceHandler(h, opts...)
}

// GoogleAuth returns unimplemented for private handler.
// This method is handled by AuthPublicGRPCHandler.
func (h *AuthPrivateGRPCHandler) GoogleAuth(
	ctx context.Context,
	req *connect.Request[authv1.GoogleAuthReq],
) (*connect.Response[authv1.GoogleAuthRes], error) {
	// Implementation outline:
	// 1. Return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("use public endpoint")).
}

// RefreshToken returns unimplemented for private handler.
func (h *AuthPrivateGRPCHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[authv1.RefreshTokenReq],
) (*connect.Response[authv1.RefreshTokenRes], error) {
	// Implementation outline:
	// 1. Return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("use public endpoint")).
}

// DeviceCode returns unimplemented for private handler.
func (h *AuthPrivateGRPCHandler) DeviceCode(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeReq],
) (*connect.Response[authv1.DeviceCodeRes], error) {
	// Implementation outline:
	// 1. Return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("use public endpoint")).
}

// DevicePoll returns unimplemented for private handler.
func (h *AuthPrivateGRPCHandler) DevicePoll(
	ctx context.Context,
	req *connect.Request[authv1.DevicePollReq],
) (*connect.Response[authv1.DevicePollRes], error) {
	// Implementation outline:
	// 1. Return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("use public endpoint")).
}

// DeviceCodeApprove approves a device code (requires authentication).
func (h *AuthPrivateGRPCHandler) DeviceCodeApprove(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
	// Implementation outline:
	// 1. Get userID from context using interceptor.UserIDFromContext(ctx).
	// 2. If userID is empty, return connect.CodeUnauthenticated error (should not happen due to interceptor).
	// 3. Create auth.DeviceCodeApproveParams with UserCode and UserID.
	// 4. Call h.svc.DeviceCodeApprove(ctx, params).
	// 5. If error, log and return appropriate error code:
	//    - "not found" -> connect.CodeNotFound
	//    - "expired" -> connect.CodeDeadlineExceeded
	//    - "already approved" -> connect.CodeAlreadyExists
	//    - otherwise -> connect.CodeInternal
	// 6. Return success response with Success=true and Message.
}

// Compile-time interface verification.
var _ authv1connect.AuthServiceHandler = (*AuthPublicGRPCHandler)(nil)
var _ authv1connect.AuthServiceHandler = (*AuthPrivateGRPCHandler)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Public: GoogleAuth success | Valid authorization code | TokenPair returned | GoogleAuth happy path |
| Public: GoogleAuth failure | Invalid code | CodeInternal error | GoogleAuth error branch |
| Public: RefreshToken success | Valid refresh token | New TokenPair returned | RefreshToken happy path |
| Public: RefreshToken failure | Invalid/expired token | CodeUnauthenticated error | RefreshToken error branch |
| Public: DeviceCode success | Empty request | DeviceCode info returned | DeviceCode happy path |
| Public: DeviceCode failure | Service error | CodeInternal error | DeviceCode error branch |
| Public: DevicePoll pending | Valid device code, not approved | Pending=true, no tokens | DevicePoll pending branch |
| Public: DevicePoll approved | Valid device code, approved | Pending=false, tokens returned | DevicePoll approved branch |
| Public: DeviceCodeApprove call | Any request | CodeUnimplemented error | Unimplemented branch |
| Private: DeviceCodeApprove success | Valid userCode, authenticated user | Success=true | DeviceCodeApprove happy path |
| Private: DeviceCodeApprove not found | Invalid userCode | CodeNotFound error | Not found error branch |
| Private: DeviceCodeApprove expired | Expired userCode | CodeDeadlineExceeded error | Expired error branch |
| Private: DeviceCodeApprove already approved | Already approved code | CodeAlreadyExists error | Already approved branch |
| Private: GoogleAuth call | Any request | CodeUnimplemented error | Unimplemented branch |

---

### Step 3: Update Auth Module to Register Both Handlers

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Module registration patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`

**Description**:
Update the auth module to register both `AuthPublicGRPCHandler` and `AuthPrivateGRPCHandler` with their respective group tags.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/auth"
	"github.com/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth/google"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// OAuth adapter
		fx.Provide(
			fx.Annotate(
				google.NewGoogleOAuthAdapter,
				fx.As(new(oauth.GoogleOAuthPort)),
			),
		),

		// User repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoUserRepository,
				fx.As(new(repository.UserRepositoryPort)),
			),
		),

		// Device code repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoDeviceCodeRepository,
				fx.As(new(repository.DeviceCodeRepositoryPort)),
			),
		),

		// Service
		fx.Provide(auth.NewService),

		// Public ConnectRPC handler (no auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthPublicGRPCHandler,
				fx.As(new(PublicConnectHandler)),
				fx.ResultTags(`group:"public_connect_handlers"`),
			),
		),

		// Private ConnectRPC handler (auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthPrivateGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Module provides public handler | fx.New with auth module | PublicConnectHandler in group | Public handler registration |
| Module provides private handler | fx.New with auth module | PrivateConnectHandler in group | Private handler registration |

---

### Step 4: Update Health Module to Use Public Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_health.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_health.go`

**Description**:
Update health module to register as a public handler (health checks should not require authentication).

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/health"
	"github.com/team-attention/cops/api/internal/service/health/inbound/grpc/connectrpc"
)

func newHealthModule() fx.Option {
	return fx.Module("health",
		// Service
		fx.Provide(health.NewService),

		// gRPC Handler (public - no auth required for health checks)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewHealthGRPCHandler,
				fx.As(new(PublicConnectHandler)),
				fx.ResultTags(`group:"public_connect_handlers"`),
			),
		),
	)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Health check without auth | No authorization header | Serving status returned | Public access |

---

### Step 5: Update Dashboard Module to Use Private Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`

**Description**:
Update dashboard module to register as a private handler (requires authentication).

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/dashboard"
	"github.com/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb"
)

func newDashboardModule() fx.Option {
	return fx.Module("dashboard",
		// MongoDB Repository Adapter
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoDashboardRepository,
				fx.As(new(repository.DashboardRepositoryPort)),
			),
		),

		// Service
		fx.Provide(dashboard.NewService),

		// gRPC Handler (private - requires auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewDashboardGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Dashboard access without auth | No authorization header | CodeUnauthenticated error | Auth interceptor |
| Dashboard access with valid auth | Valid Bearer token | Dashboard data returned | Authenticated access |

---

### Step 6: Update Project Module to Use Private Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`

**Description**:
Update project module to register as a private handler.

```go
// Change only the fx.Annotate for the gRPC handler:
fx.Provide(
	fx.Annotate(
		connectrpc.NewProjectGRPCHandler,
		fx.As(new(PrivateConnectHandler)),
		fx.ResultTags(`group:"private_connect_handlers"`),
	),
),
```

---

### Step 7: Update Aggregation Module to Use Private Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`

**Description**:
Update aggregation module to register as a private handler.

```go
// Change only the fx.Annotate for the gRPC handler:
fx.Provide(
	fx.Annotate(
		connectrpc.NewAggregationGRPCHandler,
		fx.As(new(PrivateConnectHandler)),
		fx.ResultTags(`group:"private_connect_handlers"`),
	),
),
```

---

### Step 8: Update User Module to Use Private Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`

**Description**:
Update user module to register as a private handler.

```go
// Change only the fx.Annotate for the gRPC handler:
fx.Provide(
	fx.Annotate(
		connectrpc.NewUserGRPCHandler,
		fx.As(new(PrivateConnectHandler)),
		fx.ResultTags(`group:"private_connect_handlers"`),
	),
),
```

---

### Step 9: Remove Old ConnectHandler Interface

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`: Current implementation

**Description**:
After updating all modules, remove the old `ConnectHandler` interface from `register_connectrpc.go` since it is no longer used. This was replaced by `PublicConnectHandler` and `PrivateConnectHandler` in Step 1.

---

## Summary of Handler Classification

| Module | Handler Type | Reason |
|:-------|:-------------|:-------|
| Auth (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) | Public | Authentication endpoints - users don't have tokens yet |
| Auth (DeviceCodeApprove) | Private | Requires authenticated user to approve device |
| Health | Public | Health checks should be accessible without auth |
| Dashboard | Private | User-specific data requires authentication |
| Project | Private | Project management requires authentication |
| Aggregation | Private | Data collection requires authentication |
| User | Private | User management requires authentication |

## Manual Testing Plan

After implementation, perform these tests:

1. **Google OAuth Flow**:
   - Navigate to `/auth` page
   - Click "Sign in with Google"
   - Complete Google OAuth
   - Verify redirect to dashboard without "missing authorization header" error

2. **Token Refresh Flow**:
   - Login successfully
   - Wait for token to expire (or manually expire)
   - Perform an authenticated action
   - Verify automatic token refresh works

3. **Device Code Flow**:
   - Use CLI to initiate device code flow (`cops login`)
   - Verify user code is returned without auth error
   - Poll for device status without auth error
   - Approve device from web (requires auth)
   - Verify CLI receives tokens

4. **Health Check**:
   - Call `/health.v1.HealthService/Check` without auth header
   - Verify serving status is returned

5. **Protected Endpoints**:
   - Without token: Verify 401 error for `/dashboard.v1.DashboardService/GetOverview`
   - With valid token: Verify success for all dashboard/project/session endpoints

## Quality Checklist

- [x] Every function has a concrete signature
- [x] Detailed algorithm explanation included as comments in function body
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (using existing packages only)
- [x] Execution order is clear (Steps 1-9 in sequence)
