# Implementation Plan

## Overview
Implement the CollectorService endpoint to receive session records from the daemon with only `project_id`, store them in MongoDB with the project ID as a BSON ObjectID, and return a success response with records count.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
|---------|---------|-------------|----------------------|
| gRPC/ConnectRPC | connectrpc.com/connect | N/A (existing) | Already used in project for ConnectRPC handlers |
| MongoDB | go.mongodb.org/mongo-driver/v2 | N/A (existing) | Already used for MongoDB operations |
| DI Container | go.uber.org/fx | N/A (existing) | Already used for dependency injection |

## Architecture Decisions

### Decision 1: Protobuf Schema Simplification
**Choice**: Change `SendRecordsReq` to use `string project_id` instead of `ProjectMetadata project`
**Rationale**: Projects are already registered in the system. The daemon only needs to send the project ID to associate records with a project. This simplifies the API and reduces data transfer.

### Decision 2: Project ID Conversion Strategy
**Choice**: Convert `project_id` hex string to `bson.ObjectID` in the repository layer
**Rationale**: The repository layer handles MongoDB-specific types. Converting at this layer keeps the service layer clean and follows the existing pattern in the codebase.

### Decision 3: Reuse Existing Domain Types
**Choice**: Reuse `shareddomain.SessionRecord` from `shared/domain/session.go`
**Rationale**: The existing domain types match the collector's needs. No new domain types required.

### Decision 4: Service Structure Pattern
**Choice**: Follow the aggregation service pattern with separate repository port and MongoDB adapter
**Rationale**: Maintains consistency with existing codebase patterns and enables future repository implementations if needed.

## Implementation Steps

### Step 1: Modify Protobuf Schema

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/idl/protobuf/collector/v1/collector.proto` (modify)

**Changes**:

```protobuf
// BEFORE:
message SendRecordsReq {
  ProjectMetadata project = 1;
  repeated SessionRecord records = 2;
}

// AFTER:
message SendRecordsReq {
  // Project MongoDB ObjectID as hex string
  string project_id = 1;

  // Session records to send
  repeated SessionRecord records = 2;
}
```

**Additional Changes**:
- Remove the `ProjectMetadata` message definition (lines 91-103) as it is no longer needed

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Valid project_id format | `"507f1f77bcf86cd799439011"` | Compiles successfully | Schema validation |
| Empty project_id | `""` | Compiles (validation at runtime) | Optional field |

---

### Step 2: Regenerate Protobuf Code

**Commands**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Files Modified** (auto-generated):
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/collector/v1/collector.pb.go`
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect/collector.connect.go`

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Successful generation | buf generate | No errors, files updated | Happy path |
| Import validation | go build ./shared/... | Compiles successfully | Generated code valid |

---

### Step 3: Create Repository Port Interface

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/collector/outbound/repository/collector_repo_port.go` (create)

**Functions**:

```go
package repository

import (
    "context"

    shareddomain "github.com/team-attention/cops/shared/domain"
)

// RecordBatch represents a batch of session records with project association.
type RecordBatch struct {
    ProjectID string                       // MongoDB ObjectID as hex string
    Records   []shareddomain.SessionRecord // Session records to store
}

// CollectorRepositoryPort defines the interface for collector record persistence.
type CollectorRepositoryPort interface {
    // SaveRecords saves a batch of session records to storage.
    // Returns the number of records successfully saved.
    SaveRecords(ctx context.Context, batch *RecordBatch) (int, error)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Interface compilation | N/A | Compiles successfully | Type definition |

---

### Step 4: Create MongoDB Repository Adapter

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/collector/outbound/repository/mongodb/collector_repo.go` (create)

**Functions**:

