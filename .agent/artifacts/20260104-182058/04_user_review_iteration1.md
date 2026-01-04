# Review Result

**Status**: Changes Required

## Request Summary

Code review identified a critical routing issue preventing the DeleteAccount RPC from being accessible. The UserService handler is properly implementing the DeleteAccount method and is registered to the private handler group, but the handler registration does not match the pattern used by other services that have both public and private RPCs.

## Root Cause Analysis

The 404 error occurs because **UserGRPCHandler does not receive auth interceptor options** when calling `GetHandler()`.

### How ConnectRPC Registration Works

From `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`:

```go
// Register private handlers WITH auth interceptor options
for _, handler := range params.PrivateHandlers {
    path, h := handler.GetHandler(privateOpts...)  // ✅ Passes auth interceptor
    params.App.All(path+"*", adaptor.HTTPHandler(h))
}
```

The `privateOpts` contains the auth interceptor, which is passed when calling `handler.GetHandler(privateOpts...)`.

### The Problem with UserGRPCHandler

**UserGRPCHandler implementation** (`/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go:37-39`):

```go
func (h *UserGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
    return userv1connect.NewUserServiceHandler(h, opts...)
}
```

This implementation:
1. ✅ Correctly passes `opts` to the generated handler
2. ✅ Is registered to `private_connect_handlers` group
3. ✅ SHOULD receive `privateOpts` containing auth interceptor

**BUT**, the handler implementation manually extracts and validates JWT tokens in both `GetMe` and `DeleteAccount` methods instead of relying on the auth interceptor.

### Comparison with AuthService Pattern

**AuthService uses TWO separate handlers**:

1. **AuthPublicGRPCHandler** (registered to `public_connect_handlers`):
   - Handles: GoogleAuth, RefreshToken, DeviceCode, DevicePoll
   - Returns `Unimplemented` for: DeviceCodeApprove

2. **AuthPrivateGRPCHandler** (registered to `private_connect_handlers`):
   - Handles: DeviceCodeApprove (uses `interceptor.UserIDFromContext`)
   - Returns `Unimplemented` for: GoogleAuth, RefreshToken, DeviceCode, DevicePoll

The private handler extracts userID from context via `interceptor.UserIDFromContext(ctx)`, which is set by the auth interceptor.

### Why UserService Differs

UserService has:
- `GetMe` - requires auth
- `DeleteAccount` - requires auth

Both methods manually extract JWT from the Authorization header instead of using the auth interceptor. This suggests **UserGRPCHandler was implemented before the public/private handler pattern was established**.

## Investigation Results

### Files Checked

| File | Finding |
|------|---------|
| `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go` | ✅ DeleteAccount method is implemented (lines 128-202) |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect/user.connect.go` | ✅ Generated interface includes DeleteAccount |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go` | ✅ Handler registered to `private_connect_handlers` group |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` | ✅ Private handlers receive auth interceptor options |

### Generated ConnectRPC Interface Verification

The generated code includes the DeleteAccount RPC:

```
UserServiceDeleteAccountProcedure = "/user.v1.UserService/DeleteAccount"
```

The procedure is registered in the generated handler's ServeHTTP method.

### Handler Registration Verification

Module registration is correct:

```go
fx.Provide(
    fx.Annotate(
        connectrpc.NewUserGRPCHandler,
        fx.As(new(PrivateConnectHandler)),
        fx.ResultTags(`group:"private_connect_handlers"`),
    ),
)
```

## Acceptance Criteria

The following issues must be fixed to resolve the 404 error:

- [ ] UserGRPCHandler must properly utilize the auth interceptor instead of manually extracting JWT tokens
- [ ] GetMe method should use `interceptor.UserIDFromContext(ctx)` instead of manual JWT validation
- [ ] DeleteAccount method should use `interceptor.UserIDFromContext(ctx)` instead of manual JWT validation
- [ ] Remove duplicate JWT validation code from handler methods
- [ ] Ensure handler follows the same pattern as AuthPrivateGRPCHandler

## Scope

### In Scope
- Refactor UserGRPCHandler to use auth interceptor pattern
- Remove manual JWT extraction from GetMe and DeleteAccount methods
- Use `interceptor.UserIDFromContext(ctx)` to get authenticated userID
- Ensure error handling matches auth interceptor behavior

### Out of Scope
- Any other refactoring not related to the auth interceptor issue
- Changes to the Service layer (business logic is correct)
- Changes to repository implementations
- Frontend changes

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `api/internal/service/user/inbound/grpc/connectrpc/handler.go` | 42-73 | `go/go-inbound-grpc-connectrpc.md` | GetMe manually extracts JWT instead of using auth interceptor | Use `interceptor.UserIDFromContext(ctx)` pattern from AuthPrivateGRPCHandler |
| `api/internal/service/user/inbound/grpc/connectrpc/handler.go` | 129-162 | `go/go-inbound-grpc-connectrpc.md` | DeleteAccount manually extracts JWT instead of using auth interceptor | Use `interceptor.UserIDFromContext(ctx)` pattern from AuthPrivateGRPCHandler |
| `api/internal/service/user/inbound/grpc/connectrpc/handler.go` | 28-33 | `go/go-inbound-grpc-connectrpc.md` | Handler constructor accepts `cfg *config.Config` but should not need it if using interceptor | Remove cfg dependency and use interceptor pattern |

## Correct Implementation Pattern

**Reference**: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go:207-248`

