# Implementation Plan: Split AuthService into Public and Private Services

## Overview

This plan resolves the DeviceCodeApprove routing issue by splitting the single `AuthService` proto definition into two separate services:
- `AuthService` (public): Contains unauthenticated endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll)
- `AuthPrivateService` (private): Contains authenticated endpoints (DeviceCodeApprove)

This approach leverages ConnectRPC's natural path generation where each service registers to its own unique path:
- `/auth.v1.AuthService/` for public endpoints
- `/auth.v1.AuthPrivateService/` for private endpoints

This eliminates the path conflict that causes all requests to route to the public handler.

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | N/A | N/A | No external package changes required. Only protobuf regeneration needed. |

---

## Step 1: Update Protobuf Service Definition

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Naming conventions for proto files
- `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`: Reference for service definition pattern

### `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto`

**Description**:
Split the existing `AuthService` into two services. Keep all message types unchanged. Move `DeviceCodeApprove` RPC to a new `AuthPrivateService`.

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1;authv1";

// TokenPair contains access and refresh tokens.
message TokenPair {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_at = 3;
}

// GoogleAuthReq contains Google OAuth authorization code.
message GoogleAuthReq {
  string authorization_code = 1;
  string redirect_uri = 2;
}

// GoogleAuthRes contains authentication result.
message GoogleAuthRes {
  TokenPair tokens = 1;
}

// DeviceCodeReq initiates device flow authentication.
message DeviceCodeReq {}

// DeviceCodeRes contains device code for CLI authentication.
message DeviceCodeRes {
  string device_code = 1;
  string user_code = 2;
  string verification_url = 3;
  int32 expires_in = 4;
  int32 interval = 5;
}

// DevicePollReq polls for device authentication completion.
message DevicePollReq {
  string device_code = 1;
}

// DevicePollRes contains poll result.
message DevicePollRes {
  bool pending = 1;
  TokenPair tokens = 2;
}

// RefreshTokenReq contains refresh token for token renewal.
message RefreshTokenReq {
  string refresh_token = 1;
}

// RefreshTokenRes contains new token pair.
message RefreshTokenRes {
  TokenPair tokens = 1;
}

// DeviceCodeApproveReq contains the device code to approve.
message DeviceCodeApproveReq {
  string user_code = 1;
}

// DeviceCodeApproveRes contains the approval result.
message DeviceCodeApproveRes {
  bool success = 1;
  string message = 2;
}

// AuthService handles public authentication operations (no auth required).
service AuthService {
  // GoogleAuth exchanges Google OAuth code for JWT tokens (web flow).
  rpc GoogleAuth(GoogleAuthReq) returns (GoogleAuthRes);

  // DeviceCode initiates device flow for CLI authentication.
  rpc DeviceCode(DeviceCodeReq) returns (DeviceCodeRes);

  // DevicePoll polls for device authentication completion.
  rpc DevicePoll(DevicePollReq) returns (DevicePollRes);

  // RefreshToken exchanges refresh token for new token pair.
  rpc RefreshToken(RefreshTokenReq) returns (RefreshTokenRes);
}

// AuthPrivateService handles authenticated operations (JWT required).
service AuthPrivateService {
  // DeviceCodeApprove approves a device code from the web application.
  // Requires authenticated user (JWT in Authorization header).
  rpc DeviceCodeApprove(DeviceCodeApproveReq) returns (DeviceCodeApproveRes);
}
```

**Test Scenarios**: N/A (Proto definition file)

---

## Step 2: Regenerate Code

**Files to Read**:
- `/Users/jayce/team-attention/cops/idl/protobuf/buf.gen.yaml`: Code generation configuration

**Description**:
Run buf generate to regenerate all code. This will:
1. Generate Go types and ConnectRPC handlers in `shared/gen/grpcstub/auth/v1/`
2. Generate TypeScript types and Connect-Query hooks in `web/src/gen/grpcstub/auth/v1/`

**Command**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Expected Generated Files**:

Go (Backend):
- `shared/gen/grpcstub/auth/v1/auth.pb.go` - Message types (unchanged structure)
- `shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go` - Will now contain:
  - `AuthServiceHandler` interface (GoogleAuth, DeviceCode, DevicePoll, RefreshToken)
  - `AuthPrivateServiceHandler` interface (DeviceCodeApprove)
  - `NewAuthServiceHandler()` - Returns path `/auth.v1.AuthService/`
  - `NewAuthPrivateServiceHandler()` - Returns path `/auth.v1.AuthPrivateService/`

TypeScript (Frontend):
- `web/src/gen/grpcstub/auth/v1/auth_pb.ts` - Message types + service definitions
- `web/src/gen/grpcstub/auth/v1/auth-AuthService_connectquery.ts` - Public service hooks
- `web/src/gen/grpcstub/auth/v1/auth-AuthPrivateService_connectquery.ts` - Private service hook (new file)

---

## Step 3: Update Backend Public Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: Handler implementation pattern
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger naming

### `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**:
Update `AuthPublicGRPCHandler` to implement only the new `AuthServiceHandler` interface (4 methods instead of 5). Remove the `DeviceCodeApprove` stub method.

