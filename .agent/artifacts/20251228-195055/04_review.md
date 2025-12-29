# Pre-PR Code Review

## Review Summary
- **Status**: FAIL
- **Files Reviewed**: 8
- **Issues Found**: 2 (Critical: 2, Warning: 0, Info: 0)

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go vet ./...` | PASS |

---

## Critical Issues Identified

### Critical Issue 1: Function Complexity in Repository

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
**Function**: `FindOrCreate` (lines 35-142)

**Problem**: The function body is too complex with 107 lines of code containing:
- Sequential query logic (first by ID, then by URL)
- Conditional branching based on `IsGitProject`
- Multiple error handling blocks with repeated patterns
- Document creation logic mixed with query logic

**Required Fix**: Simplify to a single `$or` query that handles all search conditions at once.

---

### Critical Issue 2: Business Logic in Repository Layer (Architecture Violation)

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
**Function**: `FindOrCreate` (lines 35-142)

**Rule Violated**: `.agent/rules/go/go-hexagonal-layout.md`, `.agent/rules/go/go-outbound.md`, `.agent/rules/go/go-service.md`

**Problem**: The repository is making a business decision:

| Current Repository Logic | Should Be In |
|-------------------------|--------------|
| `if params.IsGitProject` - deciding when to search by URL | Not needed at all |

**Key Insight**: There is NO need to check `IsGitProject` anywhere. Just pass all parameters through. Empty values won't match in MongoDB anyway. The "business logic" is already handled upstream in the CLI layer which decides what URLs to collect.

---

## Execution Plan for Execute Agent

The refactoring is extremely simple - just remove all conditionals and pass everything through.

### Design Principle: No Conditionals Needed

**Why no `if IsGitProject` check is needed anywhere**:
- Non-Git projects have empty `ConfiguredURL` and `ActualURL` (collected as empty in CLI)
- Empty strings won't match any documents in MongoDB
- The `$or` query naturally handles this: if URLs are empty, only ID matching works
- The repository just builds a query from whatever it receives

```
CLI (collects data) → Service (passes through) → Repository (builds $or query)
                                                        ↓
                                          Empty URLs = no URL condition added
                                          Non-empty URLs = URL condition added
```

---

### Step 1: Simplify Repository - Single $or Query, No Conditionals

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

```go
package mongodb

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"

    "github.com/team-attention/cops/api/internal/service/project/outbound/repository"
    "github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoProjectRepository struct {
    logger       *slog.Logger
    projectsColl *mongo.Collection
}

func NewMongoProjectRepository(l *slog.Logger, db *mongo.Database) *MongoProjectRepository {
    return &MongoProjectRepository{
        logger:       l.With(slog.String("name", "project.repository.mongodb")),
        projectsColl: db.Collection(mongoschema.ProjectCollectionName),
    }
}

// FindOrCreate finds existing project by ID or URLs, or creates a new one.
// No business logic - just builds query from provided parameters.
// Empty values are naturally filtered out when building conditions.
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
    // Build $or conditions from provided parameters
    conditions := r.buildSearchConditions(params)

    // If we have search conditions, try to find existing project
    if len(conditions) > 0 {
        project, err := r.findByConditions(ctx, conditions)
        if err != nil {
            return nil, err
        }
        if project != nil {
            r.logger.Info("found existing project",
                slog.String("projectID", project.ProjectID))
            return project, nil
        }
    }

    // Not found, create new project
    return r.createProject(ctx, params)
}

// buildSearchConditions creates $or conditions from params.
// Only adds conditions for non-empty values.
func (r *MongoProjectRepository) buildSearchConditions(params repository.FindOrCreateParams) []bson.M {
    conditions := make([]bson.M, 0, 2)

    // Add ID condition if provided
    if params.ExistingID != "" {
        if objectID, err := bson.ObjectIDFromHex(params.ExistingID); err == nil {
            conditions = append(conditions, bson.M{mongoschema.ProjectIDField: objectID})
        }
    }

    // Add URL conditions if provided (using $in for multiple URLs)
    urls := r.collectNonEmptyURLs(params.ConfiguredURL, params.ActualURL)
    if len(urls) > 0 {
        conditions = append(conditions, bson.M{
            mongoschema.ProjectRemoteURLField: bson.M{"$in": urls},
        })
    }

    return conditions
}

