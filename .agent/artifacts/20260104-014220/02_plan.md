# Implementation Plan: Daemon Token Refresh Logic

## Overview

The daemon currently sends logs to the API server but lacks token refresh logic when the access token expires. This implementation adds automatic token refresh capability to the daemon's auth service, following the pattern established in the CLI's auth service. When the daemon receives a 401 Unauthorized error from the API server, it will automatically use the refresh token to obtain a new access token, persist it to `~/.cops/auth.json`, and retry the original request.

The implementation follows hexagonal architecture patterns:
1. Add an outbound adapter (`AuthAPIPort`) for calling the RefreshToken API endpoint
2. Enhance the auth service with token refresh and persistence capabilities
3. Create a ConnectRPC interceptor for transparent auth header injection with automatic retry on 401

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | N/A | N/A | All required packages already exist in the project (`connectrpc.com/connect`, `github.com/imroc/req/v3`) |

---

## Step 1: Create Auth API Port Interface

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/auth_port.go`: Reference interface from CLI

### `/Users/jayce/team-attention/cops/daemon/internal/service/auth/outbound/api/auth_port.go`

**Description**:
Define the port interface for auth API operations. The daemon only needs the `RefreshToken` method (unlike CLI which also needs device flow methods).

```go
package api

import "context"

// TokenResult contains new tokens from refresh operation.
type TokenResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    int64
}

// AuthAPIPort defines the interface for auth API operations.
type AuthAPIPort interface {
    // RefreshToken exchanges a refresh token for a new token pair.
    RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)
}
```

**Test Scenarios**: N/A (interface definition only)

---

## Step 2: Create Auth API ConnectRPC Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`: Reference implementation from CLI
- `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/copsapi.go`: API client setup
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go`: Generated auth service client

### `/Users/jayce/team-attention/cops/daemon/internal/service/auth/outbound/api/connectrpc/auth_client.go`

**Description**:
Implement the ConnectRPC adapter for auth API operations. Uses the generated `authv1connect.AuthServiceClient` to call the RefreshToken endpoint.

```go
package connectrpc

