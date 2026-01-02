# Review Result

**Status**: Changes Required

## Request Summary

Code review identified rule violations that need to be addressed. The implementation does not fully follow project standards defined in `.agent/rules/`. Most critically, there's a bug in the Google OAuth adapter's `ExchangeCode` method that will cause token expiry calculation to always return 0. Please address the violation listed below.

## Acceptance Criteria

- [ ] Fix critical bug in `api/internal/service/auth/outbound/oauth/google/google_oauth.go:58` where `ExpiresIn` calculation is incorrect

## Scope

### In Scope
- Fix identified rule violation
- Ensure all changes follow applicable rules
- Correct the OAuth token expiry calculation bug

### Out of Scope
- Any other refactoring or improvements not related to rule violations
- Feature additions beyond fixing violations

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `api/internal/service/auth/outbound/oauth/google/google_oauth.go` | 58 | General Code Quality | **CRITICAL BUG**: ExpiresIn calculation is `token.Expiry.Sub(token.Expiry).Seconds()` which will always be 0 seconds (subtracting a time from itself) | Change to `int(time.Until(token.Expiry).Seconds())` or `int(token.Expiry.Sub(time.Now()).Seconds())` |

## Additional Context

### Files Reviewed

**Domain Models:**
- `shared/domain/account.go` - Account domain model ✅
- `shared/domain/user.go` - User domain model ✅
- `shared/domain/organization.go` - Organization and OrganizationMember domain models ✅

**MongoSchema:**
- `shared/domain/mongoschema/user.go` - User MongoSchema ✅
- `shared/domain/mongoschema/organization.go` - Organization MongoSchema ✅
- `shared/domain/mongoschema/organization_member.go` - OrganizationMember MongoSchema ✅

**Protobuf:**
- `idl/protobuf/auth/v1/auth.proto` - Auth service protobuf definition ✅

**API Module:**
- `api/internal/platform/util/jwtutil/jwtutil.go` - JWT utilities ✅
- `api/internal/platform/middleware/auth.go` - Auth middleware ✅
- `api/internal/platform/setup/config/config.go` - Config with JWT and OAuth ✅
- `api/internal/service/auth/auth_service.go` - Auth service ✅
- `api/internal/service/auth/outbound/repository/user_repo_port.go` - User repository port ✅
- `api/internal/service/auth/outbound/repository/mongodb/user_repo.go` - MongoDB user repository ✅
- `api/internal/service/auth/outbound/oauth/google_port.go` - Google OAuth port ✅
- `api/internal/service/auth/outbound/oauth/google/google_oauth.go` - Google OAuth adapter ❌ (Critical bug)
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go` - ConnectRPC handler ✅

**CLI Module:**
- `cli/internal/service/auth/auth_service.go` - CLI auth service ✅
- `cli/internal/service/auth/outbound/api/auth_port.go` - Auth API port ✅
- `cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` - Auth API client ✅
- `cli/internal/platform/setup/httpclient/httpclient.go` - HTTP client with auth ✅

**Daemon Module:**
- `daemon/internal/service/auth/auth_service.go` - Daemon auth service ✅
- `daemon/internal/platform/setup/copsapi.go` - API client with auth ✅

### Review Notes

#### Positive Findings

1. **Excellent adherence to hexagonal architecture**: The auth service follows the port/adapter pattern correctly with clear separation between service logic and outbound dependencies.

2. **Proper struct field types**: The implementation correctly uses `[]*Account` for the accounts array in User domain model, following the rule that struct array elements must use pointer types.

3. **Correct MongoSchema implementation**: All MongoSchema files follow the mandatory structure with proper field constants, `FromDomain`, and `ToDomain` methods.

4. **JWT implementation is solid**: Uses only `RegisteredClaims` with UserID in Subject field as specified in the plan.

5. **Proper error handling**: Services log errors appropriately and wrap errors with context.

6. **Security best practices**: Auth tokens are stored with 0600 permissions, tokens are never logged.

7. **ConnectRPC handler follows conventions**: Implements the `ConnectHandler` interface correctly with proper logger binding.

8. **CLI auth flow is well-structured**: Device flow implementation matches the plan with proper polling and token storage.

9. **Daemon auth service implements proper caching**: Uses mutex for thread-safe access with 30-second cache TTL.

10. **Context key naming**: Correctly uses camelCase for context keys (`userId`).

11. **Field constants**: Properly defined without dots in values, with separate constants for embedded types.

12. **OAuth configuration**: Correctly injected from main config into adapters via constructors.

13. **HTTP client pattern**: Correctly uses `imroc/req/v3` with `WithAuth()` for per-request auth token injection.

#### Critical Bug

**`api/internal/service/auth/outbound/oauth/google/google_oauth.go:58`**

The `ExpiresIn` calculation has a critical bug:
```go
ExpiresIn: int(token.Expiry.Sub(token.Expiry).Seconds()),
```

This subtracts `token.Expiry` from itself, which will always result in 0 seconds. This means all tokens will appear to expire immediately.

**Fix:**
```go
ExpiresIn: int(time.Until(token.Expiry).Seconds()),
```

or

```go
ExpiresIn: int(token.Expiry.Sub(time.Now()).Seconds()),
```

### Architecture Review

The implementation follows hexagonal architecture correctly:

```
CLI/Daemon → Auth Service → [User Repository Port, OAuth Port]
                                     ↓                    ↓
                            [MongoDB Adapter]   [Google OAuth Adapter]
