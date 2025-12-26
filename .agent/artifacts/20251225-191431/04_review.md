# Pre-PR Code Review

## Review Summary
- **Status**: FAIL
- **Files Reviewed**: 22 (11 modified + 11 newly created)
- **Issues Found**: 7 (Critical: 3, Warning: 2, Info: 2)

---

## Issues Overview

| # | Severity | File | Issue | Status |
|---|----------|------|-------|--------|
| 1 | **Critical** | Multiple files (6 files) | Constructor naming convention violation - Rule enforcement | **Fix** |
| 2 | Warning | `api/internal/service/project/project_service.go` | Duplicate DTO between Service and Repository | Fix |
| 3 | Info | `cli/cmd/internal/container/module_tracking.go` | Unnecessary "adapter" suffix in imports | Fix |
| 4 | **Critical** | `cli/internal/service/tracking/tracking_service.go` | UUID generation should NOT happen locally | Fix |
| 5 | Warning | `cli/internal/service/tracking/tracking_service.go` | Unnecessary if-else branching | Fix |
| 6 | Info | `cli/internal/service/tracking/tracking_service.go` | Should use `lo` package for filtering | Fix |
| 7 | **Critical** | `idl/protobuf/project/v1/project.proto` | Protobuf needs regeneration after user changes | Fix |

---

## Detailed Issue Analysis

### Issue 1: Constructor naming convention violation - CODEBASE-WIDE REFACTORING

**Severity**: **Critical** (Rule enforcement)
**Files**: 6 files across API modules

**Verdict**: CRITICAL FIX - Rule in `.agent/rules/go/go-outbound.md` must be followed

**Rule** (`.agent/rules/go/go-outbound.md`):
```
### Struct Names
- `{Technology or Service}{Domain}{Category}` (ex. `MongoAgentRepository`)

### Constructor Names
- `New{StructName}(...)` (ex. `NewMongoAgentRepository`)
```

**All Violations Found in Codebase**:

| File | Current Struct | Current Constructor | Expected Struct | Expected Constructor |
|------|----------------|---------------------|-----------------|---------------------|
| `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | `Adapter` | `NewAdapter` | `MongoSessionRecordRepository` | `NewMongoSessionRecordRepository` |
| `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` | `Adapter` | `NewAdapter` | `MongoDashboardRepository` | `NewMongoDashboardRepository` |
| `api/internal/service/project/outbound/repository/mongodb/project_repo.go` | `Adapter` | `NewAdapter` | `MongoProjectRepository` | `NewMongoProjectRepository` |

**Container Module Updates Required**:

| File | Current | Expected |
|------|---------|----------|
| `api/cmd/internal/container/module_aggregation.go:17` | `mongodb.NewAdapter` | `mongodb.NewMongoSessionRecordRepository` |
| `api/cmd/internal/container/module_dashboard.go:17` | `mongodb.NewAdapter` | `mongodb.NewMongoDashboardRepository` |
| `api/cmd/internal/container/module_project.go:17` | `mongodb.NewAdapter` | `mongodb.NewMongoProjectRepository` |

**Required Fix**:

1. **In `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`**:
   - Rename `type Adapter struct` to `type MongoSessionRecordRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoSessionRecordRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoSessionRecordRepository)`

2. **In `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`**:
   - Rename `type Adapter struct` to `type MongoDashboardRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoDashboardRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoDashboardRepository)`

3. **In `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`**:
   - Rename `type Adapter struct` to `type MongoProjectRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoProjectRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoProjectRepository)`

4. **In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go`**:
   - Change `mongodb.NewAdapter` to `mongodb.NewMongoSessionRecordRepository`

5. **In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go`**:
   - Change `mongodb.NewAdapter` to `mongodb.NewMongoDashboardRepository`

6. **In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`**:
   - Change `mongodb.NewAdapter` to `mongodb.NewMongoProjectRepository`

---

### Issue 2: Duplicate DTO between Service and Repository

**Severity**: Warning
**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`
**Lines**: 17-21

**Current Code**:
```go
// In project_service.go (Service layer)
type RegisterProjectResult struct {
    ProjectID string
    IsNew     bool
}