```go
package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/auth"
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
	// 1. Create handler instance with logger bound to "auth.grpc.connectrpc.public".
	// 2. Return the handler.
}

// GetHandler implements PublicConnectHandler interface.
func (h *AuthPublicGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// 1. Call authv1connect.NewAuthServiceHandler(h, opts...).
	// 2. Return path and handler (path will be "/auth.v1.AuthService/").
}

// GoogleAuth handles Google OAuth authentication.
func (h *AuthPublicGRPCHandler) GoogleAuth(
	ctx context.Context,
	req *connect.Request[authv1.GoogleAuthReq],
) (*connect.Response[authv1.GoogleAuthRes], error) {
	// 1. Create auth.GoogleAuthParams from request message.
	// 2. Call h.svc.GoogleAuth(ctx, params).
	// 3. If error, log and return connect.CodeInternal error.
	// 4. Convert tokens to authv1.TokenPair response.
	// 5. Return connect.NewResponse(res).
}

// RefreshToken handles token refresh.
func (h *AuthPublicGRPCHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[authv1.RefreshTokenReq],
) (*connect.Response[authv1.RefreshTokenRes], error) {
	// 1. Call h.svc.RefreshToken(ctx, req.Msg.RefreshToken).
	// 2. If error, log and return connect.CodeUnauthenticated error.
	// 3. Convert tokens to authv1.TokenPair response.
	// 4. Return connect.NewResponse(res).
}

// DeviceCode initiates device code flow.
func (h *AuthPublicGRPCHandler) DeviceCode(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeReq],
) (*connect.Response[authv1.DeviceCodeRes], error) {
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
	// 1. Call h.svc.DevicePoll(ctx, req.Msg.DeviceCode).
	// 2. If error, log and return connect.CodeInternal error.
	// 3. Create authv1.DevicePollRes with Pending flag.
	// 4. If not pending and tokens exist, add TokenPair to response.
	// 5. Return connect.NewResponse(res).
}

// NOTE: DeviceCodeApprove method is REMOVED - it is now in AuthPrivateGRPCHandler

// Compile-time interface verification.
var _ authv1connect.AuthServiceHandler = (*AuthPublicGRPCHandler)(nil)
```

**Test Scenarios**: N/A (Handler passes through to service layer which has its own tests)

---

## Step 4: Update Backend Private Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: Handler implementation pattern
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger naming

### `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**:
Update `AuthPrivateGRPCHandler` to implement only the new `AuthPrivateServiceHandler` interface (1 method). Remove all stub methods for public endpoints. Update `GetHandler` to use `NewAuthPrivateServiceHandler`.

```go
// AuthPrivateGRPCHandler handles private auth gRPC endpoints (authentication required).
type AuthPrivateGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
}

// NewAuthPrivateGRPCHandler creates a new private auth gRPC handler.
func NewAuthPrivateGRPCHandler(l *slog.Logger, svc *auth.Service) *AuthPrivateGRPCHandler {
	// 1. Create handler instance with logger bound to "auth.grpc.connectrpc.private".
	// 2. Return the handler.
}

// GetHandler implements PrivateConnectHandler interface.
func (h *AuthPrivateGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// 1. Call authv1connect.NewAuthPrivateServiceHandler(h, opts...).
	// 2. Return path and handler (path will be "/auth.v1.AuthPrivateService/").
}

// NOTE: GoogleAuth, RefreshToken, DeviceCode, DevicePoll stubs are REMOVED
// These methods are no longer in the AuthPrivateServiceHandler interface

// DeviceCodeApprove approves a device code (requires authentication).
func (h *AuthPrivateGRPCHandler) DeviceCodeApprove(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	// 2. If userID is empty, return connect.CodeUnauthenticated error.
	// 3. Create auth.DeviceCodeApproveParams with UserCode and UserID.
	// 4. Call h.svc.DeviceCodeApprove(ctx, params).
	// 5. If error:
	//    a. Log error with userCode.
	//    b. Determine error code based on error message:
	//       - "not found" -> connect.CodeNotFound
	//       - "expired" -> connect.CodeDeadlineExceeded
	//       - "already approved" -> connect.CodeAlreadyExists
	//       - default -> connect.CodeInternal
	//    c. Return connect.NewError with appropriate code.
	// 6. Create authv1.DeviceCodeApproveRes with success=true and message.
	// 7. Return connect.NewResponse(res).
}

// Compile-time interface verification.
var _ authv1connect.AuthPrivateServiceHandler = (*AuthPrivateGRPCHandler)(nil)
```

