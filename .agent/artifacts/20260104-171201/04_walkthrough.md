# Development Walkthrough

## Summary
Fixed Google OAuth login failure caused by the API server's global authentication interceptor blocking all ConnectRPC endpoints, including public authentication endpoints. The solution involved splitting handler registration into two distinct interfaces (PublicConnectHandler and PrivateConnectHandler), allowing authentication endpoints to bypass the auth interceptor while maintaining protection for all other endpoints.

## Code Overview

### Problem Analysis

The bug manifested when users attempted to sign in via Google OAuth. After completing Google authentication and being redirected back to the application, the frontend would send the authorization code to the backend's `GoogleAuth` endpoint. However, this request would fail with a "missing authorization header" error because:

1. The global auth interceptor (applied to ALL ConnectRPC handlers) required an Authorization header
2. Users calling the `GoogleAuth` endpoint to LOGIN couldn't have a token yet (chicken-and-egg problem)
3. This same issue affected other public auth endpoints: `RefreshToken`, `DeviceCode`, and `DevicePoll`

### Solution Architecture

Instead of implementing an endpoint whitelist within the auth interceptor, the solution separates handlers at the registration level:

- **PublicConnectHandler**: Handlers registered WITHOUT auth interceptor
- **PrivateConnectHandler**: Handlers registered WITH auth interceptor

This approach provides:
- Clear, explicit authentication policy at the module level
- No string-based endpoint matching logic
- Type-safe handler separation
- Easier to understand and maintain

### New Components

#### `PublicConnectHandler` Interface
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` (lines 19-22)
- **Purpose**: Interface for ConnectRPC handlers that do not require authentication
- **Key Method**:
  - `GetHandler(opts ...connect.HandlerOption) (string, http.Handler)`: Returns handler path and HTTP handler

#### `PrivateConnectHandler` Interface
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` (lines 24-27)
- **Purpose**: Interface for ConnectRPC handlers that require authentication
- **Key Method**:
  - `GetHandler(opts ...connect.HandlerOption) (string, http.Handler)`: Returns handler path and HTTP handler

#### `AuthPublicGRPCHandler`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` (lines 20-153)
- **Purpose**: Handles public authentication endpoints that don't require an Authorization header
- **Key Methods**:
  - `GoogleAuth()`: Exchanges Google OAuth authorization code for JWT tokens
  - `RefreshToken()`: Exchanges refresh token for new access/refresh token pair
  - `DeviceCode()`: Initiates device code flow (returns user code for CLI authentication)
  - `DevicePoll()`: Polls device code approval status
  - `DeviceCodeApprove()`: Returns CodeUnimplemented (redirects to private handler)
- **Implementation Details**:
  - Accepts requests without Authorization header
  - Performs authentication logic (validates with Google, generates tokens, etc.)
  - Returns appropriate error codes (CodeInternal, CodeUnauthenticated)

#### `AuthPrivateGRPCHandler`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` (lines 155-253)
- **Purpose**: Handles authentication endpoints that require an authenticated user
- **Key Methods**:
  - `DeviceCodeApprove()`: Approves a device code (requires authenticated user from context)
  - `GoogleAuth()`, `RefreshToken()`, `DeviceCode()`, `DevicePoll()`: Return CodeUnimplemented (redirect to public handler)
- **Implementation Details**:
  - Requires Authorization header (enforced by auth interceptor)
  - Extracts user ID from context using `interceptor.UserIDFromContext(ctx)`
  - Returns specific error codes based on failure reason (CodeNotFound, CodeDeadlineExceeded, CodeAlreadyExists)

### Modified Components

#### `registerConnectRPCServer()`
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` (lines 40-88)
- **Changes**:
  - Changed from single `ConnectHandler` interface to two separate handler groups
  - Updated params struct to collect `PublicHandlers` and `PrivateHandlers` via group tags
  - Public handlers registered WITHOUT auth interceptor options (line 59)
  - Private handlers registered WITH auth interceptor options (line 65)
  - Maintains existing lifecycle hooks for HTTP server start/stop

#### Auth Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`
- **Changes**:
  - Removed single handler registration
  - Added `AuthPublicGRPCHandler` registration with `group:"public_connect_handlers"` tag
  - Added `AuthPrivateGRPCHandler` registration with `group:"private_connect_handlers"` tag
  - Both handlers receive the same dependencies (service, config, logger)

#### Health Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_health.go`
- **Changes**: Updated handler registration to use `PublicConnectHandler` interface and `group:"public_connect_handlers"` tag
- **Reason**: Health checks should be accessible without authentication

