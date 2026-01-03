# Review Result

**Status**: Changes Required

## Request Summary

Code review identified rule violations related to CLI dependency injection. The `NewAuthAPIClient` constructor requires primitive types (`string`, `*http.Client`) that the dig framework cannot resolve. Constructors must accept typed dependencies like `*config.Config` and `*httpclient.APIHTTPClient` that are registered in the container.

## Acceptance Criteria

- [ ] Update `NewAuthAPIClient` constructor signature to accept `*config.Config` and `*httpclient.APIHTTPClient` instead of `string` and `*http.Client`
- [ ] Update `NewAuthAPIClient` implementation to extract `baseURL` from config and `*http.Client` from `APIHTTPClient`
- [ ] Verify the auth module registration in `module_auth.go` works correctly after the fix

## Scope

### In Scope
- Fix `NewAuthAPIClient` constructor to use typed dependencies
- Ensure constructor follows the same pattern as `NewProjectClient`

### Out of Scope
- Changes to other CLI constructors (they already follow correct patterns)
- Changes to service logic or behavior
- Adding new functionality

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
| ---- | ---- | ---- | ----- | ------------- |
| `cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` | 20 | `go/go-outbound.md`, `go/go-dig-container.md` | Constructor accepts `*http.Client` directly - dig cannot resolve this generic type | Change parameter to `*httpclient.APIHTTPClient` and call `.StandardHTTPClient()` internally |
| `cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` | 20 | `go/go-outbound.md`, `go/go-dig-container.md` | Constructor accepts `string` directly for baseURL - dig cannot resolve primitive types | Change parameter to `*config.Config` and extract `cfg.API.URL` internally |

## Detailed Analysis

### Current Implementation (Incorrect)

**File:** `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`

```go
// Line 20 - INCORRECT: Accepts primitive types that dig cannot resolve
func NewAuthAPIClient(l *slog.Logger, httpClient *http.Client, baseURL string) *AuthAPIClient {
	return &AuthAPIClient{
		logger: l.With(slog.String("name", "auth.api.connectrpc")),
		client: authv1connect.NewAuthServiceClient(httpClient, baseURL),
	}
}
```

**Problem:** The dig framework cannot resolve:
- `*http.Client` - generic stdlib type, not registered in container
- `string` - primitive type, cannot be distinguished from other strings

### Correct Pattern (Reference Implementation)

**File:** `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`

```go
// Line 24 - CORRECT: Accepts typed dependencies that dig can resolve
func NewProjectClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *ProjectClient {
	logger := l.With(slog.String("name", "tracking.api.connectrpc"))

	client := projectv1connect.NewProjectServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &ProjectClient{
		logger: logger,
		client: client,
	}
}
```

### Suggested Fix

Update `NewAuthAPIClient` to follow the same pattern:

```go
package connectrpc

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

type AuthAPIClient struct {
	logger *slog.Logger
	client authv1connect.AuthServiceClient
}

func NewAuthAPIClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *AuthAPIClient {
	return &AuthAPIClient{
		logger: l.With(slog.String("name", "auth.api.connectrpc")),
		client: authv1connect.NewAuthServiceClient(httpClient.StandardHTTPClient(), cfg.API.URL),
	}
}

// ... rest of the file unchanged ...
```

## Container Types Available

The following types are registered in the dig container via `module_platform.go`:

| Type | Provider | Description |
| ---- | -------- | ----------- |
| `*config.Config` | `config.LoadConfig` | Application configuration |
| `*slog.Logger` | `logger.InitLogger` | Structured logger |
| `*httpclient.APIHTTPClient` | `httpclient.InitAPIHTTPClient` | Typed HTTP client for API |
| `*cobra.Command` | `setup_cobra.NewRootCommand` | Root CLI command |

**Note:** `*http.Client` and `string` are NOT registered and cannot be injected.

## Files Reviewed

- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go` - VIOLATION FOUND
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go` - OK (uses interface type)
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` - OK (correct pattern)
- `/Users/jayce/team-attention/cops/cli/internal/service/daemon/outbound/installer/kardianos/installer.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/daemon/daemon_service.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/inbound/cli/cobra/handler.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/daemon/inbound/cli/cobra/handler.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/platform/setup/cobra/root.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go` - OK
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go` - OK
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` - Reference for available types
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_auth.go` - Container registration
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` - Reference for correct pattern
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_daemon.go` - Reference for correct pattern

## Rules Applied

- [`.agent/rules/common.md`](.agent/rules/common.md) - General code rules
- [`.agent/rules/workflow.md`](.agent/rules/workflow.md) - Workflow rules
- [`.agent/rules/go/go-outbound.md`](.agent/rules/go/go-outbound.md) - Outbound adapter guidelines (constructor dependency injection)
- [`.agent/rules/go/go-dig-container.md`](.agent/rules/go/go-dig-container.md) - Dig container patterns
- [`.agent/rules/go/go-hexagonal-layout.md`](.agent/rules/go/go-hexagonal-layout.md) - Hexagonal architecture rules
- [`.agent/rules/go/go-logging-conventions.md`](.agent/rules/go/go-logging-conventions.md) - Logger injection pattern

## Additional Context

- Requirements document: `.agent/artifacts/20260103-012308/01_requirements.md`
- Plan document: `.agent/artifacts/20260103-012308/02_plan.md`
- Review triggered by user feedback identifying dependency injection issues in CLI module
- Only 1 file requires changes out of 16 reviewed
