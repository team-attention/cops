# Development Walkthrough

## Summary
Consolidated duplicate Collector module into existing Aggregation module, simplifying the data model to use only `project_id` instead of `daemon_id`, eliminating code duplication and architectural redundancy.

## Background

### Initial Implementation Problem
The task initially aimed to implement a CollectorService endpoint for receiving session records from the daemon. However, during code review, a critical architectural issue was discovered: the `collector` module was **completely redundant** with the existing `aggregation` module.

Both modules served identical purposes:
- Receiving session records from the daemon
- Storing them in MongoDB
- Using nearly identical code structure

The duplication occurred because the research phase didn't identify that the `aggregation` module already existed and the user had previously renamed "collector" to "aggregation."

### Solution
Rather than maintaining duplicate code, the decision was made to:
1. Delete the entire Collector module
2. Update the existing Aggregation module to use `project_id` instead of `daemon_id`
3. Simplify the data model by removing unnecessary fields (`daemon_id`, `created_at`)
4. Update the Daemon to use AggregationService

## Code Overview

### Modified Components

#### Aggregation Protobuf Schema
- **Location**: `idl/protobuf/aggregation/v1/aggregation.proto`
- **Changes**: Simplified `LogBatch` message to use only `project_id`
- **Key Modifications**:
  ```protobuf
  message LogBatch {
    repeated SessionRecord records = 1;
    string project_id = 2;  // Replaced daemon_id with project_id
  }
  ```
- **Rationale**: The `daemon_id` field was redundant since session records should be associated with projects, not daemons. The `created_at` field was unnecessary metadata.

#### Aggregation Repository Port
- **Location**: `api/internal/service/aggregation/outbound/repository/port.go`
- **Changes**: Simplified `LogBatch` struct to contain only necessary fields
- **Key Structure**:
  ```go
  type LogBatch struct {
      Records   []shareddomain.SessionRecord
      ProjectID string  // Only ProjectID needed
  }
  ```
- **Rationale**: Removed `DaemonID` and `CreatedAt` fields as they provided no business value for the aggregation use case.

#### MongoDB Adapter
- **Location**: `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`
- **Changes**: Updated to convert `project_id` string to MongoDB `ObjectID`
- **Key Methods**:
  - `SaveBatch()`:
    - Converts `batch.ProjectID` from hex string to `bson.ObjectID` (lines 37-40)
    - Adds `projectId` field to all inserted documents (line 70)
    - Handles invalid project ID hex strings with proper error wrapping
  - `toDocument()`:
    - Builds BSON documents with `projectId` as `bson.ObjectID`
    - Handles optional fields (`slug`, `requestId`, message fields)
- **Error Handling**: Returns descriptive errors for invalid project ID format

#### ConnectRPC Handler
- **Location**: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
- **Changes**: Updated converter to use simplified `LogBatch` structure
- **Key Updates**:
  - Converter now extracts only `ProjectId` from protobuf message (no `DaemonId`)
  - Interface verification ensures compile-time type safety

#### Daemon Domain Model
- **Location**: `daemon/internal/platform/domain/watch.go`
- **Changes**: Simplified `LogBatch` to contain only necessary fields
- **Key Structure**:
  ```go
  type LogBatch struct {
      Records   []shareddomain.SessionRecord
      ProjectID string  // Only ProjectID needed for API transmission
  }
  ```
- **Rationale**: The daemon only needs to know which project the logs belong to; daemon identity and timestamps are not needed for log aggregation.

#### Daemon API Client
- **Location**: `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- **Changes**: Switched from CollectorService to AggregationService
- **Key Modifications**:
  - Import changed from `collectorv1` to `aggregationv1`
  - Client type changed from `CollectorServiceClient` to `AggregationServiceClient`
  - RPC call changed from `SendRecords` to `SendLogs`
  - Request payload simplified to only `Records` and `ProjectId`
- **Benefits**: Single, consistent API endpoint for log aggregation across the system

### Deleted Components

All Collector module files were completely removed:

#### Protobuf Schema
- **Deleted**: `idl/protobuf/collector/v1/collector.proto`
- **Impact**: Removed duplicate service definition

#### Service Implementation
- **Deleted**: `api/internal/service/collector/` (entire directory)
- **Included**:
  - `collector_service.go` - Duplicate business logic
  - `outbound/repository/port.go` - Duplicate repository interface
  - `outbound/repository/mongodb/adapter.go` - Duplicate MongoDB implementation
  - `inbound/grpc/connectrpc/handler.go` - Duplicate gRPC handler
  - `inbound/grpc/connectrpc/converter.go` - Duplicate converter logic

#### DI Module
- **Deleted**: `api/cmd/internal/container/module_collector.go`
- **Updated**: Removed `newCollectorModule()` from `application.go` module list

#### CLI References
- **Deleted**:
  - `cli/internal/service/tracking/outbound/api/collector_port.go`
  - `cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go`
- **Impact**: CLI no longer has any collector-related code

## Data Flow After Consolidation

```
Claude Code JSONL logs
    ↓
