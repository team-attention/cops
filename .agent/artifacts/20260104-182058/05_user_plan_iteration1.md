# Implementation Plan: Refactor UserGRPCHandler to Use Auth Interceptor

## Overview

This plan addresses the issue where UserGRPCHandler manually extracts and validates JWT tokens instead of using the auth interceptor pattern. The handler is currently registered to the `private_connect_handlers` group and receives auth interceptor options, but the implementation bypasses the interceptor by manually validating tokens.

The goal is to refactor UserGRPCHandler to follow the same pattern as AuthPrivateGRPCHandler.DeviceCodeApprove, which properly utilizes the auth interceptor by extracting userID from context using `interceptor.UserIDFromContext(ctx)`.

## Package Changes

No external package changes are required. This refactoring only involves modifying existing code to use the auth interceptor pattern.

## Implementation Steps

### Step 1: Refactor UserGRPCHandler Constructor and Struct

**Files to Read**:
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns and constructor conventions
- `.agent/rules/go/go-logging-conventions.md`: Logger binding patterns
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: Reference implementation of AuthPrivateGRPCHandler (lines 155-172)

#### `api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
Remove the `cfg *config.Config` dependency from the handler struct and constructor, as JWT validation is now handled by the auth interceptor.

**Changes**:

1. **Update imports** (lines 3-18):
   - Remove: `"github.com/team-attention/cops/api/internal/platform/setup/config"`
   - Remove: `"github.com/team-attention/cops/api/internal/platform/util/jwtutil"`
   - Remove: `"strings"`
   - Add: `"github.com/team-attention/cops/api/internal/platform/interceptor"`

2. **Modify struct definition** (lines 20-25):

```go
// UserGRPCHandler handles gRPC requests for user service.
type UserGRPCHandler struct {
    svc    *user.Service
    logger *slog.Logger
    // Removed: cfg *config.Config
}
```

3. **Update constructor** (lines 27-34):

```go
// NewUserGRPCHandler creates a new user gRPC handler.
func NewUserGRPCHandler(l *slog.Logger, svc *user.Service) *UserGRPCHandler {
    return &UserGRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "user.grpc.connectrpc")),
        // Removed: cfg parameter and field initialization
    }
}
```

### Step 2: Refactor GetMe Method

**Files to Read**:
- `api/internal/platform/interceptor/auth_interceptor.go`: UserIDFromContext function (lines 17-31)
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: DeviceCodeApprove reference implementation (lines 207-248)

#### `api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
Replace manual JWT extraction and validation with `interceptor.UserIDFromContext(ctx)`.

**Method Signature** (no change):
```go
func (h *UserGRPCHandler) GetMe(
    ctx context.Context,
    req *connect.Request[userv1.GetMeReq],
) (*connect.Response[userv1.GetMeRes], error)
```

**Implementation outline**:
```go
func (h *UserGRPCHandler) GetMe(
    ctx context.Context,
    req *connect.Request[userv1.GetMeReq],
) (*connect.Response[userv1.GetMeRes], error) {
    // 1. Extract userID from context (set by auth interceptor).
    //    - Call interceptor.UserIDFromContext(ctx)
    //    - If userID is empty string, log warning and return connect.CodeUnauthenticated error
    //      with message "user not authenticated"

    // 2. Call h.svc.GetMe(ctx, userID) to get user and organizations.

    // 3. If error returned from service:
    //    a. If error contains "user not found":
    //       - Log info with userID
    //       - Return connect.CodeNotFound error
    //    b. Otherwise:
    //       - Log error with userID and error details
    //       - Return connect.CodeInternal error

    // 4. Convert service result to protobuf response:
    //    a. If result.User is not nil:
    //       - Create domainv1.User with fields mapped from result.User:
    //         - Id: string(result.User.ID)
    //         - Email: result.User.Email
    //         - Name: result.User.Name
    //         - AvatarUrl: result.User.ProfileImageURL
    //    b. For each item in result.Organizations:
    //       - If userOrg.Organization is not nil:
    //         - Append domainv1.Organization with fields:
    //           - Id: string(userOrg.Organization.ID)
    //           - Name: userOrg.Organization.Name
    //           - Slug: userOrg.Organization.Slug
    //           - (Note: Members field intentionally not populated)

    // 5. Create userv1.GetMeRes with User and Organizations fields.

    // 6. Return connect.NewResponse with the response.
}
```

**Test Scenarios**:

| Scenario | Context State | Expected Output | Branch Covered |
|:---------|:--------------|:----------------|:---------------|
| Valid authenticated request | userID="valid-user-id" in context | Success with user and orgs | Happy path |
| Missing userID in context | userID="" (empty) in context | Error: CodeUnauthenticated, "user not authenticated" | Auth validation |
| User not found | userID in context, but service returns "user not found" error | Error: CodeNotFound | User not found branch |
| Service internal error | userID in context, service returns unexpected error | Error: CodeInternal | Error handling branch |

### Step 3: Refactor DeleteAccount Method

**Files to Read**:
- `api/internal/platform/interceptor/auth_interceptor.go`: UserIDFromContext function (lines 17-31)
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: DeviceCodeApprove reference implementation (lines 207-248)

#### `api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
Replace manual JWT extraction and validation with `interceptor.UserIDFromContext(ctx)`.

