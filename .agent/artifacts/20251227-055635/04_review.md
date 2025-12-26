# Pre-PR Code Review

## Review Summary
- **Status**: PASS
- **Files Reviewed**: 7
- **Issues Found**: 0 (Critical: 0, Warning: 0, Info: 0)

## Plan Compliance Checklist

- [x] Step 1: GlobalConfig schema updated to use shared/domain.Project
- [x] Step 2: ConfigWatcher service updated with nil checks
- [x] Step 3: Handler groups separated and start order enforced
- [x] Step 4: Verification logging added
- [x] No compilation errors (verified with `go build ./daemon/...`)
- [x] No logic errors or bugs
- [x] All imports are correct
- [x] Code follows project conventions

## Files Reviewed

### `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/global_config.go`

**Changes Made:**
- Removed local `ProjectConfig` struct
- Changed `Projects` field type from `[]ProjectConfig` to `[]*shareddomain.Project`
- Added import for `shareddomain "github.com/team-attention/cops/shared/domain"`

**Verification:**
- Matches plan Step 1 exactly
- The shared domain's `Project` struct uses `json:"gitProject"` tag (confirmed in `/Users/jayce/team-attention/cops/shared/domain/project.go` line 17)
- This fixes the JSON tag mismatch that was causing `IsGitProject` to always be `false`
- Using pointer slice `[]*shareddomain.Project` is correct as it matches CLI's behavior

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go`

**Changes Made:**
1. Line 66: Empty config creation changed from `[]domain.ProjectConfig{}` to `[]*shareddomain.Project{}`
2. Lines 99-102: Added nil check for project pointer in `buildWatchTargets`:
   ```go
   if project == nil {
       continue
   }
   ```

**Verification:**
- Matches plan Step 2 exactly
- Nil check is necessary since we now use pointer slice
- The import for `shareddomain` was already present (line 9)
- Field access `project.IsGitProject` and `project.Path` work correctly with pointer type

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/register_fsnotify.go`

**Changes Made:**
1. Added new interfaces:
   - `SubscriberHandler` - for handlers that subscribe to pubsub (must start first)
   - `PublisherHandler` - for handlers that publish to pubsub (must start after subscribers)
   - `FsnotifyHandler` - retained for handlers with no pubsub dependency

2. Changed struct from `fsnotifyParams` to `handlerParams` with three separate groups:
   - `SubscriberHandlers []SubscriberHandler \`group:"subscriber_handlers"\``
   - `PublisherHandlers []PublisherHandler \`group:"publisher_handlers"\``
   - `FsnotifyHandlers []FsnotifyHandler \`group:"fsnotify_handlers"\``

3. Renamed function from `registerFsnotify` to `registerHandlers`

4. Implemented ordered startup:
   - First: Start all subscriber handlers
   - Second: Start all publisher handlers
   - Third: Start all fsnotify handlers

5. Implemented reverse-order shutdown:
   - First: Stop fsnotify handlers
   - Second: Stop publisher handlers
   - Third: Stop subscriber handlers

6. Added detailed logging showing handler counts by type

**Verification:**
- Matches plan Step 3 exactly
- Start order ensures subscribers are ready before publishers publish
- Shutdown order is correctly reversed
- Interface definitions are clean and well-documented

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_config.go`

**Changes Made:**
- Lines 34-39: Changed `ConfigWatcherFsnotifyHandler` registration:
  - From: `fx.As(new(FsnotifyHandler))` with `group:"fsnotify_handlers"`
  - To: `fx.As(new(PublisherHandler))` with `group:"publisher_handlers"`
- Updated comment from "Inbound: FsnotifyHandler with fx.Group" to "Inbound: PublisherHandler - publishes to pubsub on config change"

**Verification:**
- Matches plan Step 3 (Change 2) exactly
- Correctly classifies ConfigWatcherFsnotifyHandler as a publisher (it publishes watch targets to pubsub)

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/module_log.go`

**Changes Made:**
- Lines 58-63: Changed `LogPubsubHandler` registration:
  - From: `fx.As(new(FsnotifyHandler))` with `group:"fsnotify_handlers"`
  - To: `fx.As(new(SubscriberHandler))` with `group:"subscriber_handlers"`
- Updated comment from "PubSub Handler (target changes -> Service.UpdateTargets)" to "SubscriberHandler - subscribes to pubsub for target updates"
- LogFsnotifyHandler remains as `FsnotifyHandler` with `group:"fsnotify_handlers"` (correct - it only watches files, no pubsub)

**Verification:**
- Matches plan Step 3 (Change 3) exactly
- Correctly classifies LogPubsubHandler as a subscriber (it subscribes to receive watch targets)
- LogFsnotifyHandler correctly remains as FsnotifyHandler (no pubsub dependency)

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/cmd/internal/container/application.go`

**Changes Made:**
- Line 17-18: Changed from:
  ```go
  // Handler registration
  fx.Invoke(registerFsnotify),
  ```
  To:
  ```go
  // Handler registration (ordered: subscribers -> publishers -> fsnotify)
  fx.Invoke(registerHandlers),
  ```

**Verification:**
- Matches plan Step 3 (Change 4) exactly
- Function name updated to match the renamed function
- Comment documents the startup order

**Status:** CORRECT

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go`

**Changes Made:**
- Lines 46-52: Added verification logging around initial config load:
  ```go
  // Initial config load - restore watches from saved config
  h.logger.Info("loading initial config for watch restoration", slog.String("path", h.configPath))
  if err := h.svc.HandleConfigChange(h.configPath); err != nil {
      h.logger.Warn("initial config load failed", slog.Any("error", err))
  } else {
      h.logger.Info("initial config loaded, watch targets published")
  }
  ```

**Verification:**
- Matches plan Step 4 exactly
- Logs before attempting to load config (visibility into restoration attempt)
- Logs success case with "watch targets published" message
- Error case already logged with "initial config load failed"

**Status:** CORRECT

---

## Build Verification

```bash
$ go build ./daemon/...
# Completed with no errors
```

**Status:** PASS

---

## Root Cause Analysis Verification

The original issue was that watches were not being restored when the daemon restarted. This was caused by two issues:

1. **JSON Tag Mismatch (Primary):** The daemon's local `ProjectConfig` struct used `json:"isGitProject"` while the CLI writes the config file using `shared/domain.Project` which has `json:"gitProject"`. This caused `IsGitProject` to always deserialize as `false`, preventing worktree discovery.

2. **Handler Start Order (Secondary):** The subscriber handler (`LogPubsubHandler`) might not be started before the publisher handler (`ConfigWatcherFsnotifyHandler`) published initial watch targets, causing the message to be dropped.

Both issues are now fixed:
1. GlobalConfig now uses `*shareddomain.Project` which has the correct `json:"gitProject"` tag
2. Handler groups are now separated with explicit start order (subscribers before publishers)

---

## Approval Notes

- All 4 planned steps implemented correctly
- Code compiles without errors
- Implementation follows project conventions (fx DI patterns, logging conventions)
- Handler separation provides clear semantic meaning (publisher vs subscriber vs fsnotify)
- Verification logging will help debug future issues

---

**Status: PASS**

Implementation is complete and correct. Ready for commit and PR creation.