// collectNonEmptyURLs gathers non-empty, unique URLs.
func (r *MongoProjectRepository) collectNonEmptyURLs(configuredURL, actualURL string) []string {
    urls := make([]string, 0, 2)
    if configuredURL != "" {
        urls = append(urls, configuredURL)
    }
    if actualURL != "" && actualURL != configuredURL {
        urls = append(urls, actualURL)
    }
    return urls
}

// findByConditions executes the $or query.
func (r *MongoProjectRepository) findByConditions(ctx context.Context, conditions []bson.M) (*repository.FindOrCreateResult, error) {
    filter := bson.M{"$or": conditions}
    var doc bson.M
    err := r.projectsColl.FindOne(ctx, filter).Decode(&doc)

    if err == mongo.ErrNoDocuments {
        return nil, nil
    }
    if err != nil {
        r.logger.Error("failed to find project", slog.Any("error", err))
        return nil, fmt.Errorf("failed to find project: %w", err)
    }

    return r.docToResult(doc), nil
}

// createProject inserts a new project document.
func (r *MongoProjectRepository) createProject(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
    // Prefer configured URL, fallback to actual URL
    remoteURL := params.ConfiguredURL
    if remoteURL == "" {
        remoteURL = params.ActualURL
    }

    newDoc := bson.M{
        mongoschema.ProjectRemoteURLField:    remoteURL,
        mongoschema.ProjectNameField:         params.Name,
        mongoschema.ProjectIsGitProjectField: params.IsGitProject,
        mongoschema.ProjectRegisteredAtField: time.Now(),
    }

    result, err := r.projectsColl.InsertOne(ctx, newDoc)
    if err != nil {
        r.logger.Error("failed to create project", slog.Any("error", err))
        return nil, fmt.Errorf("failed to create project: %w", err)
    }

    newID := result.InsertedID.(bson.ObjectID).Hex()
    r.logger.Info("created new project",
        slog.String("projectID", newID),
        slog.String("name", params.Name))

    return &repository.FindOrCreateResult{
        ProjectID:    newID,
        IsNew:        true,
        Name:         params.Name,
        IsGitProject: params.IsGitProject,
    }, nil
}

// docToResult converts a MongoDB document to FindOrCreateResult.
func (r *MongoProjectRepository) docToResult(doc bson.M) *repository.FindOrCreateResult {
    return &repository.FindOrCreateResult{
        ProjectID:    doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex(),
        IsNew:        false,
        Name:         doc[mongoschema.ProjectNameField].(string),
        IsGitProject: doc[mongoschema.ProjectIsGitProjectField].(bool),
    }
}

var _ repository.ProjectRepositoryPort = (*MongoProjectRepository)(nil)
```

**Key Points**:
1. NO `if params.IsGitProject` check - removed entirely
2. `collectNonEmptyURLs` naturally filters out empty strings
3. Single `$or` query handles all cases
4. Helper methods for readability: `buildSearchConditions`, `collectNonEmptyURLs`, `findByConditions`, `createProject`, `docToResult`

---

### Step 2: Service Layer - Just Pass Everything Through

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`

The service layer becomes a simple pass-through with no conditionals:

```go
package project

import (
    "context"
    "log/slog"

    "github.com/team-attention/cops/api/internal/service/project/outbound/repository"
)

// RegisterProjectParams contains parameters for registering a project.
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    Name                string
    IsGitProject        bool
}

// Service implements project business logic.
type Service struct {
    logger *slog.Logger
    repo   repository.ProjectRepositoryPort
}

// NewService creates a new project service.
func NewService(l *slog.Logger, repo repository.ProjectRepositoryPort) *Service {
    return &Service{
        logger: l.With(slog.String("name", "project.service")),
        repo:   repo,
    }
}

// RegisterProject registers a project or returns existing project ID if already registered.
// Simply passes all parameters to repository - no conditionals needed.
// Empty URLs naturally won't match in the database query.
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
    // Just pass everything through - no conditionals
    result, err := s.repo.FindOrCreate(ctx, repository.FindOrCreateParams{
        ExistingID:    params.ExistingProjectID,
        ConfiguredURL: params.ConfiguredRemoteURL,
        ActualURL:     params.ActualRemoteURL,
        Name:          params.Name,
        IsGitProject:  params.IsGitProject,
    })
    if err != nil {
        s.logger.Error("failed to register project", slog.Any("error", err))
        return nil, err
    }

    s.logger.Info("project registered",
        slog.String("projectID", result.ProjectID),
        slog.Bool("isNew", result.IsNew),
        slog.String("name", result.Name),
        slog.Bool("isGitProject", result.IsGitProject))

    return result, nil
}
```

