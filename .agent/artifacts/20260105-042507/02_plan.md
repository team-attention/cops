# Implementation Plan: Daemon Token Refresh Behavior Fix

## Overview

The Daemon's authentication system cannot automatically refresh expired access tokens. When a token expires, the auth interceptor sends requests without authentication, causing 401 errors for authenticated endpoints.

This plan implements a proper hexagonal architecture approach by:
1. Creating `platform/outbound/authstate/` package with Port interface and filesystem adapter (matching CLI pattern)
2. Moving token refresh logic and auth.json file handling from Service layer to Platform adapter
3. Updating the auth interceptor to use the new `AuthStatePort` and fail fast on missing tokens

## Package Changes

None required. All necessary dependencies already exist in the codebase.

## Implementation Steps

### Step 1: Create AuthState Port Interface

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Port/Adapter naming conventions and patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform outbound package guidelines
- `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/authstate_port.go`: Reference implementation for the port interface

#### `/Users/jayce/team-attention/cops/daemon/internal/platform/outbound/authstate/authstate_port.go`

**Description**:
Create a new port interface for auth state management. This interface provides access to valid access tokens with automatic refresh handling.

```go
package authstate

import "context"

// AuthStatePort defines the interface for accessing authentication state.
// This is a platform-level adapter that can be used by any service needing
// authenticated API access without depending on the auth service directly.
type AuthStatePort interface {
	// GetAccessToken returns a valid access token, refreshing if needed.
	// Returns error if not logged in or token refresh fails.
	GetAccessToken(ctx context.Context) (string, error)
}
```

**Test Scenarios**:

This is an interface definition only - no tests required.

---

### Step 2: Create Filesystem AuthState Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter structure and naming conventions
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Structured logging best practices
- `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem/authstate.go`: Reference implementation for filesystem adapter
- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/outbound/api/auth_port.go`: AuthAPIPort interface for token refresh

#### `/Users/jayce/team-attention/cops/daemon/internal/platform/outbound/authstate/filesystem/authstate.go`

**Description**:
Create a filesystem-based auth state adapter that handles reading/writing `auth.json` and automatic token refresh with 5-minute buffer. This adapter depends on `AuthAPIPort` for the actual refresh API call.

```go
package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
	authapi "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
)

// refreshBuffer is the duration before token expiry to trigger proactive refresh (5 minutes).
const refreshBuffer = int64(300)

// AuthState represents the local authentication state.
type AuthState struct {
	Tokens *TokenInfo `json:"tokens"`
}

// TokenInfo contains token data.
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// FilesystemAuthState implements AuthStatePort using filesystem storage.
type FilesystemAuthState struct {
	logger    *slog.Logger
	authPath  string
	apiClient authapi.AuthAPIPort
	mu        sync.RWMutex
	refreshMu sync.Mutex
}

// NewFilesystemAuthState creates a new filesystem-based auth state adapter.
func NewFilesystemAuthState(l *slog.Logger, apiClient authapi.AuthAPIPort, homeDir string) authstate.AuthStatePort {
	// Implementation outline:
	// 1. Construct auth path as {homeDir}/.cops/auth.json.
	// 2. Create and return FilesystemAuthState with:
	//    - Logger bound with name "platform.authstate.filesystem"
	//    - authPath set to constructed path
	//    - apiClient for token refresh API calls
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	// Implementation outline:
	// 1. Acquire read lock (mu.RLock).
	// 2. Read auth state from file via readAuthState().
	// 3. Release read lock.
	// 4. If error reading state, return error.
	// 5. If state is nil or Tokens is nil:
	//    - Return "not authenticated" error.
	// 6. Calculate current time (now) and token expiry time.
	// 7. If (tokenExpiry - now) > refreshBuffer:
	//    - Token is still valid with buffer, return cached access token.
	// 8. Log info: "access token near expiry or expired, refreshing".
	// 9. Acquire refresh lock (refreshMu.Lock) to prevent concurrent refresh.
	// 10. Double-check: Re-read state and check expiry again (another goroutine may have refreshed).
	//     - If now valid, release refresh lock and return token.
	// 11. Call apiClient.RefreshToken(ctx, state.Tokens.RefreshToken).
	// 12. If refresh fails:
	//     - Log error: "failed to refresh token" with error.
	//     - Release refresh lock.
	//     - Return wrapped error.
	// 13. Update state with new tokens from refresh result.
	// 14. Save updated state to file via saveAuthState().
	// 15. If save fails:
	//     - Log error: "failed to save refreshed tokens" with error.
	//     - Release refresh lock.
	//     - Return wrapped error.
	// 16. Log info: "token refreshed successfully".
	// 17. Release refresh lock.
	// 18. Return new access token.
}

