# Research Report

## Mode
General Research

## Request Summary
Implement CollectorService endpoint in the API server to receive session records from the daemon. The service needs to handle `SendRecords` RPC which receives project ID and session records, stores them in MongoDB, and returns success/failure response.

**Important Design Decision**: The endpoint receives only `project_id` (string) instead of full ProjectMetadata. Projects are already registered in the system, so only the ID is needed.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go` | Similar service pattern - receives log batches and saves to repository |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` | ConnectRPC handler pattern with protobuf-to-domain conversion |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | MongoDB repository pattern for saving session records |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go` | Repository port interface definition |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go` | DI module registration pattern |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go` | ConnectHandler interface definition |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect/collector.connect.go` | Generated CollectorServiceHandler interface |
| `/Users/jayce/team-attention/cops/idl/protobuf/collector/v1/collector.proto` | Protobuf schema for SendRecordsReq/Res |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go` | MongoDB field constants for session records |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Service layer implementation rules |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md` | ConnectRPC handler guidelines |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md` | Outbound adapter guidelines |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md` | DI container guidelines |

## Existing Architecture Analysis

### Service Layer Pattern (from aggregation service)

**Service struct:**
```go
type Service struct {
    logger *slog.Logger
    repo   repository.SessionRecordRepositoryPort
}

func NewService(l *slog.Logger, repo repository.SessionRecordRepositoryPort) *Service {
    return &Service{
        logger: l.With(slog.String("name", "aggregation.service")),
        repo:   repo,
    }
}
```

**Key patterns:**
- Logger as first constructor parameter
- Logger bound with service name immediately: `l.With(slog.String("name", "{domain}.service"))`
- Repository port interface as dependency
- Simple return types with success/error/count fields

### ConnectRPC Handler Pattern

**Handler struct:**
```go
type AggregationGRPCHandler struct {
    svc    *aggregationservice.Service
    logger *slog.Logger
}

func NewAggregationGRPCHandler(l *slog.Logger, svc *aggregationservice.Service) *AggregationGRPCHandler {
    return &AggregationGRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "aggregation.grpc.connectrpc")),
    }
}
```

**GetHandler method:**
```go
func (h *AggregationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
    return aggregationv1connect.NewAggregationServiceHandler(h, opts...)
}
```

**Interface verification:**
```go
var _ aggregationv1connect.AggregationServiceHandler = (*AggregationGRPCHandler)(nil)
```

### MongoDB Repository Pattern

**Repository struct:**
```go
type MongoSessionRecordRepository struct {
    logger     *slog.Logger
    collection *mongo.Collection
}

func NewMongoSessionRecordRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRecordRepository {
    return &MongoSessionRecordRepository{
        logger:     l.With(slog.String("name", "aggregation.repository.mongodb")),
        collection: db.Collection(mongoschema.SessionRecordCollectionName),
    }
}
```

**Key patterns:**
- Uses `mongoschema` field constants (e.g., `mongoschema.SessionRecordUUIDField`)
- Uses `bson.M` for document construction
- Uses `InsertMany` for batch operations
- Interface verification at end of file

### DI Module Registration Pattern

```go
func newAggregationModule() fx.Option {
    return fx.Module("aggregation",
        // Repository
        fx.Provide(
            fx.Annotate(
                mongodb.NewMongoSessionRecordRepository,
                fx.As(new(repository.SessionRecordRepositoryPort)),
            ),
        ),

        // Service
        fx.Provide(aggregationservice.NewService),

        // gRPC Handler
        fx.Provide(
            fx.Annotate(
                connectrpc.NewAggregationGRPCHandler,
                fx.As(new(ConnectHandler)),
                fx.ResultTags(`group:"connect_handlers"`),
            ),
        ),
    )
}
```

**Key patterns:**
- Use `fx.Module` with domain name
- Use `fx.Annotate` with `fx.As` for interface conversion
- Use `fx.ResultTags` with `group:"connect_handlers"` for handler registration

### Protobuf to Domain Conversion Pattern

The aggregation handler shows conversion from protobuf to domain types:

