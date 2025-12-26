# Requirements

## Request Summary
Fix port mismatch between CLI and API server where CLI is configured to connect to port 8081 but API server runs on port 8080. Additionally, remove CLI's CollectorHTTPClient since it was designed to call a "Collector server" (likely Daemon), but Daemon does not expose HTTP handlers and API server does not have a Collector endpoint.

## Acceptance Criteria

- [ ] CLI configuration default for `api.url` is changed from `http://localhost:8081` to `http://localhost:8080` to match API server's port
- [ ] Documentation (`doc/config.md`) is updated to reflect correct API URL port (8080 instead of 8081)
- [ ] Remove `CollectorConfig` and `CollectorHTTPClient` from CLI since Daemon does not expose HTTP handlers
- [ ] Remove `collector` field from tracking service and all references to `CollectorPort`
- [ ] Update `SyncProject` to clarify that sync is not currently implemented (no collector endpoint exists)

## Scope

### In Scope
- Fix CLI API URL configuration: change port from 8081 to 8080
- Update documentation to reflect correct API port (8080)
- Remove CollectorConfig and CollectorHTTPClient from CLI
- Remove collector-related code from tracking service
- Update SyncProject method to return "not implemented" error (no collector endpoint exists)
- Clean up DI container to remove collector client injection

### Out of Scope
- Changing the API server's port from 8080 to 8081
- Implementing a Collector endpoint in API server (future feature)
- Changing environment variable names beyond removing COPS_COLLECTOR_URL
- Updating web/frontend configuration (already correctly uses 8080)
- Modifying Daemon architecture

## Constraints
- API server port is fixed at 8080 (defined in `api/internal/platform/setup/config/config.go`)
- Daemon already correctly connects to 8080 via `COPS_API_URL` default
- Must maintain backward compatibility with existing environment variable overrides
- Follow project's configuration patterns (Viper for CLI, env tags for API/Daemon)

## Additional Context

### Current State Analysis

**API Server:**
- Runs on port 8080 by default (`SERVER_PORT` in `api/internal/platform/setup/config/config.go`)

**CLI:**
- Has TWO URLs configured:
  - `collector.url` → `http://localhost:8080` (SHOULD BE REMOVED - no collector exists)
  - `api.url` → `http://localhost:8081` (WRONG - should be 8080)
- Uses Viper for configuration with `COPS_` prefix
- Has `CollectorHTTPClient` that is unused (calls non-existent collector server)

**Daemon:**
- Daemon's role is to watch Claude Code JSONL logs and send data to API
- Does NOT expose HTTP handlers (only has inbound workers: fsnotify, pubsub)
- Daemon itself has an HTTP client to call the API server (correctly configured to port 8080)

**Collector Investigation:**
- CLI's `CollectorHTTPClient` was designed to call a "Collector server"
- No Collector endpoint exists in API server
- Daemon does not expose HTTP handlers
- SyncProject method tries to call collector, will always fail

**Documentation:**
- `doc/config.md` incorrectly shows `COPS_API_URL` default as `http://localhost:8081`

### Files to Modify

1. `cli/internal/platform/setup/config/config.go` - Remove CollectorConfig, fix API URL from 8081 to 8080
2. `cli/internal/platform/setup/httpclient/httpclient.go` - Remove CollectorHTTPClient
3. `cli/internal/service/tracking/tracking_service.go` - Remove collector field and update SyncProject
4. `cli/internal/service/tracking/outbound/api/collector_port.go` - DELETE (no longer needed)
5. `cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go` - DELETE (no longer needed)
6. `cli/cmd/internal/container/module_platform.go` - Remove CollectorHTTPClient initialization
7. `cli/cmd/internal/container/module_tracking.go` - Remove collector client from DI
8. `doc/config.md` - Update documentation to show correct port (8080), remove COPS_COLLECTOR_URL

### Decision Points

**Question 1: What to do with SyncProject method?**

Options:
1. **Remove SyncProject entirely** - Since no collector endpoint exists
2. **Keep SyncProject stub** - Return "not implemented" error for future implementation
3. **Implement in API** - Add collector endpoint to API server (out of scope)

**Decision: Option 2 - Keep stub with "not implemented" error**
- Preserves CLI interface for future when collector endpoint is added
- Provides clear error message to users that feature is not yet available
- Less breaking change than complete removal

## Questions Resolved

| Question                                                          | Answer                                                                                                |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| What is the correct API server port?                             | 8080 (defined in API server config)                                                                   |
| Does CLI need both collector.url and api.url?                    | NO - collector.url should be removed (no collector endpoint exists)              |
| Does CLI need a Daemon/Collector HTTP client?                             | NO - Daemon doesn't expose HTTP handlers, API doesn't have collector endpoint |
| Should we change API server port or CLI configuration?           | Change CLI configuration to match API server (8080 is the source of truth)                           |
| What to do with SyncProject functionality?                       | Keep stub method that returns "not implemented" error for future implementation |
| What environment variable controls API URL in different modules? | CLI: `COPS_API_URL`, Daemon: `COPS_API_URL`, API: `SERVER_PORT` (remove `COPS_COLLECTOR_URL`)                                      |
