# Research Report

## Mode
General Research (Updated based on Requirements v1)

## Request Summary
Fix port mismatch between CLI and API server (8081 -> 8080) and remove unused CollectorHTTPClient from CLI since no Collector endpoint exists on the API server.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
| ---- | ------ |
| `/Users/jayce/team-attention/cops/cli/internal/platform/setup/config/config.go` | Contains CollectorConfig to be removed and API URL to fix |
| `/Users/jayce/team-attention/cops/cli/internal/platform/setup/httpclient/httpclient.go` | Contains CollectorHTTPClient to be removed |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` | Contains collector field and SyncProject to be modified |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/collector_port.go` | File to DELETE |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go` | File to DELETE |
| `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` | Remove CollectorHTTPClient initialization |
| `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` | Remove collector client DI registration |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` | Contains --sync flag that calls SyncProject |
| `/Users/jayce/team-attention/cops/doc/config.md` | Update documentation for API URL port |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Rules for service layer implementation |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-dig-container.md` | Rules for DI container modifications |

## Package Candidates

No new packages needed - this is a removal/cleanup task.

## Technical Constraints

1. **API Server Port**: Fixed at 8080 (defined in `api/internal/platform/setup/config/config.go:27`)
2. **Daemon Already Correct**: Daemon's API URL default is already `http://localhost:8080`
3. **No Collector Endpoint**: API server implements `AggregationService.SendLogs`, NOT `CollectorService.SendRecords`
4. **CLI uses dig**: Uses `go.uber.org/dig` for DI, not `fx`
5. **Backward Compatibility**: Must maintain `COPS_API_URL` environment variable override

## Dependency Chain Analysis

### Files to DELETE (no dependents after removal)
1. `cli/internal/service/tracking/outbound/api/collector_port.go`
   - Defines `CollectorPort` interface
   - Only used by `tracking_service.go` and `collector_client.go`

2. `cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go`
   - Implements `CollectorPort`
   - Depends on `CollectorHTTPClient` and `CollectorConfig`
   - Only registered in DI container

### Files to MODIFY

1. **`cli/internal/platform/setup/config/config.go`**
   - Remove `CollectorConfig` struct (lines 32-36)
   - Remove `Collector CollectorConfig` field from `Config` struct (line 15)
   - Remove collector defaults (lines 65-66)
   - Remove collector initialization (lines 80-82)
   - Change API URL default from 8081 to 8080 (line 67)

2. **`cli/internal/platform/setup/httpclient/httpclient.go`**
   - Remove `CollectorHTTPClient` struct (lines 11-14)
   - Remove `InitCollectorHTTPClient` function (lines 21-28)
   - Remove `StandardHTTPClient` method for CollectorHTTPClient (lines 41-43)

3. **`cli/internal/service/tracking/tracking_service.go`**
   - Remove `collector api.CollectorPort` field from Service struct (line 33)
   - Remove `collector api.CollectorPort` parameter from NewService (line 42)
   - Remove collector initialization in NewService (line 49)
   - Modify `SyncProject` to return "not implemented" error (lines 310-352)
   - Remove import of `api` package if no longer needed

4. **`cli/cmd/internal/container/module_platform.go`**
   - Remove `httpclient.InitCollectorHTTPClient` from providers (line 22)

5. **`cli/cmd/internal/container/module_tracking.go`**
   - Remove `connectrpc.NewCollectorClient` registration (lines 31-35)
   - Remove `api` package import if no longer needed

6. **`doc/config.md`**
   - Change `COPS_API_URL` default from `http://localhost:8081` to `http://localhost:8080` (line 19)
   - Remove `COPS_COLLECTOR_URL` and `COPS_COLLECTOR_TIMEOUT` rows (lines 17-18)
   - Remove `CollectorConfig` from struct example (lines 56, 70-73)
   - Remove `collector.url` -> `COPS_COLLECTOR_URL` mapping example (line 40)
   - Remove collector URL/timeout usage examples (lines 95-104)

## Similar Implementations Found

### Example 1: Tracking Service Structure
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:28-52`
- **Relevance**: Shows current service struct with 4 ports; after removal will have 3 ports

### Example 2: DI Container Module Pattern
- **File**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go:17-55`
- **Relevance**: Shows dig pattern for registering outbound adapters with `dig.As`

### Example 3: HTTP Client Setup Pattern
- **File**: `/Users/jayce/team-attention/cops/cli/internal/platform/setup/httpclient/httpclient.go:16-37`
- **Relevance**: Shows APIHTTPClient pattern to keep; CollectorHTTPClient pattern to remove

## Additional Information for Planning

### Critical Discovery: Collector vs Aggregation
- **CLI's CollectorClient** calls `CollectorService.SendRecords` (collector.proto)
- **API server implements** `AggregationService.SendLogs` (aggregation.proto)
- **These are incompatible** - CLI's collector would never work even if it was used
- **Daemon also has this bug** - it calls `CollectorService.SendRecords` but API implements `AggregationService.SendLogs`
- **However**: Fixing the Daemon is OUT OF SCOPE for this task

### SyncProject Behavior After Change
The `SyncProject` method should:
1. Keep the method signature for future implementation
2. Return `errutil.NotImplementedf("sync is not yet implemented")` or similar
3. The `--sync` flag in `cops add` command will trigger this and show the error message
4. This provides clear feedback that the feature is not available

### Port Configuration Summary
| Component | Config Key | Current Default | Required Default |
| --------- | ---------- | --------------- | ---------------- |
| CLI API URL | `api.url` | `http://localhost:8081` | `http://localhost:8080` |
| CLI Collector URL | `collector.url` | `http://localhost:8080` | **REMOVE** |
| Daemon API URL | `COPS_API_URL` | `http://localhost:8080` | No change needed |
| API Server Port | `SERVER_PORT` | `8080` | No change needed |
| Doc API URL | `COPS_API_URL` | `http://localhost:8081` | `http://localhost:8080` |
| Doc Collector URL | `COPS_COLLECTOR_URL` | `http://localhost:8080` | **REMOVE** |

### Import Cleanup Notes
After removing collector-related code:
1. `tracking_service.go`: Remove `"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"` import if `api.CollectorPort` was the only usage (check if `api.ProjectPort` is still used - it is, so import stays)
2. `module_tracking.go`: Remove unused imports after removing collector client registration

### Directory Structure After Cleanup
```
cli/internal/service/tracking/outbound/api/
    collector_port.go     # DELETE
    project_port.go       # KEEP
    connectrpc/
        collector_client.go  # DELETE
        project_client.go    # KEEP
```
