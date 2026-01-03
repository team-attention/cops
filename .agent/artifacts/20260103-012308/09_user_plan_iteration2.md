# Implementation Plan: Fix CLI Dependency Injection Issue

## Overview

Fix the dependency injection issue in the CLI auth module by updating the `NewAuthAPIClient` constructor to accept typed dependencies (`*config.Config` and `*httpclient.APIHTTPClient`) instead of primitive types (`string` and `*http.Client`) that the dig container cannot resolve. This follows the same pattern already established in `NewProjectClient`.

## Package Changes

No package changes required. All necessary dependencies are already available in the codebase.

## Implementation Steps

### Step 1: Update AuthAPIClient Constructor Signature

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md` - Outbound adapter constructor dependency injection rules
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-dig-container.md` - Dig container patterns and type resolution rules
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` - Reference implementation showing correct pattern
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` - Current implementation to be fixed

#### `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`

**Description**:
Update the constructor to follow the correct dependency injection pattern by accepting `*config.Config` and `*httpclient.APIHTTPClient` instead of `*http.Client` and `string`. Extract the required values internally to maintain the same initialization behavior.

**Changes**:

1. **Add Import Statements**:
```go
import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)
```

2. **Update Constructor Signature and Implementation**:
```go
// NewAuthAPIClient creates a new ConnectRPC auth client.
//
// Dependencies:
// - l: Logger for structured logging
// - cfg: Configuration containing API server URL
// - httpClient: Typed HTTP client wrapper providing standard http.Client
//
// Returns:
// - *AuthAPIClient: Initialized auth API client ready for use
func NewAuthAPIClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *AuthAPIClient {
	// Implementation outline:
	// 1. Create logger with service-specific context using l.With()
	//    - Add "name" field with value "auth.api.connectrpc"
	// 2. Extract standard http.Client from typed wrapper
	//    - Call httpClient.StandardHTTPClient() to get *http.Client
	// 3. Extract base URL from config
	//    - Use cfg.API.URL to get API server URL string
	// 4. Create ConnectRPC service client
	//    - Call authv1connect.NewAuthServiceClient with extracted http.Client and base URL
	// 5. Return initialized AuthAPIClient struct
	//    - Set logger field to scoped logger from step 1
	//    - Set client field to service client from step 4
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid dependencies | `logger`, `cfg.API.URL="http://localhost:8080"`, `httpClient` | `*AuthAPIClient` with initialized logger and client | Happy path |
| Logger with parent context | `logger.With(slog.String("parent", "test"))`, `cfg`, `httpClient` | Logger includes both parent context and "name" field | Logger binding |

**Note**: Constructor does not perform validation as typed dependencies guarantee correct types. The dig container ensures all dependencies are non-nil at injection time.

### Step 2: Verify Container Registration

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_auth.go` - Auth module container registration
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` - Platform module showing available types

#### Verification Only - No Changes Required

**Description**:
Verify that the container registration in `module_auth.go` correctly uses `dig.Provide` with `dig.As` for interface conversion. No changes should be needed as the registration pattern is already correct.

**Expected Registration Pattern**:
```go
func newAuthModule(c *dig.Container) error {
	// API client registration with dig.As for interface casting
	if err := c.Provide(
		connectrpc.NewAuthAPIClient,
		dig.As(new(api.AuthAPIPort)),
	); err != nil {
		return err
	}

	// Service registration
	if err := c.Provide(auth.NewService); err != nil {
		return err
	}

	// CLI handler registration with dig.As + dig.Group
	return c.Provide(
		cobra.NewAuthCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
```

**Types Available in Container** (from `module_platform.go`):
- `*config.Config` - Provided by `config.LoadConfig`
- `*slog.Logger` - Provided by `logger.InitLogger`
- `*httpclient.APIHTTPClient` - Provided by `httpclient.InitAPIHTTPClient`
- `*cobra.Command` - Provided by `setup_cobra.NewRootCommand`

**Verification Steps**:
1. Confirm `connectrpc.NewAuthAPIClient` is registered with `dig.Provide`
2. Confirm `dig.As(new(api.AuthAPIPort))` is used for interface conversion
3. Confirm no manual wrapper functions are used
4. Confirm the constructor will receive the correct typed dependencies automatically

## Quality Checklist

Before implementation is complete, verify:
- [x] Constructor signature matches reference pattern from `NewProjectClient`
- [x] Constructor accepts `*config.Config` and `*httpclient.APIHTTPClient` (typed dependencies)
- [x] Constructor no longer accepts `string` or `*http.Client` (primitive types)
- [x] Import statements include `config` and `httpclient` packages
- [x] Implementation extracts `cfg.API.URL` and `httpClient.StandardHTTPClient()` internally
- [x] Logger binding uses `l.With(slog.String("name", "auth.api.connectrpc"))`
- [x] Container registration pattern in `module_auth.go` uses `dig.As` correctly
- [x] No changes to service logic or behavior
- [x] Test scenarios cover constructor initialization

## Reference Implementation

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` (Lines 24-36)

This file demonstrates the correct pattern:
- Accepts `*config.Config` and `*httpclient.APIHTTPClient` as parameters
- Extracts `cfg.API.URL` for base URL
- Calls `httpClient.StandardHTTPClient()` for standard http.Client
- Binds logger with service-specific context
- Returns initialized struct with all dependencies

## Architecture Alignment

This fix ensures the auth module follows the same dependency injection patterns as other CLI modules:
- **Platform Setup**: `config` and `httpclient` are initialized in `module_platform.go`
- **Typed Dependencies**: All outbound adapters accept typed wrappers, not primitives
- **Container Resolution**: dig can resolve all dependencies without manual wrappers
- **Consistent Pattern**: All ConnectRPC clients follow the same constructor signature pattern
