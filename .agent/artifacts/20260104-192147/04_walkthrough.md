# Development Walkthrough

## Summary
Fixed the "501 Not Implemented" error in the device login flow by splitting the AuthService protobuf definition into two separate services (AuthService and AuthPrivateService), eliminating the path conflict that caused DeviceCodeApprove requests to route to the wrong handler.

## Problem Overview

### The Issue
When users clicked the "Approve Device" button during device login, the request failed with HTTP 501 and error message "[unimplemented] use authenticated endpoint".

### Root Cause
The system had two ConnectRPC handlers for authentication:
- **AuthPublicGRPCHandler**: Handles unauthenticated endpoints (GoogleAuth, DeviceCode, DevicePoll, RefreshToken)
- **AuthPrivateGRPCHandler**: Handles authenticated endpoints (DeviceCodeApprove)

Both handlers implemented the same `AuthServiceHandler` interface, causing them to register to the same path `/auth.v1.AuthService/*`. When Fiber registered routes, the public handler took precedence, routing all requests (including DeviceCodeApprove) to the public handler.

The public handler's DeviceCodeApprove method intentionally returned HTTP 501 "use authenticated endpoint", while the private handler contained the actual implementation.

### The Solution
Split the single `AuthService` proto definition into two separate services:
- **AuthService**: Public endpoints (no authentication required)
- **AuthPrivateService**: Private endpoints (JWT authentication required)

This leverages ConnectRPC's natural path generation where each service registers to its own unique path:
- `/auth.v1.AuthService/*` for public endpoints
- `/auth.v1.AuthPrivateService/*` for private endpoints (including DeviceCodeApprove)

## Code Overview

### New Services

#### `AuthPrivateService` (Protobuf)
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto`
- **Purpose**: Separate proto service for authenticated endpoints
- **Key Methods**:
  - `DeviceCodeApprove()`: Approves a device code for CLI authentication (requires JWT)

#### Modified Service: `AuthService` (Protobuf)
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto`
- **Changes**: Removed `DeviceCodeApprove` RPC method, now contains only public endpoints
- **Remaining Methods**:
  - `GoogleAuth()`: OAuth flow for web login
  - `DeviceCode()`: Initiates device flow for CLI
  - `DevicePoll()`: Polls for device approval status
  - `RefreshToken()`: Refreshes access token

### Modified Components

#### `AuthPublicGRPCHandler`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`
- **Changes**:
  - Now implements only `AuthServiceHandler` interface (4 methods instead of 5)
  - Removed `DeviceCodeApprove()` stub method
  - `GetHandler()` continues to use `authv1connect.NewAuthServiceHandler()`
  - Registers to path: `/auth.v1.AuthService/*`

#### `AuthPrivateGRPCHandler`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`
- **Changes**:
  - Now implements only `AuthPrivateServiceHandler` interface (1 method)
  - Removed stub methods for public endpoints (GoogleAuth, DeviceCode, DevicePoll, RefreshToken)
  - Updated `GetHandler()` to use `authv1connect.NewAuthPrivateServiceHandler()`
  - Registers to path: `/auth.v1.AuthPrivateService/*`
- **Key Implementation**:
  - Extracts user ID from context using `interceptor.UserIDFromContext(ctx)`
  - Returns `connect.CodeUnauthenticated` if user not authenticated
  - Maps errors to appropriate gRPC codes (NotFound, DeadlineExceeded, AlreadyExists)

#### Frontend Hook: `useApproveDevice`
- **Location**: `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts`
- **Changes**: Updated import to use generated `AuthPrivateService` hook
  - Before: Imported from `auth-AuthService_connectquery`
  - After: Imports from `auth-AuthPrivateService_connectquery`
- **Behavior**: No functional changes, just routes to the correct service endpoint

### Generated Code Changes

#### Backend (Go)
- **Location**: `/Users/jayce/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go`
- **Generated**:
  - `AuthServiceHandler` interface with 4 methods (GoogleAuth, DeviceCode, DevicePoll, RefreshToken)
  - `AuthPrivateServiceHandler` interface with 1 method (DeviceCodeApprove)
  - `NewAuthServiceHandler()` returns path `/auth.v1.AuthService/`
  - `NewAuthPrivateServiceHandler()` returns path `/auth.v1.AuthPrivateService/`
  - Path constants:
    - `AuthServiceGoogleAuthProcedure = "/auth.v1.AuthService/GoogleAuth"`
    - `AuthPrivateServiceDeviceCodeApproveProcedure = "/auth.v1.AuthPrivateService/DeviceCodeApprove"`

#### Frontend (TypeScript)
- **Location**: `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/auth/v1/`
- **Generated**:
  - `auth-AuthService_connectquery.ts`: Public service hooks (GoogleAuth, DeviceCode, etc.)
  - `auth-AuthPrivateService_connectquery.ts`: Private service hook (DeviceCodeApprove) - **NEW FILE**
  - `auth_pb.ts`: Updated message types and service definitions

### No Changes Required