```go
package mongodb

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"

    "github.com/team-attention/cops/api/internal/service/collector/outbound/repository"
    shareddomain "github.com/team-attention/cops/shared/domain"
    "github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoCollectorRepository implements CollectorRepositoryPort using MongoDB.
type MongoCollectorRepository struct {
    logger     *slog.Logger
    collection *mongo.Collection
}

// NewMongoCollectorRepository creates a new MongoDB collector repository adapter.
func NewMongoCollectorRepository(l *slog.Logger, db *mongo.Database) *MongoCollectorRepository {
    return &MongoCollectorRepository{
        logger:     l.With(slog.String("name", "collector.repository.mongodb")),
        collection: db.Collection(mongoschema.SessionRecordCollectionName),
    }
}

// SaveRecords saves a batch of session records to MongoDB.
// Returns the number of records successfully saved.
func (r *MongoCollectorRepository) SaveRecords(ctx context.Context, batch *repository.RecordBatch) (int, error) {
    // Implementation outline:
    // 1. Return 0 if batch is empty
    // 2. Parse project_id to bson.ObjectID
    // 3. Convert records to BSON documents with projectId field
    // 4. InsertMany into collection
    // 5. Return count of inserted documents
}

// toDocument converts a SessionRecord to a BSON document with projectId.
func toDocument(record shareddomain.SessionRecord, projectID bson.ObjectID) bson.M {
    // Implementation outline:
    // 1. Create base document with all session record fields using mongoschema constants
    // 2. Add projectId field with the provided ObjectID
    // 3. Add createdAt timestamp
    // 4. Conditionally add optional fields (slug, requestId)
    // 5. Conditionally add message fields if present
    // 6. Return document
}

// Compile-time interface verification.
var _ repository.CollectorRepositoryPort = (*MongoCollectorRepository)(nil)
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Empty batch | `&RecordBatch{Records: []}` | `(0, nil)` | Early return for empty |
| Valid project_id | `"507f1f77bcf86cd799439011"` | Records saved with ObjectID | Happy path |
| Invalid project_id hex | `"invalid-hex"` | `(0, error)` | ObjectID parse error |
| MongoDB insert error | Mock failure | `(0, error)` | Database error handling |
| Single record | 1 record batch | `(1, nil)` | Single insert |
| Multiple records | 5 record batch | `(5, nil)` | Batch insert |
| Record with message | Record with Message field | Document includes message fields | Optional message fields |
| Record without message | Record with nil Message | Document excludes message fields | Nil message handling |

---

### Step 5: Create Collector Service

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/collector/collector_service.go` (create)

**Functions**:

```go
package collector

import (
    "context"
    "log/slog"

    "github.com/team-attention/cops/api/internal/service/collector/outbound/repository"
)

// Service handles session record collection operations.
type Service struct {
    logger *slog.Logger
    repo   repository.CollectorRepositoryPort
}

// NewService creates a new collector service.
func NewService(l *slog.Logger, repo repository.CollectorRepositoryPort) *Service {
    return &Service{
        logger: l.With(slog.String("name", "collector.service")),
        repo:   repo,
    }
}

// CollectRecordsResult contains the result of record collection.
type CollectRecordsResult struct {
    Success         bool
    RecordsReceived int32
    ErrorMessage    string
}

// CollectRecords processes a batch of session records and saves them to storage.
func (s *Service) CollectRecords(ctx context.Context, batch *repository.RecordBatch) *CollectRecordsResult {
    // Implementation outline:
    // 1. If batch is empty, return success with 0 count
    // 2. Log incoming batch info (projectId, record count)
    // 3. Call repo.SaveRecords
    // 4. If error, log error and return failure result
    // 5. Return success result with saved count
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Empty batch | `&RecordBatch{Records: []}` | `{Success: true, RecordsReceived: 0}` | Empty batch early return |
| Valid batch | 5 records | `{Success: true, RecordsReceived: 5}` | Happy path |
| Repository error | Mock repo error | `{Success: false, ErrorMessage: "..."}` | Error handling |

---

### Step 6: Create ConnectRPC Handler

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/collector/inbound/grpc/connectrpc/handler.go` (create)

**Functions**:

```go
package connectrpc

import (
    "context"
    "log/slog"
    "net/http"

    "connectrpc.com/connect"

    collectorservice "github.com/team-attention/cops/api/internal/service/collector"
    "github.com/team-attention/cops/api/internal/service/collector/outbound/repository"
    shareddomain "github.com/team-attention/cops/shared/domain"
    collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect"
)

// CollectorGRPCHandler handles collector service gRPC endpoints.
type CollectorGRPCHandler struct {
    svc    *collectorservice.Service
    logger *slog.Logger
}

// NewCollectorGRPCHandler creates a new collector gRPC handler.
func NewCollectorGRPCHandler(l *slog.Logger, svc *collectorservice.Service) *CollectorGRPCHandler {
    return &CollectorGRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "collector.grpc.connectrpc")),
    }
}

// GetHandler implements ConnectHandler interface.
func (h *CollectorGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
    return collectorv1connect.NewCollectorServiceHandler(h, opts...)
}

// SendRecords implements collectorv1connect.CollectorServiceHandler.
func (h *CollectorGRPCHandler) SendRecords(
    ctx context.Context,
    req *connect.Request[collectorv1.SendRecordsReq],
) (*connect.Response[collectorv1.SendRecordsRes], error) {
    // Implementation outline:
    // 1. Validate project_id is not empty
    // 2. Convert protobuf records to domain types
    // 3. Create RecordBatch with projectId and records
    // 4. Call service.CollectRecords
    // 5. Return response with result
}

// Compile-time interface verification.
var _ collectorv1connect.CollectorServiceHandler = (*CollectorGRPCHandler)(nil)
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Empty project_id | `{project_id: ""}` | `{success: false}` | Validation error |
| Valid request | `{project_id: "...", records: [...]}` | `{success: true, records_received: N}` | Happy path |
| Empty records | `{project_id: "...", records: []}` | `{success: true, records_received: 0}` | Empty records |

---

### Step 7: Create Protobuf-to-Domain Converter

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/collector/inbound/grpc/connectrpc/converter.go` (create)

**Functions**:

```go
package connectrpc

import (
    shareddomain "github.com/team-attention/cops/shared/domain"
    collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
)

// convertRecordsToDomain converts protobuf session records to domain types.
func convertRecordsToDomain(records []*collectorv1.SessionRecord) []shareddomain.SessionRecord {
    // Implementation outline:
    // 1. Create result slice with same length
    // 2. For each protobuf record, convert to domain SessionRecord
    // 3. Return result slice
}

// convertSessionType converts protobuf type string to domain SessionType.
func convertSessionType(t string) shareddomain.SessionType {
    // Implementation outline:
    // 1. Switch on type string
    // 2. Map to corresponding shareddomain.SessionType constant
    // 3. Default to SessionTypeUser for unknown types
}

// convertMessage converts protobuf message fields to domain Message.
func convertMessage(record *collectorv1.SessionRecord) *shareddomain.Message {
    // Implementation outline:
    // 1. Create Message with role and content from record
    // 2. If usage metadata present, create Usage struct
    // 3. Return Message pointer
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Empty records | `[]` | `[]` | Empty slice |
| Single record | 1 pb record | 1 domain record | Basic conversion |
| All session types | Various types | Correct domain types | Type mapping |
| Unknown session type | `"unknown"` | `SessionTypeUser` | Default case |
| Record with usage | Record with usage data | Message with Usage | Usage conversion |
| Record without usage | Record without usage | Message with nil Usage | Nil usage handling |

---

### Step 8: Create DI Module

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_collector.go` (create)

**Functions**:

```go
package container

import (
    "go.uber.org/fx"

    collectorservice "github.com/team-attention/cops/api/internal/service/collector"
    "github.com/team-attention/cops/api/internal/service/collector/inbound/grpc/connectrpc"
    "github.com/team-attention/cops/api/internal/service/collector/outbound/repository"
    "github.com/team-attention/cops/api/internal/service/collector/outbound/repository/mongodb"
)

func newCollectorModule() fx.Option {
    return fx.Module("collector",
        // Repository
        fx.Provide(
            fx.Annotate(
                mongodb.NewMongoCollectorRepository,
                fx.As(new(repository.CollectorRepositoryPort)),
            ),
        ),

        // Service
        fx.Provide(collectorservice.NewService),

        // gRPC Handler
        fx.Provide(
            fx.Annotate(
                connectrpc.NewCollectorGRPCHandler,
                fx.As(new(ConnectHandler)),
                fx.ResultTags(`group:"connect_handlers"`),
            ),
        ),
    )
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Module registration | fx.New with module | No errors | DI wiring |
| Handler collection | Group tag | Handler in connect_handlers group | Group registration |

---

### Step 9: Register Collector Module in Application

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go` (modify)

