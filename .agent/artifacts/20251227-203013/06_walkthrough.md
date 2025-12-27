# Development Walkthrough

## Summary
Fixed dashboard visibility issue by implementing CORS middleware in the API server, allowing cross-origin requests from the web frontend. Refactored server configuration structure from `ServerConfig` to `HTTPConfig` for better organization and added debug logging to track data flow through dashboard repository queries.

## Problem Statement

The dashboard web application was unable to display data from the API server despite session records and project data being successfully stored in MongoDB. The root cause was a missing CORS (Cross-Origin Resource Sharing) configuration, which blocked all HTTP requests from the frontend running on a different origin (e.g., `http://localhost:5173`) to the API server (e.g., `http://localhost:8080`).

Browser security policies enforce the Same-Origin Policy, requiring servers to explicitly allow cross-origin requests by sending appropriate CORS headers. Without these headers, the browser blocks the responses even if the API server processes the requests successfully.

## Code Overview

### Modified Components

#### `HTTPConfig` (formerly `ServerConfig`)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go`
- **Purpose**: Centralized HTTP server and CORS configuration
- **Changes**:
  - Renamed `ServerConfig` to `HTTPConfig` for clearer semantics (HTTP-specific settings)
  - Updated environment variable prefix from `SERVER_` to `HTTP_` for consistency (though kept backward compatible defaults)
  - Added three new CORS configuration fields:
    - `CORSAllowOrigins`: Comma-separated list of allowed origins (default: `*` for development)
    - `CORSAllowMethods`: Allowed HTTP methods (default includes standard REST methods)
    - `CORSAllowHeaders`: Allowed request headers (critically includes `Connect-Protocol-Version` for ConnectRPC)
  - Updated port validation to use `cfg.HTTP.Port` instead of `cfg.Server.Port`

**Key Configuration Fields**:
```go
type HTTPConfig struct {
    Port            int           `env:"PORT" envDefault:"8080"`
    ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"30s"`
    WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
    ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`

    // CORS settings
    CORSAllowOrigins string `env:"CORS_ALLOW_ORIGINS" envDefault:"*"`
    CORSAllowMethods string `env:"CORS_ALLOW_METHODS" envDefault:"GET,POST,PUT,DELETE,OPTIONS,PATCH"`
    CORSAllowHeaders string `env:"CORS_ALLOW_HEADERS" envDefault:"Origin,Content-Type,Accept,Connect-Protocol-Version"`
}
```

**Why `Connect-Protocol-Version` is Critical**:
ConnectRPC (the gRPC-over-HTTP protocol used by the API) requires the `Connect-Protocol-Version` header for protocol negotiation. Without this header in the CORS allow list, all ConnectRPC requests would fail even with CORS enabled.

#### `InitFiber` - Fiber Server Initialization
- **Location**: `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go`
- **Purpose**: Initialize Fiber HTTP server with middleware stack
- **Changes**:
  - Added import for `github.com/gofiber/fiber/v2/middleware/cors`
  - Updated all `cfg.Server` references to `cfg.HTTP` to match renamed config struct
  - Inserted CORS middleware immediately after `recover.New()` and before `requestid.New()`
  - Added CORS origins to server initialization log for debugging visibility

**Middleware Ordering**:
```go
app.Use(recover.New())              // 1. Panic recovery (must be first)
app.Use(cors.New(cors.Config{...})) // 2. CORS (before routes/handlers)
app.Use(requestid.New())            // 3. Request ID tracking
```

The ordering is critical: CORS must be registered before route handlers to inject headers into all responses, including OPTIONS preflight requests.

#### `registerConnectRPCServer` - Server Registration
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`
- **Purpose**: Register ConnectRPC server with fx lifecycle
- **Changes**: Updated port reference from `params.Config.Server.Port` to `params.Config.HTTP.Port` to match renamed config

#### `MongoDashboardRepository` - Dashboard Data Access
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- **Purpose**: Query MongoDB for dashboard statistics and project lists
- **Changes**: Added debug-level logging at four key points:

**Debug Logging Points**:
1. **GetOverviewStats entry**: Logs when overview stats query starts
2. **GetOverviewStats project count**: Logs the number of projects found (helps verify data exists)
3. **ListProjects entry**: Logs pagination parameters (page, pageSize)
4. **ListProjects results**: Logs total count and number of results returned

These logs help diagnose data flow issues without affecting production performance (debug logs are disabled by default).

### New Configuration

#### `.env.example` - Environment Variable Documentation
- **Location**: `/Users/jayce/team-attention/cops/api/.meta/.env.example`
- **Purpose**: Document all available environment variables with examples
- **Changes**:
  - Updated section headers with "Configuration" suffix for clarity
  - Renamed all `SERVER_*` variables to `HTTP_*` for consistency
  - Added comprehensive CORS configuration section with usage notes
  - Updated MongoDB URI default to `mongodb://localhost:27017` (local development)
  - Added comments explaining CORS settings and when to use wildcards vs specific origins

