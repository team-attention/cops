# Development Walkthrough

## Summary
Added project ID tracking to the daemon by reading local configuration files (`{projectPath}/.cops/config.json`) and populating `LogBatch.ProjectID` when sending logs to the API. Records are now grouped by ProjectID in the buffer to send accurate per-project batches. Projects and worktrees without local config are skipped from watching.

## Code Overview

### New Components

#### `LocalConfigPort` (Interface)
- **Location**: `daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go`
- **Purpose**: Defines interface for reading local project configurations
- **Key Methods**:
  - `LoadLocalConfig(projectPath string) (*LocalConfig, error)`: Loads the local configuration from `{projectPath}/.cops/config.json`

#### `FilesystemLocalConfigAdapter` (Adapter)
- **Location**: `daemon/internal/service/configwatcher/outbound/localconfig/filesystem/filesystem_localconfig.go`
- **Purpose**: Implements LocalConfigPort using filesystem operations
- **Implementation Details**:
  - Reads `.cops/config.json` from project directories
  - Uses `sonic` for JSON parsing (consistent with existing daemon code)
  - Returns error if file does not exist or cannot be parsed
  - Implements compile-time interface verification

### Modified Components

#### `WatchTarget` Domain Model
- **Location**: `daemon/internal/platform/domain/watch.go`
- **Changes**:
  - Added `ProjectID shareddomain.ID` field (line 26)
  - Changed `LogBatch.ProjectID` from `string` to `shareddomain.ID` (line 39)
- **Rationale**: Using shared domain ID type ensures type consistency across the system

```diff
 type WatchTarget struct {
-	ProjectPath string          // Original project path from GlobalConfig
-	ClaudeDir   string          // ~/.claude/projects/{encoded-path}
-	Type        WatchTargetType // Type of watch target
+	ProjectPath string               // Original project path from GlobalConfig
+	ClaudeDir   string               // ~/.claude/projects/{encoded-path}
+	Type        WatchTargetType      // Type of watch target
+	ProjectID   shareddomain.ID      // Project ID from local config
 }

 type LogBatch struct {
 	Records   []shareddomain.SessionRecord // Session records from shared domain
-	ProjectID string                       // Project ID (for aggregation API)
+	ProjectID shareddomain.ID              // Project ID (for aggregation API)
 }
```

#### `ConfigWatcher Service`
- **Location**: `daemon/internal/service/configwatcher/configwatcher_service.go`
- **Changes**:
  - Added `localConfigPort localconfig.LocalConfigPort` field to Service struct
  - Updated `NewService()` constructor to accept LocalConfigPort dependency
  - Modified `buildWatchTargets()` to load ProjectID from local config for each project and worktree
  - Added skip logic: projects/worktrees without local config are skipped with warning logs
  - Added `loadProjectID()` helper method
- **Key Behavior**: Each worktree reads its own `.cops/config.json` independently (no inheritance from parent)

```diff
 type Service struct {
-	logger     *slog.Logger
-	pubsub     pubsub.WriterPort[[]domain.WatchTarget]
-	configPath string
+	logger          *slog.Logger
+	pubsub          pubsub.WriterPort[[]domain.WatchTarget]
+	configPath      string
+	localConfigPort localconfig.LocalConfigPort
 }

 func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
 	var targets []domain.WatchTarget

 	for _, project := range cfg.Projects {
+		// Load ProjectID from local config - skip if not found
+		projectID, err := s.loadProjectID(project.Path)
+		if err != nil {
+			s.logger.Warn("skipping project without local config (project not registered)",
+				slog.String("path", project.Path),
+				slog.Any("error", err),
+			)
+			continue
+		}
+
 		// Add main project directory
 		targets = append(targets, domain.WatchTarget{
 			ProjectPath: project.Path,
 			ClaudeDir:   pathutil.GetClaudeProjectDir(project.Path),
 			Type:        domain.WatchTargetRoot,
+			ProjectID:   projectID,
 		})

 		// Add worktrees if git project
 		if project.IsGitProject {
 			// ... worktree logic
 			for _, wt := range worktrees[1:] {
+				worktreeProjectID, err := s.loadProjectID(wt)
+				if err != nil {
+					s.logger.Warn("skipping worktree without local config (worktree not registered)",
+						slog.String("worktree", wt),
+						slog.String("parentProject", project.Path),
+						slog.Any("error", err),
+					)
+					continue
+				}
+
 				targets = append(targets, domain.WatchTarget{
 					ProjectPath: wt,
 					ClaudeDir:   pathutil.GetClaudeProjectDir(wt),
 					Type:        domain.WatchTargetWorktree,
+					ProjectID:   worktreeProjectID,
 				})
 			}
 		}
 	}

 	return targets
 }
```

