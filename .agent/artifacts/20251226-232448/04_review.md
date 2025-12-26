# Pre-PR Code Review

**STATUS: FAIL**

## Critical Issue: Duplicate Service Modules

The `collector` module is **completely redundant** with the existing `aggregation` module. Both modules serve the identical purpose of receiving session records from the daemon and storing them in MongoDB.

### Evidence of Duplication

| Aspect | Aggregation Module | Collector Module |
|--------|-------------------|------------------|
| **Purpose** | "log collection operations" | "session record collection operations" |
| **Protobuf Service** | `AggregationService.SendLogs` | `CollectorService.SendRecords` |
| **Service Method** | `CollectLogs(LogBatch)` | `CollectRecords(RecordBatch)` |
| **Data Model** | `SessionRecord` with daemon_id | `SessionRecord` with project_id |
| **Repository** | `SaveBatch(LogBatch)` | `SaveRecords(RecordBatch)` |
| **Location** | `api/internal/service/aggregation/` | `api/internal/service/collector/` |

Both modules have nearly identical code structure, just with different naming conventions.

### Root Cause

The issue occurred because the research phase did not identify that:
1. The `aggregation` module already exists and serves the same purpose
2. The user previously renamed "collector" → "aggregation"
3. The daemon should use the existing `AggregationService.SendLogs` RPC, not create a new endpoint

### Recommended Action: Consolidate to Aggregation Module

**DELETE** the newly created `collector` module entirely and **UPDATE** the existing `aggregation` module to support `project_id` instead of `daemon_id`.

## Review Summary
- **Files Reviewed**: 13 (6 new, 7 modified)
- **Issues Found**: 2 (Critical: 1 - Duplicate Module, Warning: 1 - File Naming)
- **Build Status**: PASS (but implementation is architecturally incorrect)

## Plan Step Verification

| Step | Description | Status | Notes |
|------|-------------|--------|-------|
| 1 | Modify Protobuf Schema | PASS | `project_id` field added, `ProjectMetadata` removed |
| 2 | Regenerate Protobuf Code | PASS | Generated code includes `ProjectId` field |
| 3 | Create Repository Port Interface | PASS | `CollectorRepositoryPort` with `SaveRecords` method |
| 4 | Create MongoDB Repository Adapter | PASS | Proper ObjectID conversion, BSON document building |
| 5 | Create Collector Service | PASS | Logger binding, result struct, error handling |
| 6 | Create ConnectRPC Handler | PASS | Interface verification, proper response mapping |
| 7 | Create Protobuf-to-Domain Converter | PASS | All session types handled, usage conversion complete |
| 8 | Create DI Module | PASS | fx.Annotate with fx.As pattern used |
| 9 | Register Collector Module | PASS | Added to application.go |
| 10 | Update Daemon Client | PASS | Uses `ProjectId` instead of `Project` |

## Files Reviewed

### `/Users/jayce/team-attention/cops/idl/protobuf/collector/v1/collector.proto`

**Status**: PASS

Changes correctly implement the plan:
- Removed `ProjectMetadata` message definition
- Changed `SendRecordsReq` to use `string project_id = 1` instead of `ProjectMetadata project = 1`
- Comments follow project conventions

### `/Users/jayce/team-attention/cops/shared/gen/grpcstub/collector/v1/collector.pb.go`

**Status**: PASS

Generated code correctly reflects schema changes:
- `SendRecordsReq.ProjectId` field present
- `ProjectMetadata` type removed

### `/Users/jayce/team-attention/cops/api/internal/service/collector/outbound/repository/port.go`

**Status**: PASS with WARNING

Implementation matches plan:
- `RecordBatch` struct with `ProjectID` and `Records`
- `CollectorRepositoryPort` interface with `SaveRecords` method

#### Warning
- **File Naming**: File is named `port.go` but plan specified `collector_repo_port.go`
- **Verdict**: ACCEPTABLE - The codebase has mixed conventions (`port.go` in aggregation, `{domain}_repo_port.go` in dashboard/project). The current naming follows the aggregation pattern which is acceptable.

### `/Users/jayce/team-attention/cops/api/internal/service/collector/outbound/repository/mongodb/adapter.go`

**Status**: PASS

Implementation correctly handles:
- Empty batch early return (line 34-36)
- ObjectID parsing with proper error handling (line 39-46)
- BSON document building using `mongoschema` constants
- Optional field handling (slug, requestId, message fields)
- Compile-time interface verification (line 132)
- Logger binding follows convention: `"collector.repository.mongodb"`