import (
    "context"
    "log/slog"

    "connectrpc.com/connect"

    "github.com/team-attention/cops/daemon/internal/platform/setup"
    "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
    authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

// AuthAPIClient implements AuthAPIPort using ConnectRPC.
type AuthAPIClient struct {
    logger *slog.Logger
    client authv1connect.AuthServiceClient
}

// NewAuthAPIClient creates a new ConnectRPC auth API client adapter.
func NewAuthAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *AuthAPIClient {
    // Implementation outline:
    // 1. Create logger with component name "auth.api.connectrpc".
    // 2. Create authv1connect.AuthServiceClient using:
    //    - apiClient.StandardHTTPClient() as HTTP client
    //    - cfg.API.URL as base URL
    // 3. Return initialized AuthAPIClient struct.
}

// RefreshToken exchanges a refresh token for a new token pair.
func (c *AuthAPIClient) RefreshToken(ctx context.Context, refreshToken string) (*api.TokenResult, error) {
    // Implementation outline:
    // 1. Create connect.NewRequest with authv1.RefreshTokenReq containing refreshToken.
    // 2. Call c.client.RefreshToken(ctx, req).
    // 3. If error:
    //    a. Log error with slog.Any("error", err).
    //    b. Return nil, err.
    // 4. Extract tokens from resp.Msg.Tokens.
    // 5. Return &api.TokenResult with AccessToken, RefreshToken, ExpiresAt.
}

// Compile-time interface verification.
var _ api.AuthAPIPort = (*AuthAPIClient)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Successful refresh | Valid refresh token | TokenResult with new tokens | Happy path |
| Invalid refresh token | Expired/invalid token | Error from API | Error handling |
| Network error | Valid token, network failure | Network error | Error handling |

---

## Step 3: Enhance Auth Service with Token Refresh and Persistence

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions
- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go`: Current daemon auth service
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go`: Reference implementation with refresh logic

### `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go`

**Description**:
Enhance the existing auth service with:
1. Token refresh capability via AuthAPIPort
2. Persistence of new tokens to auth.json
3. Mutex protection for concurrent refresh operations
4. Cache invalidation after token refresh

```go
package auth

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
)

// AuthState mirrors CLI auth state structure.
type AuthState struct {
    Tokens *TokenInfo `json:"tokens"`
}

// TokenInfo contains token data.
type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    int64  `json:"expiresAt"`
}

// Service manages daemon authentication state.
type Service struct {
    logger      *slog.Logger
    authAPI     api.AuthAPIPort
    authPath    string
    mu          sync.RWMutex
    refreshMu   sync.Mutex // Separate mutex for token refresh to prevent concurrent refresh attempts
    cachedState *AuthState
    lastLoad    time.Time
}

// NewService creates a new daemon auth service.
func NewService(l *slog.Logger, authAPI api.AuthAPIPort, homeDir string) *Service {
    // Implementation outline:
    // 1. Create logger with component name "daemon.auth.service".
    // 2. Construct authPath as filepath.Join(homeDir, ".cops", "auth.json").
    // 3. Return initialized Service struct with authAPI.
}

// GetAccessToken returns current access token, reloading from file if stale.
// Does NOT perform token refresh - use RefreshAccessToken for that.
func (s *Service) GetAccessToken() (string, error) {
    // Implementation outline (UNCHANGED from current implementation):
    // 1. Acquire read lock, check if reload needed (nil cache or >30s since last load).
    // 2. Release read lock.
    // 3. If reload needed, call s.reloadAuthState().
    // 4. Acquire read lock.
    // 5. Check if cachedState or Tokens is nil -> return "not authenticated" error.
    // 6. Check if token expired (ExpiresAt <= now) -> return "access token expired" error.
    // 7. Return access token.
}

// GetRefreshToken returns the current refresh token.
func (s *Service) GetRefreshToken() (string, error) {
    // Implementation outline:
    // 1. Acquire read lock, check if reload needed.
    // 2. Release read lock.
    // 3. If reload needed, call s.reloadAuthState().
    // 4. Acquire read lock.
    // 5. Check if cachedState or Tokens is nil -> return "not authenticated" error.
    // 6. Return refresh token.
}

// RefreshAccessToken uses the refresh token to obtain a new access token.
// Thread-safe: only one refresh operation can occur at a time.
// Returns the new access token on success.
func (s *Service) RefreshAccessToken(ctx context.Context) (string, error) {
    // Implementation outline:
    // 1. Acquire refreshMu lock (prevents concurrent refresh attempts).
    // 2. Defer unlock of refreshMu.
    // 3. Call s.GetRefreshToken() to get current refresh token.
    // 4. If error, return error (not authenticated).
    // 5. Log info "refreshing access token".
    // 6. Call s.authAPI.RefreshToken(ctx, refreshToken).
    // 7. If error:
    //    a. Log error "failed to refresh token".
    //    b. Return error.
    // 8. Create new AuthState with updated tokens from result.
    // 9. Call s.saveAuthState(newState) to persist to file.
    // 10. If save error:
    //     a. Log error "failed to save refreshed tokens".
    //     b. Return error.
    // 11. Acquire write lock on s.mu.
    // 12. Update s.cachedState with newState.
    // 13. Update s.lastLoad to time.Now().
    // 14. Release write lock.
    // 15. Log info "token refreshed successfully".
    // 16. Return new access token.
}

// IsAuthenticated checks if valid auth state exists.
func (s *Service) IsAuthenticated() bool {
    // Implementation outline (UNCHANGED):
    // 1. Call GetAccessToken.
    // 2. Return err == nil.
}

// InvalidateCache forces reload of auth state from file on next access.
func (s *Service) InvalidateCache() {
    // Implementation outline:
    // 1. Acquire write lock.
    // 2. Set s.cachedState to nil.
    // 3. Set s.lastLoad to zero time.
    // 4. Release write lock.
}

// reloadAuthState reloads auth state from file.
func (s *Service) reloadAuthState() error {
    // Implementation outline (UNCHANGED from current implementation):
    // 1. Acquire write lock.
    // 2. Defer unlock.
    // 3. Check if auth file exists -> set cachedState to nil, update lastLoad, return "auth file not found" error.
    // 4. Read file contents.
    // 5. Unmarshal JSON into AuthState.
    // 6. Update cachedState and lastLoad.
    // 7. Return nil.
}

// saveAuthState writes auth state to file with secure permissions.
func (s *Service) saveAuthState(state *AuthState) error {
    // Implementation outline:
    // 1. Get directory path from s.authPath.
    // 2. Create directory with 0700 permissions if not exists.
    // 3. Marshal state to indented JSON.
    // 4. Write to s.authPath with 0600 permissions.
    // 5. Return any error.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| GetAccessToken - valid token | Cached valid token | Access token string | Happy path |
| GetAccessToken - expired token | Cached expired token | Error "access token expired" | Expiry check |
| GetAccessToken - not authenticated | No auth file | Error "not authenticated" | Auth check |
| GetRefreshToken - valid | Cached state with refresh token | Refresh token string | Happy path |
| GetRefreshToken - not authenticated | No auth file | Error "not authenticated" | Auth check |
| RefreshAccessToken - success | Valid refresh token | New access token | Happy path |
| RefreshAccessToken - API failure | Invalid refresh token | Error from API | API error |
| RefreshAccessToken - save failure | Valid refresh, no write permission | Error saving | Save error |
| RefreshAccessToken - concurrent calls | Multiple goroutines | Only one refresh executes | Mutex protection |
| InvalidateCache | Any state | Cache cleared | Cache invalidation |

---

## Step 4: Create Auth Interceptor for ConnectRPC

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/platform/middleware/`: Reference for middleware patterns in API server
- `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/copsapi.go`: Current API client setup

### `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`

**Description**:
Create a ConnectRPC interceptor that:
1. Adds Authorization header with Bearer token to all outgoing requests (if authenticated)
2. Intercepts 401 Unauthenticated responses
3. Automatically refreshes token and retries the request once
4. Handles edge cases gracefully when token refresh is not possible

**Edge Case Handling**:
- **User not logged in (no auth.json or no tokens)**: Send request WITHOUT Authorization header. Let server return 401. Do NOT retry (no refresh token available).
- **Refresh token is expired/invalid**: Return original 401 error. Do NOT retry indefinitely. Log warning for debugging.

```go
package interceptor

import (
    "context"
    "log/slog"

    "connectrpc.com/connect"

    "github.com/team-attention/cops/daemon/internal/service/auth"
)

// AuthInterceptor adds authentication to ConnectRPC requests and handles token refresh.
type AuthInterceptor struct {
    logger      *slog.Logger
    authService *auth.Service
}

// NewAuthInterceptor creates a new authentication interceptor.
func NewAuthInterceptor(l *slog.Logger, authService *auth.Service) *AuthInterceptor {
    // Implementation outline:
    // 1. Create logger with component name "auth.interceptor".
    // 2. Return initialized AuthInterceptor.
}

// WrapUnary implements connect.Interceptor for unary RPCs.
func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    // Implementation outline:
    // 1. Return a function that:
    //    a. Check if req.Spec().IsClient is true (only intercept client requests).
    //       - If not client request, call next(ctx, req) directly and return.
    //
    //    b. Try to get access token from authService.GetAccessToken().
    //    c. If GetAccessToken returns error (user not logged in or token expired):
    //       - Log debug "no valid access token available, sending request without auth".
    //       - Do NOT set Authorization header.
    //       - Call next(ctx, req) to execute the request.
    //       - If error and connect.CodeOf(err) == connect.CodeUnauthenticated:
    //         * Log warn "request failed with 401, user may need to re-authenticate via CLI".
    //         * Return original error (no retry possible without valid refresh token).
    //       - Return response/error.
    //
    //    d. If GetAccessToken succeeds:
    //       - Set Authorization header: req.Header().Set("Authorization", "Bearer "+token).
    //       - Call next(ctx, req) to execute the request.
    //
    //    e. If error and connect.CodeOf(err) == connect.CodeUnauthenticated:
    //       - Log info "received 401, attempting token refresh".
    //       - Call authService.RefreshAccessToken(ctx).
    //       - If refresh error:
    //         * Log warn "token refresh failed, user may need to re-authenticate via CLI",
    //           with slog.Any("error", refreshErr).
    //         * Return original 401 error (don't retry indefinitely).
    //       - Update request header with new token.
    //       - Log debug "retrying request with new token".
    //       - Call next(ctx, req) again (single retry).
    //       - Return retry result.
    //
    //    f. Return original response/error for non-401 errors.
}

// WrapStreamingClient implements connect.Interceptor for streaming client RPCs.
func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
    // Implementation outline:
    // 1. Return a function that:
    //    a. Try to get access token from authService.GetAccessToken().
    //    b. Call next(ctx, spec) to get connection.
    //    c. If GetAccessToken succeeded (no error):
    //       - Set header: conn.RequestHeader().Set("Authorization", "Bearer "+token).
    //    d. If GetAccessToken failed:
    //       - Log debug "no valid access token for streaming request".
    //       - Do NOT set Authorization header (let server decide).
    //    e. Return connection.
    // Note: Streaming does not support automatic retry on 401 (would require reconnection).
}

// WrapStreamingHandler implements connect.Interceptor for streaming handler RPCs.
// This is a no-op for client-side interceptor.
func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
    // Implementation outline:
    // 1. Return next unchanged (no server-side handling needed).
}

// Compile-time interface verification.
var _ connect.Interceptor = (*AuthInterceptor)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Request with valid token | Valid access token | Request with Authorization header, success | Happy path |
| Request with expired token, refresh succeeds | Expired token, valid refresh | Retry succeeds with new token | 401 retry success |
| Request with expired token, refresh fails | Expired token, invalid refresh | Original 401 error returned | 401 retry failure |
| User not logged in (no auth.json) | GetAccessToken fails | Request sent without header, 401 returned | No auth - not logged in |
| User not logged in, server returns 401 | No tokens, server 401 | 401 error with warning log | No auth - 401 handling |
| Non-401 error | Network error | Original error (no retry) | Other errors |
| Streaming with valid token | Valid access token | Stream with Authorization header | Streaming happy path |
| Streaming without auth | GetAccessToken fails | Stream without header | Streaming no auth |

---

## Step 5: Update API Client Setup to Use Interceptor

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-setup.md`: Platform setup patterns
- `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/copsapi.go`: Current API client setup
- `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`: Auth interceptor (from Step 4)

### `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/copsapi.go`

**Description**:
Modify the existing APIClient setup to accept and store the auth interceptor. The interceptor will be used when creating ConnectRPC clients.

```go
package setup

import (
    "net/http"

    "connectrpc.com/connect"
    "github.com/imroc/req/v3"
)

// APIClient is an HTTP client configured for the COps API server.
type APIClient struct {
    *req.Client
    interceptor connect.Interceptor
}

// InitAPIClient creates a new HTTP client for the COps API server.
func InitAPIClient(cfg *Config) *APIClient {
    // Implementation outline (mostly unchanged):
    // 1. Create req.Client with BaseURL and Timeout from config.
    // 2. Return APIClient with Client set, interceptor nil initially.
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIClient) StandardHTTPClient() *http.Client {
    // Implementation outline (UNCHANGED):
    // 1. Return c.Client.GetClient().
}

// WithAuth returns a cloned client with auth header set.
func (c *APIClient) WithAuth(accessToken string) *req.Client {
    // Implementation outline (UNCHANGED):
    // 1. Return c.Client.Clone().SetCommonBearerAuthToken(accessToken).
}

// SetInterceptor sets the ConnectRPC interceptor for authenticated requests.
func (c *APIClient) SetInterceptor(interceptor connect.Interceptor) {
    // Implementation outline:
    // 1. Set c.interceptor = interceptor.
}

// ConnectOptions returns the connect.ClientOption for using the auth interceptor.
// Returns empty slice if no interceptor is set.
func (c *APIClient) ConnectOptions() []connect.ClientOption {
    // Implementation outline:
    // 1. If c.interceptor is nil, return empty slice.
    // 2. Return []connect.ClientOption{connect.WithInterceptors(c.interceptor)}.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| ConnectOptions with interceptor | Interceptor set | Options with interceptor | Interceptor set |
| ConnectOptions without interceptor | No interceptor | Empty options slice | No interceptor |

---

## Step 6: Update Aggregation API Client to Use Interceptor

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`: Current API client

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`

**Description**:
Modify the aggregation API client to use the auth interceptor from APIClient when creating the ConnectRPC client.

```go
package connectrpc

import (
    "context"
    "log/slog"
    "strings"

    "connectrpc.com/connect"

    "github.com/team-attention/cops/daemon/internal/platform/domain"
    "github.com/team-attention/cops/daemon/internal/platform/setup"
    "github.com/team-attention/cops/daemon/internal/platform/util/errutil"
    aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
    logger *slog.Logger
    client aggregationv1connect.AggregationServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
    // Implementation outline:
    // 1. Create AggregationServiceClient using:
    //    - apiClient.StandardHTTPClient() as HTTP client
    //    - cfg.API.URL as base URL
    //    - apiClient.ConnectOptions()... to pass interceptor options
    // 2. Return initialized APIClient.
}

// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
    // Implementation outline (UNCHANGED - interceptor handles auth automatically):
    // 1. Create aggregationv1.SendLogsReq with batch data.
    // 2. Call c.client.SendLogs(ctx, connect.NewRequest(req)).
    // 3. Handle errors (payload too large, etc.).
    // 4. Return result.
}
```

**Test Scenarios**: Existing tests should continue to pass. Auth header injection is handled transparently by interceptor.

---

## Step 7: Update fx Module Registration

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container patterns
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_platform.go`: Platform module
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_log.go`: Log module

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_auth.go` (NEW FILE)