**Key Points**:
1. NO `if params.IsGitProject` check
2. Just pass all parameters directly to repository
3. The "business logic" is implicit: CLI collects URLs for Git projects, empty for non-Git

---

### Summary of Changes

| Layer | Before | After |
|-------|--------|-------|
| **Repository** | 107 lines, sequential queries, `if IsGitProject` check | ~75 lines, single `$or` query, no conditionals |
| **Service** | Thin wrapper, just passes params through | Still a thin wrapper, unchanged in purpose |
| **Interface** | No change needed | No change needed |

**Benefits of This Approach**:
1. **Simplest possible implementation** - no conditionals anywhere
2. **Single query** instead of multiple sequential queries
3. **Natural filtering** - empty values don't add conditions
4. **Clear separation** - CLI decides what to collect, layers just pass through
5. **Easier to test** - no branching logic to cover

---

## Files Reviewed (Previously Passing)

### `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Plan Step**: Step 1 - Fix API Validation for Non-Git Projects

**Implementation Verification** (Original):
- [x] Removed validation that required at least one search criterion
- [x] Restructured logic to: ExistingID -> URL (git projects only) -> Create new
- [x] Non-Git projects without existingID now proceed directly to creation
- [x] ExistingID lookup is now step 1 (was combined with URL)
- [x] URL-based dedup only applies when `IsGitProject == true`
- [x] Error handling maintains proper pattern (check ErrNoDocuments separately)
- [x] Logging updated with descriptive messages ("found by ID", "found by URL")

**New Issues Found**:
- [CRITICAL] Function complexity - needs refactoring into helper methods
- [CRITICAL] Unnecessary `if params.IsGitProject` check - should be removed entirely

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**Plan Step**: Step 2 - Add Parent Detection Method to Service Layer

**Implementation Verification**:
- [x] `ParentProjectInfo` struct added with exported name
- [x] Fields: `ID domain.ID`, `Name string`, `Path string` - all value types (correct per `go-struct.md` for required fields)
- [x] `FindParentProject(targetPath string) (*ParentProjectInfo, error)` implemented
- [x] Path expansion using existing `pathutil.ExpandPath`
- [x] Uses `s.configRepo.LoadGlobalConfig()` as planned (proper layer separation)
- [x] Walks up directory tree correctly (parent of target path first)
- [x] Returns `nil, nil` when no parent found (not an error)
- [x] Root check handles both "/" and "." cases

**Rule Compliance**:
- [x] Follows `go-service.md` pattern
- [x] Uses existing dependencies (no new imports required)
- [x] Error handling uses `errutil` package

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`

**Plan Step**: Step 4 - Update Add Command to Pass Service to TUI

**Implementation Verification**:
- [x] `runAddTUI` call updated to include `h.svc` parameter
- [x] Minimal change, correctly passes service reference

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`

**Plan Step**: Step 3 - Add Parent Detection Step to CLI TUI

**Implementation Verification**:
- [x] New step constant `stepParentDetection = iota` added at position 0
- [x] `stepGitSelection` correctly shifted to position 1
- [x] `parentProject *tracking.ParentProjectInfo` field added to addModel
- [x] `parentCursor int` field added for Yes/No selection
- [x] `service *tracking.Service` field added for parent detection
- [x] `newAddModel` signature updated with service parameter
- [x] Initial step set to `stepParentDetection`
- [x] `Init()` returns `m.detectParentProject` command
- [x] `parentDetectionMsg` type defined with parent and err fields
- [x] `detectParentProject()` method calls service correctly
- [x] `runAddTUI` signature updated with service parameter

**Rule Compliance**:
- [x] Follows existing TUI pattern in codebase
- [x] Import of tracking package for `ParentProjectInfo` type

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`

**Plan Step**: Step 3 - Add Parent Detection Step to CLI TUI (Update logic)

**Implementation Verification**:
- [x] `parentDetectionMsg` case handler added in `Update()`
- [x] Error handling: sets `m.err` and quits on error
- [x] If parent is nil, proceeds to `stepGitSelection` with `detectGitRepos` command
- [x] If parent exists, stays on `stepParentDetection` for user confirmation
- [x] `stepParentDetection` case added in KeyMsg switch
- [x] `updateParentSelection` method implemented:
  - [x] Handles ctrl+c, esc, n, N for cancel
  - [x] Handles up/down (k/j) for cursor navigation
  - [x] Handles enter for selection (cursor 0 = Yes, cursor 1 = No)
  - [x] Handles y/Y for quick Yes selection
- [x] Cursor bounds checked correctly (0 to 1)

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`

