# Requirements

## Request Summary

Implement the missing CollectorService endpoint in the API server. The daemon is currently unable to send session records to the API server because the `/collector.v1.CollectorService/SendRecords` endpoint returns 404 with "unimplemented" error. This endpoint needs to be implemented following the existing hexagonal architecture pattern used in the API server for other services (health, dashboard, aggregation, project).

## Acceptance Criteria

- [ ] Criterion 1: API server successfully receives SendRecords RPC calls from daemon without 404 errors
- [ ] Criterion 2: SendRecords endpoint validates incoming ProjectMetadata and SessionRecord data
- [ ] Criterion 3: Session records are persisted to MongoDB with proper schema design
- [ ] Criterion 4: SendRecords returns appropriate response with records_received count and success status
- [ ] Criterion 5: Error handling follows project conventions using errutil package
- [ ] Criterion 6: Implementation follows hexagonal architecture with proper service/inbound/outbound separation
- [ ] Criterion 7: Logger is properly injected and bound with appropriate naming conventions
- [ ] Criterion 8: ConnectRPC handler is registered to DI container using fx.Annotate pattern
- [ ] Criterion 9: MongoDB repository implements proper data model conversion from protobuf to domain types

## Scope

### In Scope

- Item 1: Create collector service with business logic (`api/internal/service/collector/collector_service.go`)
- Item 2: Create ConnectRPC inbound handler (`api/internal/service/collector/inbound/grpc/connectrpc/handler.go`)
- Item 3: Create MongoDB outbound repository adapter (`api/internal/service/collector/outbound/repository/mongodb/collector_repo.go`)
- Item 4: Create repository port interface (`api/internal/service/collector/outbound/repository/collector_repo_port.go`)
- Item 5: Register collector module in DI container (`api/cmd/internal/container/module_collector.go`)
- Item 6: Design MongoDB schema for storing session records with proper indexing
- Item 7: Handle conversion from protobuf types (collectorv1.SessionRecord) to domain models

### Out of Scope

- Item 1: Dashboard UI visualization of collected session data (separate feature)
- Item 2: Data retention policies and cleanup jobs (can be added later)
- Item 3: Advanced analytics or aggregation queries (separate service)
- Item 4: Authentication/authorization for collector endpoint (daemon is trusted for now)
- Item 5: Rate limiting or quota management (not needed initially)
- Item 6: WebSocket streaming or real-time updates (batch processing is sufficient)

## Constraints

- Must follow hexagonal architecture pattern used in existing API services (health, dashboard, aggregation, project)
- Must use ConnectRPC for gRPC implementation (not standard gRPC)
- Must follow file structure conventions: `inbound/grpc/connectrpc/` for handlers
- Must use `fx.Annotate` with `fx.As` and `fx.ResultTags` for DI registration (see register_connectrpc.go)
- Logger must be injected as first parameter and bound with pattern `collector.service`, `collector.grpc.connectrpc`, `collector.repository.mongodb`
- Must use `*slog.Logger` type for logging (not structured logger interface)
- Must return domain models from service layer, not protobuf types
- Repository must accept `*mongo.Database` from platform setup (not initialize MongoDB itself)
- Error handling must use `github.com/team-attention/cops/api/internal/platform/util/errutil` package
- Must verify interface implementation with compile-time check: `var _ collectorv1connect.CollectorServiceHandler = (*CollectorGRPCHandler)(nil)`

## Additional Context

- Protobuf schema is already defined in `idl/protobuf/collector/v1/collector.proto`
- Generated ConnectRPC code exists in `shared/gen/grpcstub/collector/v1/collectorv1connect/`
- Daemon is already calling SendRecords endpoint via `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- Daemon converts domain.LogBatch to collectorv1.SendRecordsReq with ProjectMetadata and SessionRecords
- SessionRecord contains: UUID, ParentUUID, SessionID, Type, Role, Content, Timestamp, CWD, GitBranch, Version, UsageMetadata
- ProjectMetadata contains: ID, Name, Path, GitProject flag
- UsageMetadata contains token counts for tracking Claude API usage
- Similar implementations exist in health, dashboard, aggregation services that can be referenced for patterns
- MongoDB connection is initialized in `api/internal/platform/setup/mongodb/mongodb.go` and provided via DI

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should records be validated before storage? | Yes - validate required fields (UUID, SessionID, Timestamp) and log warnings for malformed data without blocking entire batch |
| What should happen with duplicate UUIDs? | Use upsert strategy - update existing record if UUID exists, insert if new. This handles daemon retries gracefully |
| Should there be a batch size limit? | No hard limit initially - accept batches as sent by daemon. Can add limits later if performance issues arise |
| What indexes are needed on MongoDB collection? | Primary: UUID (unique), Secondary: SessionID + Timestamp for session queries, ProjectID for project filtering |
| Should successful storage be acknowledged per-record or per-batch? | Per-batch - return total records_received and success=true if batch processed. Log individual record errors but don't fail batch |
| Should we store ProjectMetadata separately or embedded? | Store ProjectMetadata fields within each SessionRecord document for query flexibility. Can normalize later if needed |
| What timezone should timestamps use? | UTC - convert protobuf Timestamp to time.Time in UTC for storage |
