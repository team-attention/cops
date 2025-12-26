# Research Report

## Mode
General Research

## Request Summary
Implement project ID tracking in the daemon so that when sending logs to the API, the `LogBatch.ProjectID` field is populated correctly. Currently, the daemon reads the global config (`~/.cops/config.json`) to get project paths, but needs to also read local config files (`{projectPath}/.cops/config.json`) to get project IDs and associate them with log records.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` | Core file to modify - contains Flush() where LogBatch.ProjectID is set (line 169) |
| `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` | Contains WatchTarget struct that needs ProjectID field added |
| `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` | Loads global config and builds WatchTargets - needs to also load local configs |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go` | Reference implementation for LoadLocalConfig() |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go` | LocalConfig struct definition with ID field |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Rules for service layer implementation |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md` | Rules for outbound adapter pattern |

## Current Architecture Analysis

### Data Flow Overview
```
~/.cops/config.json (GlobalConfig)
         |
         v
ConfigWatcher Service --> buildWatchTargets() --> []WatchTarget (ProjectPath, ClaudeDir, Type)
         |
         v (pubsub)
LogWatcher Service --> UpdateTargets() --> watches ClaudeDir for JSONL files
         |
         v (fsnotify events)
HandleFileChange() --> AddRecords() --> buffer ([]SessionRecord)
         |
         v (flush ticker)
Flush() --> LogBatch{Records, ProjectID: ""} --> API
```

### Key Components

#### 1. ConfigWatcher Service (`configwatcher_service.go`)
- **Purpose**: Watches global config file for changes
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go`
- **Key Method**: `buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget`
- **Current Behavior**:
  - Loads global config with project paths
  - For each project, creates WatchTarget with ProjectPath and ClaudeDir
  - Adds worktrees for git projects
  - **Does NOT** read local config files to get ProjectID

#### 2. WatchTarget Domain Model (`watch.go`)
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`
- **Current Fields**:
  ```go
  type WatchTarget struct {
      ProjectPath string          // Original project path from GlobalConfig
      ClaudeDir   string          // ~/.claude/projects/{encoded-path}
      Type        WatchTargetType // Type of watch target
  }
  ```
- **Missing**: `ProjectID` field to store the ID from local config

#### 3. LogWatcher Service (`log_service.go`)
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`
- **Key Issue**: Line 167-170
  ```go
  batch := domain.LogBatch{
      Records:   records,
      ProjectID: "", // TODO: Implement project ID tracking from config
  }
  ```
- **Problem**: No way to associate records with their project ID
- **Buffer**: Stores `[]shareddomain.SessionRecord` without project association

#### 4. LogBatch Domain Model (`watch.go`)
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`
- **Current Fields**:
  ```go
  type LogBatch struct {
      Records   []shareddomain.SessionRecord
      ProjectID string
  }
  ```

### CLI Reference Implementation

The CLI already has config adapter that reads local configs:

**Port Definition** (`config_port.go`):
```go
type LocalConfig struct {
    ID domain.ID `json:"id"`
}

type ConfigPort interface {
    LoadLocalConfig(projectPath string) (*LocalConfig, error)
    LocalConfigExists(projectPath string) bool
    // ...
}
```

**Implementation** (`filesystem_config.go`):
```go
func (a *FilesystemConfigAdapter) LoadLocalConfig(projectPath string) (*config.LocalConfig, error) {
    configPath := filepath.Join(projectPath, localConfigDirName, localConfigFileName)
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    var cfg config.LocalConfig
    if err := sonic.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

## Package Candidates

### Problem 1: Local Config File Reading

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| sonic | `/bytedance/sonic` | Already used in daemon for JSON parsing (configwatcher_service.go:8), consistent with existing code |

**Decision**: Use `sonic` (already a dependency) - no new package needed.

## Technical Constraints

1. **Worktrees share ProjectID**: Worktrees of a project should use the same ProjectID as the main project (local config is at root, not in worktree)
2. **Missing config is valid**: Project may not have a local config yet (API registration in progress) - should log warning and continue with empty ProjectID
3. **Buffer doesn't track source**: Current buffer stores records without knowing which project they came from - need architectural change
4. **Single ProjectID per batch**: Current LogBatch assumes all records have same ProjectID - may need to send multiple batches

## Similar Implementations Found

### Example 1: CLI Config Adapter Pattern
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go:94-109`
- **Relevance**: Shows exact implementation of LoadLocalConfig() that daemon should follow

### Example 2: ConfigWatcher buildWatchTargets
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go:84-120`
- **Relevance**: Shows where ProjectID loading should be integrated

### Example 3: Existing pathutil in daemon
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/platform/util/pathutil/pathutil.go`
- **Relevance**: Contains path encoding utilities already used in daemon

## Recommended Implementation Approach

### Option A: Add ProjectID to WatchTarget (Recommended)

**Pros:**
- ProjectID flows through existing pubsub mechanism
- LogWatcher can build ClaudeDir to ProjectID mapping at UpdateTargets()
- Minimal architectural change

**Changes Required:**