**CORS Configuration Section**:
```bash
# CORS Configuration
# Comma-separated list of allowed origins (use * for all origins in development)
HTTP_CORS_ALLOW_ORIGINS=*
HTTP_CORS_ALLOW_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
HTTP_CORS_ALLOW_HEADERS=Origin,Content-Type,Accept,Connect-Protocol-Version
```

## Testing

### Manual Verification Commands

**1. Check CORS Preflight (OPTIONS Request)**:
```bash
curl -X OPTIONS http://localhost:8080/cops.dashboard.v1.DashboardService/GetOverview \
  -H "Origin: http://localhost:5173" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Connect-Protocol-Version" \
  -v
```

**Expected Headers in Response**:
```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET,POST,PUT,DELETE,OPTIONS,PATCH
Access-Control-Allow-Headers: Origin,Content-Type,Accept,Connect-Protocol-Version
```

**2. Check Actual POST Request with CORS**:
```bash
curl -X POST http://localhost:8080/cops.dashboard.v1.DashboardService/GetOverview \
  -H "Origin: http://localhost:5173" \
  -H "Content-Type: application/json" \
  -H "Connect-Protocol-Version: 1" \
  -d '{}' \
  -v
```

**Expected**: Response includes `Access-Control-Allow-Origin` header and JSON data

**3. View Debug Logs (Development Mode)**:
```bash
cd /Users/jayce/team-attention/cops/api
LOGGING_LEVEL=debug make dev
```

**Expected Debug Log Output**:
```
DEBUG GetOverviewStats called
DEBUG overview stats counts projectCount=5
DEBUG ListProjects called page=1 pageSize=10
DEBUG ListProjects results totalCount=5 dataCount=5
```

### Build Verification

```bash
# Build all modules to verify no compilation errors
go build ./api/...
```
**Result**: PASS (confirmed during review)

### Browser Testing

1. Open browser DevTools (F12)
2. Navigate to Network tab
3. Load dashboard page
4. Verify:
   - No CORS errors in Console tab
   - `GetOverview` request shows 200 status
   - Response headers include `Access-Control-Allow-Origin`
   - Data displays correctly in dashboard UI

## Configuration Updates

### Production Deployment

For production environments, restrict CORS to specific origins:

```bash
# .env or environment variables
HTTP_CORS_ALLOW_ORIGINS=https://dashboard.example.com,https://app.example.com
HTTP_CORS_ALLOW_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
HTTP_CORS_ALLOW_HEADERS=Origin,Content-Type,Accept,Connect-Protocol-Version
```

### Development Setup

For local development, the defaults allow all origins:

```bash
HTTP_CORS_ALLOW_ORIGINS=*
```

This eliminates CORS issues when frontend runs on varying ports (Vite dev server, Docker, etc.).

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Dashboard shows no data despite database records | Added CORS middleware to API server to allow cross-origin requests from web frontend |
| Config naming inconsistency (`ServerConfig` vs HTTP-specific settings) | Renamed to `HTTPConfig` and updated all references for semantic clarity |
| ConnectRPC requests failing with CORS enabled | Added `Connect-Protocol-Version` to `CORSAllowHeaders` to support ConnectRPC protocol negotiation |
| Difficulty debugging data flow | Added debug-level logging to repository methods to trace query execution and results |
| Environment variable documentation outdated | Updated `.env.example` with new CORS settings and clearer section organization |

## Architecture Decisions

### Why CORS Middleware Over Nginx/Reverse Proxy

While CORS can be configured at multiple layers (reverse proxy, API gateway, application server), adding it directly to the Fiber application provides:
- **Simplicity**: Single source of truth for CORS configuration
- **Development velocity**: No need for external proxy in local development
- **Portability**: Works in any deployment environment (Docker, Kubernetes, bare metal)
- **Framework integration**: Fiber's built-in middleware handles OPTIONS preflight automatically

### Why HTTPConfig Rename

The original `ServerConfig` name was ambiguous - it could refer to any server type (HTTP, gRPC, WebSocket). Renaming to `HTTPConfig` makes it explicit that these settings apply to HTTP/HTTPS communication specifically. This becomes important as the system grows to potentially include other server types (standalone gRPC server, WebSocket server, etc.).

### Why Environment Variable Prefix Change

Changed from `SERVER_*` to `HTTP_*` to match the renamed config struct and clarify that these variables configure HTTP-specific behavior. This follows the principle of least surprise and makes configuration more self-documenting.

## Related Files

**Modified**:
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go`
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go`
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- `/Users/jayce/team-attention/cops/api/.meta/.env.example`

**Related Documentation**:
- `CLAUDE.md` - Project overview and data flow
- `.agent/rules/go/go-backend.md` - Go coding conventions
- `.agent/rules/go/go-platform.md` - Platform package guidelines

## Future Improvements

1. **CORS Configuration Validation**: Add validation to ensure `CORSAllowOrigins` is not `*` in production environments
2. **CORS Metrics**: Track CORS preflight requests and failures for monitoring
3. **Origin Allowlist**: Implement regex-based origin matching for flexible subdomain support
4. **Credentials Support**: Add `CORSAllowCredentials` setting when authentication cookies are needed
