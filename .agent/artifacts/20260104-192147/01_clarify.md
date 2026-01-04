# Requirements

## Request Summary

Fix the "501 Not Implemented" error when approving a device during the Device Login flow. The issue occurs when a user clicks the "Approve Device" button on the device approval page. The error message "[unimplemented] use authenticated endpoint" and HTTP 501 status indicate that the DeviceCodeApprove RPC is calling the public handler instead of the authenticated private handler.

## Root Cause Analysis

The system has two separate handlers for AuthService:
- **AuthPublicGRPCHandler**: For unauthenticated endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll)
- **AuthPrivateGRPCHandler**: For authenticated endpoints (DeviceCodeApprove)

Currently, both handlers are registered to the same service path (`/auth.v1.AuthService/`), causing the public handler to respond to all requests. The DeviceCodeApprove method in the public handler intentionally returns a 501 error with "use authenticated endpoint", while the private handler contains the actual implementation.

## Acceptance Criteria

- [ ] DeviceCodeApprove RPC calls route to AuthPrivateGRPCHandler instead of AuthPublicGRPCHandler
- [ ] Authenticated requests to DeviceCodeApprove succeed with 200 status
- [ ] Device approval flow works end-to-end: user clicks "Approve Device" button and device is successfully linked
- [ ] Public endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) continue to work without authentication
- [ ] Unauthenticated requests to DeviceCodeApprove return 401 Unauthenticated (not 501)
- [ ] All other auth endpoints remain functional

## Scope

### In Scope
- Modify ConnectRPC handler registration to prevent path conflicts between public and private handlers
- Ensure DeviceCodeApprove routes to the authenticated handler
- Verify the auth interceptor properly validates JWT tokens for DeviceCodeApprove

### Out of Scope
- Changes to the auth service business logic (DeviceCodeApprove implementation is already correct)
- Changes to the frontend components or hooks
- Modifications to the device code generation or validation logic
- Updates to the database schema or repositories

## Constraints

- **Architecture constraint**: Must maintain separation between public and private handlers as per hexagonal architecture
- **No breaking changes**: Cannot break existing public endpoints (GoogleAuth, DeviceCode, DevicePoll, RefreshToken)
- **Security requirement**: DeviceCodeApprove MUST require authentication via JWT token
- **ConnectRPC limitation**: Both handlers implement the same AuthServiceHandler interface, causing path conflicts

## Additional Context

### Current Handler Registration (module_auth.go)

```go
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

### Current Registration in register_connectrpc.go

```go
// Register public handlers WITHOUT any interceptor options
for _, handler := range params.PublicHandlers {
    path, h := handler.GetHandler()
    params.App.All(path+"*", adaptor.HTTPHandler(h))
}

// Register private handlers WITH auth interceptor options
for _, handler := range params.PrivateHandlers {
    path, h := handler.GetHandler(privateOpts...)
    params.App.All(path+"*", adaptor.HTTPHandler(h))
}
```

### Problem

Both handlers call `authv1connect.NewAuthServiceHandler(h, opts...)` which generates the same path (`/auth.v1.AuthService/`). When both are registered to Fiber using `app.All(path+"*", ...)`, the public handler registration takes precedence, causing all requests to route to the public handler.

### Relevant Files

- Backend:
  - `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` - Handler implementations
  - `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go` - DI registration
  - `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` - Handler registration
  - `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto` - Service definition

- Frontend:
  - `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts` - Hook calling the RPC
  - `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts` - Transport with auth interceptor

### Questions Resolved

| Question | Answer |
|----------|--------|
| Is the DeviceCodeApprove service implementation correct? | Yes, the implementation in AuthPrivateGRPCHandler is correct and includes authentication checks via interceptor.UserIDFromContext(ctx) |
| Is the frontend sending authentication tokens? | Yes, the connect-transport.ts has an auth interceptor that adds "Authorization: Bearer {token}" headers |
| Why does the public handler have a DeviceCodeApprove method? | Because both handlers implement the same authv1connect.AuthServiceHandler interface (all methods required), but the public handler returns CodeUnimplemented for authenticated-only methods |
| What routing mechanism is used? | Fiber web framework with gofiber/adaptor for mounting ConnectRPC handlers |
| How are public and private handlers differentiated? | Public handlers registered without interceptor options, private handlers registered with auth interceptor options - but both currently register to the same path |