**Plan Step**: Step 3 - Add Parent Detection Step to CLI TUI (View logic)

**Implementation Verification**:
- [x] `stepParentDetection` case added in `View()` switch
- [x] `viewParentConfirmation()` method implemented:
  - [x] Shows "Checking for parent projects..." when parentProject is nil
  - [x] Displays title "Parent Project Detected"
  - [x] Shows parent project name and path
  - [x] Renders Yes/No options with cursor
  - [x] Shows help text with navigation instructions
- [x] Uses existing style patterns (`titleStyle`, `cursorStyle`, `helpStyle`)

---

### `/Users/jayce/team-attention/cops/daemon/internal/platform/util/pathutil/pathutil.go`

**Plan Step**: Step 6 - Add Path Decode Function to Daemon Pathutil

**Implementation Verification**:
- [x] `DecodeClaudeProjectDir(claudeDir string) string` function added
- [x] Gets home directory and handles error (returns empty)
- [x] Builds prefix: `~/.claude/projects/` with proper separator
- [x] Checks HasPrefix for prefix validation
- [x] Returns empty string for invalid prefix
- [x] Extracts encoded portion after prefix
- [x] Returns empty string for empty encoded part
- [x] Decodes by replacing `-` with `/`
- [x] Validates result starts with `/`
- [x] Returns decoded path or empty string on failure

**Rule Compliance**:
- [x] Follows `go-platform.md` util package guidelines

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`

**Plan Step**: Step 5 - Implement Hierarchical Project Matching in Daemon

**Implementation Verification**:
- [x] `projectPathToID map[string]shareddomain.ID` field added to Service struct
- [x] Map initialized in `NewService`
- [x] `UpdateTargets` builds both mappings:
  - [x] `newClaudeDirMapping` for exact matches
  - [x] `newProjectPathMapping` for hierarchical matching
- [x] Both mappings updated atomically
- [x] Logging updated to show both mapping counts
- [x] `GetProjectIDForClaudeDir` updated:
  - [x] Tries exact match first (existing behavior preserved)
  - [x] On miss, decodes claudeDir using `pathutil.DecodeClaudeProjectDir`
  - [x] Returns empty on decode failure
  - [x] Iterates projectPathToID for prefix matching
  - [x] Uses `strings.HasPrefix(decodedPath, projectPath+"/")` for proper prefix check (avoids `/a/bc` matching `/a/b`)
  - [x] Tracks longest match for most specific project
  - [x] Returns matched ID or empty if no match

**Rule Compliance**:
- [x] Follows `go-service.md` pattern
- [x] Import of `strings` package added
- [x] Import of `pathutil` added for decode function
- [x] Thread-safe with existing mutex

---

## Plan Compliance Summary

| Step | Description | Status |
|------|-------------|--------|
| 1 | Fix API Validation for Non-Git Projects | NEEDS REFACTOR |
| 2 | Add Parent Detection Method to Service Layer | COMPLETE |
| 3 | Add Parent Detection Step to CLI TUI | COMPLETE |
| 4 | Update Add Command to Pass Service to TUI | COMPLETE |
| 5 | Implement Hierarchical Project Matching in Daemon | COMPLETE |
| 6 | Add Path Decode Function to Daemon Pathutil | COMPLETE |

## Architecture Compliance

- [ ] **Hexagonal Architecture**: Repository contains unnecessary conditional logic (VIOLATION)
- [x] **Layer Separation**: TUI does not directly access ConfigPort; uses Service method
- [x] **Go Struct Rules**: `ParentProjectInfo` uses value types for required fields
- [x] **Logging Conventions**: All new log statements use structured logging with slog
- [x] **Error Handling**: Uses errutil package in service layer

## Notes

1. **Untracked File**: `shared/domain/record.go` appears in git status but is unrelated to this implementation plan.

2. **Test Coverage**: Unit tests for the new functionality should be added in a follow-up task.

## Blocking Issues for PR

This review **FAILS** due to architecture violations. The Execute Agent must:

1. **Simplify repository** - Use single `$or` query instead of sequential queries
2. **Remove ALL conditionals** - No `if params.IsGitProject` check anywhere (not in repo, not in service)
3. **Extract helper methods** - Break down the complex function for readability

The fix is simple: just pass all parameters through and let empty values naturally filter themselves out.

Once these changes are made, request a re-review.