#### `LogWatcher Service`
- **Location**: `daemon/internal/service/logwatcher/log_service.go`
- **Changes**:
  - **Removed**: `daemonID string` field and `uuid` dependency (no longer needed)
  - **Removed**: Old `buffer []shareddomain.SessionRecord` field
  - **Removed**: `AddRecords()` method
  - **Added**: `claudeDirToProject map[string]shareddomain.ID` to map ClaudeDir paths to ProjectIDs
  - **Added**: `bufferByProject map[shareddomain.ID][]shareddomain.SessionRecord` to group records by project
  - **Modified**: `UpdateTargets()` now builds and maintains claudeDirToProject mapping
  - **Added**: `AddRecordsForClaudeDir()` method to add records associated with a specific ClaudeDir
  - **Modified**: `Flush()` now sends separate batches for each project
  - **Added**: `GetProjectIDForClaudeDir()` helper method
- **Key Behavior**: Records from different projects are separated and sent in individual batches with accurate ProjectID

```diff
 type Service struct {
-	logger      *slog.Logger
-	fileWatcher filesystem.FileWatchPort // Outbound: fsnotify Add/Remove
-	apiClient   api.APIClientPort        // Outbound: API transmission
-	watchedDirs map[string]bool
-	buffer      []shareddomain.SessionRecord
-	daemonID    string
-	mu          sync.Mutex
+	logger             *slog.Logger
+	fileWatcher        filesystem.FileWatchPort // Outbound: fsnotify Add/Remove
+	apiClient          api.APIClientPort        // Outbound: API transmission
+	watchedDirs        map[string]bool
+	claudeDirToProject map[string]shareddomain.ID
+	bufferByProject    map[shareddomain.ID][]shareddomain.SessionRecord
+	mu                 sync.Mutex
 }

 func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
 	s.mu.Lock()
 	defer s.mu.Unlock()

-	// Build set of new directories
+	// Build set of new directories and ProjectID mapping
 	newDirs := make(map[string]bool)
+	newMapping := make(map[string]shareddomain.ID)
 	for _, t := range targets {
 		newDirs[t.ClaudeDir] = true
+		newMapping[t.ClaudeDir] = t.ProjectID
 	}

 	// ... watch management logic ...

+	// Update ClaudeDir to ProjectID mapping
+	s.claudeDirToProject = newMapping
+
 	s.logger.Info("updated watch targets",
 		slog.Int("watching", len(s.watchedDirs)),
+		slog.Int("mappings", len(s.claudeDirToProject)),
 	)

 	return nil
 }

-func (s *Service) AddRecords(records []shareddomain.SessionRecord) {
+func (s *Service) AddRecordsForClaudeDir(claudeDir string, records []shareddomain.SessionRecord) {
 	s.mu.Lock()
 	defer s.mu.Unlock()
-	s.buffer = append(s.buffer, records...)
+
+	projectID := s.claudeDirToProject[claudeDir]
+	s.bufferByProject[projectID] = append(s.bufferByProject[projectID], records...)
 }

 func (s *Service) Flush(ctx context.Context) error {
 	s.mu.Lock()
-	if len(s.buffer) == 0 {
+	if len(s.bufferByProject) == 0 {
 		s.mu.Unlock()
 		return nil
 	}

 	// Take ownership of buffer
-	records := s.buffer
-	s.buffer = []shareddomain.SessionRecord{}
+	bufferedRecords := s.bufferByProject
+	s.bufferByProject = make(map[shareddomain.ID][]shareddomain.SessionRecord)
 	s.mu.Unlock()

-	batch := domain.LogBatch{
-		Records:   records,
-		ProjectID: "", // TODO: Implement project ID tracking from config
-	}
+	var totalCount int
+	var lastErr error

-	s.logger.Info("flushing log batch",
-		slog.Int("count", len(records)),
-	)
+	for projectID, records := range bufferedRecords {
+		if len(records) == 0 {
+			continue
+		}

-	if err := s.apiClient.SendLogs(ctx, batch); err != nil {
-		// Put records back in buffer on failure
-		s.mu.Lock()
-		s.buffer = append(records, s.buffer...)
-		s.mu.Unlock()
+		batch := domain.LogBatch{
+			Records:   records,
+			ProjectID: projectID,
+		}

-		return fmt.Errorf("failed to send logs: %w", err)
-	}
+		s.logger.Info("flushing log batch",
+			slog.Int("count", len(records)),
+			slog.String("projectID", projectID.String()),
+		)
+
+		if err := s.apiClient.SendLogs(ctx, batch); err != nil {
+			// Put records back in buffer on failure
+			s.mu.Lock()
+			s.bufferByProject[projectID] = append(records, s.bufferByProject[projectID]...)
+			s.mu.Unlock()
+
+			lastErr = fmt.Errorf("failed to send logs for project %s: %w", projectID.String(), err)
+			s.logger.Error("failed to send logs",
+				slog.String("projectID", projectID.String()),
+				slog.Any("error", err),
+			)
+			continue
+		}
+
+		totalCount += len(records)
+	}
+
+	if lastErr != nil {
+		return lastErr
+	}

 	return nil
 }
```