**Description**:
Create a new fx module for auth service dependencies.

```go
package container

import (
    "os"

    "go.uber.org/fx"

    "github.com/team-attention/cops/daemon/internal/service/auth"
    "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
    connectrpc "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api/connectrpc"
)

func newAuthModule() fx.Option {
    return fx.Module("auth",
        // Provide home directory for auth file path
        fx.Provide(func() string {
            // Implementation outline:
            // 1. Call os.UserHomeDir().
            // 2. If error, return ".".
            // 3. Return home directory.
        }),

        // Outbound: AuthAPIPort
        fx.Provide(fx.Annotate(
            connectrpc.NewAuthAPIClient,
            fx.As(new(api.AuthAPIPort)),
        )),

        // Service
        fx.Provide(auth.NewService),
    )
}
```

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_platform.go`

**Description**:
Add auth interceptor initialization to platform module.

```go
package container

import (
    "go.uber.org/fx"

    "github.com/team-attention/cops/daemon/internal/platform/interceptor"
    "github.com/team-attention/cops/daemon/internal/platform/setup"
)

func newPlatformModule() fx.Option {
    return fx.Module("platform",
        // Configuration (root - no dependencies)
        fx.Provide(setup.LoadConfig),

        // Logger (depends on config)
        fx.Provide(setup.InitLogger),

        // SQLite DB (depends on config and logger)
        fx.Provide(setup.InitSQLite),

        // API Client (depends on config)
        fx.Provide(setup.InitAPIClient),

        // Auth Interceptor (depends on logger and auth service)
        fx.Provide(interceptor.NewAuthInterceptor),

        // Target PubSub (depends on logger)
        fx.Provide(setup.InitTargetPubSub),

        // Log Watcher (shared fsnotify.Watcher)
        fx.Provide(setup.InitLogWatcher),

        // Invoke to wire interceptor to API client (side effect)
        fx.Invoke(func(apiClient *setup.APIClient, interceptor *interceptor.AuthInterceptor) {
            // Implementation outline:
            // 1. Call apiClient.SetInterceptor(interceptor).
        }),
    )
}
```

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/application.go`

