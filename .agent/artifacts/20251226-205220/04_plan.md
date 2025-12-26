# Implementation Plan

## Overview
Remove unused CollectorHTTPClient and CollectorPort from CLI, fix API URL default port from 8081 to 8080, and update SyncProject to return "not implemented" error.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| N/A | N/A | N/A | This is a removal/cleanup task - no new packages needed |

## Architecture Decisions

### Decision 1: Remove Collector Infrastructure Entirely
**Choice**: Delete all collector-related code rather than deprecating or disabling
**Rationale**: The collector endpoint was never implemented on the API server (API implements `AggregationService.SendLogs`, not `CollectorService.SendRecords`). The code has no function and adds confusion.

### Decision 2: SyncProject Returns NotImplemented Error
**Choice**: Keep the `SyncProject` method signature but return `errutil.NotImplementedf("sync is not yet implemented")`
**Rationale**: Maintains API compatibility for the `--sync` flag while providing clear feedback that the feature is not available. The error message is user-friendly and indicates future availability.

### Decision 3: Keep ProjectPort and APIHTTPClient
**Choice**: Retain `api.ProjectPort` and `httpclient.APIHTTPClient` - only remove collector-specific code
**Rationale**: The project registration API (`RegisterProject`) is actively used and working. Only the collector-related code is dead.

## Implementation Steps

### Step 1: Fix API URL Default Port in Config

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/platform/setup/config/config.go` (modify)

**Changes**:

1. Remove `CollectorConfig` struct (lines 32-36):
```go
// DELETE THIS ENTIRE BLOCK
type CollectorConfig struct {
	URL     string
	Timeout time.Duration
}
```

2. Remove `Collector CollectorConfig` field from `Config` struct (line 15):
```go
// BEFORE
type Config struct {
	App       AppConfig
	Logging   LoggingConfig
	Collector CollectorConfig  // DELETE THIS LINE
	API       APIConfig
	Daemon    DaemonConfig
}

// AFTER
type Config struct {
	App     AppConfig
	Logging LoggingConfig
	API     APIConfig
	Daemon  DaemonConfig
}
```

3. Remove collector defaults (lines 65-66):
```go
// DELETE THESE TWO LINES
v.SetDefault("collector.url", "http://localhost:8080")
v.SetDefault("collector.timeout", "30s")
```

4. Change API URL default from 8081 to 8080 (line 67):
```go
// BEFORE
v.SetDefault("api.url", "http://localhost:8081")

// AFTER
v.SetDefault("api.url", "http://localhost:8080")
```

5. Remove collector initialization from Config struct literal (lines 80-82):
```go
// DELETE THIS BLOCK FROM THE Config{} LITERAL
Collector: CollectorConfig{
	URL:     v.GetString("collector.url"),
	Timeout: v.GetDuration("collector.timeout"),
},
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Default API URL | No env vars | `cfg.API.URL == "http://localhost:8080"` | Default value |
| Custom API URL | `COPS_API_URL=http://custom:9000` | `cfg.API.URL == "http://custom:9000"` | Env override |
| Config struct has no Collector field | N/A | Compile success without CollectorConfig | Type safety |

### Step 2: Remove CollectorHTTPClient from httpclient Package

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/platform/setup/httpclient/httpclient.go` (modify)

**Changes**:

1. Remove `CollectorHTTPClient` struct (lines 11-14):
```go
// DELETE THIS BLOCK
type CollectorHTTPClient struct {
	*req.Client
}
```

2. Remove `InitCollectorHTTPClient` function (lines 21-28):
```go
// DELETE THIS ENTIRE FUNCTION
func InitCollectorHTTPClient(cfg *config.Config) *CollectorHTTPClient {
	client := req.C().
		SetBaseURL(cfg.Collector.URL).
		SetTimeout(cfg.Collector.Timeout)

	return &CollectorHTTPClient{Client: client}
}
```

3. Remove `StandardHTTPClient` method for CollectorHTTPClient (lines 41-43):
```go
// DELETE THIS METHOD
func (c *CollectorHTTPClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}
```

**Final file structure**:
```go
package httpclient

import (
	"net/http"

	"github.com/imroc/req/v3"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
)

// APIHTTPClient is an HTTP client configured for the API server.
type APIHTTPClient struct {
	*req.Client
}