```go
// Correct pattern from AuthPrivateGRPCHandler.DeviceCodeApprove
func (h *AuthPrivateGRPCHandler) DeviceCodeApprove(
    ctx context.Context,
    req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
    // 1. Get userID from context (set by auth interceptor)
    userID := interceptor.UserIDFromContext(ctx)
    if userID == "" {
        return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
    }

    // 2. Call service with userID from context
    // ... rest of implementation
}
```

**UserGRPCHandler should follow this pattern**:

```go
type UserGRPCHandler struct {
    svc    *user.Service
    logger *slog.Logger
    // Remove: cfg *config.Config (not needed with interceptor)
}

func NewUserGRPCHandler(l *slog.Logger, svc *user.Service) *UserGRPCHandler {
    return &UserGRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "user.grpc.connectrpc")),
        // Remove: cfg
    }
}

func (h *UserGRPCHandler) GetMe(
    ctx context.Context,
    req *connect.Request[userv1.GetMeReq],
) (*connect.Response[userv1.GetMeRes], error) {
    // Get userID from context (set by auth interceptor)
    userID := interceptor.UserIDFromContext(ctx)
    if userID == "" {
        return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
    }

    // Call service with userID from context
    result, err := h.svc.GetMe(ctx, userID)
    // ... rest of implementation
}

func (h *UserGRPCHandler) DeleteAccount(
    ctx context.Context,
    req *connect.Request[userv1.DeleteAccountReq],
) (*connect.Response[userv1.DeleteAccountRes], error) {
    // Get userID from context (set by auth interceptor)
    userID := interceptor.UserIDFromContext(ctx)
    if userID == "" {
        return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
    }

    // Get confirmation phrase from request
    confirmationPhrase := req.Msg.ConfirmationPhrase

    // Call service with userID from context
    result, err := h.svc.DeleteAccount(ctx, userID, confirmationPhrase)
    // ... rest of implementation
}
```

## Additional Context

- Requirements document: `.agent/artifacts/20260104-182058/01_clarify.md`
- Plan document: `.agent/artifacts/20260104-182058/02_plan.md`
- Initial review: `.agent/artifacts/20260104-182058/03_review.md`
- Review triggered by 404 error on DeleteAccount RPC endpoint

## Rules References

The following rules were applied during this review:
- [`.agent/rules/go/go-inbound.md`](.agent/rules/go/go-inbound.md) - Handler interface patterns
- [`.agent/rules/go/go-inbound-grpc-connectrpc.md`](.agent/rules/go/go-inbound-grpc-connectrpc.md) - ConnectRPC handler patterns
- [`.agent/rules/common.md`](.agent/rules/common.md) - Code consistency