// In project_repo_port.go (Repository layer)
type FindOrCreateResult struct {
    ProjectID string
    IsNew     bool
}
```

**Problem**: `RegisterProjectResult` in the Service layer is an exact duplicate of `FindOrCreateResult` in the Repository layer. This creates unnecessary code duplication.

**Rule Violated**: `.agent/rules/go/go-port-adapter-pattern.md` - DTO Model Patterns
> **Be Conservative with DTOs**: Check for existing models before creating DTOs. Minimize code. Only create DTOs when absolutely necessary.

**Required Fix**:
- Remove `RegisterProjectResult` from the Service layer
- Reuse `repository.FindOrCreateResult` directly in the Service

**Suggested Change in** `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`:
```go
// Before
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*RegisterProjectResult, error) {
    result, err := s.repo.FindOrCreate(...)
    // ...
    return &RegisterProjectResult{
        ProjectID: result.ProjectID,
        IsNew:     result.IsNew,
    }, nil
}

// After
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
    result, err := s.repo.FindOrCreate(...)
    // ...
    return result, nil  // Return directly, no conversion needed
}
```

---

### Issue 3: Unnecessary "adapter" suffix in imports

**Severity**: Info
**File**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`
**Lines**: 7-13

**Verdict**: VALID - Requires fix

**Current Code (CLI module - INCORRECT)**:
```go
import (
    "go.uber.org/dig"

    "github.com/team-attention/cops/cli/internal/service/tracking"
    cliadapter "github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
    connectrpcadapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
    configport "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
    filesystemadapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem"
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
    jsonladapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"
)
```

**Existing Patterns in Codebase**:

**API modules** (`module_aggregation.go`, `module_dashboard.go`) - No alias when no conflict:
```go
import (
    "github.com/.../outbound/repository/mongodb"  // No alias
    "github.com/.../inbound/grpc/connectrpc"      // No alias
)
// Usage: mongodb.NewAdapter, connectrpc.NewHandler
```

**Daemon module** (`module_log.go`) - Alias only when there's a naming conflict:
```go
import (
    fsnotifyadapter "github.com/.../outbound/filesystem/fsnotify"
    fsnotifyhandler "github.com/.../inbound/worker/fsnotify"
)
// Pattern: {package}adapter or {package}handler when there's a naming conflict
```

**Established Convention**:
- Use **no alias** when package name does not conflict
- Use `{package}adapter` or `{package}handler` suffix **ONLY** when there is a naming conflict

**Problem**: CLI module uses aliases where no conflicts exist.

**Required Fix** in `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`:
```go
import (
    "go.uber.org/dig"

    "github.com/team-attention/cops/cli/internal/service/tracking"
    "github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"       // No alias needed
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc" // No alias needed
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"         // Was configport
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem" // No alias needed
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
    "github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"   // No alias needed
)
// Then update usages: cobra.NewHandler, connectrpc.NewClient, config.Port, filesystem.NewAdapter, jsonl.NewParser
```

---

### Issue 4: UUID generation should NOT happen locally (CRITICAL)

**Severity**: **Critical**
**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
**Lines**: 127-132 (and lines 117-119, 137-142)

**Current Code**:
```go
// Lines 127-132: When no remote URL exists
} else {
    // No remote URL, use existing or generate new
    if existingProjectID != "" {
        projectID = domain.ID(existingProjectID)
    } else {
        projectID = domain.NewID(uuid.New().String())  // PROBLEM HERE
    }
    s.logger.Debug("no remote URL, using local ID generation",
        slog.String("id", projectID.String()))
}

// Lines 117-119: API failure fallback
} else {
    projectID = domain.NewID(uuid.New().String())  // PROBLEM HERE
}

// Lines 137-142: Non-git project fallback
} else {
    // Non-git project: use existing or generate new UUID
    if existingProjectID != "" {
        projectID = domain.ID(existingProjectID)
    } else {
        projectID = domain.NewID(uuid.New().String())  // PROBLEM HERE
    }
```

**Problem**: Local UUID generation defeats the entire purpose of API-coordinated Project IDs. The core requirement is that the API server is the single source of truth for Project IDs.

**Expected Behavior**:
1. If no remote URL exists, send empty strings to the API and let the server handle it
2. If API is unreachable, either:
   - Fail gracefully with an error (preferred for data consistency)
   - OR use existing local ID if available, but NEVER generate new UUID locally

