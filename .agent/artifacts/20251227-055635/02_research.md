# Research Report

## Mode
General Research

## Request Summary
The Daemon does not restore directory watches on restart. When the daemon bootstraps, it should automatically read `~/.cops/config.json` and set up watches for all registered projects, but this is not happening correctly.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go` | ConfigWatcher bootstrap flow - calls HandleConfigChange on Start |
| `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` | Core service that builds WatchTargets and publishes to pubsub |
| `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/pubsub/handler.go` | LogPubsubHandler that should receive WatchTargets and call UpdateTargets |
| `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go` | Daemon's GlobalConfig domain model (SCHEMA MISMATCH) |
| `/Users/jayce/team-attention/cops/shared/domain/project.go` | CLI's Project domain model (actual schema) |
| `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/register_fsnotify.go` | FsnotifyHandler lifecycle registration |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-domain.md` | Rules for domain model definitions |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md` | Rules for struct field types |

## Root Cause Analysis

### Issue 1: Schema Mismatch (CRITICAL)

The daemon's `GlobalConfig` domain model does not match the actual config file schema written by the CLI.

**Daemon's domain model** (`/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go:4-13`):
```go
type GlobalConfig struct {
    Projects []ProjectConfig `json:"projects"`
}

type ProjectConfig struct {
    Path         string `json:"path"`           // Project root directory
    Name         string `json:"name,omitempty"` // Display name (optional)
    IsGitProject bool   `json:"isGitProject"`   // CLI determines this when adding
}
```

**Actual config file** (`~/.cops/config.json`):
```json
{
  "projects": [
    {
      "id": "694e924aacaeed28b71a2fa4",
      "name": "cops",
      "path": "/Users/jayce/team-attention/cops",
      "gitProject": true,
      "claudeDir": "/Users/jayce/.claude/projects/-Users-jayce-team-attention-cops",
      "registeredAt": "2025-12-26T22:48:58.688216+09:00"
    }
  ]
}
```

**CLI's domain model** (`/Users/jayce/team-attention/cops/shared/domain/project.go:14-21`):
```go
type Project struct {
    ProjectAbstract                          // ID, Name, Path
    IsGitProject bool      `json:"gitProject"`          // Note: "gitProject" not "isGitProject"
    ClaudeDir    string    `json:"claudeDir"`
    Worktrees    []string  `json:"worktrees,omitempty"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

**Schema differences:**
| Field | Daemon Model | Actual Schema | Impact |
|-------|-------------|---------------|--------|
| `id` | Missing | Present | N/A (not needed) |
| `isGitProject` vs `gitProject` | `json:"isGitProject"` | `json:"gitProject"` | **BREAKS PARSING** - always false |
| `claudeDir` | Missing | Present | N/A (computed by daemon) |
| `registeredAt` | Missing | Present | N/A (not needed) |

**Result**: When daemon parses the config, `IsGitProject` is always `false` because the JSON tag is wrong (`isGitProject` vs `gitProject`). This causes worktrees to never be discovered.

### Issue 2: Race Condition on Bootstrap (POTENTIAL)

Looking at the bootstrap sequence:

1. **fx.Lifecycle OnStart** calls all handlers' `Start()` methods
2. **ConfigWatcherFsnotifyHandler.Start()** (`line 47`) calls `HandleConfigChange()` immediately
3. **HandleConfigChange()** (`configwatcher_service.go:57`) calls `pubsub.Publish(targets)`
4. **LogPubsubHandler.Start()** (`line 38`) subscribes to pubsub

**The problem**: The handlers are registered in `register_fsnotify.go` using a group, and `fx` may start them in undefined order. If `ConfigWatcherFsnotifyHandler.Start()` runs before `LogPubsubHandler.Start()`:
- The pubsub message is published
- But there are NO subscribers yet
- The message is dropped (see `inmemory/pubsub.go:44-46` - drops when no subscribers)

**Evidence from pubsub implementation** (`/Users/jayce/team-attention/cops/daemon/internal/platform/pkg/pubsub/inmemory/pubsub.go:40-48`):
```go
func (ps *PubSub[T]) Publish(msg T) error {
    // ...
    for _, ch := range ps.subscribers {
        select {
        case ch <- msg:
            // Message sent successfully
        default:
            // Channel is full, drop message for this subscriber
            ps.logger.Warn("subscriber channel full, message dropped")
        }
    }
    return nil
}
```

If `ps.subscribers` is empty (no one subscribed yet), the loop body never executes and the message is silently lost.

## Bootstrap Flow Analysis

### Current Flow

```
fx.New() -> Run()
  |-- newPlatformModule()  -> InitTargetPubSub() creates *PubSub
  |-- newConfigModule()    -> Provides ConfigWatcherFsnotifyHandler
  |-- newLogModule()       -> Provides LogPubsubHandler
  +-- fx.Invoke(registerFsnotify)
        +-- fx.Lifecycle.Append(OnStart)
              +-- For each handler in group "fsnotify_handlers":
                    handler.Start(ctx)
```

### Handler Start Order Issue

The handlers are collected via `group:"fsnotify_handlers"`:
- `ConfigWatcherFsnotifyHandler` (from module_config.go)
- `LogFsnotifyHandler` (from module_log.go)
- `LogPubsubHandler` (from module_log.go)

**Order is undefined** - fx collects them based on module load order and provides them as a slice, but the iteration order in `registerFsnotify` is:
```go
for _, handler := range p.Handlers {
    if err := handler.Start(ctx); err != nil {
        return err
    }
}
```

### ConfigWatcherFsnotifyHandler.Start() Flow

Location: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go:43-60`

```go
func (h *ConfigWatcherFsnotifyHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(context.Background())

    // Initial config load - THIS PUBLISHES TO PUBSUB IMMEDIATELY
    if err := h.svc.HandleConfigChange(h.configPath); err != nil {
        h.logger.Warn("initial config load failed", slog.Any("error", err))
    }

    // Watch config file
    if err := h.watcher.Add(h.configPath); err != nil {
        return err
    }

    h.logger.Info("config watcher started", slog.String("path", h.configPath))

    go h.loop()
    return nil
}
```

### HandleConfigChange() Flow

Location: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go:44-58`

```go
func (s *Service) HandleConfigChange(path string) error {
    cfg, err := s.loadConfig(path)  // Loads GlobalConfig with WRONG SCHEMA
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    targets := s.buildWatchTargets(cfg)  // Builds targets, but IsGitProject=false

    s.logger.Info("config loaded and targets built",
        slog.Int("projects", len(cfg.Projects)),
        slog.Int("targets", len(targets)),
    )

    return s.pubsub.Publish(targets)  // Published, but maybe no subscribers yet
}
```

### LogPubsubHandler.Start() Flow

Location: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/pubsub/handler.go:36-43`

```go
func (h *LogPubsubHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(context.Background())
    h.targetCh = h.targetReader.Subscribe()  // Subscribes HERE

    h.logger.Info("log pubsub handler started")
    go h.loop()
    return nil
}
```

## Technical Constraints

1. Must use existing `fx` lifecycle management
2. Must follow hexagonal architecture patterns
3. Must not break existing config file watching functionality
4. The daemon's GlobalConfig model should match CLI's Project model for the fields it needs

## Similar Implementations Found

### Example 1: CLI's GlobalConfig (correct schema)
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go:6-8`
- **Relevance**: Shows the correct approach - CLI uses `[]*domain.Project` directly

```go
type GlobalConfig struct {
    Projects []*domain.Project `json:"projects"`
}
```

### Example 2: CLI's Project domain model
- **File**: `/Users/jayce/team-attention/cops/shared/domain/project.go:14-21`
- **Relevance**: The actual schema being written to config files

```go
type Project struct {
    ProjectAbstract                          // ID, Name, Path
    IsGitProject bool      `json:"gitProject"`  // Note the JSON tag
    ClaudeDir    string    `json:"claudeDir"`
    Worktrees    []string  `json:"worktrees,omitempty"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

## Package Candidates

No new packages are needed for this fix. The solution uses existing infrastructure.

## Proposed Solution

### Fix 1: Update GlobalConfig Schema (CRITICAL)

**File to modify**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go`

**Change**: Replace daemon-specific `ProjectConfig` struct with `shared/domain.Project` to ensure schema compatibility.

```go
package domain

import (
    shareddomain "github.com/team-attention/cops/shared/domain"
)

// GlobalConfig represents ~/.cops/config.json structure.
type GlobalConfig struct {
    Projects []*shareddomain.Project `json:"projects"`
}

// No need for ProjectConfig - use shareddomain.Project directly
```

**Impact on configwatcher_service.go**: Update `buildWatchTargets` to use `project.IsGitProject` (which correctly maps to `json:"gitProject"`).

### Fix 2: Ensure Subscription Before Publication (CRITICAL)

**Problem**: The pubsub message may be published before any subscribers exist.

**Solution Options**:

#### Option A: Ensure handler start order (Recommended)
Explicitly control the order of handler starts in `register_fsnotify.go` by separating subscriber registration:

1. Start all subscriber handlers first (LogPubsubHandler)
2. Then start publisher handlers (ConfigWatcherFsnotifyHandler)

**Implementation**: Add a separate group for subscriber handlers or use fx.Invoke ordering.

#### Option B: Use buffered pubsub with replay
Modify the pubsub to buffer the last message and replay it to new subscribers.

**Not recommended**: Adds complexity and may cause other issues.

#### Option C: Initial load in LogPubsubHandler
Have LogPubsubHandler request initial targets after subscribing.

**Implementation**: Add a method to ConfigWatcher service that the LogPubsubHandler can call after subscription.

### Recommended Implementation Order

1. **Fix schema mismatch** (Fix 1) - This is critical and may be the only issue
2. **Test if watches work** - After schema fix, test if the race condition actually happens
3. **Fix subscription order** (Fix 2, Option A) - Only if race condition is confirmed

## Additional Information for Planning

### Files Requiring Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `daemon/internal/platform/domain/global_config.go` | Replace | Use `shared/domain.Project` instead of custom struct |
| `daemon/internal/service/configwatcher/configwatcher_service.go` | Minor update | Update field access to use new schema |
| `daemon/cmd/internal/container/register_fsnotify.go` | Potential | May need to control handler start order |

### Testing Strategy

1. Start daemon with existing config file
2. Check logs for "config loaded and targets built" with correct target count
3. Check logs for "updated watch targets" showing directories being watched
4. Verify file changes in Claude directories trigger log processing

### Log Messages to Verify

On successful bootstrap:
```
INFO config loaded and targets built projects=1 targets=1
INFO updated watch targets watching=1 mappings=1
```

If schema mismatch (current behavior):
```
INFO config loaded and targets built projects=1 targets=1
# But targets may have incorrect data due to IsGitProject=false
```

If race condition (no subscribers):
```
DEBUG publishing message subscribers=0
INFO updated watch targets watching=0 mappings=0
# Or no "updated watch targets" log at all
```
