# Implementation Plan: Fix Missing Authorization Header in `cops add .`

## Overview

The `cops add .` command fails with "unauthenticated: missing authorization header" because the `ProjectClient.RegisterProject` method does not set the Authorization header when calling the API. This plan fixes the issue by:

1. Creating a shared platform outbound adapter (`platform/outbound/authstate/`) for access token retrieval
2. Updating `ProjectPort` interface to accept `accessToken` as a separate parameter
3. Updating `ProjectClient.RegisterProject` to accept the token and set Authorization header
4. Injecting the new `AuthStatePort` into `tracking.Service` to fetch the access token

**Architecture Compliance**: This plan follows the Service Independence rule from `go-hexagonal-layout.md`:
- Services cannot directly import other services
- Shared functionality goes in `platform/outbound/` as a platform adapter
- The new `authstate` adapter reads `~/.cops/auth.json` and handles token refresh via the auth API

---

## Step 1: Create Platform Outbound AuthState Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform package guidelines
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter pattern
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go`: Reference for token fetching logic
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/auth_port.go`: Auth API port for token refresh

### Create Directory Structure

```
cli/internal/platform/outbound/authstate/
├── authstate_port.go       # Port interface
└── filesystem/
    └── authstate.go        # Filesystem implementation
```

### `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/authstate_port.go`

**Description**:
Define the port interface for accessing authentication state. This provides a clean abstraction for getting access tokens that can be used across services without violating service independence.

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

**Test Scenarios**: N/A (interface definition only)

### `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem/authstate.go`

**Description**:
Implement the AuthStatePort using filesystem storage for auth state and the auth API for token refresh. This implementation mirrors the logic from `auth.Service.GetAccessToken` but as a platform-level adapter.

```go
package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	authapi "github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
)

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
}

// NewFilesystemAuthState creates a new filesystem-based auth state adapter.
func NewFilesystemAuthState(l *slog.Logger, apiClient authapi.AuthAPIPort) authstate.AuthStatePort {
	// 1. Get user home directory, default to "." if error.
	// 2. Construct auth path as ~/.cops/auth.json.
	// 3. Create and return FilesystemAuthState with logger binding.
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	// 1. Read auth state from file:
	//    a. Check if auth file exists, return "not logged in" error if not.
	//    b. Read and unmarshal JSON file.
	//    c. Validate state has tokens, return "not logged in" error if nil.
	// 2. Check token expiry:
	//    a. Calculate time until expiry (tokenExpiry - now).
	//    b. If more than 300 seconds (5 min buffer), return current access token.
	// 3. Refresh token if near expiry:
	//    a. Log that token is being refreshed.
	//    b. Call apiClient.RefreshToken(ctx, refreshToken).
	//    c. If error, log and return error.
	//    d. Update state with new tokens.
	//    e. Save updated state to file.
	//    f. Log success.
	// 4. Return access token.
}

// readAuthState reads auth state from file.
func (a *FilesystemAuthState) readAuthState() (*AuthState, error) {
	// 1. Check if file exists using os.Stat.
	// 2. If not exists, return nil, nil (not logged in).
	// 3. Read file contents.
	// 4. Unmarshal JSON into AuthState.
	// 5. Return state.
}

// saveAuthState writes auth state to file with secure permissions.
func (a *FilesystemAuthState) saveAuthState(state *AuthState) error {
	// 1. Ensure .cops directory exists (os.MkdirAll with 0700).
	// 2. Marshal state to JSON with indentation.
	// 3. Write file with 0600 permissions.
}

// Compile-time interface verification
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid token not expired | Auth file exists, token valid | Returns access token | Happy path |
| Token near expiry | Auth file exists, token expires in < 300s | Refreshes and returns new token | Token refresh |
| Not logged in (no file) | Auth file does not exist | Error: "not logged in" | No auth file |
| Not logged in (no tokens) | Auth file exists, tokens nil | Error: "not logged in" | Empty tokens |
| Token refresh fails | Token expired, API refresh fails | Error from API | Refresh error |
| File read error | Auth file corrupted | Error: "failed to read/parse" | File error |

---

## Step 2: Register AuthState Adapter in DI Container

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_auth.go`: Auth module DI patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-dig-container.md`: Dig container patterns

### `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` (New File)

**Description**:
Create a new module to register platform-level outbound adapters. This keeps platform adapters organized separately from service modules.

```go
package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem"
)

// newPlatformModule registers platform-level outbound adapters.
func newPlatformModule(c *dig.Container) error {
	// 1. Provide FilesystemAuthState as AuthStatePort.
	//    Pattern: c.Provide(filesystem.NewFilesystemAuthState, dig.As(new(authstate.AuthStatePort)))
}
```

**Note**: The `filesystem.NewFilesystemAuthState` constructor requires `authapi.AuthAPIPort` which is already provided by the auth module. Dig will automatically resolve this dependency.

### Update Container Initialization

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/container.go`: Container initialization

The container initialization file needs to call `newPlatformModule`. Check if there's a central place where modules are registered and add the platform module call there.