**Method Signature** (no change):
```go
func (h *UserGRPCHandler) DeleteAccount(
    ctx context.Context,
    req *connect.Request[userv1.DeleteAccountReq],
) (*connect.Response[userv1.DeleteAccountRes], error)
```

**Implementation outline**:
```go
func (h *UserGRPCHandler) DeleteAccount(
    ctx context.Context,
    req *connect.Request[userv1.DeleteAccountReq],
) (*connect.Response[userv1.DeleteAccountRes], error) {
    // 1. Extract userID from context (set by auth interceptor).
    //    - Call interceptor.UserIDFromContext(ctx)
    //    - If userID is empty string, log warning and return connect.CodeUnauthenticated error
    //      with message "user not authenticated"

    // 2. Get confirmation phrase from req.Msg.ConfirmationPhrase.

    // 3. Call h.svc.DeleteAccount(ctx, userID, confirmationPhrase).

    // 4. If error returned from service:
    //    a. If error contains "confirmation phrase":
    //       - Log info with userID
    //       - Return connect.CodeInvalidArgument error
    //    b. If error contains "user not found":
    //       - Log info with userID
    //       - Return connect.CodeNotFound error
    //    c. Otherwise:
    //       - Log error with userID and error details
    //       - Return connect.CodeInternal error

    // 5. Create userv1.DeleteAccountRes with:
    //    - Success: result.Success
    //    - Message: result.Message

    // 6. Return connect.NewResponse with the response.
}
```

**Test Scenarios**:

| Scenario | Context State | Request Data | Expected Output | Branch Covered |
|:---------|:--------------|:-------------|:----------------|:---------------|
| Valid delete request | userID in context | confirmationPhrase="DELETE MY ACCOUNT" | Success with message | Happy path |
| Missing userID in context | userID="" (empty) | confirmationPhrase="DELETE MY ACCOUNT" | Error: CodeUnauthenticated, "user not authenticated" | Auth validation |
| Invalid confirmation phrase | userID in context | confirmationPhrase="wrong" | Error: CodeInvalidArgument | Validation branch |
| User not found | userID in context | confirmationPhrase="DELETE MY ACCOUNT", but user doesn't exist | Error: CodeNotFound | User not found branch |
| Service internal error | userID in context | confirmationPhrase="DELETE MY ACCOUNT", service fails | Error: CodeInternal | Error handling branch |

### Step 4: Verify DI Container Registration

**Files to Read**:
- `api/cmd/internal/container/module_user.go`: Current DI registration (lines 41-48)
- `.agent/rules/go/go-container.md`: Container patterns and fx.Provide conventions

#### `api/cmd/internal/container/module_user.go`

**Description**:
Verify that the DI container registration is correct after removing the `cfg` parameter from NewUserGRPCHandler. The registration should automatically adapt because fx will inject only the required dependencies (logger and service).

**No changes required** - fx will automatically inject the correct parameters based on the updated constructor signature:

```go
// ConnectRPC handler (private - requires auth)
fx.Provide(
    fx.Annotate(
        connectrpc.NewUserGRPCHandler,  // Now expects (l *slog.Logger, svc *user.Service)
        fx.As(new(PrivateConnectHandler)),
        fx.ResultTags(`group:"private_connect_handlers"`),
    ),
),
```

**Verification steps**:
1. Check that NewUserGRPCHandler constructor signature matches: `func NewUserGRPCHandler(l *slog.Logger, svc *user.Service) *UserGRPCHandler`
2. Verify that fx can resolve both dependencies:
   - `*slog.Logger` is provided by platform module
   - `*user.Service` is provided by user module (line 39)
3. Confirm that no compilation errors occur after changes

## Quality Checklist

- [x] Every function has a concrete signature (GetMe and DeleteAccount signatures unchanged)
- [x] Detailed algorithm explanation included as comments in both method implementations
- [x] Both methods have test scenarios covering all branches (auth validation, happy path, error handling)
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All dependencies are clearly specified (removed cfg, using interceptor.UserIDFromContext)
- [x] Execution order is clear: Step 1 (struct/constructor) → Step 2 (GetMe) → Step 3 (DeleteAccount) → Step 4 (verify DI)

## Expected Outcome

After implementing this plan:

1. UserGRPCHandler will no longer manually validate JWT tokens
2. The handler will use `interceptor.UserIDFromContext(ctx)` to extract userID set by the auth interceptor
3. The `cfg *config.Config` dependency will be removed from the handler
4. Both GetMe and DeleteAccount methods will follow the same pattern as AuthPrivateGRPCHandler.DeviceCodeApprove
5. The handler will properly utilize the auth interceptor options passed via `GetHandler(opts...)`
6. The 404 error on DeleteAccount RPC endpoint will be resolved

## References

- Review document: `/Users/jayce/team-attention/cops/.agent/artifacts/20260104-182058/04_user_review_iteration1.md`
- Reference implementation: `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go` (AuthPrivateGRPCHandler.DeviceCodeApprove, lines 207-248)
- Auth interceptor: `/Users/jayce/team-attention/cops/api/internal/platform/interceptor/auth_interceptor.go` (UserIDFromContext, lines 17-31)
- Rules:
  - `.agent/rules/go/go-inbound-grpc-connectrpc.md` - ConnectRPC handler patterns
  - `.agent/rules/go/go-logging-conventions.md` - Logger binding patterns
  - `.agent/rules/go/go-container.md` - DI container patterns
