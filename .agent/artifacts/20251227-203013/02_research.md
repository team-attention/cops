# Research Report

## Mode
General Research

## Request Summary
Investigate why session records and project data exist in MongoDB but do not appear in the web dashboard. The user confirmed data is being stored successfully by the daemon, but the dashboard is not displaying it.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File                                                                                   | Reason                                                      |
| -------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`         | Defines collection name constants for sessionRecords        |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`                | Defines collection name constants for projects              |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` | Dashboard repository that reads from MongoDB - key file to fix |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | Aggregation repository that writes to MongoDB               |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` | Project repository - writes projects to MongoDB             |
| `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go`         | Fiber server setup - missing CORS middleware                |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`                      | Rules for outbound adapter implementation                   |

## Collection Name Analysis

### Collection Names Defined (shared/domain/mongoschema/)

| Collection            | Constant Name                      | Value            |
| --------------------- | ---------------------------------- | ---------------- |
| Projects              | `ProjectCollectionName`            | `"projects"`     |
| Session Records       | `SessionRecordCollectionName`      | `"sessionRecords"` |

### Collection Name Usage in Code

| Service       | File                                                                      | Collection Reference                         |
| ------------- | ------------------------------------------------------------------------- | -------------------------------------------- |
| Dashboard     | `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:31` | `db.Collection(mongoschema.ProjectCollectionName)` |
| Dashboard     | `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:32` | `db.Collection(mongoschema.SessionRecordCollectionName)` |
| Aggregation   | `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:27` | `db.Collection(mongoschema.SessionRecordCollectionName)` |
| Project       | `api/internal/service/project/outbound/repository/mongodb/project_repo.go:25` | `db.Collection(mongoschema.ProjectCollectionName)` |

**Finding: Collection names are CONSISTENT across all services.** Both writer (aggregation, project) and reader (dashboard) use the same constants from `mongoschema` package.

## Data Flow Analysis

### Write Path (Daemon -> API -> MongoDB)
```
1. Daemon parses JSONL logs from ~/.claude/projects/{encoded-path}/
2. Daemon calls API via gRPC: AggregationService.SendLogs
3. API Handler: api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go
4. API Service: api/internal/service/aggregation/aggregation_service.go
5. API Repository: api/internal/service/aggregation/outbound/repository/mongodb/adapter.go
   -> Writes to collection: "sessionRecords"
```

### Read Path (Web -> API -> MongoDB)
```
1. Web calls API via ConnectRPC: DashboardService.GetOverview
   - Hook: web/src/feature/dashboard/hook/use-get-overview.ts
   - Transport: web/src/shared/service/connect-transport.ts (baseUrl: localhost:8080)
2. API Handler: api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go
3. API Service: api/internal/service/dashboard/dashboard_service.go
4. API Repository: api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go
   -> Reads from collections: "projects", "sessionRecords"
```

### Project Registration Path
```
1. CLI: cops add [directory]
2. CLI calls API via gRPC: ProjectService.RegisterProject
3. API Handler: api/internal/service/project/inbound/grpc/connectrpc/handler.go
4. API Service: api/internal/service/project/project_service.go
5. API Repository: api/internal/service/project/outbound/repository/mongodb/project_repo.go
   -> FindOrCreate in collection: "projects" (only stores remoteUrl)
```

## Potential Root Causes

### 1. CRITICAL: Missing CORS Middleware
**Evidence:**
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go` initializes Fiber with only `recover` and `requestid` middleware
- No CORS middleware is configured
- Web frontend runs on a different port (likely 5173 for Vite dev server) than API (8080)
- Browser will block cross-origin requests without proper CORS headers

**Impact:** Frontend cannot communicate with API at all - all gRPC requests will fail with CORS errors.

**Fix Location:** `/Users/jayce/team-attention/cops/api/internal/platform/setup/server/fiber.go`

### 2. POTENTIAL: Project Data Not Fully Populated
**Evidence:**
- `project_repo.go:87-89` creates new project with only `remoteUrl` field:
  ```go
  newDoc := bson.M{
      mongoschema.ProjectRemoteURLField: remoteURL,
  }
  ```
