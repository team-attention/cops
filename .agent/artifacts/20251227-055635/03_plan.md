# Implementation Plan

## Overview

Fix the daemon watch restoration issue by correcting the schema mismatch between daemon's `GlobalConfig` domain model and the actual config file format written by the CLI, and ensure proper pubsub subscription order on bootstrap.

## Selected Packages

No new packages are required. The solution uses existing infrastructure.

## Architecture Decisions

### Decision 1: Use shared/domain.Project for GlobalConfig

**Choice**: Replace daemon-specific `ProjectConfig` struct with `shared/domain.Project` to ensure schema compatibility.

**Rationale**: The CLI writes the config file using `shared/domain.Project`, which has `json:"gitProject"` for the `IsGitProject` field. The daemon's `ProjectConfig` uses `json:"isGitProject"`, causing a JSON tag mismatch that results in `IsGitProject` always being `false` when parsed.

### Decision 2: Separate Handler Groups for Ordered Startup

**Choice**: Create separate handler groups (`subscriber_handlers` and `publisher_handlers`) and start them in explicit order.

**Rationale**: The current implementation uses a single `fsnotify_handlers` group, and fx does not guarantee handler start order. If `ConfigWatcherFsnotifyHandler.Start()` runs before `LogPubsubHandler.Start()`, the pubsub message is published with no subscribers, and the message is silently dropped.

### Decision 3: Add Verification Logging

**Choice**: Add explicit logging at bootstrap to show watch restoration status.

**Rationale**: This provides visibility into whether watches are being correctly restored, making debugging easier.

## Implementation Steps

### Step 1: Update GlobalConfig to Use shared/domain.Project

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go` (modify)

**Current Code** (lines 1-13):
```go
package domain

// GlobalConfig represents ~/.cops/config.json structure.
type GlobalConfig struct {
	Projects []ProjectConfig `json:"projects"`
}

// ProjectConfig represents a project entry in GlobalConfig.
type ProjectConfig struct {
	Path         string `json:"path"`           // Project root directory
	Name         string `json:"name,omitempty"` // Display name (optional)
	IsGitProject bool   `json:"isGitProject"`   // CLI determines this when adding
}
```

**New Code**:
```go
package domain

import (
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// GlobalConfig represents ~/.cops/config.json structure.
type GlobalConfig struct {
	Projects []*shareddomain.Project `json:"projects"`
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Config with gitProject=true | JSON with `"gitProject": true` | `IsGitProject` is `true` | Correct JSON tag parsing |
| Config with gitProject=false | JSON with `"gitProject": false` | `IsGitProject` is `false` | False case |
| Config with missing gitProject | JSON without gitProject field | `IsGitProject` defaults to `false` | Missing field default |

### Step 2: Update ConfigWatcher Service to Use New Schema

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` (modify)

**Change 1: Update loadConfig empty config creation** (lines 64-71):

**Current Code**:
```go
if os.IsNotExist(err) {
	// Create empty config file if it doesn't exist
	emptyConfig := &domain.GlobalConfig{Projects: []domain.ProjectConfig{}}
	if err := s.saveConfig(path, emptyConfig); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}
	s.logger.Info("created empty config file", slog.String("path", path))
	return emptyConfig, nil
}
```

**New Code**:
```go
if os.IsNotExist(err) {
	// Create empty config file if it doesn't exist
	emptyConfig := &domain.GlobalConfig{Projects: []*shareddomain.Project{}}
	if err := s.saveConfig(path, emptyConfig); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}
	s.logger.Info("created empty config file", slog.String("path", path))
	return emptyConfig, nil
}
```

**Change 2: Update buildWatchTargets to use shared domain** (lines 93-153):

**Current Code** (line 99-119):
```go
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
```

**New Code**:
```go
for _, project := range cfg.Projects {
	if project == nil {
		continue
	}

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
```

Note: The only change is adding a nil check for the project pointer. The field access `project.IsGitProject` now correctly maps to `json:"gitProject"` via the shared domain.

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Git project config | Project with IsGitProject=true | Worktrees are discovered | Git project branch |
| Non-git project config | Project with IsGitProject=false | No worktree discovery | Non-git project branch |
| Nil project in slice | Config with nil project entry | Skip nil entry gracefully | Nil check branch |
| Empty config file | Empty or non-existent config | Empty targets, no error | Empty config branch |

### Step 3: Fix Handler Start Order for Pubsub Subscription

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/register_fsnotify.go` (modify)
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_log.go` (modify)
- `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_config.go` (modify)

**Approach**: Create two separate handler interfaces and groups:
1. `SubscriberHandler` - Handlers that subscribe to pubsub (must start first)
2. `PublisherHandler` - Handlers that publish to pubsub (must start after subscribers)

**Change 1: Update register_fsnotify.go** (complete rewrite):

**Current Code** (lines 1-53):
```go
package container

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
)