**Test Scenarios**: N/A (DI wiring - verified at runtime)

---

## Step 3: Update ProjectPort Interface Signature

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/user/outbound/api/user_port.go`: Reference showing correct interface pattern with `accessToken` as separate parameter

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`

**Description**:
Update the `ProjectPort` interface to add `accessToken string` as a separate parameter to `RegisterProject`. The `RegisterProjectParams` struct remains unchanged - the access token goes in the HTTP header, not the request body.

**Current Interface** (line 41-45):
```go
// ProjectPort defines the interface for project API operations.
type ProjectPort interface {
	// RegisterProject registers a project or returns existing project ID if already registered.
	// Performs duplicate detection using remote URLs and optional existing project ID.
	RegisterProject(ctx context.Context, params RegisterProjectParams) (*RegisterProjectResult, error)
}
```

**Updated Interface**:
```go
// ProjectPort defines the interface for project API operations.
type ProjectPort interface {
	// RegisterProject registers a project or returns existing project ID if already registered.
	// Performs duplicate detection using remote URLs and optional existing project ID.
	// Requires valid access token for authentication.
	RegisterProject(ctx context.Context, accessToken string, params RegisterProjectParams) (*RegisterProjectResult, error)
}
```

**Changes**:
- Add `accessToken string` as the second parameter (after `ctx`, before `params`)
- Add comment noting authentication requirement

**Test Scenarios**: N/A (interface definition only)

---

## Step 4: Update ProjectClient Implementation

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/user/outbound/api/connectrpc/user_client.go`: Reference implementation showing correct Authorization header pattern

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`

**Description**:
Update the `RegisterProject` method to accept `accessToken string` as a parameter and set the Authorization header, following the exact pattern from `UserAPIClient.GetMe`.

**Current Implementation** (lines 38-67):
```go
// RegisterProject registers a project or returns existing project ID if already registered.
func (c *ProjectClient) RegisterProject(ctx context.Context, params api.RegisterProjectParams) (*api.RegisterProjectResult, error) {
	req := connect.NewRequest(&projectv1.RegisterProjectReq{
		ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
		ActualRemoteUrl:     params.ActualRemoteURL,
		ExistingProjectId:   params.ExistingProjectID,
		Name:                params.Name,
		IsGitProject:        params.IsGitProject,
		OrganizationId:      params.OrganizationID,
	})

	resp, err := c.client.RegisterProject(ctx, req)
	// ... rest of implementation
}
```

**Updated Implementation**:
```go
// RegisterProject registers a project or returns existing project ID if already registered.
func (c *ProjectClient) RegisterProject(ctx context.Context, accessToken string, params api.RegisterProjectParams) (*api.RegisterProjectResult, error) {
	// 1. Create the request with project data from params.
	// 2. Set the Authorization header using Bearer token format.
	//    Pattern: req.Header().Set("Authorization", "Bearer "+accessToken)
	// 3. Call the gRPC client's RegisterProject method.
	// 4. If error, log and return error.
	// 5. Map response to RegisterProjectResult and return.
}
```

**Specific Changes**:
1. Add `accessToken string` parameter after `ctx context.Context`
2. After creating the request (`req := connect.NewRequest(...)`), add:
   ```go
   req.Header().Set("Authorization", "Bearer "+accessToken)
   ```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid token and project data | Valid accessToken, valid project params | Success with ProjectID | Happy path |
| Empty access token | accessToken = "" | API returns 401 error | Error handling (server-side rejection) |
| API connection error | Valid token, unreachable server | Error returned | Error handling |
| Duplicate project | Valid token, existing project | Success with existing ProjectID, IsNew=false | Idempotent registration |

---

## Step 5: Inject AuthStatePort and Fetch Token in Tracking Service

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/user/user_service.go`: Reference showing auth service injection pattern
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service layer patterns

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**Description**:
Update the Service struct to include `AuthStatePort` dependency (not `auth.Service` directly), update the constructor to accept it, and modify `AddProject` to fetch the access token before calling the API.

#### Import Addition

Add the authstate port import:

```go
import (
	// ... existing imports ...
	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
)
```

#### Service Struct Update

**Current Struct** (lines 36-43):
```go
// Service provides tracking operations.
type Service struct {
	logger *slog.Logger

	configRepo config.ConfigPort
	parser     parser.ParserPort
	project    api.ProjectPort
}
```

**Updated Struct**:
```go
// Service provides tracking operations.
type Service struct {
	logger *slog.Logger

	authState  authstate.AuthStatePort
	configRepo config.ConfigPort
	parser     parser.ParserPort
	project    api.ProjectPort
}
```

#### Constructor Update

**Current Constructor** (lines 45-58):
```go
// NewService creates a new tracking service.
func NewService(
	l *slog.Logger,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	project api.ProjectPort,
) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "tracking.service")),
		configRepo: configRepo,
		parser:     parser,
		project:    project,
	}
}
```

**Updated Constructor**:
```go
// NewService creates a new tracking service.
func NewService(
	l *slog.Logger,
	authState authstate.AuthStatePort,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	project api.ProjectPort,
) *Service {
	// 1. Create logger with service name binding.
	// 2. Initialize Service struct with all dependencies including authState.
	// 3. Return pointer to Service.
}
```

#### AddProject Method Update

**Location**: Insert token fetching after line 114 (after getting git remote URLs) and before line 117 (the API call).

**Insert this code block** (after `actualURL = gitutil.GetActualRemoteURL(projectPath)` block):

```go
	// Get access token for API authentication
	accessToken, err := s.authState.GetAccessToken(ctx)
	if err != nil {
		s.logger.Error("failed to get access token", slog.Any("error", err))
		return nil, errutil.Internalf("authentication failed: %v", err)
	}