#### `Fsnotify Handler`
- **Location**: `daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go`
- **Changes**:
  - Added `path/filepath` import
  - Extract ClaudeDir from file path using `filepath.Dir(event.Name)`
  - Call `AddRecordsForClaudeDir(claudeDir, records)` instead of old `AddRecords(records)`
- **Rationale**: Handler receives file paths containing ClaudeDir, which is used to determine project association

```diff
+import (
+	"path/filepath"
+)

 func (h *LogFsnotifyHandler) handleFileEvent(event fsnotify.Event) {
 	// ... event filtering logic ...

 	if len(records) > 0 {
-		h.svc.AddRecords(records)
+		// Extract ClaudeDir from file path (parent directory)
+		claudeDir := filepath.Dir(event.Name)
+		h.svc.AddRecordsForClaudeDir(claudeDir, records)
 		h.filePositions[event.Name] = newOffset
 		// ... position saving logic ...
 	}
 }
```

#### `DI Container`
- **Location**: `daemon/cmd/internal/container/module_config.go`
- **Changes**:
  - Added imports for `localconfig` and `filesystem` packages
  - Added `fx.Provide` for `FilesystemLocalConfigAdapter` with `fx.As` type conversion to `LocalConfigPort`
- **Rationale**: Follows hexagonal architecture DI pattern - concrete adapter is provided and converted to port interface

```diff
 import (
+	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
+	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/filesystem"
 )

 func newConfigModule() fx.Option {
 	return fx.Module("config",
 		// ... existing providers ...

+		// Outbound: LocalConfigPort
+		fx.Provide(fx.Annotate(
+			filesystem.NewFilesystemLocalConfigAdapter,
+			fx.As(new(localconfig.LocalConfigPort)),
+		)),

 		// Service (pure business logic)
 		fx.Provide(configwatcher.NewService),
 		// ...
 	)
 }
```

## Data Flow

### Before (Without ProjectID Tracking)

```
Claude Code → JSONL logs → Daemon watches → Parses records → Buffer (flat) → API (ProjectID empty)
```

### After (With ProjectID Tracking)

```
1. ConfigWatcher:
   GlobalConfig → Read local config (.cops/config.json) → Extract ProjectID → WatchTarget with ProjectID → Publish

2. LogWatcher:
   Receive WatchTargets → Build claudeDirToProject mapping → Watch directories

3. Log Processing:
   JSONL file change → Extract ClaudeDir from path → Look up ProjectID → Add to bufferByProject[ProjectID]

4. Flush:
   For each ProjectID in buffer → Create LogBatch with ProjectID → Send to API → Clear buffer
```

## New Feature Integration

### 1. Local Config Reading (Port/Adapter Pattern)