### `/Users/jayce/team-attention/cops/api/internal/service/collector/collector_service.go`

**Status**: PASS

Implementation correctly handles:
- Logger binding: `"collector.service"` (line 19)
- Empty batch early return (line 33-38)
- Logging with structured fields (line 40-43)
- Error handling with proper result struct (line 51-54)
- Port type injection (line 13)

### `/Users/jayce/team-attention/cops/api/internal/service/collector/inbound/grpc/connectrpc/handler.go`

**Status**: PASS

Implementation correctly handles:
- Logger binding: `"collector.grpc.connectrpc"` (line 26)
- Empty project_id validation (line 41-47)
- Proper ConnectRPC response pattern
- Compile-time interface verification (line 71)

### `/Users/jayce/team-attention/cops/api/internal/service/collector/inbound/grpc/connectrpc/converter.go`

**Status**: PASS

Implementation correctly handles:
- All session types mapped (user, assistant, system, summary, file-history-snapshot, queue-operation)
- Default to `SessionTypeUser` for unknown types
- Message conversion with nil checks
- Usage metadata conversion including cache creation

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_collector.go`

**Status**: PASS

DI registration follows fx patterns:
- Repository with `fx.As(new(repository.CollectorRepositoryPort))`
- Handler with `fx.As(new(ConnectHandler))` and `group:"connect_handlers"` tag

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`

**Status**: PASS

Collector module registered correctly at line 18.

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`

**Status**: PASS

Client updated to use `ProjectId` field instead of `Project`:
```go
req := &collectorv1.SendRecordsReq{
    ProjectId: batch.ProjectID,
    Records:   convertRecords(batch.Records),
}
```

## Architecture Compliance

### Hexagonal Architecture
- **Port/Adapter Pattern**: PASS - Port interface in `outbound/repository/`, adapter in `outbound/repository/mongodb/`
- **Inbound Structure**: PASS - `inbound/grpc/connectrpc/` follows the mandatory directory structure
- **Service Layer**: PASS - Business logic in `collector_service.go`, depends on port interface

### Logger Binding Convention
- **Repository**: `"collector.repository.mongodb"` - PASS
- **Service**: `"collector.service"` - PASS
- **Handler**: `"collector.grpc.connectrpc"` - PASS

### DI Pattern
- **fx.Annotate with fx.As**: PASS - Used for interface type conversion
- **Group Registration**: PASS - Handler registered to `connect_handlers` group

### ObjectID Conversion
- **Location**: Repository layer - PASS (as per plan decision)
- **Error Handling**: PASS - Returns error for invalid hex strings

## Build Verification

```bash
go build ./api/...   # PASS
go build ./daemon/... # PASS
```

Both modules compile successfully without errors.

## Consolidation Plan: Merge Collector into Aggregation

To eliminate the duplication, follow these steps:

### Step 1: Update Aggregation Protobuf Schema

**File**: `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`

**Simplify LogBatch to only use project_id:**
```protobuf
message LogBatch {
  repeated SessionRecord records = 1;
  string project_id = 2;  // Replace daemon_id with project_id
}
```

**Changes:**
- Remove `daemon_id` field (line 62)
- Remove `created_at` field (line 63)
- Replace with single `project_id` field

### Step 2: Update Aggregation Repository

**File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go`

**Simplify LogBatch to only have ProjectID:**
```go
type LogBatch struct {
    Records   []shareddomain.SessionRecord
    ProjectID string  // Replace DaemonID with ProjectID
}
```

**Changes:**
- Remove `DaemonID` field
- Remove `CreatedAt` field
- Keep only `Records` and `ProjectID`

### Step 3: Update MongoDB Adapter

**File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`

**Update SaveBatch to convert project_id to ObjectID:**
```go
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *LogBatch) error {
    if len(batch.Records) == 0 {
        return nil
    }

    // Convert project_id to ObjectID
    projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)
    if err != nil {
        return fmt.Errorf("invalid project ID: %w", err)
    }

    // Build BSON documents with projectId field
    docs := make([]interface{}, len(batch.Records))
    for i, record := range batch.Records {
        doc := bson.M{
            mongoschema.SessionRecordUUIDField:      record.UUID,
            mongoschema.SessionRecordProjectIDField: projectObjID,  // Add projectId
            // ...existing fields...
        }
        docs[i] = doc
    }

    _, err = r.collection.InsertMany(ctx, docs)
    return err
}
```

**Changes:**
- Always convert `batch.ProjectID` to ObjectID (no optional check)
- Remove DaemonID-related logic
- Add `projectId` field to all inserted documents

### Step 4: Update Aggregation Handler

**File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`

