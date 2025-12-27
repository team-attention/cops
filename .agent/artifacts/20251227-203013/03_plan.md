# Implementation Plan

## Overview

Fix dashboard visibility by adding CORS middleware to the Fiber server and adding diagnostic logging to the dashboard repository. The CORS middleware is the critical fix that blocks all cross-origin requests from the web frontend.

## Selected Packages

| Problem         | Package                                | Context7 ID        | Reason for Selection                                           |
| --------------- | -------------------------------------- | ------------------ | -------------------------------------------------------------- |
| CORS Middleware | github.com/gofiber/fiber/v2/middleware/cors | (Fiber v2 stdlib)  | Native Fiber v2 middleware, already included, zero-config option |
| Logging         | log/slog                               | (Go stdlib)        | Already in use throughout the project                          |

## Architecture Decisions

### Decision 1: CORS Configuration Approach
**Choice**: Rename `ServerConfig` to `HTTPConfig` and include CORS settings within it
**Rationale**:
- Logically groups all HTTP server-related settings (ports, timeouts, CORS) under one config
- More intuitive naming: `HTTPConfig` is clearer than `ServerConfig` for HTTP-specific settings
- CORS is an HTTP concern, not a separate cross-cutting concern
- Avoids creating too many top-level config structs
- The project already uses Fiber v2 (`github.com/gofiber/fiber/v2 v2.52.10`)
- Built-in middleware requires no additional dependencies
- Development mode allows all origins; production mode restricts to specific origins

### Decision 2: CORS Headers for ConnectRPC
**Choice**: Include `Connect-Protocol-Version` in allowed headers
**Rationale**:
- ConnectRPC uses custom headers for protocol negotiation
- Missing this header will cause ConnectRPC requests to fail even with CORS enabled
- Standard headers: `Origin`, `Content-Type`, `Accept`, `Connect-Protocol-Version`

### Decision 3: Debug Logging Strategy
**Choice**: Add debug-level logging at key points in dashboard repository
**Rationale**:
- Uses existing `slog.Logger` infrastructure
- Debug level logs are disabled in production by default
- Provides visibility into data flow without affecting performance

### Decision 4: Project Field Backfill
**Choice**: Defer project field population to a future task
**Rationale**:
- CORS is the primary blocker - fix it first
- Dashboard uses `mongoutil.Get` which returns empty strings for missing fields (no crash)
- Project name/path display is a cosmetic issue, not a functional blocker
- Can be addressed after confirming CORS fix works

## Implementation Steps