```

**Inbound adapters:**
- ConnectRPC handler (API)
- CLI commands

**Outbound adapters:**
- MongoDB repository
- Google OAuth client

**Service layer:**
- Authentication business logic
- Token generation
- User creation/lookup

All layers are properly isolated with interfaces at boundaries.

### Security Review

✅ **JWT tokens**: Using HS256 with proper validation
✅ **Token storage**: File permissions set to 0600
✅ **Error messages**: No sensitive data exposed in errors
✅ **Logging**: Tokens are never logged
✅ **Token refresh**: Implements 5-minute buffer before expiry
✅ **Device flow**: Proper handling of pending/complete states
❌ **Token expiry calculation**: Bug will cause incorrect expiry time (see violation above)

### Code Quality Observations

#### What Was Done Well

1. **Domain Model Design**: Clean separation between Account (embedded) and User models
2. **Repository Pattern**: Proper use of `$elemMatch` for querying embedded accounts array
3. **Error Handling**: Consistent error wrapping with context throughout all layers
4. **Logging Strategy**: Structured logging with appropriate log levels and context
5. **Constructor Pattern**: All constructors follow `New{ComponentName}` pattern consistently
6. **Interface Verification**: All adapters include `var _ Port = (*Adapter)(nil)` checks
7. **Middleware Implementation**: Auth middleware correctly extracts and validates JWT, stores userId in fiber locals
8. **CLI UX**: Clear separation between login initiation and polling for good UX
9. **Daemon Integration**: Proper caching strategy with mutex protection and TTL
10. **Protobuf Conventions**: Follows naming conventions with `Req`/`Res` suffixes, snake_case fields

#### Areas That Work Correctly

1. **Thread Safety**: Daemon auth service properly uses RWMutex for concurrent access
2. **File System Security**: Auth file created with 0700 directory and 0600 file permissions
3. **Token Lifecycle**: Proper handling of access token expiry with automatic refresh
4. **Google OAuth Integration**: Device flow correctly handles authorization_pending and slow_down states
5. **Configuration Management**: Environment-based config with sensible defaults
6. **Dependency Injection**: Clean constructor injection throughout all modules

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](.agent/rules/common.md)
- [`.agent/rules/go/go-struct.md`](.agent/rules/go/go-struct.md)
- [`.agent/rules/go/go-platform-domain.md`](.agent/rules/go/go-platform-domain.md)
- [`.agent/rules/go/go-platform-domain-mongoschema.md`](.agent/rules/go/go-platform-domain-mongoschema.md)
- [`.agent/rules/go/go-hexagonal-layout.md`](.agent/rules/go/go-hexagonal-layout.md)
- [`.agent/rules/go/go-service.md`](.agent/rules/go/go-service.md)
- [`.agent/rules/go/go-outbound.md`](.agent/rules/go/go-outbound.md)
- [`.agent/rules/go/go-inbound.md`](.agent/rules/go/go-inbound.md)
- [`.agent/rules/go/go-inbound-grpc-connectrpc.md`](.agent/rules/go/go-inbound-grpc-connectrpc.md)
- [`.agent/rules/idl/protobuf.md`](.agent/rules/idl/protobuf.md)