**Description**:
Add auth module to application composition.

```go
package container

import (
    "time"

    "go.uber.org/fx"
)

// Run creates and runs the fx application.
func Run() {
    fx.New(
        // Modules (auth must come before platform to provide auth.Service for interceptor)
        newAuthModule(),
        newPlatformModule(),
        newConfigModule(),
        newLogModule(),

        // Handler registration (ordered: subscribers -> publishers -> fsnotify)
        fx.Invoke(registerHandlers),

        // Lifecycle timeouts
        fx.StartTimeout(30*time.Second),
        fx.StopTimeout(30*time.Second),
    ).Run()
}
```

**Test Scenarios**: Integration test - verify daemon starts successfully with new module structure.

---

## Dependency Graph

```
                    +------------------+
                    |   application    |
                    +--------+---------+
                             |
        +--------------------+--------------------+
        |                    |                    |
        v                    v                    v
+---------------+   +---------------+   +---------------+
|  authModule   |   |platformModule |   |   logModule   |
+-------+-------+   +-------+-------+   +-------+-------+
        |                   |                   |
        v                   v                   v
+---------------+   +---------------+   +---------------+
| auth.Service  |<--| AuthInterceptor|   |  logwatcher   |
+-------+-------+   |(interceptor)  |   |   .Service    |
        |           +-------+-------+   +-------+-------+
        v                   |                   |
+---------------+           v                   |
| AuthAPIClient |   +---------------+           |
| (ConnectRPC)  |   |   APIClient   |<----------+
+---------------+   |   (setup)     |
                    +---------------+
```

