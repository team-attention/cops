# Requirements

## Request Summary

When the Daemon service is stopped and restarted (bootstrap), it does not restore the directory watches that were previously configured. The ConfigWatcher handler loads the global config file on startup and triggers `HandleConfigChange`, but this is a one-time initialization. The directories need to be re-watched each time the daemon starts, based on the projects listed in `~/.cops/config.json`.

## Problem Analysis

After analyzing the codebase:

1. **Current Bootstrap Flow**:
   - `ConfigWatcherFsnotifyHandler.Start()` calls `HandleConfigChange()` once on startup (line 47 in `/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go`)
   - This publishes the WatchTargets to the pubsub, which the LogWatcher service should subscribe to
   - However, there is no explicit subscription or initialization mechanism shown

2. **Missing Component**:
   - The LogWatcher service has an `UpdateTargets()` method that sets up watches
   - But there's no visible mechanism that calls `UpdateTargets()` when the daemon bootstraps

## Clarifying Questions

Before proceeding with implementation, I need to clarify the following:

### 1. Current Pubsub Subscription
**Question**: Is there an existing pubsub subscription mechanism that connects the ConfigWatcher's published WatchTargets to the LogWatcher's `UpdateTargets()` method?
- If yes, where is this subscription set up?
- If no, should we create an inbound handler for LogWatcher that subscribes to the WatchTarget pubsub?

### 2. Expected Behavior on Bootstrap
**Question**: When the daemon starts, what should happen?
- Should it:
  a) Automatically read `~/.cops/config.json` and set up all watches for registered projects?
  b) Wait for a config file change event before setting up watches?
  c) Both (initial load + watch for changes)?

**Current observation**: The code does (c) - initial load + watch for changes, but the wiring may be incomplete.

### 3. State Persistence
**Question**: The LogWatcher service maintains in-memory state (`watchedDirs`, `claudeDirToProject`, `bufferByProject`). Should any of this state persist across daemon restarts?
- File read offsets (to resume from where it left off)?
- Buffered records (to avoid data loss)?
- Or is it acceptable to start fresh on each restart?

### 4. Project Registration Flow
**Question**: When a new project is registered via `cops add`:
- Does it immediately update `~/.cops/config.json`?
- Does the config file watcher detect this change and trigger `HandleConfigChange`?
- Should the watches be set up automatically, or does the user need to restart the daemon?

**Current observation**: The global config at `~/.cops/config.json` shows the newer format with `id`, `gitProject`, `claudeDir`, `registeredAt` fields, but the domain model in `daemon/internal/platform/domain/global_config.go` only has `Path`, `Name`, and `IsGitProject` fields. This suggests a schema mismatch.

### 5. Schema Mismatch
**Question**: The `GlobalConfig` domain model differs from the actual config file format:
- Domain model: `Path`, `Name`, `IsGitProject`
- Actual file: `id`, `name`, `path`, `gitProject`, `claudeDir`, `registeredAt`

Should we:
- Update the domain model to match the actual config schema?
- Or update the config file format to match the domain model?

### 6. Verification Method
**Question**: How can we verify that the fix works?
- What is the expected log output when daemon starts successfully with watches restored?
- Is there a way to query the daemon to see which directories are currently being watched?
- Should we add health check endpoints that show watch status?

## Acceptance Criteria

- [ ] Criterion 1: When daemon starts, it MUST automatically read `~/.cops/config.json` and set up watches for all registered projects
- [ ] Criterion 2: When daemon starts, it MUST set up watches for all git worktrees associated with registered projects (if applicable)
- [ ] Criterion 3: Log output clearly indicates which directories are being watched on startup
- [ ] Criterion 4: When a config file change is detected, watches are updated accordingly (add new, remove obsolete)
- [ ] Criterion 5: The GlobalConfig domain model matches the actual config file schema
- [ ] Criterion 6: When daemon is stopped and restarted, all previous watches are restored without manual intervention

## Scope

### In Scope
- Fix the daemon bootstrap process to restore directory watches from `~/.cops/config.json`
- Ensure proper pubsub subscription between ConfigWatcher and LogWatcher
- Resolve schema mismatch between GlobalConfig domain model and actual config file
  - Daemon should use `shared/domain.Project` to access `id`, `claudeDir`, `gitProject` fields
  - Remove unnecessary `registeredAt` field from config file (CLI modification)
- Add appropriate logging to verify watch restoration on startup
- Document the bootstrap flow and watch lifecycle

### Out of Scope
- State persistence for file read offsets (will start fresh on each restart unless explicitly requested)
- UI/dashboard for visualizing watch status (command-line logging only)
- Automatic retry logic for failed watch operations
- Performance optimization for large numbers of watches

## Constraints
- Must use existing `fx` lifecycle management
- Must follow hexagonal architecture patterns (inbound/outbound separation)
- Must not break existing functionality (config file watching, log file watching when working)
- Should maintain backward compatibility with existing config files if possible

## Additional Context
- The daemon uses `go.uber.org/fx` for dependency injection and lifecycle management
- fsnotify is used for file system watching
- The pubsub is in-memory (`inmemory.PubSub`)
- The ConfigWatcher and LogWatcher are separate services that communicate via pubsub

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| (Awaiting user input) | (Awaiting user input) |
