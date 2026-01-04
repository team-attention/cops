# Development Walkthrough

## Summary

Fixed the Daemon's token refresh behavior by refactoring authentication state management from the service layer to a platform-level outbound adapter, implementing proactive token refresh with a 5-minute buffer, and updating the auth interceptor to fail fast on missing tokens instead of sending unauthenticated requests.

## Code Overview

### New Components

#### `AuthStatePort`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/outbound/authstate/authstate_port.go`
- **Purpose**: Platform-level port interface for accessing authentication state with automatic token refresh
- **Key Methods**:
  - `GetAccessToken(ctx context.Context) (string, error)`: Returns a valid access token, refreshing if needed

**Design Decision**: This interface was placed in `platform/outbound/authstate/` rather than `service/auth/outbound/` because authentication state is a cross-cutting infrastructure concern that can be used by any service requiring API authentication, not just the auth service itself.

#### `FilesystemAuthState`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/outbound/authstate/filesystem/authstate.go`
- **Purpose**: Filesystem-based adapter implementing `AuthStatePort` with proactive token refresh logic
- **Key Features**:
  - **Proactive Refresh**: Refreshes tokens 5 minutes before expiry (300-second buffer)
  - **Thread-Safe Refresh**: Uses `refreshMu` mutex to prevent concurrent refresh attempts with double-check pattern
  - **Automatic Retry**: Re-checks token validity after acquiring refresh lock (handles race condition where another goroutine may have already refreshed)
  - **Secure File Handling**: Creates `.cops` directory with 0700 permissions, writes `auth.json` with 0600 permissions
- **Key Methods**:
  - `GetAccessToken(ctx context.Context) (string, error)`: Main entry point with proactive refresh logic
  - `readAuthState() (*AuthState, error)`: Reads and parses `~/.cops/auth.json`
  - `saveAuthState(state *AuthState) error`: Writes updated tokens to disk with secure permissions

**Token Refresh Flow**:
```
1. Read current state (with RLock)
2. Check if token is valid with 5-min buffer: (expiresAt - now) > 300
3. If valid, return cached token immediately
4. If near expiry or expired:
   a. Acquire refresh lock (prevents concurrent refresh)
   b. Double-check: Re-read state (another goroutine may have refreshed)
   c. Call API to refresh token
   d. Update state with new tokens
   e. Save to disk
   f. Return new access token
```

**Dependencies**: Depends on `AuthAPIPort` (from `service/auth/outbound/api/`) for the actual refresh API call.

### Modified Components

#### `AuthInterceptor`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`
- **Changes**:
  - **Dependency Change**: Now depends on `AuthStatePort` interface instead of `*auth.Service`
  - **Fail-Fast Behavior**: Returns `CodeUnauthenticated` error immediately when `GetAccessToken()` fails
  - **Removed Fallback**: Deleted the problematic logic that sent requests without authentication when no token was available
  - **Simplified Flow**: Removed 401 retry logic since proactive refresh in adapter handles token expiry before requests fail

**Before** (Problematic):
```go
token, err := i.authService.GetAccessToken()
if err != nil {
    // Sent request WITHOUT auth - problematic for authenticated endpoints
    i.logger.Debug("no valid access token available, sending request without auth")
    return next(ctx, req)
}
```

**After** (Fail-Fast):
```go
token, err := i.authState.GetAccessToken(ctx)
if err != nil {
    i.logger.Error("failed to get access token", slog.Any("error", err))
    return nil, connect.NewError(connect.CodeUnauthenticated, err)
}
```

#### `module_platform.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_platform.go`
- **Changes**:
  - **Added Home Directory Provider**: Provides home directory string for auth state adapter initialization
  - **Added FilesystemAuthState Provider**: Registers `FilesystemAuthState` with `fx.As` to provide `AuthStatePort` interface
  - **Updated Dependencies**: `AuthInterceptor` now receives `AuthStatePort` instead of `*auth.Service`

**New Providers**:
```go
// Home directory for auth state adapter
fx.Provide(func() string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "."
    }
    return homeDir
}),

// FilesystemAuthState adapter
fx.Provide(fx.Annotate(
    filesystem.NewFilesystemAuthState,
    fx.As(new(authstate.AuthStatePort)),
)),
```

#### `module_auth.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_auth.go`
- **Changes**:
  - **Removed Home Directory Provider**: Moved to platform module since it's now used by platform adapter
  - **Removed auth.Service Provider**: Service layer no longer manages auth state - functionality moved to platform adapter
  - **Kept AuthAPIPort Provider**: Still needed by `FilesystemAuthState` for token refresh API calls

**Minimal Module** (Only provides API client):
```go
func newAuthModule() fx.Option {
    return fx.Module("auth",
        fx.Provide(fx.Annotate(
            connectrpc.NewAuthAPIClient,
            fx.As(new(api.AuthAPIPort)),
        )),
    )
}
```

