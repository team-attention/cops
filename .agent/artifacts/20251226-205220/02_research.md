# Research Report

## Mode
General Research

## Request Summary
Fix port mismatch between CLI and API server where CLI is configured to connect to port 8081 but API server runs on port 8080. Additionally, verify if CLI needs an HTTP client for Daemon, or if only the API client is needed.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File                                                                | Reason                                                           |
| ------------------------------------------------------------------- | ---------------------------------------------------------------- |
| `/Users/jayce/team-attention/cops/cli/internal/platform/setup/config/config.go` | Contains the CLI configuration with incorrect `api.url` default (line 67) |
| `/Users/jayce/team-attention/cops/doc/config.md`                    | Documentation that needs to be updated (line 19)                 |
| `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go` | Confirms API server runs on port 8080 (line 27)                  |
| `/Users/jayce/team-attention/cops/cli/internal/platform/setup/httpclient/httpclient.go` | Shows two HTTP clients: CollectorHTTPClient and APIHTTPClient    |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-setup.md` | Rules for configuration patterns                                 |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`     | Rules for Go struct definitions                                  |

## Package Candidates

No new packages are required for this task. This is a configuration fix using existing infrastructure.

## Technical Constraints

1. API server port is fixed at 8080 (defined in `api/internal/platform/setup/config/config.go` line 27: `env:"SERVER_PORT" envDefault:"8080"`)
2. Daemon correctly uses port 8080 (defined in `daemon/internal/platform/setup/config.go` line 37: `env:"COPS_API_URL" envDefault:"http://localhost:8080"`)
3. CLI uses Viper with `COPS_` prefix for environment variables
4. Must maintain backward compatibility with existing environment variable overrides (`COPS_API_URL`)

## Similar Implementations Found

### Example 1: Daemon Configuration (Correct Implementation)
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/config.go:37`
- **Relevance**: Shows the correct API URL default of `http://localhost:8080`

```go
// APIConfig holds API server connection settings.
type APIConfig struct {
    URL     string        `env:"COPS_API_URL" envDefault:"http://localhost:8080"`
    Timeout time.Duration `env:"COPS_API_TIMEOUT" envDefault:"30s"`
}
```

### Example 2: API Server Port Configuration (Source of Truth)
- **File**: `/Users/jayce/team-attention/cops/api/internal/platform/setup/config/config.go:27`
- **Relevance**: Defines the actual port the API server runs on (8080)

```go
// ServerConfig holds HTTP server settings.
type ServerConfig struct {
    Port            int           `env:"SERVER_PORT" envDefault:"8080"`
    // ...
}
```

## Investigation Results

### Question: Does CLI need a Daemon HTTP client?

**Answer: NO - CLI does NOT have a Daemon HTTP client and does NOT need one.**

**Findings:**
1. CLI has only TWO HTTP clients in `/Users/jayce/team-attention/cops/cli/internal/platform/setup/httpclient/httpclient.go`:
   - `CollectorHTTPClient` - Used for sending session records to API's collector endpoint
   - `APIHTTPClient` - Used for project registration with API server

2. CLI's `DaemonConfig` (lines 44-47 in `cli/internal/platform/setup/config/config.go`) only contains:
   - `BinaryPath string` - Path to the daemon binary for installation purposes

3. CLI's daemon service (`/Users/jayce/team-attention/cops/cli/internal/service/daemon/daemon_service.go`) uses an `InstallerPort` interface to:
   - Install the daemon as a system service
   - Uninstall the daemon service
   - Check the daemon service status

4. Daemon does NOT expose HTTP handlers:
   - Searched for `http.Handler`, `http.Server`, `fiber` - no results found
   - Daemon only has inbound workers (fsnotify-based file watchers)
   - Daemon's role is watching/monitoring logs, not serving HTTP requests

**Conclusion:** No changes needed for Daemon HTTP client since it does not exist. The original requirements assumed there might be a separate Daemon HTTP client, but CLI only communicates with the Daemon through system service management (install/uninstall/status), not HTTP calls.

## Summary of Required Changes

### Change 1: Fix CLI API URL Default
- **File**: `/Users/jayce/team-attention/cops/cli/internal/platform/setup/config/config.go`
- **Line**: 67
- **Change**: `v.SetDefault("api.url", "http://localhost:8081")` -> `v.SetDefault("api.url", "http://localhost:8080")`

### Change 2: Update Documentation
- **File**: `/Users/jayce/team-attention/cops/doc/config.md`
- **Line**: 19
- **Change**: Update `COPS_API_URL` default from `http://localhost:8081` to `http://localhost:8080`

## Additional Information for Planning

1. **Scope is minimal**: Only two files need to be modified
2. **No structural changes**: This is a simple default value fix
3. **No Daemon HTTP client exists**: The investigation confirmed CLI does not have and does not need an HTTP client to communicate with Daemon
4. **Both changes are straightforward**: Single-line edits in each file
5. **Testing**: After changes, verify CLI can connect to API server on port 8080

## Port Configuration Summary

| Component | Configuration | Default Port | Status |
| --------- | ------------- | ------------ | ------ |
| API Server | `SERVER_PORT` | 8080 | Correct (source of truth) |
| Daemon | `COPS_API_URL` | http://localhost:8080 | Correct |
| CLI Collector URL | `COPS_COLLECTOR_URL` | http://localhost:8080 | Correct |
| CLI API URL | `COPS_API_URL` | http://localhost:8081 | **WRONG - needs fix** |
| Documentation | doc/config.md | http://localhost:8081 | **WRONG - needs fix** |