### Step 1: Rename ServerConfig to HTTPConfig and Add CORS Settings

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go` (modify)

**Changes**:

1. Rename `ServerConfig` to `HTTPConfig` and add CORS fields:

```go
// HTTPConfig holds HTTP server and CORS settings.
type HTTPConfig struct {
	Port            int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" envDefault:"30s"`

	// CORS settings
	CORSAllowOrigins string `env:"CORS_ALLOW_ORIGINS" envDefault:"*"`
	CORSAllowMethods string `env:"CORS_ALLOW_METHODS" envDefault:"GET,POST,PUT,DELETE,OPTIONS,PATCH"`
	CORSAllowHeaders string `env:"CORS_ALLOW_HEADERS" envDefault:"Origin,Content-Type,Accept,Connect-Protocol-Version"`
}
```

2. Update Config struct (replace `Server ServerConfig` with `HTTP HTTPConfig`):
```go
type Config struct {
	App     AppConfig
	HTTP    HTTPConfig    // Renamed from Server
	Logging LoggingConfig
	MongoDB MongoDBConfig
}
```

3. Update validation function to use `cfg.HTTP.Port` instead of `cfg.Server.Port` (line 82):
```go
// Validate port range
if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
	return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", cfg.HTTP.Port)
}
```

**Test Scenarios**:
| Scenario                   | Input                               | Expected Output                     | Branch Covered     |
| -------------------------- | ----------------------------------- | ----------------------------------- | ------------------ |
| Default config             | No env vars                         | AllowOrigins="*"                    | Default path       |
| Custom origins             | CORS_ALLOW_ORIGINS="http://localhost:5173" | AllowOrigins="http://localhost:5173" | Custom value path |
| Empty headers              | CORS_ALLOW_HEADERS=""               | AllowHeaders=""                     | Empty string path  |

---

### Step 2: Add CORS Middleware to Fiber Server and Update Config References

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go` (modify)
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` (modify - update Port reference)

**Changes**:

Add import for CORS middleware (line 7, after existing imports):
```go
import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
)
```

Update fiber.Config to use `cfg.HTTP` instead of `cfg.Server` and add CORS middleware:
```go
func InitFiber(cfg *config.Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.HTTP.ReadTimeout,     // Changed from cfg.Server
		WriteTimeout:          cfg.HTTP.WriteTimeout,    // Changed from cfg.Server
		DisableStartupMessage: cfg.App.Env == "production",
		ErrorHandler:          newErrorHandler(logger),
	})

	// Middleware
	app.Use(recover.New())

	// CORS middleware - must be before routes and other middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.HTTP.CORSAllowOrigins,  // Use HTTP config
		AllowMethods: cfg.HTTP.CORSAllowMethods,  // Use HTTP config
		AllowHeaders: cfg.HTTP.CORSAllowHeaders,  // Use HTTP config
	}))

	app.Use(requestid.New())

	logger.Info("Fiber app initialized",
		slog.String("name", cfg.App.Name),
		slog.String("env", cfg.App.Env),
		slog.String("cors_origins", cfg.HTTP.CORSAllowOrigins),
	)

	return app
}
```

Update `register_connectrpc.go` to use `cfg.HTTP.Port` instead of `cfg.Server.Port` (line 44):
```go
// In OnStart hook
addr := fmt.Sprintf(":%d", params.Config.HTTP.Port)  // Changed from Server.Port
```

**Test Scenarios**:
| Scenario                      | Input                               | Expected Output                                    | Branch Covered        |
| ----------------------------- | ----------------------------------- | -------------------------------------------------- | --------------------- |
| OPTIONS preflight request     | OPTIONS /dashboard.v1.DashboardService/GetOverview | 200 with CORS headers                              | Preflight handling    |
| Cross-origin POST request     | POST from localhost:5173            | Response with Access-Control-Allow-Origin header   | CORS header injection |
| Same-origin request           | POST from localhost:8080            | Response with CORS headers (no change in behavior) | Pass-through          |

---

### Step 3: Add Debug Logging to Dashboard Repository

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` (modify)

**Changes**:

Add debug logging to `GetOverviewStats` method (after line 37, at start of method):
```go
func (r *MongoDashboardRepository) GetOverviewStats(ctx context.Context) (*repository.OverviewStats, error) {
	r.logger.Debug("GetOverviewStats called")
	stats := &repository.OverviewStats{}
	// ... existing code ...
```

Add debug logging after project count query (after line 78):
```go
	stats.ProjectCount = int32(projectCount)
	r.logger.Debug("overview stats counts",
		slog.Int64("projectCount", projectCount),
	)
```

Add debug logging to `ListProjects` method (at start of method, after line 122):
```go
func (r *MongoDashboardRepository) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	r.logger.Debug("ListProjects called",
		slog.Int32("page", params.Page),
		slog.Int32("pageSize", params.PageSize),
	)
	// ... existing code ...
```

Add debug logging after parsing facet result (after line 211):
```go
	// Extract total count
	var totalCount int64
	if len(facetResult.Metadata) > 0 {
		totalCount = facetResult.Metadata[0].TotalCount
	}
	r.logger.Debug("ListProjects results",
		slog.Int64("totalCount", totalCount),
		slog.Int("dataCount", len(facetResult.Data)),
	)
```

**Test Scenarios**:
| Scenario              | Input                 | Expected Output                              | Branch Covered     |
| --------------------- | --------------------- | -------------------------------------------- | ------------------ |
| Empty database        | No projects/sessions  | Debug log: "projectCount: 0"                 | Empty result path  |
| Data exists           | 5 projects, 10 sessions | Debug log: "projectCount: 5"               | Data present path  |
| Pagination            | Page 2, PageSize 10   | Debug log: "page: 2, pageSize: 10"           | Pagination path    |

---

### Step 4: Verify CORS Headers in Response