## File Summary

| File | Action | Description |
| :--- | :----- | :---------- |
| `daemon/internal/service/auth/outbound/api/auth_port.go` | CREATE | Port interface for auth API |
| `daemon/internal/service/auth/outbound/api/connectrpc/auth_client.go` | CREATE | ConnectRPC adapter for RefreshToken |
| `daemon/internal/service/auth/auth_service.go` | MODIFY | Add RefreshAccessToken, GetRefreshToken, saveAuthState, InvalidateCache |
| `daemon/internal/platform/interceptor/auth_interceptor.go` | CREATE | ConnectRPC interceptor for auth header injection and 401 retry |
| `daemon/internal/platform/setup/copsapi.go` | MODIFY | Add SetInterceptor, ConnectOptions methods |
| `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` | MODIFY | Use ConnectOptions from APIClient |
| `daemon/cmd/internal/container/module_auth.go` | CREATE | fx module for auth dependencies |
| `daemon/cmd/internal/container/module_platform.go` | MODIFY | Add auth interceptor provider and invoke |
| `daemon/cmd/internal/container/application.go` | MODIFY | Add auth module to composition |

## Execution Order

1. Step 1: Create auth API port interface
2. Step 2: Create auth API ConnectRPC adapter
3. Step 3: Enhance auth service (depends on Step 1 port interface)
4. Step 4: Create auth interceptor (depends on Step 3 auth service)
5. Step 5: Update API client setup (depends on Step 4 interceptor)
6. Step 6: Update aggregation API client (depends on Step 5 API client)
7. Step 7: Update fx module registration (depends on all previous steps)