#### DI Container Module
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`
- **Why**: The existing DI registration already correctly separates public and private handlers into different groups. The handlers now implement different interfaces, but the registration pattern remains the same.

## Path Routing

After this implementation, requests route to the correct handlers:

| Endpoint | Service | Full Path | Handler | Auth Required |
|----------|---------|-----------|---------|---------------|
| GoogleAuth | AuthService | `/auth.v1.AuthService/GoogleAuth` | AuthPublicGRPCHandler | No |
| DeviceCode | AuthService | `/auth.v1.AuthService/DeviceCode` | AuthPublicGRPCHandler | No |
| DevicePoll | AuthService | `/auth.v1.AuthService/DevicePoll` | AuthPublicGRPCHandler | No |
| RefreshToken | AuthService | `/auth.v1.AuthService/RefreshToken` | AuthPublicGRPCHandler | No |
| DeviceCodeApprove | AuthPrivateService | `/auth.v1.AuthPrivateService/DeviceCodeApprove` | AuthPrivateGRPCHandler | Yes (JWT) |

The path separation ensures:
- Public handlers register to `/auth.v1.AuthService/*`
- Private handlers register to `/auth.v1.AuthPrivateService/*`
- No path conflict occurs
- DeviceCodeApprove correctly routes to the authenticated handler

## Testing

### Build Verification
All code successfully compiles:
```bash
go build ./api/... ./shared/...  # Result: SUCCESS
cd web && npm run build          # Result: SUCCESS
```

### Code Generation
```bash
cd idl/protobuf && buf generate  # Result: SUCCESS
# Generated Go stubs in shared/gen/grpcstub/auth/v1/
# Generated TypeScript stubs in web/src/gen/grpcstub/auth/v1/
```

### Verification Commands Run

| Command | Purpose | Result |
|---------|---------|--------|
| `buf generate` | Regenerate protobuf code | SUCCESS |
| `go build ./api/... ./shared/...` | Build Go modules | SUCCESS |
| `npm run build` (in web/) | Build frontend | SUCCESS |

### Manual Testing Required

The device login flow should be tested end-to-end:

1. **CLI initiates device flow**:
   - Run: `cops login` or equivalent CLI command
   - Should receive device code and verification URL

2. **User approves in browser**:
   - Open verification URL
   - Should see device approval page with user code
   - Click "Approve Device" button
   - Should succeed with HTTP 200 (not 501)

3. **CLI receives tokens**:
   - CLI should poll and receive JWT tokens
   - Authentication should complete successfully

4. **Public endpoints continue working**:
   - GoogleAuth: Web login flow works
   - DeviceCode: CLI can initiate device flow
   - DevicePoll: CLI can poll for approval
   - RefreshToken: Token refresh works

5. **Authentication enforcement**:
   - DeviceCodeApprove without JWT: Returns 401 Unauthenticated
   - DeviceCodeApprove with valid JWT: Returns 200 Success

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Both handlers registered to same path `/auth.v1.AuthService/` | Split proto service into AuthService (public) and AuthPrivateService (private) to generate different paths |
| Public handler took precedence in route registration | Service split ensures handlers register to different paths: `/auth.v1.AuthService/*` and `/auth.v1.AuthPrivateService/*` |
| Frontend imported from wrong service | Updated `use-approve-device.ts` to import from `auth-AuthPrivateService_connectquery` |

## Related Tickets

No explicit ticket reference was provided. This fix resolves the DeviceCodeApprove 501 error in the device login flow.

## Key Decisions

### Why split the service instead of other solutions?

**Alternative approaches considered**:
1. **Custom path prefixes**: Manually prefix handler paths (e.g., `/auth/public/*`, `/auth/private/*`)
   - Rejected: Breaks ConnectRPC conventions, requires custom routing logic
2. **Middleware-based routing**: Route based on auth header presence
   - Rejected: Still has path conflict, adds complexity
3. **Separate HTTP routers**: Mount public and private handlers to different base paths
   - Rejected: Requires infrastructure changes, complicates deployment

**Chosen approach: Proto service split**:
- Leverages ConnectRPC's built-in path generation
- Clean separation of public vs private APIs
- No custom routing logic needed
- Type-safe at compile time
- Follows RESTful principle of resource separation

### Why keep both handlers in the same file?

Both `AuthPublicGRPCHandler` and `AuthPrivateGRPCHandler` remain in `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` because:
- They share the same service layer (`auth.Service`)
- They're both ConnectRPC implementations for the same domain (auth)
- File organization follows `inbound/{protocol}/{implementation}/` pattern
- Separation is by interface (Port), not by file

### Why no changes to DI container?

The existing DI registration pattern already separated handlers into `public_connect_handlers` and `private_connect_handlers` groups. The handlers now implement different interfaces (`AuthServiceHandler` vs `AuthPrivateServiceHandler`), but the group-based registration works identically.

## Architecture Notes

This change maintains **Hexagonal Architecture** principles:
- **Inbound adapters** (handlers) remain in `inbound/grpc/connectrpc/`
- **Service layer** unchanged (business logic intact)
- **Port/Adapter pattern** preserved (handlers implement ConnectRPC-generated interfaces)
- **Dependency injection** through fx groups continues to work

The separation is purely at the **protocol definition level** (proto services) and **interface level** (handler interfaces), not at the business logic level.