**Files to Create/Modify**:
- No code changes - manual verification step

**Verification Process**:

1. Start the API server with debug logging enabled:
   ```bash
   cd /Users/jayce/team-attention/cops/api && LOGGING_LEVEL=debug make dev
   ```

2. Test CORS preflight from command line:
   ```bash
   curl -X OPTIONS http://localhost:8080/dashboard.v1.DashboardService/GetOverview \
     -H "Origin: http://localhost:5173" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: Content-Type,Connect-Protocol-Version" \
     -v
   ```

3. Expected response headers:
   ```
   Access-Control-Allow-Origin: *
   Access-Control-Allow-Methods: GET,POST,PUT,DELETE,OPTIONS,PATCH
   Access-Control-Allow-Headers: Origin,Content-Type,Accept,Connect-Protocol-Version
   ```

4. Test actual POST request:
   ```bash
   curl -X POST http://localhost:8080/dashboard.v1.DashboardService/GetOverview \
     -H "Origin: http://localhost:5173" \
     -H "Content-Type: application/json" \
     -H "Connect-Protocol-Version: 1" \
     -d '{}' \
     -v
   ```

5. Check API server logs for debug output showing data counts.

**Test Scenarios**:
| Scenario                    | Input                          | Expected Output                              | Branch Covered     |
| --------------------------- | ------------------------------ | -------------------------------------------- | ------------------ |
| CORS preflight succeeds     | OPTIONS request                | 200 with all CORS headers                    | Preflight          |
| CORS POST succeeds          | POST with Origin header        | 200 with response data and CORS headers      | Cross-origin POST  |
| Debug logs appear           | Any API call with LOGGING_LEVEL=debug | Debug logs in server output                  | Logging            |

---

## Execution Order

1. **Step 1: Rename ServerConfig to HTTPConfig and Add CORS Settings** (no dependencies)
   - Modify `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go`

2. **Step 2: Add CORS Middleware and Update Config References** (depends on Step 1)
   - Modify `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go`
   - Modify `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`
   - Requires HTTPConfig with CORS fields from Step 1

3. **Step 3: Add Debug Logging** (no dependencies, can run parallel with Step 1-2)
   - Modify `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

4. **Step 4: Verify** (depends on Steps 1-3)
   - Manual testing to confirm CORS works and data flows correctly

## Notes for Execute Agent

1. **Import Order**: When adding the CORS import, maintain alphabetical order within import groups (stdlib first, then external packages).

2. **Middleware Order Matters**: CORS middleware MUST be added before request handlers are registered. The current code structure adds middleware before routes, so adding after `recover.New()` is correct.

3. **No go.mod Changes**: The CORS middleware is part of Fiber v2 which is already a dependency. No need to run `go get`.

4. **Test with Browser DevTools**: After deployment, open browser DevTools Network tab and check that:
   - No CORS errors in console
   - Response headers include `Access-Control-Allow-Origin`
   - API requests from frontend succeed

5. **Environment Variable Override**: For production, set `CORS_ALLOW_ORIGINS` to specific origin(s):
   ```
   CORS_ALLOW_ORIGINS=https://dashboard.example.com
   ```

6. **ConnectRPC Headers**: The `Connect-Protocol-Version` header is critical for ConnectRPC. Without it in `AllowHeaders`, ConnectRPC requests will fail even with CORS enabled.

7. **Debug Logging Level**: The debug logs added in Step 3 only appear when `LOGGING_LEVEL=debug` is set. In production (`LOGGING_LEVEL=info`), these logs are hidden.

8. **Line Number References**: The line numbers referenced are based on the files read during planning. If files have been modified since, adjust insertions accordingly.

## Rollback Plan

If the CORS fix causes issues:

1. **Quick Disable**: Set environment variable to empty origins:
   ```
   CORS_ALLOW_ORIGINS=""
   ```

2. **Full Rollback**: Revert the modified files:
   - `config/config.go` - revert HTTPConfig back to ServerConfig, remove CORS fields
   - `server/fiber.go` - remove CORS import and middleware, revert cfg.HTTP to cfg.Server
   - `register_connectrpc.go` - revert cfg.HTTP.Port to cfg.Server.Port