Daemon LogWatcher
    ↓
daemon.LogBatch {ProjectID, Records}
    ↓
AggregationService.SendLogs (ConnectRPC)
    ↓
aggregationv1.LogBatch {project_id, records}
    ↓
repository.LogBatch {ProjectID, Records}
    ↓
MongoDB (projectId as ObjectID)
```

## Migration Details: daemon_id → project_id

### Why the Migration Was Necessary

**Problem**: The original design used `daemon_id` to identify which daemon sent the logs.

**Issue**: This approach was architecturally incorrect because:
1. Session records belong to **projects**, not daemons
2. A single daemon can watch multiple projects
3. Multiple daemons (or daemon restarts) can send logs for the same project
4. Dashboard queries need to filter by **project**, not by daemon
5. The daemon is just a transport mechanism, not a business entity

**Solution**: Use `project_id` to directly associate session records with projects.

### Implementation Changes

**Before (with daemon_id)**:
```protobuf
message LogBatch {
  repeated SessionRecord records = 1;
  string daemon_id = 2;
  google.protobuf.Timestamp created_at = 3;
}
```

**After (with project_id)**:
```protobuf
message LogBatch {
  repeated SessionRecord records = 1;
  string project_id = 2;  // Direct project association
}
```

### Data Mapping

| Field | Before | After | Rationale |
|-------|--------|-------|-----------|
| **Identifier** | `daemon_id` | `project_id` | Session records belong to projects, not daemons |
| **Timestamp** | `created_at` | (removed) | Not needed; each SessionRecord has its own timestamp |
| **MongoDB Field** | `daemonId` (ObjectID) | `projectId` (ObjectID) | Direct project reference for queries |

### Query Impact

**Before**: To get all sessions for a project, you would need to:
1. Find all daemons watching that project
2. Query sessions by daemon IDs
3. Handle daemon restarts (new daemon_id, same project)

**After**: Direct project-based queries:
```javascript
db.sessionRecords.find({ projectId: ObjectId("...") })
```

## Testing

### Build Verification
All modules compile successfully:
```bash
go build ./api/...     # PASS - No errors
go build ./daemon/...  # PASS - No errors
go build ./cli/...     # PASS - No errors
go build ./shared/...  # PASS - No errors
```

### Code Generation
Protobuf regeneration successful:
```bash
cd idl/protobuf && buf generate  # PASS
```

### Interface Compliance
Compile-time interface verification passes:
- `MongoSessionRecordRepository` implements `SessionRecordRepositoryPort`
- `AggregationGRPCHandler` implements `ConnectHandler`

### Module Isolation
- No circular dependencies
- All imports resolve correctly
- No references to deleted Collector module

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Initial implementation duplicated existing Aggregation module | Deleted entire Collector module, consolidated into Aggregation |
| `daemon_id` was architecturally incorrect (sessions belong to projects, not daemons) | Migrated to `project_id` for direct project association |
| Unnecessary `created_at` field in LogBatch | Removed; each SessionRecord has its own timestamp |
| Stale generated files remain after proto deletion | Acceptable; files not imported by code (can be cleaned manually) |

## Architecture Benefits

### Single Source of Truth
- Only `aggregation` module handles session record collection
- No duplicate code paths or services
- Clear ownership and responsibility

### Simplified Data Model
- Direct project association via `project_id`
- No unnecessary daemon tracking
- Cleaner MongoDB queries by project

### Better Naming
- "Aggregation" better represents the service purpose (collecting and aggregating logs)
- Consistent with user's previous naming convention

### Reduced Maintenance
- Fewer files to maintain
- Single implementation to update for changes
- Less cognitive overhead for developers

## Related Tickets
This work was part of the CollectorService implementation task, which pivoted to a consolidation effort after architectural review identified the duplication issue.