// FsnotifyHandler is implemented by Inbound handlers that process fsnotify events.
type FsnotifyHandler interface {
	// Start begins the event loop (non-blocking).
	Start(ctx context.Context) error
	// Stop gracefully shuts down the event loop.
	Stop(ctx context.Context) error
}

type fsnotifyParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *slog.Logger
	Handlers  []FsnotifyHandler `group:"fsnotify_handlers"`
}

// registerFsnotify collects all FsnotifyHandler implementations and manages their lifecycle.
func registerFsnotify(p fsnotifyParams) {
	p.Logger.Info("registering fsnotify handlers",
		slog.Int("count", len(p.Handlers)),
	)

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			for _, handler := range p.Handlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("all fsnotify handlers started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop handlers in reverse order
			for i := len(p.Handlers) - 1; i >= 0; i-- {
				handler := p.Handlers[i]
				_ = handler.Stop(ctx)
				// Continue stopping other handlers even if one fails
			}
			p.Logger.Info("all fsnotify handlers stopped")
			return nil
		},
	})
}
```

**New Code**:
```go
package container

import (
	"context"
	"log/slog"

	"go.uber.org/fx"
)

// SubscriberHandler is implemented by handlers that subscribe to pubsub.
// These handlers MUST be started before PublisherHandlers.
type SubscriberHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// PublisherHandler is implemented by handlers that publish to pubsub.
// These handlers MUST be started after SubscriberHandlers.
type PublisherHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// FsnotifyHandler is implemented by handlers that only watch files (no pubsub).
type FsnotifyHandler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type handlerParams struct {
	fx.In

	Lifecycle          fx.Lifecycle
	Logger             *slog.Logger
	SubscriberHandlers []SubscriberHandler `group:"subscriber_handlers"`
	PublisherHandlers  []PublisherHandler  `group:"publisher_handlers"`
	FsnotifyHandlers   []FsnotifyHandler   `group:"fsnotify_handlers"`
}

// registerHandlers manages the lifecycle of all handlers with correct start order.
// Order: SubscriberHandlers -> PublisherHandlers -> FsnotifyHandlers
func registerHandlers(p handlerParams) {
	totalHandlers := len(p.SubscriberHandlers) + len(p.PublisherHandlers) + len(p.FsnotifyHandlers)
	p.Logger.Info("registering handlers",
		slog.Int("subscriber_handlers", len(p.SubscriberHandlers)),
		slog.Int("publisher_handlers", len(p.PublisherHandlers)),
		slog.Int("fsnotify_handlers", len(p.FsnotifyHandlers)),
		slog.Int("total", totalHandlers),
	)

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 1. Start subscriber handlers first (they need to subscribe before publishers publish)
			for _, handler := range p.SubscriberHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("subscriber handlers started", slog.Int("count", len(p.SubscriberHandlers)))

			// 2. Start publisher handlers (they can now publish to subscribers)
			for _, handler := range p.PublisherHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("publisher handlers started", slog.Int("count", len(p.PublisherHandlers)))

			// 3. Start fsnotify handlers (no pubsub dependency)
			for _, handler := range p.FsnotifyHandlers {
				if err := handler.Start(ctx); err != nil {
					return err
				}
			}
			p.Logger.Info("fsnotify handlers started", slog.Int("count", len(p.FsnotifyHandlers)))

			p.Logger.Info("all handlers started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop in reverse order: fsnotify -> publishers -> subscribers
			for i := len(p.FsnotifyHandlers) - 1; i >= 0; i-- {
				_ = p.FsnotifyHandlers[i].Stop(ctx)
			}
			for i := len(p.PublisherHandlers) - 1; i >= 0; i-- {
				_ = p.PublisherHandlers[i].Stop(ctx)
			}
			for i := len(p.SubscriberHandlers) - 1; i >= 0; i-- {
				_ = p.SubscriberHandlers[i].Stop(ctx)
			}
			p.Logger.Info("all handlers stopped")
			return nil
		},
	})
}
```

**Change 2: Update module_config.go** (lines 34-39):

**Current Code**:
```go
// Inbound: FsnotifyHandler with fx.Group
fx.Provide(fx.Annotate(
	fsnotifyhandler.NewConfigWatcherFsnotifyHandler,
	fx.As(new(FsnotifyHandler)),
	fx.ResultTags(`group:"fsnotify_handlers"`),
)),
```

**New Code**:
```go
// Inbound: PublisherHandler - publishes to pubsub on config change
fx.Provide(fx.Annotate(
	fsnotifyhandler.NewConfigWatcherFsnotifyHandler,
	fx.As(new(PublisherHandler)),
	fx.ResultTags(`group:"publisher_handlers"`),
)),
```

**Change 3: Update module_log.go** (lines 51-63):

**Current Code**:
```go
// Inbound 1: Fsnotify Handler (reads watcher.Events)
fx.Provide(fx.Annotate(
	fsnotifyhandler.NewLogFsnotifyHandler,
	fx.As(new(FsnotifyHandler)),
	fx.ResultTags(`group:"fsnotify_handlers"`),
)),

