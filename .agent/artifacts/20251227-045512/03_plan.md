# Implementation Plan

## Overview

Add project ID tracking to the daemon by reading local config files (`{projectPath}/.cops/config.json`) and populating `LogBatch.ProjectID` when sending logs to the API. Records will be grouped by ProjectID in the buffer to send accurate per-project batches. Projects and worktrees without local config will be skipped from watching.

## Selected Packages

| Problem          | Package | Context7 ID        | Reason for Selection                                      |
| ---------------- | ------- | ------------------ | --------------------------------------------------------- |
| JSON Parsing     | sonic   | `/bytedance/sonic` | Already used in daemon for JSON parsing (no new packages) |
| Path Operations  | stdlib  | N/A                | Using `path/filepath` and `os` from standard library      |

## Architecture Decisions

### Decision 1: Add ProjectID to WatchTarget Using domain.ID

**Choice**: Add `ProjectID domain.ID` field to the existing `WatchTarget` struct in daemon domain, using the shared domain ID type.

**Rationale**: Using `domain.ID` from the shared module ensures type consistency across the system. ProjectID naturally flows through the existing pubsub mechanism from ConfigWatcher to LogWatcher.

### Decision 2: Create LocalConfig Port/Adapter for ConfigWatcher

**Choice**: Create a dedicated `localconfig` outbound port and filesystem adapter in the configwatcher service to read local project configs. LocalConfig struct uses `ID domain.ID` field.

**Rationale**: Follows hexagonal architecture pattern. ConfigWatcher is responsible for loading configs, so the local config reading logic belongs there rather than in LogWatcher.

### Decision 3: Group Buffer by ProjectID

**Choice**: Change the buffer from `[]SessionRecord` to `map[domain.ID][]SessionRecord` where the key is ProjectID.

**Rationale**: Enables sending separate LogBatch per project with accurate ProjectID. Records from different projects are naturally separated.

### Decision 4: Track ClaudeDir to ProjectID Mapping in LogWatcher

**Choice**: Maintain a `claudeDirToProject map[string]domain.ID` in LogWatcher, updated via `UpdateTargets()`.

**Rationale**: The fsnotify handler receives file paths containing ClaudeDir. This mapping enables determining which project a file change belongs to without needing to re-read config files.

### Decision 5: Each Worktree Reads Its Own Local Config

**Choice**: Each worktree reads its own `{worktreePath}/.cops/config.json` file independently. Worktrees do NOT inherit ProjectID from parent project.

**Rationale**: Worktrees are separate working directories and may be associated with different project registrations. Each worktree should have its own `.cops/config.json` if it needs to be watched.

### Decision 6: Skip Targets Without Local Config

**Choice**: When building WatchTargets, if LoadLocalConfig fails (file not found), skip adding that target entirely and log a warning.

**Rationale**: Only registered projects should be watched. Projects without local config are not registered with the API, so watching them would result in orphaned data. The warning helps users understand why a project is not being watched.

## Implementation Steps

### Step 1: Add ProjectID Field to WatchTarget

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` (modify)

**Functions**:

```go
import (
    shareddomain "github.com/team-attention/cops/shared/domain"
)

// WatchTarget represents a directory to watch for Claude Code logs.
type WatchTarget struct {
    ProjectPath string                // Original project path from GlobalConfig
    ClaudeDir   string                // ~/.claude/projects/{encoded-path}
    Type        WatchTargetType       // Type of watch target
    ProjectID   shareddomain.ID       // Project ID from local config
}
```

**Test Scenarios**:
| Scenario              | Input                                  | Expected Output                         | Branch Covered       |
| --------------------- | -------------------------------------- | --------------------------------------- | -------------------- |
| Root project target   | WatchTarget with ProjectID set         | Struct contains ProjectID               | Normal instantiation |
| Worktree target       | WatchTarget with its own ProjectID     | Struct contains worktree's ProjectID    | Worktree case        |

---

### Step 2: Create LocalConfig Port for ConfigWatcher

**Files to Create**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go` (create)

