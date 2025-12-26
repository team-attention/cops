# Development Walkthrough: Daemon Watch Restoration Fix

## Summary

Fixed the daemon's inability to restore file watches on restart by correcting a JSON schema mismatch between the CLI and daemon, and ensuring proper handler startup order for pubsub communication. This ensures that when the daemon restarts, it correctly loads the saved project configuration and restores watches on all registered directories.

## Problem Statement

### Issue
When the daemon restarted, it failed to restore file watches on previously registered project directories. Users would need to manually re-register projects after every daemon restart.

### Root Causes

1. **JSON Schema Mismatch (Primary)**: The daemon used a local `ProjectConfig` struct with `json:"isGitProject"` tag, while the CLI writes config files using `shared/domain.Project` which has `json:"gitProject"`. This mismatch caused `IsGitProject` to always deserialize as `false`, preventing git worktree discovery.

2. **Handler Startup Race Condition (Secondary)**: The `LogPubsubHandler` (subscriber) might start after `ConfigWatcherFsnotifyHandler` (publisher), causing the initial watch targets message published during daemon startup to be dropped.

## Code Overview

### Modified Components

#### 1. GlobalConfig Schema Unification
**Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go`

**Changes**:
- Removed daemon-specific `ProjectConfig` struct
- Changed `Projects` field from `[]ProjectConfig` to `[]*shareddomain.Project`
- Added import for `shared/domain`

**Why**: Ensures the daemon reads the exact same schema that the CLI writes, eliminating JSON tag mismatches.

**Before**:
```go
type GlobalConfig struct {
    Projects []ProjectConfig `json:"projects"`
}

type ProjectConfig struct {
    Path         string `json:"path"`
    Name         string `json:"name,omitempty"`
    IsGitProject bool   `json:"isGitProject"`  // ❌ Wrong tag
}
```

**After**:
```go
import shareddomain "github.com/team-attention/cops/shared/domain"

type GlobalConfig struct {
    Projects []*shareddomain.Project `json:"projects"`
}
// Now uses shared Project with json:"gitProject" ✅
```

---

#### 2. ConfigWatcher Service Updates
**Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go`

**Changes**:
1. Updated empty config creation to use `[]*shareddomain.Project{}`
2. Added nil check in `buildWatchTargets` loop for pointer safety

**Why**: Adapts service logic to work with pointer slice and ensures robustness.

**Key Change (lines 99-102)**:
```go
for _, project := range cfg.Projects {
    if project == nil {  // ✅ Nil check for pointer safety
        continue
    }
    // ... rest of logic
}
```

---

#### 3. Handler Startup Order Enforcement
**Location**: `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/register_fsnotify.go`

**Changes**:
- Split single `FsnotifyHandler` interface into three distinct types:
  - `SubscriberHandler` - handlers that subscribe to pubsub (must start first)
  - `PublisherHandler` - handlers that publish to pubsub (must start after subscribers)
  - `FsnotifyHandler` - handlers with no pubsub dependency
- Renamed function from `registerFsnotify` to `registerHandlers`
- Implemented explicit startup order: Subscribers → Publishers → Fsnotify
- Implemented reverse shutdown order

**Why**: Guarantees that pubsub subscribers are ready before publishers send messages, preventing message loss on startup.

**Startup Sequence**:
```go
// 1. Start subscriber handlers first
for _, handler := range p.SubscriberHandlers {
    handler.Start(ctx)
}

// 2. Start publisher handlers (subscribers now ready)
for _, handler := range p.PublisherHandlers {
    handler.Start(ctx)
}

// 3. Start fsnotify handlers (no pubsub dependency)
for _, handler := range p.FsnotifyHandlers {
    handler.Start(ctx)
}
```

---

#### 4. Handler Classification Updates

**ConfigWatcher Handler** (`daemon/cmd/internal/container/module_config.go`):
```go
// Changed from:
fx.As(new(FsnotifyHandler))
fx.ResultTags(`group:"fsnotify_handlers"`)

// To:
fx.As(new(PublisherHandler))  // ✅ Publishes watch targets
fx.ResultTags(`group:"publisher_handlers"`)
```

**Log Pubsub Handler** (`daemon/cmd/internal/container/module_log.go`):
```go
// Changed from:
fx.As(new(FsnotifyHandler))
fx.ResultTags(`group:"fsnotify_handlers"`)

// To:
fx.As(new(SubscriberHandler))  // ✅ Subscribes to watch targets
fx.ResultTags(`group:"subscriber_handlers"`)
```

**Log Fsnotify Handler** (`daemon/cmd/internal/container/module_log.go`):
```go
// Remained as:
fx.As(new(FsnotifyHandler))  // ✅ Only watches files, no pubsub
fx.ResultTags(`group:"fsnotify_handlers"`)
```

---

#### 5. Verification Logging
**Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go`

**Changes**:
- Added log message before initial config load: "loading initial config for watch restoration"
- Added log message on success: "initial config loaded, watch targets published"

**Why**: Provides visibility into watch restoration process for debugging.

**Added Logs**:
```go
h.logger.Info("loading initial config for watch restoration",
    slog.String("path", h.configPath))