### Deleted Components

#### `auth_service.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go` (DELETED)
- **Reason**: Token refresh and `auth.json` management moved to platform adapter (`FilesystemAuthState`). The service layer no longer manages authentication state - this is now a platform infrastructure concern.
- **Preserved**: The `outbound/api/` subdirectory containing `AuthAPIPort` interface and `connectrpc.AuthAPIClient` implementation remain, as they're used by the platform adapter.

## Architecture Improvements

### Hexagonal Architecture Compliance

The refactoring improves adherence to hexagonal architecture principles:

1. **Platform Infrastructure**: Authentication state is now properly categorized as platform infrastructure (`platform/outbound/authstate/`) rather than business logic
2. **Port-Adapter Pattern**: Clear separation between `AuthStatePort` interface and `FilesystemAuthState` implementation
3. **Dependency Inversion**: Interceptor depends on `AuthStatePort` abstraction, not concrete implementation
4. **Single Responsibility**: `FilesystemAuthState` owns file I/O and token refresh; services don't manage infrastructure concerns

### Dependency Flow

```
AuthInterceptor (platform)
    ↓ depends on
AuthStatePort (interface)
    ↑ implemented by
FilesystemAuthState (adapter)
    ↓ depends on
AuthAPIPort (interface)
    ↑ implemented by
AuthAPIClient (adapter)
```

### Thread Safety

The implementation uses a two-lock pattern for thread-safe token refresh:

- **`mu sync.RWMutex`**: Read-write lock for reading auth state from disk
  - Multiple goroutines can read simultaneously (RLock)
  - Prevents reads during writes
- **`refreshMu sync.Mutex`**: Exclusive lock for refresh operations
  - Only one goroutine can refresh at a time
  - Double-check pattern after acquiring lock prevents redundant refresh attempts

**Race Condition Prevention**: If two goroutines detect near-expiry simultaneously:
1. Both acquire RLock and read state
2. Both detect token needs refresh
3. First goroutine acquires `refreshMu`, performs refresh
4. Second goroutine waits for `refreshMu`
5. Second goroutine re-reads state after acquiring lock
6. Second goroutine sees valid token, returns without refreshing

## Testing

### Verification Commands Run

```bash
go build ./daemon/...  # Result: PASS
```

### Test Coverage

No unit tests were added in this implementation. The following test scenarios were identified but not implemented:

**FilesystemAuthState**:
- Token valid with buffer (>5min remaining)
- Token near expiry (<5min remaining)
- Token already expired
- Not authenticated (no auth file)
- Token refresh API call fails
- Concurrent refresh (double-check pattern)

**AuthInterceptor**:
- Valid token available
- Token refresh during GetAccessToken
- GetAccessToken fails (not authenticated)
- GetAccessToken fails (refresh failed)
- Streaming with valid/invalid token

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Service layer managing file I/O for auth state | Moved auth state management to platform outbound adapter following hexagonal architecture |
| No automatic token refresh in Daemon | Implemented proactive refresh with 5-minute buffer in `FilesystemAuthState.GetAccessToken()` |
| Interceptor sending requests without auth on token failure | Changed to fail-fast: return `CodeUnauthenticated` error immediately |
| Potential concurrent refresh attempts | Implemented double-check pattern with `refreshMu` mutex |
| Home directory provider in wrong module | Moved from auth module to platform module (used by platform adapter) |

## Related Changes

This implementation followed the architectural pattern established in the CLI:

- **Reference Implementation**: `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem/authstate.go`
- **Pattern Consistency**: Both CLI and Daemon now use the same platform-level `AuthStatePort` pattern for authentication state management
- **Code Reuse Opportunity**: Future work could extract this to a shared package to avoid duplication between CLI and Daemon

## Key Decisions

1. **Platform vs Service Layer**: Authentication state management was moved from `service/auth/` to `platform/outbound/authstate/` because it's infrastructure, not business logic
2. **5-Minute Refresh Buffer**: Matches CLI behavior to ensure consistent token refresh timing across components
3. **Fail-Fast on No Token**: Interceptor now returns error immediately rather than attempting unauthenticated requests, preventing confusing 401 errors downstream
4. **Double-Check Pattern**: Prevents redundant refresh API calls when multiple goroutines detect near-expiry simultaneously
5. **Preserved AuthAPIPort**: Kept the auth service's API client adapter since it's reused by the platform adapter for refresh operations

## Conclusion

This refactoring successfully fixes the Daemon's token refresh behavior while improving architectural alignment with hexagonal architecture principles. The authentication state is now properly categorized as platform infrastructure with clear separation of concerns, proactive token refresh prevents request failures, and the fail-fast behavior in the interceptor provides clearer error handling for authentication failures.