**Functions**:

```go
package localconfig

import (
    "github.com/team-attention/cops/shared/domain"
)

// LocalConfig represents the {projectPath}/.cops/config.json structure.
type LocalConfig struct {
    ID domain.ID `json:"id"`
}

// LocalConfigPort defines the interface for reading local project configs.
type LocalConfigPort interface {
    // LoadLocalConfig loads the local configuration from {projectPath}/.cops/config.json.
    // Returns nil, error if file does not exist or cannot be parsed.
    LoadLocalConfig(projectPath string) (*LocalConfig, error)
}
```

**Test Scenarios**:
| Scenario            | Input             | Expected Output        | Branch Covered           |
| ------------------- | ----------------- | ---------------------- | ------------------------ |
| Interface contract  | N/A               | Compiles successfully  | Interface definition     |

---

### Step 3: Create Filesystem Adapter for LocalConfig

**Files to Create**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/filesystem/filesystem_localconfig.go` (create)

**Functions**:

```go
package filesystem

import (
    "log/slog"
    "os"
    "path/filepath"

    "github.com/bytedance/sonic"
    "github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
)

const (
    localConfigDirName  = ".cops"
    localConfigFileName = "config.json"
)

// FilesystemLocalConfigAdapter implements LocalConfigPort using filesystem.
type FilesystemLocalConfigAdapter struct {
    logger *slog.Logger
}

// NewFilesystemLocalConfigAdapter creates a new filesystem local config adapter.
func NewFilesystemLocalConfigAdapter(l *slog.Logger) *FilesystemLocalConfigAdapter {
    return &FilesystemLocalConfigAdapter{
        logger: l.With(slog.String("name", "configwatcher.localconfig.filesystem")),
    }
}

// LoadLocalConfig loads the local configuration from {projectPath}/.cops/config.json.
func (a *FilesystemLocalConfigAdapter) LoadLocalConfig(projectPath string) (*localconfig.LocalConfig, error) {
    configPath := filepath.Join(projectPath, localConfigDirName, localConfigFileName)

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }

    var cfg localconfig.LocalConfig
    if err := sonic.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}

// Compile-time interface verification
var _ localconfig.LocalConfigPort = (*FilesystemLocalConfigAdapter)(nil)
```

**Test Scenarios**:
| Scenario               | Input                              | Expected Output                   | Branch Covered          |
| ---------------------- | ---------------------------------- | --------------------------------- | ----------------------- |
| Valid config file      | Project with `.cops/config.json`   | LocalConfig with ID populated     | Happy path              |
| Missing config file    | Project without `.cops/` directory | Error (os.ErrNotExist)            | File not found branch   |
| Malformed JSON         | Invalid JSON in config file        | Error (sonic unmarshal error)     | JSON parse error branch |
| Missing ID field       | `{}` in config file                | LocalConfig with empty ID         | Empty ID case           |

---

### Step 4: Update ConfigWatcher Service to Load LocalConfigs and Skip Unregistered

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` (modify)

**Functions**:

```go
import (
    shareddomain "github.com/team-attention/cops/shared/domain"
    "github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
)

// Service contains pure business logic for config watching.
type Service struct {
    logger          *slog.Logger
    pubsub          pubsub.WriterPort[[]domain.WatchTarget]
    configPath      string
    localConfigPort localconfig.LocalConfigPort // NEW: for loading local project configs
}

// NewService creates a new ConfigWatcher service.
func NewService(
    l *slog.Logger,
    ps pubsub.WriterPort[[]domain.WatchTarget],
    cfg *setup.Config,
    localConfigPort localconfig.LocalConfigPort, // NEW parameter
) *Service {
    return &Service{
        logger:          l.With(slog.String("name", "configwatcher.service")),
        pubsub:          ps,
        configPath:      cfg.Cops.GlobalConfigPath,
        localConfigPort: localConfigPort,
    }
}

// buildWatchTargets builds watch targets from global config.
// This includes main project directories and git worktrees.
// Projects and worktrees without local config are skipped.
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
    var targets []domain.WatchTarget

    for _, project := range cfg.Projects {
        // Load ProjectID from local config - skip if not found
        projectID, err := s.loadProjectID(project.Path)
        if err != nil {
            s.logger.Warn("skipping project without local config (project not registered)",
                slog.String("path", project.Path),
                slog.Any("error", err),
            )
            continue
        }

        // Add main project directory
        targets = append(targets, domain.WatchTarget{
            ProjectPath: project.Path,
            ClaudeDir:   pathutil.GetClaudeProjectDir(project.Path),
            Type:        domain.WatchTargetRoot,
            ProjectID:   projectID,
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
            // Each worktree reads its own local config
            for _, wt := range worktrees[1:] {
                worktreeProjectID, err := s.loadProjectID(wt)
                if err != nil {
                    s.logger.Warn("skipping worktree without local config (worktree not registered)",
                        slog.String("worktree", wt),
                        slog.String("parentProject", project.Path),
                        slog.Any("error", err),
                    )
                    continue
                }

                targets = append(targets, domain.WatchTarget{
                    ProjectPath: wt,
                    ClaudeDir:   pathutil.GetClaudeProjectDir(wt),
                    Type:        domain.WatchTargetWorktree,
                    ProjectID:   worktreeProjectID, // Use worktree's own ID
                })
            }
        }
    }

    return targets
}

// loadProjectID loads the ProjectID from the local config file.
// Returns error if config file is not found or cannot be read.
func (s *Service) loadProjectID(projectPath string) (shareddomain.ID, error) {
    localCfg, err := s.localConfigPort.LoadLocalConfig(projectPath)
    if err != nil {
        return "", err
    }
    return localCfg.ID, nil
}
```

**Test Scenarios**:
| Scenario                              | Input                                      | Expected Output                                    | Branch Covered                   |
| ------------------------------------- | ------------------------------------------ | -------------------------------------------------- | -------------------------------- |
| Project with local config             | GlobalConfig with registered project       | WatchTarget with ProjectID populated               | Happy path                       |
| Project without local config          | GlobalConfig with unregistered project     | Project skipped, warning logged                    | Missing config - skip branch     |
| Project with registered worktrees     | Git project with 2 registered worktrees    | 3 targets, each with own ProjectID                 | Worktree with own config         |
| Project with unregistered worktrees   | Git project, worktrees have no config      | 1 target (root only), worktrees skipped            | Worktree missing config - skip   |
| Mixed: some worktrees registered      | 2 worktrees, 1 registered                  | 2 targets (root + 1 worktree)                      | Partial worktree registration    |
| All projects unregistered             | GlobalConfig with no registered projects   | Empty targets list                                 | All skipped case                 |

---