**Required Fix**:
- Remove all `uuid.New().String()` calls for new project ID generation
- When no remote URL exists, still call the API with empty strings
- Let the API server be the sole generator of Project IDs
- If API fails and no existing ID exists, return an error instead of generating locally

**Suggested Approach**:
```go
// Always attempt API registration, even with empty URLs
result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
    ConfiguredRemoteURL: configuredURL,  // May be empty
    ActualRemoteURL:     actualURL,      // May be empty
    ExistingProjectID:   existingProjectID,
})
if err != nil {
    // If we have an existing local ID, use it
    if existingProjectID != "" {
        projectID = domain.ID(existingProjectID)
        s.logger.Warn("failed to register with API, using existing local ID",
            slog.Any("error", err))
    } else {
        // No existing ID and API unreachable - FAIL
        return nil, errutil.Internalf("cannot register project: API unreachable and no existing local ID")
    }
} else {
    projectID = result.ProjectID
}
```

---

### Issue 5: Unnecessary if-else branching

**Severity**: Warning
**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
**Line**: 105 onwards

**Current Code**:
```go
// Lines 99-145: Complex branching for git vs non-git projects
if isGitProject {
    configuredURL, _ := gitutil.GetRemoteURL(projectPath)
    actualURL := gitutil.GetActualRemoteURL(projectPath)

    // Only call API if at least one remote URL exists
    if configuredURL != "" || actualURL != "" {
        result, err := s.project.RegisterProject(...)
        if err != nil {
            // fallback logic...
        } else {
            projectID = result.ProjectID
        }
    } else {
        // No remote URL fallback...
    }
} else {
    // Non-git project fallback...
}
```

**Problem**: Overly complex branching structure with redundant logic. The API should handle empty URLs gracefully, eliminating the need for client-side branching.

**Expected Behavior**: Just send whatever URLs exist (even if empty) to the API. The API can handle it.