// readAuthState reads auth state from file (must hold mu.RLock or mu.Lock).
func (a *FilesystemAuthState) readAuthState() (*AuthState, error) {
	// Implementation outline:
	// 1. Check if file exists using os.Stat.
	// 2. If file does not exist, return nil, nil (not logged in).
	// 3. Read file contents using os.ReadFile.
	// 4. If read error, return wrapped error.
	// 5. Unmarshal JSON into AuthState struct.
	// 6. If unmarshal error, return wrapped error.
	// 7. Return state and nil error.
}

// saveAuthState writes auth state to file with secure permissions.
func (a *FilesystemAuthState) saveAuthState(state *AuthState) error {
	// Implementation outline:
	// 1. Ensure .cops directory exists using os.MkdirAll with 0700 permissions.
	// 2. If mkdir fails, return wrapped error.
	// 3. Marshal state to JSON with indentation using json.MarshalIndent.
	// 4. If marshal fails, return wrapped error.
	// 5. Write file using os.WriteFile with 0600 permissions.
	// 6. If write fails, return wrapped error.
	// 7. Return nil.
}

// Compile-time interface verification.
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Token valid with buffer (>5min remaining) | Token expires in 10 minutes | Returns cached access token | Happy path, no refresh |
| Token near expiry (<5min remaining) | Token expires in 3 minutes | Calls refresh, returns new token | Proactive refresh |
| Token already expired | Token expired 5 minutes ago | Calls refresh, returns new token | Expired token refresh |
| Not authenticated (no auth file) | auth.json does not exist | Returns "not authenticated" error | Nil state check |
| Not authenticated (nil tokens) | auth.json with null tokens | Returns "not authenticated" error | Nil tokens check |
| Token refresh API call fails | API returns error | Returns wrapped refresh error | Refresh API failure |
| Token refresh save fails | Disk write fails | Returns wrapped save error | Save failure |
| Concurrent refresh (double-check) | Two goroutines call GetAccessToken | Only one refresh occurs | Double-check after lock |
| Auth file read error | Permission denied | Returns wrapped read error | File read failure |
| Auth file parse error | Malformed JSON | Returns wrapped parse error | JSON unmarshal failure |

---

### Step 3: Update Auth Interceptor to Use AuthStatePort

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Structured logging best practices
- `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`: Current interceptor implementation

#### `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go`

**Description**:
Update the interceptor to depend on `AuthStatePort` instead of `*auth.Service`. Modify `WrapUnary` to return error immediately when token cannot be obtained (fail fast). Remove the logic that sends requests without authentication.

```go
package interceptor

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
)

// AuthInterceptor adds authentication to ConnectRPC requests.
type AuthInterceptor struct {
	logger    *slog.Logger
	authState authstate.AuthStatePort
}

// NewAuthInterceptor creates a new authentication interceptor.
func NewAuthInterceptor(l *slog.Logger, authState authstate.AuthStatePort) *AuthInterceptor {
	// Implementation outline:
	// 1. Create AuthInterceptor with:
	//    - Logger bound with name "auth.interceptor"
	//    - authState for token retrieval
	// 2. Return pointer to created interceptor.
}

// WrapUnary implements connect.Interceptor for unary RPCs.
func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Implementation outline:
		// 1. Check if request is a client request (req.Spec().IsClient).
		//    - If not client request, call next(ctx, req) directly.
		// 2. Call authState.GetAccessToken(ctx) to get token.
		// 3. If GetAccessToken returns error:
		//    - Log error: "failed to get access token" with error details.
		//    - Return nil and connect.NewError(connect.CodeUnauthenticated, err).
		//    - DO NOT send request without authentication.
		// 4. Set Authorization header with "Bearer " + token.
		// 5. Execute request with next(ctx, req).
		// 6. Return response and error from request execution.
		//    - NOTE: Remove 401 retry logic since proactive refresh handles expiry.
	}
}

// WrapStreamingClient implements connect.Interceptor for streaming client RPCs.
func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		// Implementation outline:
		// 1. Get connection from next(ctx, spec).
		// 2. Call authState.GetAccessToken(ctx) to get token.
		// 3. If GetAccessToken returns error:
		//    - Log warn: "no valid access token for streaming request" with error.
		//    - Return connection without Authorization header.
		//    - (Streaming connections cannot return error here, so we log and continue)
		// 4. Set Authorization header on connection with "Bearer " + token.
		// 5. Return connection.
	}
}

// WrapStreamingHandler implements connect.Interceptor for streaming handler RPCs.
// This is a no-op for client-side interceptor.
func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// Compile-time interface verification.
var _ connect.Interceptor = (*AuthInterceptor)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid token available | GetAccessToken returns token | Request sent with Authorization header | Happy path |
| Token refresh during GetAccessToken | Token near expiry | Request sent with refreshed token (handled by adapter) | Proactive refresh in adapter |
| GetAccessToken fails (not authenticated) | No auth file | Returns CodeUnauthenticated error | Fail fast on no token |
| GetAccessToken fails (refresh failed) | Refresh API error | Returns CodeUnauthenticated error | Fail fast on refresh error |
| Non-client request | Server-side handler | Passes through without auth | Spec check bypass |
| Streaming with valid token | GetAccessToken returns token | Connection with Authorization header | Streaming happy path |
| Streaming with no token | GetAccessToken fails | Connection without header, logged | Streaming fallback |

---

### Step 4: Update DI Container Registration

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: fx module patterns and `fx.As` usage
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_auth.go`: Current auth module setup
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_platform.go`: Current platform module setup

#### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_platform.go`