```

**Update the API call** (lines 117-124):

**Current Call**:
```go
	result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
		ConfiguredRemoteURL: configuredURL,
		ActualRemoteURL:     actualURL,
		ExistingProjectID:   existingProjectID,
		Name:                name,
		IsGitProject:        isGitProject,
		OrganizationID:      params.OrganizationID,
	})
```

**Updated Call**:
```go
	result, err := s.project.RegisterProject(ctx, accessToken, api.RegisterProjectParams{
		ConfiguredRemoteURL: configuredURL,
		ActualRemoteURL:     actualURL,
		ExistingProjectID:   existingProjectID,
		Name:                name,
		IsGitProject:        isGitProject,
		OrganizationID:      params.OrganizationID,
	})
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid path with auth | Valid path, logged in user | Success, project registered | Happy path |
| Not logged in | Valid path, no auth state | Error: "authentication failed: not logged in" | Auth error |
| Token refresh needed | Valid path, expired token | Success after token refresh | Token refresh |
| API error with existing local ID | Valid path, API unreachable, has local config | Success with local ID | Fallback path |
| API error without local ID | Valid path, API unreachable, no local config | Error: "cannot register project" | Error handling |
| Git project | Git repo path | Success with git URLs | Git detection |
| Non-git project (--no-git) | Path with NoGit=true | Success without git URLs | NoGit flag |

---

## Step 6: Verify DI Wiring

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/container.go`: Container initialization order

### Verification Steps

1. Ensure `newPlatformModule` is called in the container initialization
2. Ensure module order is correct:
   - Auth module must be registered before platform module (provides `AuthAPIPort`)
   - Platform module must be registered before tracking module (provides `AuthStatePort`)

**Expected Module Order**:
```go
// In container initialization
newAuthModule(c)      // Provides auth.Service and AuthAPIPort
newPlatformModule(c)  // Provides AuthStatePort (depends on AuthAPIPort)
newTrackingModule(c)  // Uses AuthStatePort
```

**Test Scenarios**: N/A (DI wiring - verified at compile time and runtime)

---

## Implementation Order

1. **Step 1**: Create `platform/outbound/authstate/` adapter (new files, no dependencies)
2. **Step 2**: Register adapter in DI container (depends on Step 1)
3. **Step 3**: Update `ProjectPort` interface to add `accessToken string` parameter
4. **Step 4**: Update `ProjectClient.RegisterProject` to accept token and set header (depends on Step 3)
5. **Step 5**: Update `tracking.Service` to inject `AuthStatePort` and fetch token (depends on Steps 1-4)
6. **Step 6**: Verify DI wiring works at compile time (depends on all previous steps)

---

## Files Changed Summary

| File | Change Type | Description |
|:-----|:------------|:------------|
| `cli/internal/platform/outbound/authstate/authstate_port.go` | Create | Define `AuthStatePort` interface |
| `cli/internal/platform/outbound/authstate/filesystem/authstate.go` | Create | Implement filesystem-based auth state adapter |
| `cli/cmd/internal/container/module_platform.go` | Create | Register platform outbound adapters in DI |
| `cli/cmd/internal/container/container.go` | Modify | Call `newPlatformModule` in initialization |
| `cli/internal/service/tracking/outbound/api/project_port.go` | Modify | Add `accessToken string` parameter to interface |
| `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` | Modify | Add `accessToken string` parameter, set Authorization header |
| `cli/internal/service/tracking/tracking_service.go` | Modify | Add `authState AuthStatePort` field, update constructor, fetch token |

---

## Architecture Compliance

This plan follows the Service Independence rule:

| Rule | Compliance |
|:-----|:-----------|
| Services cannot import other services | YES - `tracking` does not import `auth` service |
| Platform outbound for shared functionality | YES - `authstate` is in `platform/outbound/` |
| Port/Adapter pattern | YES - `AuthStatePort` interface with `FilesystemAuthState` implementation |
| DI for dependencies | YES - `AuthStatePort` injected via constructor |

---

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (no candidates)
- [x] Execution order is clear and dependencies are explicit
- [x] AccessToken is a separate method parameter, NOT in RegisterProjectParams struct
- [x] Pattern matches UserAPIPort.GetMe and UserAPIClient.GetMe exactly
- [x] No cross-service imports (tracking does not import auth service)
- [x] Shared functionality in platform/outbound/ following existing patterns