```go
func convertToDomain(pb *aggregationv1.LogBatch) *repository.LogBatch {
    records := make([]shareddomain.SessionRecord, len(pb.GetRecords()))
    for i, r := range pb.GetRecords() {
        records[i] = shareddomain.SessionRecord{
            UUID:        r.GetUuid(),
            ParentUUID:  r.GetParentUuid(),
            SessionID:   r.GetSessionId(),
            Type:        convertSessionType(r.GetType()),
            Timestamp:   r.GetTimestamp().AsTime(),
            // ... more fields
        }
    }
    return &repository.LogBatch{
        Records:   records,
        DaemonID:  pb.GetDaemonId(),
    }
}
```

## Package Candidates

### Problem 1: No external packages needed

The CollectorService implementation uses only existing project dependencies:
- `connectrpc.com/connect` - Already used for ConnectRPC handlers
- `go.mongodb.org/mongo-driver/v2` - Already used for MongoDB operations
- `go.uber.org/fx` - Already used for DI

No new packages are required for this implementation.

## Technical Constraints

1. **Must implement `collectorv1connect.CollectorServiceHandler` interface** - The generated interface requires implementing `SendRecords` method
2. **Must register with `connect_handlers` group** - For automatic registration with Fiber server
3. **Must use existing mongoschema field constants** - To maintain consistency with session record storage
4. **Should reuse existing domain types** - `shareddomain.SessionRecord`, `shareddomain.Usage`, etc.
5. **Project ID handling** - The protobuf includes project metadata with optional ID; need to handle project lookup/creation
6. **Record deduplication** - Consider using UUID as unique key to prevent duplicate records

## Similar Implementations Found

### Example 1: Aggregation Service (Primary Reference)
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go:1-61`
- **Relevance**: Near-identical use case - receives log batches with session records and saves to MongoDB

### Example 2: Aggregation gRPC Handler
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go:1-141`
- **Relevance**: Shows protobuf-to-domain conversion pattern for session records

### Example 3: Project Service Handler
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go:1-64`
- **Relevance**: Simple handler pattern with service call and response building

### Example 4: MongoDB Session Record Repository
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:1-116`
- **Relevance**: Shows how to save session records to MongoDB with field mapping

### Example 5: DI Module Registration
- **File**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go:1-35`
- **Relevance**: Complete module registration pattern to follow

## Directory Structure to Create

```
api/internal/service/collector/
├── collector_service.go                      # Core service implementation
├── inbound/
│   └── grpc/
│       └── connectrpc/
│           ├── handler.go                    # Handler struct + GetHandler
│           └── converter.go                  # Protobuf-to-domain conversion
└── outbound/
    └── repository/
        ├── collector_repo_port.go            # Repository interface
        └── mongodb/
            └── collector_repo.go             # MongoDB implementation
```

## Additional Information for Planning

### Key Differences from Aggregation Service

1. **Project ID Only**: CollectorService receives only `project_id` (string), while Aggregation uses `DaemonID`
2. **Project ID Field**: Each session record needs `projectId` field in MongoDB (as `bson.ObjectID`)
3. **Simpler Response**: Only needs `records_received` count and `success` boolean
4. **No Project Lookup Needed**: Project is already registered; just store the projectId with each record

### Generated CollectorServiceHandler Interface

```go
type CollectorServiceHandler interface {
    SendRecords(context.Context, *connect.Request[v1.SendRecordsReq]) (*connect.Response[v1.SendRecordsRes], error)
}
```

### SendRecordsReq Structure

```protobuf
message SendRecordsReq {
  string project_id = 1;  // Project MongoDB ObjectID (hex string)
  repeated SessionRecord records = 2;
}
```

**Note**: The protobuf schema has been simplified to only send `project_id` instead of full `ProjectMetadata`. Projects are already registered in the system, so metadata fields (name, path, git_project) are not needed.

### MongoDB Session Record Fields

The following field constants are available in `mongoschema`:
- `SessionRecordProjectIDField = "projectId"` - Set from incoming `project_id` field
- All other fields match the aggregation repository pattern

**Implementation note**: Convert `project_id` string (hex) to `bson.ObjectID` when storing in MongoDB.

### Application Registration

The collector module needs to be added to:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go` - Add `newCollectorModule()` call