**Test Scenarios**: N/A (Handler passes through to service layer which has its own tests)

---

## Step 5: Update DI Container Module

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: DI patterns with fx
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`: Handler registration

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`

**Description**:
No changes required. The existing module registration already correctly separates public and private handlers into different groups. The handlers will now implement different interfaces but the DI pattern remains the same.

```go
// Existing code - NO CHANGES NEEDED
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
```

The registration in `register_connectrpc.go` also remains unchanged because:
- Public handlers register to `/auth.v1.AuthService/*`
- Private handlers register to `/auth.v1.AuthPrivateService/*`
- No path conflict occurs

---

## Step 6: Update Frontend Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook patterns and gRPC integration

### `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts`

**Description**:
Update the import to use the new `AuthPrivateService` generated hook. The generated file will be named `auth-AuthPrivateService_connectquery.ts`.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { deviceCodeApprove } from '@/gen/grpcstub/auth/v1/auth-AuthPrivateService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useApproveDevice provides a mutation hook for approving device codes.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useApproveDevice = () => {
  // 1. Return useMutation with deviceCodeApprove from AuthPrivateService.
  // 2. Pass transport for API communication.
  return useMutation(deviceCodeApprove, { transport })
}
```

**Test Scenarios**: N/A (Frontend hook - manual testing via device login flow)

---

## Step 7: Verify Build and Tests

**Description**:
Ensure all code compiles and existing tests pass.

**Commands**:
```bash
# Build Go modules
cd /Users/jayce/team-attention/cops && go build ./api/... ./shared/...

# Run Go tests (if any exist for auth service)
cd /Users/jayce/team-attention/cops && go test ./api/...

# Build frontend
cd /Users/jayce/team-attention/cops/web && npm run build

# Type check frontend
cd /Users/jayce/team-attention/cops/web && npm run type-check
```

---

## Summary of Changes

| File | Action | Description |
| :--- | :----- | :---------- |
| `idl/protobuf/auth/v1/auth.proto` | Modify | Split AuthService into AuthService (public) and AuthPrivateService (private) |
| `shared/gen/grpcstub/auth/v1/*` | Regenerate | Generated Go code from buf generate |
| `web/src/gen/grpcstub/auth/v1/*` | Regenerate | Generated TypeScript code from buf generate |
| `api/internal/service/auth/inbound/grpc/connectrpc/handler.go` | Modify | Update handlers to implement separate interfaces |
| `web/src/feature/auth/hook/use-approve-device.ts` | Modify | Update import to use AuthPrivateService hook |

## Path Routing After Implementation

| Endpoint | Service | Path | Auth Required |
| :------- | :------ | :--- | :------------ |
| GoogleAuth | AuthService | `/auth.v1.AuthService/GoogleAuth` | No |
| DeviceCode | AuthService | `/auth.v1.AuthService/DeviceCode` | No |
| DevicePoll | AuthService | `/auth.v1.AuthService/DevicePoll` | No |
| RefreshToken | AuthService | `/auth.v1.AuthService/RefreshToken` | No |
| DeviceCodeApprove | AuthPrivateService | `/auth.v1.AuthPrivateService/DeviceCodeApprove` | Yes |

## Verification Checklist

After implementation, verify:
- [ ] `buf generate` completes without errors
- [ ] Go code compiles (`go build ./api/... ./shared/...`)
- [ ] Frontend builds (`npm run build` in web/)
- [ ] Device login flow works end-to-end:
  1. CLI initiates device code flow
  2. User opens verification URL in browser
  3. User clicks "Approve Device" button
  4. CLI receives tokens and completes authentication
- [ ] Public endpoints (GoogleAuth, DeviceCode, DevicePoll, RefreshToken) work without auth
- [ ] DeviceCodeApprove returns 401 without auth token
- [ ] DeviceCodeApprove returns 200 with valid auth token