**Update SendLogs to use simplified LogBatch:**
```go
func (h *AggregationGRPCHandler) SendLogs(ctx context.Context, req *connect.Request[aggregationv1.SendLogsReq]) (*connect.Response[aggregationv1.SendLogsRes], error) {
    batch := convertToDomain(req.Msg.GetBatch())
    // batch now has: Records, ProjectID (no DaemonID, no CreatedAt)

    result := h.svc.CollectLogs(ctx, batch)

    return connect.NewResponse(&aggregationv1.SendLogsRes{
        Success:        result.Success,
        ProcessedCount: result.ProcessedCount,
        ErrorMessage:   result.ErrorMessage,
    }), nil
}
```

**Update converter function:**
```go
func convertToDomain(pb *aggregationv1.LogBatch) *repository.LogBatch {
    records := make([]shareddomain.SessionRecord, len(pb.GetRecords()))
    for i, r := range pb.GetRecords() {
        records[i] = convertSessionRecord(r)
    }

    return &repository.LogBatch{
        Records:   records,
        ProjectID: pb.GetProjectId(),  // Only projectId, no daemonId
    }
}
```

### Step 5: Update Daemon Domain Model

**File**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`

**Simplify LogBatch structure:**
```go
type LogBatch struct {
    Records   []shareddomain.SessionRecord
    ProjectID string  // Only projectId needed
}
```

**Changes:**
- Remove `DaemonID` field
- Remove `CreatedAt` field
- Remove `ProjectName`, `ProjectPath`, `IsGitProject` fields (not needed)
- Keep only `Records` and `ProjectID`

---

### Step 6: Update Daemon Client

**File**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`

**Change to use AggregationService with simplified LogBatch:**
```go
import (
    aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

type APIClient struct {
    logger *slog.Logger
    client aggregationv1connect.AggregationServiceClient  // Change from CollectorServiceClient
}

func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
    client := aggregationv1connect.NewAggregationServiceClient(
        apiClient.StandardHTTPClient(),
        cfg.API.URL,
    )

    return &APIClient{
        logger: l.With(slog.String("name", "log.api.connectrpc")),
        client: client,
    }
}

func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
    req := &aggregationv1.SendLogsReq{
        Batch: &aggregationv1.LogBatch{
            Records:   convertRecords(batch.Records),
            ProjectId: batch.ProjectID,  // Only projectId
        },
    }

    resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
    if err != nil {
        return err
    }

    if !resp.Msg.Success {
        c.logger.Warn("API returned failure")
    }

    c.logger.Debug("logs sent",
        slog.Int("processed", int(resp.Msg.ProcessedCount)),
    )

    return nil
}
```

**Changes:**
- Import `aggregationv1` instead of `collectorv1`
- Use `AggregationServiceClient` instead of `CollectorServiceClient`
- Send only `Records` and `ProjectId` in LogBatch
- Remove `DaemonId` and `CreatedAt` fields

---

### Step 7: Delete Collector Module Files

**Delete these files:**
- `/Users/jayce/team-attention/cops/api/internal/service/collector/` (entire directory)
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_collector.go`
- `/Users/jayce/team-attention/cops/idl/protobuf/collector/v1/collector.proto`

**Remove from application.go:**
- Remove `newCollectorModule()` from module list

---

### Step 8: Regenerate Protobuf Code

```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

This will update aggregation stubs and remove collector stubs.

---

### Step 9: Verify Build

```bash
go build ./api/...
go build ./daemon/...
```

### Migration Benefits

1. **Single Source of Truth**: Only `aggregation` module handles session record collection
2. **Simplified Data Model**: Only `project_id` field, no unnecessary `daemon_id` or `created_at`
3. **Cleaner Architecture**: Removes duplicate code and protobuf schemas
4. **Clear Naming**: "Aggregation" better represents the service purpose than "Collector"
5. **Consistent with Original Design**: Returns to the user's previous naming convention

## Decision Required

**Should we proceed with consolidation?**

If YES:
- Abandon the current collector implementation
- Follow the consolidation plan above
- Create a new PR with consolidated changes

If NO:
- Need justification for maintaining two identical modules
- Consider renaming to clarify distinct responsibilities