**Port (Interface)**:
- `LocalConfigPort` defines the contract for loading local configs
- Located in `outbound/localconfig/` (ConfigWatcher's outbound dependency)

**Adapter (Implementation)**:
- `FilesystemLocalConfigAdapter` implements the port using filesystem operations
- Uses `sonic` for JSON parsing (consistent with existing daemon code)
- Handles file not found and JSON parsing errors

**DI Integration**:
- Adapter is provided via `fx.Provide` with `fx.As` type conversion
- ConfigWatcher receives the port interface as a dependency
- Follows hexagonal architecture pattern

### 2. Skip Logic for Unregistered Projects

**Behavior**:
- ConfigWatcher calls `loadProjectID()` for each project and worktree
- If local config file does not exist or cannot be read, project/worktree is skipped
- Warning logs are generated with clear context (path, error details)

**Rationale**:
- Only registered projects should be watched
- Projects without local config are not registered with the API
- Watching them would result in orphaned data (no valid ProjectID)

**Example Log Output**:
```
WARN skipping project without local config (project not registered) path=/home/user/unregistered-project error=open /home/user/unregistered-project/.cops/config.json: no such file or directory
WARN skipping worktree without local config (worktree not registered) worktree=/home/user/project/worktrees/feature-branch parentProject=/home/user/project error=...
```

### 3. Worktree Independence

**Key Decision**: Each worktree reads its own `.cops/config.json` file

**Implementation**:
```go
for _, wt := range worktrees[1:] {
    worktreeProjectID, err := s.loadProjectID(wt)  // Read worktree's own config
    if err != nil {
        // Skip if worktree not registered
        continue
    }

    targets = append(targets, domain.WatchTarget{
        ProjectPath: wt,
        ClaudeDir:   pathutil.GetClaudeProjectDir(wt),
        Type:        domain.WatchTargetWorktree,
        ProjectID:   worktreeProjectID,  // Use worktree's own ID
    })
}
```

**Rationale**:
- Worktrees are separate working directories
- May be associated with different project registrations
- Each worktree should have its own `.cops/config.json` if it needs to be watched
- No inheritance from parent project

### 4. Buffer Grouping by ProjectID

**Before**:
```go
buffer []shareddomain.SessionRecord  // Flat list, all projects mixed
```

**After**:
```go
bufferByProject map[shareddomain.ID][]shareddomain.SessionRecord  // Grouped by project
```

**Benefits**:
- Enables sending separate LogBatch per project with accurate ProjectID
- Records from different projects are naturally separated
- Simplifies batch creation in Flush()

### 5. ClaudeDir to ProjectID Mapping

**Purpose**: Enable fsnotify handler to determine which project a file change belongs to

**Implementation**:
```go
claudeDirToProject map[string]shareddomain.ID  // Maps ClaudeDir to ProjectID

// Updated when targets change
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
    newMapping := make(map[string]shareddomain.ID)
    for _, t := range targets {
        newMapping[t.ClaudeDir] = t.ProjectID
    }
    s.claudeDirToProject = newMapping
}

// Used when adding records
func (s *Service) AddRecordsForClaudeDir(claudeDir string, records []shareddomain.SessionRecord) {
    projectID := s.claudeDirToProject[claudeDir]
    s.bufferByProject[projectID] = append(s.bufferByProject[projectID], records...)
}
```

**Rationale**:
- fsnotify handler receives file paths containing ClaudeDir
- This mapping enables determining project association without re-reading config files
- Efficient lookup during high-frequency file change events

### 6. Multiple Batches per Flush

**Old Behavior**:
```go
// Single batch for all records
batch := domain.LogBatch{
    Records:   records,
    ProjectID: "",  // Empty, not tracked
}
apiClient.SendLogs(ctx, batch)
```

**New Behavior**:
```go
// Separate batch for each project
for projectID, records := range bufferedRecords {
    batch := domain.LogBatch{
        Records:   records,
        ProjectID: projectID,
    }

    if err := apiClient.SendLogs(ctx, batch); err != nil {
        // Put records back on failure
        bufferByProject[projectID] = append(records, bufferByProject[projectID]...)
        lastErr = err
        continue  // Continue processing other projects
    }
}
```

**Error Recovery**:
- Failed batches are put back in buffer (per-project)
- Processing continues for other projects even if one fails
- Last error is returned after attempting all batches

## Testing

- **Build Verification**:
  ```bash
  go build ./daemon/...  # Result: PASS
  ```

- **Full Build Verification**:
  ```bash
  go mod tidy && go build ./...  # Result: PASS
  ```

## Issues & Resolutions

| Issue | Resolution |
|-------|-----------|
| Type inconsistency for ProjectID | Changed `LogBatch.ProjectID` from `string` to `shareddomain.ID` to match `WatchTarget.ProjectID` and ensure type consistency across the system |
| daemonID field no longer needed | Removed `daemonID string` field and `uuid` dependency from LogWatcher as ProjectID now serves as the primary identifier |
| Need to determine project for file changes | Added `claudeDirToProject` mapping in LogWatcher, populated via `UpdateTargets()`, enabling efficient project lookup during fsnotify events |
| Worktree config inheritance unclear | Explicitly documented that each worktree reads its own `.cops/config.json` independently (no inheritance from parent) |
| Unregistered projects would send empty ProjectID | Added skip logic in ConfigWatcher to prevent watching projects/worktrees without local config, with clear warning logs |
| Single batch for all projects not accurate | Changed buffer to `map[shareddomain.ID][]shareddomain.SessionRecord` to group by project and send separate batches in Flush() |

## Related Tickets

This implementation follows hexagonal architecture principles and maintains consistency with existing daemon patterns:
- Port/Adapter pattern for LocalConfigPort
- Dependency injection via `fx` container
- Structured logging with context binding
- Error handling without panics
- Mutex-protected concurrent access
- Compile-time interface verification