**Changes**:

```go
// Add newCollectorModule() to the fx.New() call
func Run() {
    fx.New(
        // Modules
        newPlatformModule(),
        newHealthModule(),
        newAggregationModule(),
        newDashboardModule(),
        newProjectModule(),
        newCollectorModule(),  // <-- ADD THIS LINE

        // Registrations (invoked for side effects)
        fx.Invoke(registerConnectRPCServer),

        // Lifecycle timeouts
        fx.StartTimeout(30*time.Second),
        fx.StopTimeout(30*time.Second),
    ).Run()
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Application startup | Run() | Server starts with collector endpoint | Integration |

---

### Step 10: Update Daemon Client to Send Only project_id

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` (modify)
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` (modify)

**Changes to api_client.go**:

```go
// BEFORE:
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
    req := &collectorv1.SendRecordsReq{
        Project: &collectorv1.ProjectMetadata{
            Id:         batch.ProjectID,
            Name:       batch.ProjectName,
            Path:       batch.ProjectPath,
            GitProject: batch.IsGitProject,
        },
        Records: convertRecords(batch.Records),
    }
    // ...
}

// AFTER:
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
    req := &collectorv1.SendRecordsReq{
        ProjectId: batch.ProjectID,
        Records:   convertRecords(batch.Records),
    }
    // ...
}
```

**Changes to watch.go (optional cleanup)**:
The `LogBatch` struct can optionally have unused fields removed (`ProjectName`, `ProjectPath`, `IsGitProject`), but this may require checking other usages. If these fields are used elsewhere, keep them for backward compatibility.

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Send logs with project_id | Valid LogBatch | Request sent with project_id | Happy path |
| Build success | go build ./daemon/... | Compiles successfully | API compatibility |

---

## Execution Order

1. **Step 1**: Modify Protobuf Schema (no dependencies)
2. **Step 2**: Regenerate Protobuf Code (depends on Step 1)
3. **Step 3**: Create Repository Port Interface (depends on Step 2 for imports)
4. **Step 4**: Create MongoDB Repository Adapter (depends on Step 3)
5. **Step 5**: Create Collector Service (depends on Step 3)
6. **Step 6**: Create ConnectRPC Handler (depends on Step 2, 5)
7. **Step 7**: Create Protobuf-to-Domain Converter (depends on Step 2)
8. **Step 8**: Create DI Module (depends on Steps 4, 5, 6)
9. **Step 9**: Register Collector Module (depends on Step 8)
10. **Step 10**: Update Daemon Client (depends on Step 2)

## Notes for Execute Agent

1. **Protobuf Regeneration**: After modifying the proto file, run `cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate` before proceeding with Go code changes.

2. **Build Verification**: After completing all steps, verify both modules compile:
   ```bash
   go build ./api/...
   go build ./daemon/...
   ```

3. **Directory Structure**: Create the following directory structure before adding files:
   ```
   api/internal/service/collector/
   ├── collector_service.go
   ├── inbound/
   │   └── grpc/
   │       └── connectrpc/
   │           ├── handler.go
   │           └── converter.go
   └── outbound/
       └── repository/
           ├── collector_repo_port.go
           └── mongodb/
               └── collector_repo.go
   ```

4. **MongoDB Field Mapping**: Use existing `mongoschema.SessionRecord*` constants for field names. The `SessionRecordProjectIDField` constant already exists.

5. **ObjectID Conversion**: Use `bson.ObjectIDFromHex(projectID)` to convert the hex string. Handle the error case where the hex string is invalid.

6. **Converter Pattern**: Follow the existing pattern in `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` for the protobuf-to-domain conversion functions.

7. **LogBatch Cleanup**: The `LogBatch` struct in the daemon still has `ProjectName`, `ProjectPath`, and `IsGitProject` fields. These can be removed in a follow-up cleanup if they are not used elsewhere in the daemon codebase.