**Required Fix**:
- Simplify to always call the API with whatever values are available
- Remove the `if configuredURL != "" || actualURL != ""` check
- Remove the separate `isGitProject` branch (the API doesn't care)

**Suggested Simplified Logic**:
```go
// Get URLs (empty strings if not available)
configuredURL := ""
actualURL := ""
if isGitProject {
    configuredURL, _ = gitutil.GetRemoteURL(projectPath)
    actualURL = gitutil.GetActualRemoteURL(projectPath)
}

// Always call API
result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
    ConfiguredRemoteURL: configuredURL,
    ActualRemoteURL:     actualURL,
    ExistingProjectID:   existingProjectID,
})
if err != nil {
    // Handle error...
} else {
    projectID = result.ProjectID
}
```

---

### Issue 6: Should use `lo` package for filtering

**Severity**: Info
**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
**Lines**: 304-313

**Current Code**:
```go
// Filter out project with matching path
filtered := make([]*domain.Project, 0, len(globalConfig.Projects))
found := false
for _, p := range globalConfig.Projects {
    if p.Path != absPath {
        filtered = append(filtered, p)
    } else {
        found = true
    }
}
```

**Problem**: Manual slice filtering with for-loop is verbose. The `samber/lo` package provides cleaner functional utilities.

**Rule Reference**: `.agent/rules/common.md` - Dependency Rule
> If there are packages available, use them as much as possible. Those packages must be well-tested.

**Required Fix**:
- Use `lo.Filter` from `samber/lo` package for cleaner code

**Suggested Change**:
```go
import "github.com/samber/lo"

// ...

filtered := lo.Filter(globalConfig.Projects, func(p *domain.Project, _ int) bool {
    return p.Path != absPath
})
found := len(filtered) < len(globalConfig.Projects)
```

---

### Issue 7: Protobuf needs regeneration (CRITICAL)

**Severity**: **Critical**
**File**: `idl/protobuf/project/v1/project.proto`

**Problem**: User has modified the Protobuf definition - removed Git Branch concept from Project. The generated Go code needs to be regenerated.

**Reason Provided**: Main branch is the only one managed as a project; branches only differ in Sessions.

**Action Required**:
1. User to confirm protobuf changes are complete
2. Run: `cd idl/protobuf && buf generate`
3. Verify generated code in `shared/gen/grpcstub/project/v1/`
4. Update any code that depends on removed fields

---

## Execution Plan for Execute Agent

To pass this review, Execute Agent must complete the following changes **in order**:

### Phase 1: Protobuf Regeneration (Blocking)

1. **Confirm user's protobuf changes are finalized**
2. **Regenerate protobuf**:
   ```bash
   cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
   ```

### Phase 2: API Server Fixes - Naming Convention Refactoring (Issue 1 - CRITICAL)

3. **Rename aggregation repository** in `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`:
   - Rename `type Adapter struct` to `type MongoSessionRecordRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoSessionRecordRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoSessionRecordRepository)`

4. **Rename dashboard repository** in `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`:
   - Rename `type Adapter struct` to `type MongoDashboardRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoDashboardRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoDashboardRepository)`

5. **Rename project repository** in `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`:
   - Rename `type Adapter struct` to `type MongoProjectRepository struct`
   - Rename `func NewAdapter(...)` to `func NewMongoProjectRepository(...)`
   - Update all method receivers from `(a *Adapter)` to `(r *MongoProjectRepository)`

6. **Update container module references**:
   - In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_aggregation.go:17`: Change `mongodb.NewAdapter` to `mongodb.NewMongoSessionRecordRepository`
   - In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_dashboard.go:17`: Change `mongodb.NewAdapter` to `mongodb.NewMongoDashboardRepository`
   - In `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go:17`: Change `mongodb.NewAdapter` to `mongodb.NewMongoProjectRepository`

### Phase 3: API Server Fixes - DTO Consolidation (Issue 2)

7. **Consolidate DTOs** in `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`:
   - Remove `RegisterProjectResult` struct
   - Change `RegisterProject` return type to `*repository.FindOrCreateResult`
   - Update handler that calls this method if needed

### Phase 4: CLI Fixes (Critical)

8. **Remove local UUID generation** (Issue 4) in `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`:
   - Remove `github.com/google/uuid` import
   - Remove all `uuid.New().String()` calls
   - Always call API for project registration
   - If API fails and no existing ID, return error instead of generating UUID locally

9. **Simplify branching logic** (Issue 5) in same file:
   - Remove the `if configuredURL != "" || actualURL != ""` conditional
   - Remove separate `isGitProject` branches for ID generation
   - Unified flow: get URLs -> call API -> handle result

10. **Refactor to use `lo.Filter`** (Issue 6) in same file:
    - Add `github.com/samber/lo` dependency: `go get github.com/samber/lo`
    - Replace manual filtering loop with `lo.Filter`

11. **Remove unnecessary import aliases** (Issue 3) in `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`:
    - Remove aliases from imports where no naming conflict exists
    - Update usages to use package names directly: `cobra.NewHandler`, `connectrpc.NewClient`, `config.Port`, `filesystem.NewAdapter`, `jsonl.NewParser`

---

## Test Verification Requirements

After all fixes are applied:

- [ ] Protobuf regeneration completed: `cd idl/protobuf && buf generate`
- [ ] All tests pass: `go test ./cli/...`
- [ ] All tests pass: `go test ./api/...`
- [ ] Build succeeds: `go build ./cli/...`
- [ ] Build succeeds: `go build ./api/...`
- [ ] No new linter warnings

---

## Summary

This review identified **7 issues**, **all require fixes** (no skips):

| Priority | Count | Description | Action |
|----------|-------|-------------|--------|
| Critical | 3 | Naming convention (codebase-wide), Local UUID generation, Protobuf regeneration | **Fix** |
| Warning | 2 | DTO duplication, Complex branching | **Fix** |
| Info | 2 | Import aliases, lo.Filter usage | **Fix** |

**Issue 1 (Constructor Naming)** has been escalated from SKIP to **CRITICAL FIX** per user feedback. The rule in `.agent/rules/go/go-outbound.md` must be enforced across the entire codebase. This requires renaming 3 repository structs and updating 3 container module references.

The most critical issues are:
1. **Issue 1 (Naming Convention)** - All existing code violating `.agent/rules/go/go-outbound.md` must be fixed
2. **Issue 4 (Local UUID Generation)** - Fundamentally violates the core requirement that the API server should be the single source of truth for Project IDs
3. **Issue 7 (Protobuf Regeneration)** - Blocks all other work

**Status**: FAIL - All 7 issues must be resolved before PR creation.