### Step 5: Update LogWatcher Service with ProjectID Tracking (Remove daemonID)

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` (modify)

**Functions**:

```go
import (
    shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service contains pure business logic for log file watching and processing.
type Service struct {
    logger             *slog.Logger
    fileWatcher        filesystem.FileWatchPort
    apiClient          api.APIClientPort
    watchedDirs        map[string]bool
    claudeDirToProject map[string]shareddomain.ID                   // NEW: maps ClaudeDir to ProjectID
    bufferByProject    map[shareddomain.ID][]shareddomain.SessionRecord // NEW: records grouped by ProjectID
    mu                 sync.Mutex
    // NOTE: daemonID field removed
}

// NewService creates a new Log service.
func NewService(
    l *slog.Logger,
    fileWatcher filesystem.FileWatchPort,
    apiClient api.APIClientPort,
    cfg *setup.Config,
) *Service {
    return &Service{
        logger:             l.With(slog.String("name", "log.service")),
        fileWatcher:        fileWatcher,
        apiClient:          apiClient,
        watchedDirs:        make(map[string]bool),
        claudeDirToProject: make(map[string]shareddomain.ID),
        bufferByProject:    make(map[shareddomain.ID][]shareddomain.SessionRecord),
        // NOTE: daemonID initialization removed
    }
}

// UpdateTargets updates the directories to watch.
// Called by Inbound (pubsub) when target configuration changes.
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Build set of new directories and ProjectID mapping
    newDirs := make(map[string]bool)
    newMapping := make(map[string]shareddomain.ID)
    for _, t := range targets {
        newDirs[t.ClaudeDir] = true
        newMapping[t.ClaudeDir] = t.ProjectID
    }

    // Remove watches for directories no longer in targets
    for dir := range s.watchedDirs {
        if !newDirs[dir] {
            if err := s.fileWatcher.Remove(dir); err != nil {
                s.logger.Warn("failed to remove watch",
                    slog.String("dir", dir),
                    slog.Any("error", err),
                )
            }
            delete(s.watchedDirs, dir)
        }
    }

    // Add watches for new directories
    for dir := range newDirs {
        if !s.watchedDirs[dir] {
            if err := s.fileWatcher.Add(dir); err != nil {
                s.logger.Debug("failed to add watch (directory may not exist)",
                    slog.String("dir", dir),
                    slog.Any("error", err),
                )
                continue
            }
            s.watchedDirs[dir] = true
        }
    }

    // Update ClaudeDir to ProjectID mapping
    s.claudeDirToProject = newMapping

    s.logger.Info("updated watch targets",
        slog.Int("watching", len(s.watchedDirs)),
        slog.Int("mappings", len(s.claudeDirToProject)),
    )

    return nil
}

// AddRecordsForClaudeDir adds records to the buffer, associating them with the given ClaudeDir.
func (s *Service) AddRecordsForClaudeDir(claudeDir string, records []shareddomain.SessionRecord) {
    s.mu.Lock()
    defer s.mu.Unlock()

    projectID := s.claudeDirToProject[claudeDir]
    s.bufferByProject[projectID] = append(s.bufferByProject[projectID], records...)
}

// Flush sends buffered records to the API server.
// Sends separate batches for each project.
func (s *Service) Flush(ctx context.Context) error {
    s.mu.Lock()
    if len(s.bufferByProject) == 0 {
        s.mu.Unlock()
        return nil
    }

    // Take ownership of buffer
    bufferedRecords := s.bufferByProject
    s.bufferByProject = make(map[shareddomain.ID][]shareddomain.SessionRecord)
    s.mu.Unlock()

    var totalCount int
    var lastErr error

    for projectID, records := range bufferedRecords {
        if len(records) == 0 {
            continue
        }

        batch := domain.LogBatch{
            Records:   records,
            ProjectID: projectID,
        }

        s.logger.Info("flushing log batch",
            slog.Int("count", len(records)),
            slog.String("projectID", projectID.String()),
        )

        if err := s.apiClient.SendLogs(ctx, batch); err != nil {
            // Put records back in buffer on failure
            s.mu.Lock()
            s.bufferByProject[projectID] = append(records, s.bufferByProject[projectID]...)
            s.mu.Unlock()

            lastErr = fmt.Errorf("failed to send logs for project %s: %w", projectID.String(), err)
            s.logger.Error("failed to send logs",
                slog.String("projectID", projectID.String()),
                slog.Any("error", err),
            )
            continue
        }

        totalCount += len(records)
    }

    if lastErr != nil {
        return lastErr
    }

    return nil
}