1. **Add field to WatchTarget** (`daemon/internal/platform/domain/watch.go`):
   ```go
   type WatchTarget struct {
       ProjectPath string
       ClaudeDir   string
       Type        WatchTargetType
       ProjectID   string  // NEW: from {projectPath}/.cops/config.json
   }
   ```

2. **Create outbound adapter for local config** (new files):
   - `daemon/internal/service/configwatcher/outbound/config/config_port.go` - interface
   - `daemon/internal/service/configwatcher/outbound/config/filesystem/filesystem_config.go` - implementation

3. **Update ConfigWatcher Service** to load local configs in buildWatchTargets()

4. **Update LogWatcher Service**:
   - Add `claudeDirToProjectID map[string]string` field
   - Update `UpdateTargets()` to build mapping
   - Update `Flush()` to use mapping based on record's source directory

5. **Track source directory per record** - Need to modify buffer to track which ClaudeDir each record came from

### Option B: Read config on every flush (Simpler but less efficient)

**Pros:**
- Less code change
- Always fresh ProjectID

**Cons:**
- File I/O on every flush
- Needs reverse mapping from ClaudeDir to ProjectPath

### Decision: Go with Option A

Option A follows the existing architecture pattern and avoids repeated file I/O.

## Implementation Details

### Buffer Architecture Change

Current buffer issue: Records are stored without source information.

**Proposed Change** - Instead of:
```go
buffer []shareddomain.SessionRecord
```

Use:
```go
type BufferedRecord struct {
    Record    shareddomain.SessionRecord
    ClaudeDir string  // Source directory
}
buffer []BufferedRecord
```

Or group by project:
```go
bufferByProject map[string][]shareddomain.SessionRecord  // key: ProjectID
```

### Flush Strategy

With records grouped by project, Flush() can:
1. Iterate over each project's records
2. Send separate LogBatch per project (with correct ProjectID)
3. Or send single batch with first project's ID (simpler, but loses granularity)

**Recommendation**: Group by ProjectID and send multiple batches - provides accurate per-project tracking.

## Additional Information for Planning

### File Position Tracking
The daemon already tracks file positions in SQLite (`stateRepo`). The file path in the position could be used to derive the ClaudeDir and thus ProjectID, but this would require reverse-engineering the path structure.

### Error Handling Strategy
- **Config not found**: Log warning, use empty ProjectID (project not yet registered with API)
- **Config parse error**: Log warning, use empty ProjectID
- **Multiple projects in batch**: Send separate batches per project

### Worktree Handling
Worktrees don't have their own `.cops/config.json`. They should inherit the ProjectID from the main project. The `buildWatchTargets()` function already tracks the parent project path for worktrees.

## Code Snippets from Key Files

### Current WatchTarget Definition
```go
// daemon/internal/platform/domain/watch.go:22-27
type WatchTarget struct {
    ProjectPath string          // Original project path from GlobalConfig
    ClaudeDir   string          // ~/.claude/projects/{encoded-path}
    Type        WatchTargetType // Type of watch target
}
```

### Current LogBatch Definition
```go
// daemon/internal/platform/domain/watch.go:35-39
type LogBatch struct {
    Records   []shareddomain.SessionRecord // Session records from shared domain
    ProjectID string                       // Project ID (for aggregation API)
}
```

### Current Flush Implementation
```go
// daemon/internal/service/logwatcher/log_service.go:154-186
func (s *Service) Flush(ctx context.Context) error {
    s.mu.Lock()
    if len(s.buffer) == 0 {
        s.mu.Unlock()
        return nil
    }

    // Take ownership of buffer
    records := s.buffer
    s.buffer = []shareddomain.SessionRecord{}
    s.mu.Unlock()

    batch := domain.LogBatch{
        Records:   records,
        ProjectID: "", // TODO: Implement project ID tracking from config
    }

    s.logger.Info("flushing log batch",
        slog.Int("count", len(records)),
    )

    if err := s.apiClient.SendLogs(ctx, batch); err != nil {
        // Put records back in buffer on failure
        s.mu.Lock()
        s.buffer = append(records, s.buffer...)
        s.mu.Unlock()

        return fmt.Errorf("failed to send logs: %w", err)
    }

    return nil
}
```

### Current buildWatchTargets Implementation
```go
// daemon/internal/service/configwatcher/configwatcher_service.go:84-120
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
    var targets []domain.WatchTarget

    for _, project := range cfg.Projects {
        // Add main project directory
        targets = append(targets, domain.WatchTarget{
            ProjectPath: project.Path,
            ClaudeDir:   pathutil.GetClaudeProjectDir(project.Path),
            Type:        domain.WatchTargetRoot,
        })

        // Add worktrees if git project
        if project.IsGitProject {
            worktrees, err := gitutil.GetWorktrees(project.Path)
            if err != nil {
                s.logger.Warn("failed to get worktrees",
                    slog.String("path", project.Path),
                    slog.Any("error", err),
                )
                continue
            }

            // Skip first element (main repo) as it's already added
            for _, wt := range worktrees[1:] {
                targets = append(targets, domain.WatchTarget{
                    ProjectPath: wt,
                    ClaudeDir:   pathutil.GetClaudeProjectDir(wt),
                    Type:        domain.WatchTargetWorktree,
                })
            }
        }
    }

    return targets
}
```