if err := h.svc.HandleConfigChange(h.configPath); err != nil {
    h.logger.Warn("initial config load failed", slog.Any("error", err))
} else {
    h.logger.Info("initial config loaded, watch targets published")
}
```

---

## Data Flow After Fix

### Daemon Bootstrap Sequence

```
1. registerHandlers() called
   ↓
2. SubscriberHandlers.Start()
   → LogPubsubHandler subscribes to "watch_targets" topic
   ↓
3. PublisherHandlers.Start()
   → ConfigWatcherFsnotifyHandler.Start()
      → Loads ~/.cops/config.json (using shared/domain.Project)
      → IsGitProject correctly parsed (json:"gitProject" ✅)
      → Builds watch targets (including worktrees)
      → Publishes watch targets to pubsub
   ↓
4. LogPubsubHandler receives watch targets
   → LogService.UpdateTargets() called
   → File watchers registered for all directories
   ↓
5. FsnotifyHandlers.Start()
   → LogFsnotifyHandler starts watching files
```

### Key Improvements

**Before Fix**:
- `IsGitProject` always `false` → No worktrees discovered
- Subscriber might not be ready → Message dropped
- Result: No watches restored on restart

**After Fix**:
- `IsGitProject` correctly parsed → Worktrees discovered
- Subscriber guaranteed ready → Message received
- Result: All watches restored on restart ✅

---

## Files Modified

| File | Lines Changed | Description |
|------|---------------|-------------|
| `daemon/internal/platform/domain/global_config.go` | 13 → 10 | Removed `ProjectConfig`, use `shared/domain.Project` |
| `daemon/internal/service/configwatcher/configwatcher_service.go` | 2 locations | Empty config creation + nil check |
| `daemon/cmd/internal/container/register_fsnotify.go` | Complete rewrite | Handler groups + ordered startup |
| `daemon/cmd/internal/container/module_config.go` | Lines 34-39 | Classify as `PublisherHandler` |
| `daemon/cmd/internal/container/module_log.go` | Lines 58-63 | Classify as `SubscriberHandler` |
| `daemon/cmd/internal/container/application.go` | Line 17-18 | Rename invoke to `registerHandlers` |
| `daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go` | Lines 46-52 | Add verification logs |

---

## Testing

### Verification Steps

1. **Start daemon**:
   ```bash
   cd daemon && make dev
   ```

2. **Check logs for successful startup**:
   ```
   [INFO] registering handlers subscriber_handlers=1 publisher_handlers=1 fsnotify_handlers=1 total=3
   [INFO] subscriber handlers started count=1
   [INFO] loading initial config for watch restoration path=/Users/.../.cops/config.json
   [INFO] config loaded and targets built projects=X targets=Y
   [INFO] initial config loaded, watch targets published
   [INFO] publisher handlers started count=1
   [INFO] updated watch targets watching=Y mappings=Y
   [INFO] fsnotify handlers started count=1
   [INFO] all handlers started
   ```

3. **Verify git projects**:
   - For projects with `"gitProject": true` in config
   - Check logs for worktree discovery messages
   - Verify watches established on worktree directories

4. **Restart daemon**:
   - Stop daemon (Ctrl+C)
   - Restart with `make dev`
   - Confirm watches restored (same log messages appear)

### Expected Behavior

- **Git projects**: Daemon discovers and watches worktree directories
- **Non-git projects**: Daemon watches main project directory only
- **Restart**: All previous watches restored automatically
- **No errors**: Clean startup with all handlers in correct order

---

## Impact Analysis

### What Was Fixed
- Daemon now correctly restores file watches on restart
- Git worktrees are properly discovered and monitored
- Handler startup order prevents race conditions
- Better observability with detailed logging

### What Was NOT Changed
- CLI project registration logic (unchanged)
- Config file format (already correct)
- Service business logic (unchanged)
- File watching mechanism (unchanged)

### Breaking Changes
**None** - This is a bug fix with no API or behavior changes from user perspective.

---

## Related Context

### Configuration Files
- **Global config**: `~/.cops/config.json` (written by CLI, read by daemon)
- **Project config**: `{project}/.cops/config.json` (contains ProjectID)

### Key Domain Model
`shared/domain.Project` (line 17):
```go
type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`  // ✅ Correct tag
    ClaudeDir    string    `json:"claudeDir"`
    Worktrees    []string  `json:"worktrees,omitempty"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

### Architectural Context
- Uses `fx` for dependency injection and lifecycle management
- Pubsub pattern for inter-service communication within daemon
- Hexagonal architecture with clear inbound/outbound separation

---

## Future Improvements

1. **Config validation**: Add schema validation to detect mismatches earlier
2. **Health checks**: Expose metric for number of active watches
3. **Config migration**: Consider version field for future schema changes
4. **Integration tests**: Add tests for daemon restart scenarios

---

## Commit Information

**Branch**: main
**Commit**: Not yet committed (changes in working directory)
**Previous commit**: 470bce3 - feat: implement project ID tracking in daemon