// GetProjectIDForClaudeDir returns the ProjectID for a given ClaudeDir.
// Returns empty ID if not found.
func (s *Service) GetProjectIDForClaudeDir(claudeDir string) shareddomain.ID {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.claudeDirToProject[claudeDir]
}
```

**Test Scenarios**:
| Scenario                         | Input                                      | Expected Output                               | Branch Covered                 |
| -------------------------------- | ------------------------------------------ | --------------------------------------------- | ------------------------------ |
| UpdateTargets with ProjectIDs    | Targets with ProjectID values              | claudeDirToProject mapping populated          | Happy path                     |
| AddRecordsForClaudeDir           | ClaudeDir with records                     | Records added to correct project bucket       | Buffer grouping                |
| Flush with single project        | Buffer with one project's records          | Single batch sent with ProjectID              | Single project flush           |
| Flush with multiple projects     | Buffer with two projects' records          | Two separate batches sent                     | Multi-project flush            |
| Flush with empty buffer          | No records in buffer                       | Returns nil immediately                       | Empty buffer early return      |
| Flush with API failure           | API returns error                          | Records put back, error returned              | API failure recovery           |
| AddRecords for unknown ClaudeDir | ClaudeDir not in mapping                   | Records added with empty ProjectID            | Unknown ClaudeDir case         |

---

### Step 6: Update LogBatch Domain to Use domain.ID

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/log_batch.go` or equivalent (modify)

**Functions**:

```go
import (
    shareddomain "github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of log records to send to the API.
type LogBatch struct {
    Records   []shareddomain.SessionRecord
    ProjectID shareddomain.ID
}
```

**Test Scenarios**:
| Scenario              | Input                    | Expected Output                    | Branch Covered       |
| --------------------- | ------------------------ | ---------------------------------- | -------------------- |
| LogBatch with ID      | LogBatch with ProjectID  | Struct contains domain.ID type     | Type verification    |

---

### Step 7: Update Fsnotify Handler to Pass ClaudeDir

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go` (modify)

**Functions**:

```go
func (h *LogFsnotifyHandler) handleFileEvent(event fsnotify.Event) {
    // Only process JSONL files
    if !strings.HasSuffix(event.Name, ".jsonl") {
        return
    }

    // Only process write and create events
    if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
        return
    }

    offset := h.filePositions[event.Name]
    records, newOffset, err := h.svc.HandleFileChange(event.Name, offset)
    if err != nil {
        h.logger.Debug("failed to handle file change",
            slog.String("path", event.Name),
            slog.Any("error", err),
        )
        return
    }

    if len(records) > 0 {
        // Extract ClaudeDir from file path (parent directory)
        claudeDir := filepath.Dir(event.Name)
        h.svc.AddRecordsForClaudeDir(claudeDir, records)
        h.filePositions[event.Name] = newOffset

        // Save position to SQLite
        if err := h.stateRepo.SaveFilePosition(h.ctx, &domain.FilePosition{
            Path:   event.Name,
            Offset: newOffset,
        }); err != nil {
            h.logger.Warn("failed to save file position",
                slog.String("path", event.Name),
                slog.Any("error", err),
            )
        }
    }
}
```

**Import to Add**:
```go
import (
    "path/filepath"
    // ... existing imports
)
```

**Test Scenarios**:
| Scenario                       | Input                                       | Expected Output                           | Branch Covered            |
| ------------------------------ | ------------------------------------------- | ----------------------------------------- | ------------------------- |
| JSONL file change              | Write event for `.jsonl` file               | AddRecordsForClaudeDir called with dir    | Happy path                |
| Non-JSONL file                 | Write event for `.txt` file                 | Function returns early, no records added  | File extension filter     |
| Directory extraction           | `/home/user/.claude/projects/-foo/log.jsonl`| ClaudeDir = `/home/user/.claude/projects/-foo` | Path parsing             |

---

### Step 8: Remove Deprecated AddRecords Method

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` (modify)

**Change**:
Remove the old `AddRecords` method entirely as it's replaced by `AddRecordsForClaudeDir`.

```go
// DELETE this method:
// func (s *Service) AddRecords(records []shareddomain.SessionRecord) {
//     s.mu.Lock()
//     defer s.mu.Unlock()
//     s.buffer = append(s.buffer, records...)
// }
```

Also remove any unused imports (e.g., `github.com/google/uuid` if no longer needed after removing daemonID).