// Inbound 2: PubSub Handler (target changes -> Service.UpdateTargets)
fx.Provide(fx.Annotate(
	pubsubhandler.NewLogPubsubHandler,
	fx.As(new(FsnotifyHandler)),
	fx.ResultTags(`group:"fsnotify_handlers"`),
)),
```

**New Code**:
```go
// Inbound 1: Fsnotify Handler (reads watcher.Events)
fx.Provide(fx.Annotate(
	fsnotifyhandler.NewLogFsnotifyHandler,
	fx.As(new(FsnotifyHandler)),
	fx.ResultTags(`group:"fsnotify_handlers"`),
)),

// Inbound 2: SubscriberHandler - subscribes to pubsub for target updates
fx.Provide(fx.Annotate(
	pubsubhandler.NewLogPubsubHandler,
	fx.As(new(SubscriberHandler)),
	fx.ResultTags(`group:"subscriber_handlers"`),
)),
```

**Change 4: Update application.go** (line 18):

**Current Code**:
```go
// Handler registration
fx.Invoke(registerFsnotify),
```

**New Code**:
```go
// Handler registration (ordered: subscribers -> publishers -> fsnotify)
fx.Invoke(registerHandlers),
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Normal startup | Daemon starts | Subscribers started before publishers | Ordered startup |
| Config change on startup | Config exists | LogPubsubHandler receives targets | Initial config load |
| Graceful shutdown | SIGTERM | Handlers stopped in reverse order | Shutdown order |

### Step 4: Add Verification Logging

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go` (modify)

**Change: Add detailed logging on initial config load** (lines 46-49):

**Current Code**:
```go
// Initial config load
if err := h.svc.HandleConfigChange(h.configPath); err != nil {
	h.logger.Warn("initial config load failed", slog.Any("error", err))
}
```

**New Code**:
```go
// Initial config load - restore watches from saved config
h.logger.Info("loading initial config for watch restoration", slog.String("path", h.configPath))
if err := h.svc.HandleConfigChange(h.configPath); err != nil {
	h.logger.Warn("initial config load failed", slog.Any("error", err))
} else {
	h.logger.Info("initial config loaded, watch targets published")
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Config exists | Valid config file | "initial config loaded, watch targets published" | Success log |
| Config missing | No config file | Creates empty config, logs warning | Empty config creation |
| Config invalid | Malformed JSON | "initial config load failed" | Error handling |

## Execution Order

1. **Step 1**: Update GlobalConfig schema (no dependencies)
2. **Step 2**: Update ConfigWatcher service (depends on Step 1)
3. **Step 3**: Fix handler start order (no code dependencies, but logically depends on understanding the issue)
4. **Step 4**: Add verification logging (no dependencies)

**Recommended execution**: Steps 1 and 2 together, then Steps 3 and 4 together.

## Notes for Execute Agent

1. **Import Statement**: When updating `global_config.go`, add the import for `shareddomain "github.com/team-attention/cops/shared/domain"`.

2. **Import Statement for configwatcher_service.go**: The file already imports `shareddomain`, so no new import is needed. Just verify the import alias matches.

3. **Function Rename**: In `register_fsnotify.go`, the function is renamed from `registerFsnotify` to `registerHandlers`. Update the `fx.Invoke` call in `application.go` accordingly.

4. **Testing**: After implementation, start the daemon and check logs for:
   - `loading initial config for watch restoration`
   - `subscriber handlers started`
   - `publisher handlers started`
   - `config loaded and targets built projects=X targets=Y`
   - `updated watch targets watching=X mappings=X`

5. **Verify Fix**: Ensure `IsGitProject` is correctly parsed by checking if worktrees are discovered for git projects. The log message `skipping project without local config (project not registered)` for worktrees indicates worktree discovery is happening (even if the worktree is not registered).

6. **Order of Changes**: The schema fix (Steps 1-2) is the critical fix. The handler order fix (Step 3) is a preventive measure against a potential race condition. Both should be implemented.