#### Dashboard Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`
- **Changes**: Updated handler registration to use `PrivateConnectHandler` interface and `group:"private_connect_handlers"` tag
- **Reason**: Dashboard data requires authenticated user

#### Project Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`
- **Changes**: Updated handler registration to use `PrivateConnectHandler` interface and `group:"private_connect_handlers"` tag
- **Reason**: Project operations require authenticated user

#### Aggregation Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`
- **Changes**: Updated handler registration to use `PrivateConnectHandler` interface and `group:"private_connect_handlers"` tag
- **Reason**: Data aggregation requires authenticated user

#### User Module Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`
- **Changes**: Updated handler registration to use `PrivateConnectHandler` interface and `group:"private_connect_handlers"` tag
- **Reason**: User management requires authenticated user

## Testing

### Build Verification
The implementation was verified with the following commands:

```bash
# Build all Go modules
go build ./...  # Result: PASS (no compilation errors)

# Verify specific api package
cd api && go build ./...  # Result: PASS
```

### Manual Testing Required

To fully verify the fix, the following manual tests should be performed:

1. **Google OAuth Flow**:
   - Navigate to `/auth` page in web browser
   - Click "Sign in with Google"
   - Complete Google OAuth authentication
   - Verify successful redirect to dashboard WITHOUT "missing authorization header" error
   - Verify JWT tokens are stored in browser

2. **Token Refresh Flow**:
   - Login successfully via Google OAuth
   - Wait for access token to expire (or manually expire)
   - Perform an authenticated action (e.g., load dashboard)
   - Verify automatic token refresh works without errors

3. **Device Code Flow** (CLI authentication):
   - Run `cops login` command
   - Verify device code and user code returned WITHOUT auth error
   - Navigate to verification URL in browser
   - Login and approve device code (requires authentication)
   - Verify CLI receives tokens successfully

4. **Health Check** (public endpoint):
   - Call `POST /health.v1.HealthService/Check` WITHOUT Authorization header
   - Verify response: `{"status": "SERVING"}` (no 401 error)

5. **Protected Endpoints** (private endpoints):
   - Call `POST /dashboard.v1.DashboardService/GetOverview` WITHOUT Authorization header
   - Verify response: 401 Unauthorized with "missing authorization header" error
   - Call same endpoint WITH valid Bearer token
   - Verify response: Dashboard data returned successfully

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Global auth interceptor blocking all endpoints including public auth endpoints | Split handler registration into PublicConnectHandler and PrivateConnectHandler interfaces |
| Need to maintain both public and private methods in AuthService | Split AuthGRPCHandler into AuthPublicGRPCHandler and AuthPrivateGRPCHandler, both implementing the same service interface |
| Compile-time verification needed for split handlers | Added interface verification: `var _ authv1connect.AuthServiceHandler = (*AuthPublicGRPCHandler)(nil)` |
| DeviceCodeApprove needs authenticated user context | Placed in AuthPrivateGRPCHandler which goes through auth interceptor, extracts user ID from context |
| Health checks were blocked by auth interceptor | Moved HealthGRPCHandler to public handlers group |
| Need clear separation of authentication requirements per module | Updated all module registrations to explicitly declare Public vs Private handler type |

## Implementation Patterns Followed

### Hexagonal Architecture
- **Inbound adapters**: `/api/internal/service/auth/inbound/grpc/connectrpc/`
- **Service layer**: `/api/internal/service/auth/`
- **Outbound adapters**: OAuth adapters, repository adapters (not modified in this fix)
- **Dependency injection**: `go.uber.org/fx` with group tags

### Port-Adapter Pattern
- **Port**: `PublicConnectHandler` and `PrivateConnectHandler` interfaces
- **Adapters**: Concrete handler implementations (AuthPublicGRPCHandler, AuthPrivateGRPCHandler, etc.)
- **Registration**: `fx.Annotate` with `fx.As` for interface casting and `fx.ResultTags` for group collection

### Logger Binding Conventions
- Public handler: `logger: l.With(slog.String("name", "auth.grpc.connectrpc.public"))`
- Private handler: `logger: l.With(slog.String("name", "auth.grpc.connectrpc.private"))`

### Error Handling
- Uses ConnectRPC error codes: `connect.CodeInternal`, `connect.CodeUnauthenticated`, `connect.CodeNotFound`, etc.
- Structured logging with context: `slog.String()`, `slog.Any()`

## Related Tickets
- Bug Fix: Google OAuth login failure with "missing authorization header" error