// InitAPIHTTPClient creates a new HTTP client for the API server.
func InitAPIHTTPClient(cfg *config.Config) *APIHTTPClient {
	client := req.C().
		SetBaseURL(cfg.API.URL).
		SetTimeout(cfg.API.Timeout)

	return &APIHTTPClient{Client: client}
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIHTTPClient) StandardHTTPClient() *http.Client {
	return c.Client.GetClient()
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Compile without CollectorHTTPClient | N/A | Build success | Type removal |
| APIHTTPClient still works | Valid config | Client with correct base URL | Functionality preserved |

### Step 3: Delete Collector Port and Client Files

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/collector_port.go` (DELETE)
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go` (DELETE)

**Changes**:

Use `rm` to delete these files entirely:
```bash
rm /Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/collector_port.go
rm /Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Files deleted | N/A | Files no longer exist | File removal |
| Build still succeeds | N/A | No compilation errors | No broken imports |

### Step 4: Update Tracking Service - Remove Collector Dependency

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (modify)

**Changes**:

1. Remove `api` import if only used for CollectorPort (check first - ProjectPort is also in api package, so import stays):
```go
// KEEP THIS IMPORT - ProjectPort is still used
"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
```

2. Remove `collector` field from Service struct (line 33):
```go
// BEFORE
type Service struct {
	logger *slog.Logger

	configRepo config.ConfigPort
	parser     parser.ParserPort
	collector  api.CollectorPort  // DELETE THIS LINE
	project    api.ProjectPort
}

// AFTER
type Service struct {
	logger *slog.Logger

	configRepo config.ConfigPort
	parser     parser.ParserPort
	project    api.ProjectPort
}
```

3. Remove `collector` parameter from NewService (line 42):
```go
// BEFORE
func NewService(
	l *slog.Logger,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	collector api.CollectorPort,  // DELETE THIS LINE
	project api.ProjectPort,
) *Service {

// AFTER
func NewService(
	l *slog.Logger,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	project api.ProjectPort,
) *Service {
```

4. Remove collector initialization in NewService constructor (line 49):
```go
// BEFORE
return &Service{
	logger:     l.With(slog.String("name", "tracking.service")),
	configRepo: configRepo,
	parser:     parser,
	collector:  collector,  // DELETE THIS LINE
	project:    project,
}

// AFTER
return &Service{
	logger:     l.With(slog.String("name", "tracking.service")),
	configRepo: configRepo,
	parser:     parser,
	project:    project,
}
```

5. Replace entire `SyncProject` method body (lines 310-352) with not-implemented error:
```go
// SyncProject syncs session records for a project to the collector.
// This feature is not yet implemented.
func (s *Service) SyncProject(ctx context.Context, projectID domain.ID) error {
	return errutil.NotImplementedf("sync is not yet implemented")
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| NewService with 4 params | Valid dependencies | Service created | Constructor |
| SyncProject called | Any projectID | Error: "sync is not yet implemented" | Not implemented path |
| AddProject with --sync | params.Sync=true | Warning logged, add succeeds | Sync warning path |

### Step 5: Update DI Container - Remove Collector Registration

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` (modify)
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` (modify)

**Changes in module_platform.go**:

1. Remove `httpclient.InitCollectorHTTPClient` from providers (line 22):
```go
// BEFORE
providers := []interface{}{
	// Configuration (root - no dependencies)
	config.LoadConfig,

	// Logger (depends on config)
	logger.InitLogger,

	// HTTP clients (depends on config)
	httpclient.InitCollectorHTTPClient,  // DELETE THIS LINE
	httpclient.InitAPIHTTPClient,

	// Cobra root command (depends on logger)
	setup_cobra.NewRootCommand,
}

// AFTER
providers := []interface{}{
	// Configuration (root - no dependencies)
	config.LoadConfig,

	// Logger (depends on config)
	logger.InitLogger,

	// HTTP clients (depends on config)
	httpclient.InitAPIHTTPClient,

	// Cobra root command (depends on logger)
	setup_cobra.NewRootCommand,
}
```

**Changes in module_tracking.go**:

1. Remove collector client registration (lines 31-35):
```go
// DELETE THIS ENTIRE BLOCK
if err := c.Provide(
	connectrpc.NewCollectorClient,
	dig.As(new(api.CollectorPort)),
); err != nil {
	return err
}
```

2. Remove unused import if `api.CollectorPort` was the only usage from api package:
```go
// CHECK: api package is still needed for api.ProjectPort
// KEEP the import:
"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
```

**Final module_tracking.go structure**:
```go
package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"
)

// newTrackingModule registers all tracking-related providers.
func newTrackingModule(c *dig.Container) error {
	// Outbound adapters with dig.As for interface casting
	if err := c.Provide(
		filesystem.NewFilesystemConfigAdapter,
		dig.As(new(config.ConfigPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		jsonl.NewJSONLParser,
		dig.As(new(parser.ParserPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		connectrpc.NewProjectClient,
		dig.As(new(api.ProjectPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(tracking.NewService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cobra.NewTrackingCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Container builds successfully | N/A | No DI errors | All providers registered |
| Service receives 4 dependencies | N/A | Service created with 4 ports | Constructor injection |

### Step 6: Update Documentation

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/doc/config.md` (modify)

**Changes**:

1. Remove collector rows from environment variable table (lines 17-18):
```markdown
<!-- DELETE THESE TWO ROWS -->
| `COPS_COLLECTOR_URL` | `http://localhost:8080` | Collector server URL |
| `COPS_COLLECTOR_TIMEOUT` | `30s` | Collector server timeout |
```

2. Change API URL default from 8081 to 8080 (line 19):
```markdown
<!-- BEFORE -->
| `COPS_API_URL` | `http://localhost:8081` | API server URL |

<!-- AFTER -->
| `COPS_API_URL` | `http://localhost:8080` | API server URL |
```

3. Remove collector key mapping example (line 40):
```markdown
<!-- DELETE THIS LINE -->
- `collector.url` -> `COPS_COLLECTOR_URL`
```

4. Remove CollectorConfig from struct example (lines 56, 70-73):
```go
// BEFORE (line 56)
type Config struct {
    App       AppConfig
    Logging   LoggingConfig
    Collector CollectorConfig  // DELETE THIS LINE
    API       APIConfig
}

// DELETE ENTIRE CollectorConfig STRUCT (lines 70-73)
type CollectorConfig struct {
    URL     string
    Timeout time.Duration
}
```

5. Remove collector usage examples (lines 95-104):
```markdown
<!-- DELETE THIS ENTIRE SECTION -->
### Collector server URL setting
\`\`\`bash
export COPS_COLLECTOR_URL=http://collector.example.com:8080
cops add . --sync
\`\`\`

### Collector timeout setting
\`\`\`bash
export COPS_COLLECTOR_TIMEOUT=60s
cops add . --sync
\`\`\`
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Documentation matches code | N/A | No collector references, API URL is 8080 | Documentation accuracy |

## Execution Order

1. **Step 1**: Fix config.go (no dependencies)
2. **Step 2**: Update httpclient.go (depends on Step 1 for config changes)
3. **Step 3**: Delete collector port/client files (no dependencies, can run parallel with Step 2)
4. **Step 4**: Update tracking_service.go (depends on Step 3 for import cleanup)
5. **Step 5**: Update DI container modules (depends on Steps 2, 3, 4)
6. **Step 6**: Update documentation (no code dependencies, can run last)

**Recommended execution**: Steps 1-3 can be done in parallel, then Steps 4-5, then Step 6.

## Notes for Execute Agent

1. **Import Cleanup**: After removing collector code, verify imports are correct. The `api` package import should remain in `tracking_service.go` and `module_tracking.go` because `api.ProjectPort` is still used.

2. **Build Verification**: After each step, run `go build ./cli/...` to ensure no compilation errors.

3. **SyncProject Behavior**: The `--sync` flag still exists in the CLI. When used, `SyncProject` returns an error which is logged as a warning (see `add.go` line 186-189). The add operation still succeeds.

4. **File Deletion**: Use `rm` command for file deletion in Step 3. Do not use Edit tool for deletion.

5. **Korean Documentation**: The `doc/config.md` file is written in Korean. Maintain Korean language when updating.

6. **No Test Files**: This task does not include test file updates. If tests exist that reference collector code, they will fail and need separate handling.