- Dashboard reads `name`, `path`, `gitBranch` fields which may not exist in the database
- `dashboard_repo.go:219-222` tries to read fields that were never written:
  ```go
  Name: mongoutil.Get[string](doc, mongoschema.ProjectNameField),
  Path: mongoutil.Get[string](doc, mongoschema.ProjectPathField),
  ```

**Impact:** Projects appear in dashboard but with empty name/path fields, or aggregation pipelines may fail silently.

**Fix Location:** `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

### 3. POTENTIAL: Empty Results Not Logged/Debugged
**Evidence:**
- Dashboard repository methods silently return empty results when no data is found
- No debug logging when aggregation pipelines return empty results
- Frontend shows "No projects tracked yet" without indicating if API call succeeded

**Impact:** Hard to distinguish between "no data" and "API call failed"

### 4. POTENTIAL: MongoDB Aggregation Pipeline Issues
**Evidence:**
- Complex aggregation pipelines in `dashboard_repo.go` for joining projects with session records
- `$lookup` stages reference field names that must match exactly
- If `projectId` field is stored incorrectly (e.g., as string vs ObjectID), joins will fail silently

**Specific Concern:** Line 131 uses `$expr` comparison:
```go
bson.M{"$match": bson.M{
    "$expr": bson.M{"$eq": bson.A{"$" + mongoschema.SessionRecordProjectIDField, "$$projectId"}},
}},
```
This requires `projectId` in sessionRecords to be an ObjectID, not a string.

**Fix Location:** Verify data types match between writer and reader.

## Similar Implementations Found

### Example 1: Aggregation Repository Write Pattern
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:36-46`
- **Relevance**: Shows correct pattern for converting projectId to ObjectID before storage

### Example 2: Dashboard Repository Read Pattern
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:284-292`
- **Relevance**: Shows how projectId is used in $match stage - expects ObjectID type

## Package Candidates

### Problem 1: CORS Middleware

| Package                  | Context7 ID                | Why Better Than Alternatives                                |
| ------------------------ | -------------------------- | ----------------------------------------------------------- |
| gofiber/fiber/v2/middleware/cors | (stdlib Fiber)      | Native Fiber middleware, zero config for development        |
| rs/cors                  | `/rs/cors`                 | More customizable, but requires adapter for Fiber           |

**Recommendation:** Use Fiber's built-in CORS middleware - simplest solution for this project.

### Problem 2: Debugging/Logging

| Package  | Context7 ID   | Why Better Than Alternatives                      |
| -------- | ------------- | ------------------------------------------------- |
| log/slog | (stdlib)      | Already in use, structured logging, zero deps     |

**Recommendation:** Add debug-level logs to trace data flow.

## Technical Constraints
- Fiber v2 is used as HTTP framework
- ConnectRPC is used for gRPC communication
- MongoDB driver v2 is used
- Frontend uses TanStack Query with ConnectRPC transport
- All collection names use camelCase convention

## Recommendations for Fixing

### Priority 1: Add CORS Middleware (Immediate)
```go
// In api/internal/platform/setup/server/fiber.go
import "github.com/gofiber/fiber/v2/middleware/cors"

app.Use(cors.New(cors.Config{
    AllowOrigins: "*", // Or specific origins for production
    AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
    AllowHeaders: "Origin, Content-Type, Accept, Connect-Protocol-Version",
}))
```

### Priority 2: Verify MongoDB Data Structure
- Check MongoDB directly: what fields exist in `projects` collection?
- Check MongoDB directly: what type is `projectId` in `sessionRecords` collection?
- If type mismatch exists, either fix the writer or adjust the reader

### Priority 3: Add Debug Logging
- Add debug-level logs in dashboard repository before and after aggregation
- Log the count of results at each stage

## Additional Information for Planning

1. **No CORS = Complete Failure**: If CORS is the issue, the frontend cannot communicate with the API at all. This should be checked first.

2. **Docker Compose Configuration**: The API runs on port 8080, MongoDB on 27017. Web runs via Vite on a different port.

3. **Environment Variables**: Web transport uses `VITE_API_URL` env var with fallback to `http://localhost:8080`.

4. **No Authentication**: The current implementation has no auth middleware, so auth is not blocking requests.

5. **Data Type Sensitivity**: MongoDB aggregation `$lookup` with `$expr` is type-sensitive. String "abc123" will not match ObjectID("abc123").