**Description**:
Add the `FilesystemAuthState` provider to the platform module and update the `AuthInterceptor` to receive `AuthStatePort` instead of `*auth.Service`.

```go
package container

import (
	"os"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/interceptor"
	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate/filesystem"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Implementation outline:
		// 1. Keep existing providers:
		//    - setup.LoadConfig
		//    - setup.InitLogger
		//    - setup.InitSQLite
		//    - setup.InitAPIClient
		//    - setup.InitTargetPubSub
		//    - setup.InitLogWatcher
		//
		// 2. Provide home directory for auth state adapter:
		//    fx.Provide(func() string {
		//        homeDir, err := os.UserHomeDir()
		//        if err != nil {
		//            return "."
		//        }
		//        return homeDir
		//    }),
		//
		// 3. Add new provider for FilesystemAuthState:
		//    fx.Provide(fx.Annotate(
		//        filesystem.NewFilesystemAuthState,
		//        fx.As(new(authstate.AuthStatePort)),
		//    )),
		//
		// 4. Keep AuthInterceptor provider (constructor signature will change):
		//    fx.Provide(interceptor.NewAuthInterceptor),
		//
		// 5. Keep fx.Invoke for wiring interceptor to API client.
	)
}
```

#### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_auth.go`

**Description**:
Remove the home directory provider (moved to platform module). Keep the `AuthAPIPort` provider since it's still needed by the authstate adapter. Remove `auth.Service` if no longer needed.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
	connectrpc "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api/connectrpc"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// Implementation outline:
		// 1. Remove home directory provider (moved to platform module).
		//
		// 2. Keep AuthAPIPort provider:
		//    fx.Provide(fx.Annotate(
		//        connectrpc.NewAuthAPIClient,
		//        fx.As(new(api.AuthAPIPort)),
		//    )),
		//
		// 3. Remove auth.NewService (no longer needed - token management moved to platform adapter).
	)
}
```

**Test Scenarios**:

DI container configuration - verified through successful application startup.

---

### Step 5: Remove Auth Service

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go`: Current auth service

#### Delete: `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go`

**Description**:
Since token refresh and auth.json management has moved to the platform adapter, and the interceptor no longer uses `*auth.Service`, this file is no longer needed. Delete it entirely.

**Files to Delete**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go`

**Note**: The `outbound/api/` subdirectory containing `AuthAPIPort` and `connectrpc.AuthAPIClient` must be kept since the platform adapter depends on them.

---

## Summary of Changes

| File | Change Type | Description |
| :--- | :---------- | :---------- |
| `daemon/internal/platform/outbound/authstate/authstate_port.go` | Create | New port interface for auth state access |
| `daemon/internal/platform/outbound/authstate/filesystem/authstate.go` | Create | Filesystem adapter with token refresh logic |
| `daemon/internal/platform/interceptor/auth_interceptor.go` | Modify | Use `AuthStatePort` instead of `*auth.Service`; fail fast on missing token; remove 401 retry logic |
| `daemon/cmd/internal/container/module_platform.go` | Modify | Add home directory provider; Add `FilesystemAuthState` provider with `fx.As` |
| `daemon/cmd/internal/container/module_auth.go` | Modify | Remove home directory provider; Remove `auth.NewService` |
| `daemon/internal/service/auth/auth_service.go` | Delete | Token management moved to platform adapter |

## Execution Order

1. **Step 1**: Create `authstate_port.go` - Define the port interface
2. **Step 2**: Create `filesystem/authstate.go` - Implement the adapter with token refresh
3. **Step 3**: Modify `auth_interceptor.go` - Update to use `AuthStatePort`
4. **Step 4**: Modify container modules - Wire the new adapter, move home directory provider
5. **Step 5**: Delete `auth_service.go` - Remove redundant code
6. **Step 6**: Verify compilation - Run `go build ./daemon/...`

## Quality Verification

After implementation:
- [ ] `go build ./daemon/...` succeeds
- [ ] `go vet ./daemon/...` reports no issues
- [ ] Token refresh is logged at Info level when near expiry
- [ ] Failed refresh is logged at Error level
- [ ] Interceptor returns error (not sends without auth) when token unavailable
- [ ] Auth.json file handling is isolated to `platform/outbound/authstate/filesystem/`
- [ ] Service layer does not directly manage file I/O for auth state