**Test Scenarios**:
| Scenario                    | Input | Expected Output                    | Branch Covered    |
| --------------------------- | ----- | ---------------------------------- | ----------------- |
| Method removal verification | N/A   | Code compiles without AddRecords   | Cleanup           |
| Unused import removal       | N/A   | uuid import removed if unused      | Import cleanup    |

---

### Step 9: Update DI Container for ConfigWatcher

**Files to Modify**:
- Find the fx module file that registers ConfigWatcher and add the LocalConfigPort dependency

**Locate file**: Need to search for ConfigWatcher DI registration

```go
// In the container module for configwatcher:

// Add import
import (
    "github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/filesystem"
)

// Add to fx.Provide:
fx.Provide(filesystem.NewFilesystemLocalConfigAdapter)
```

**Test Scenarios**:
| Scenario              | Input                 | Expected Output                           | Branch Covered   |
| --------------------- | --------------------- | ----------------------------------------- | ---------------- |
| DI wiring             | Application start     | ConfigWatcher gets LocalConfigPort        | DI setup         |

## Execution Order

1. **Step 1**: Add ProjectID field to WatchTarget (no dependencies)
2. **Step 2**: Create LocalConfig port interface (no dependencies)
3. **Step 3**: Create Filesystem adapter for LocalConfig (depends on Step 2)
4. **Step 4**: Update ConfigWatcher Service (depends on Steps 1, 2, 3)
5. **Step 5**: Update LogWatcher Service (depends on Step 1)
6. **Step 6**: Update LogBatch Domain (depends on Step 5)
7. **Step 7**: Update Fsnotify Handler (depends on Step 5)
8. **Step 8**: Remove deprecated AddRecords method (depends on Steps 5, 7)
9. **Step 9**: Update DI Container (depends on Steps 3, 4)

## Notes for Execute Agent

1. **Import paths**: Use the exact import paths as shown:
   - `github.com/team-attention/cops/shared/domain` (import as `shareddomain`)
   - `github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig`
   - `github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/filesystem`

2. **Directory creation**: Steps 2 and 3 require creating new directories:
   - `daemon/internal/service/configwatcher/outbound/localconfig/`
   - `daemon/internal/service/configwatcher/outbound/localconfig/filesystem/`

3. **DI Container location**: The container registration for ConfigWatcher needs to be found. Look in:
   - `daemon/cmd/internal/container/` or similar
   - Search for `configwatcher.NewService`

4. **Remove old buffer field**: In Step 5, remove the old `buffer []shareddomain.SessionRecord` field entirely from the Service struct.

5. **Remove daemonID**: In Step 5, remove the `daemonID string` field and its `uuid.New().String()` initialization. Also remove the `github.com/google/uuid` import if it becomes unused.

6. **Use domain.ID type consistently**:
   - `WatchTarget.ProjectID` -> `shareddomain.ID`
   - `LocalConfig.ID` -> `domain.ID`
   - `LogBatch.ProjectID` -> `shareddomain.ID`
   - Map keys and values should use the appropriate ID type

7. **Skip logic for missing configs**: In Step 4, the `buildWatchTargets` function must use `continue` to skip projects and worktrees when `loadProjectID` returns an error. Do NOT add targets with empty ProjectID.

8. **Each worktree reads its own config**: In Step 4, each worktree must call `loadProjectID(wt)` with its own path, not use the parent's ProjectID.

9. **Compile verification**: After each step, run `go build ./daemon/...` to verify compilation.

10. **Testing order**: Test Steps 1-4 together (config loading), then Steps 5-8 together (record buffering), then Step 9 (DI wiring).

11. **Error handling**: The `Flush` method continues sending other project batches even if one fails. This is intentional to maximize data delivery.

12. **Empty ProjectID handling**: With the skip logic, records should never have empty ProjectID. However, if an unknown ClaudeDir is encountered in AddRecordsForClaudeDir, it will use an empty ID from the map lookup. This is an edge case that shouldn't occur in normal operation.
