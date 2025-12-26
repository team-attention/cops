# Requirements

## Request Summary

Fix two critical issues in the C-Ops daemon to enable proper session tracking: (1) Implement project ID tracking so the daemon can associate log records with the correct registered project, and (2) Update the daemon to use the new AggregationService endpoint instead of the deprecated CollectorService which is causing 404 errors.

## Acceptance Criteria

- [ ] Criterion 1: Daemon reads project ID from local config file (`{projectPath}/.cops/config.json`) for each watched directory
- [ ] Criterion 2: Daemon maps ProjectPath from WatchTarget to corresponding project ID when building LogBatch
- [ ] Criterion 3: LogBatch.ProjectID field is populated with actual project ID (not empty string) when sending to API
- [ ] Criterion 4: Daemon uses `aggregation.v1.AggregationService/SendLogs` endpoint (already implemented correctly in connectrpc adapter)
- [ ] Criterion 5: API server successfully receives and processes log batches from daemon without 404 errors
- [ ] Criterion 6: Daemon handles cases where local config file does not exist (graceful degradation or appropriate error handling)

## Scope

### In Scope
- Item 1: Modify `daemon/internal/service/logwatcher/log_service.go` to resolve ProjectPath to ProjectID
- Item 2: Add mechanism to read local config (`{projectPath}/.cops/config.json`) in daemon
- Item 3: Store ProjectPath-to-ProjectID mapping in logwatcher service
- Item 4: Verify AggregationService endpoint is correctly configured in API server (already done, just needs confirmation)
- Item 5: Handle missing local config scenarios (e.g., project not registered via CLI)

### Out of Scope
- Item 1: Creating new protobuf definitions (AggregationService already exists and is correctly implemented)
- Item 2: Modifying CLI project registration logic (already working correctly)
- Item 3: Database schema changes (Project schema already has ID field)
- Item 4: Dashboard or web UI changes
- Item 5: Backwards compatibility with CollectorService (service was renamed, not a breaking change)

## Constraints
- Technical constraint 1: Must use existing local config file structure (`{projectPath}/.cops/config.json` containing `{"id": "..."}`)
- Technical constraint 2: Cannot modify protobuf schema (already correct with `project_id` field)
- Technical constraint 3: Must maintain hexagonal architecture pattern (port/adapter separation)
- Technical constraint 4: Should handle edge cases like missing config files without crashing daemon

## Additional Context

### Current Data Flow
1. **CLI Registration**: When user runs `cops add [directory]`, CLI:
   - Calls API to register project (gets project ID from MongoDB)
   - Saves project to global config (`~/.cops/config.json`) with full project details including ID
   - Saves local config (`{projectPath}/.cops/config.json`) with just `{"id": "projectID"}`

2. **Daemon Watching**: Daemon currently:
   - Watches `~/.cops/config.json` for project changes
   - Builds WatchTarget list with ProjectPath and ClaudeDir
   - Monitors Claude Code logs in each directory
   - Creates LogBatch with **empty ProjectID** (the TODO at line 169)
   - Sends to API using correct AggregationService endpoint

3. **Missing Link**: Daemon needs to:
   - Read local config from `{projectPath}/.cops/config.json`
   - Map WatchTarget.ProjectPath to ProjectID when creating LogBatch
   - Populate LogBatch.ProjectID before sending to API

### File Locations
- Local config location: `{projectPath}/.cops/config.json`
- Local config structure: `{"id": "694e924aacaeed28b71a2fa4"}` (just project ID)
- Global config location: `~/.cops/config.json`
- Global config structure: Contains full Project array with ID, name, path, etc.
- TODO location: `daemon/internal/service/logwatcher/log_service.go:167-170`

### API Endpoint Status
- Old endpoint (causing 404): `collector.v1.CollectorService/SendRecords` (NOT IN USE)
- New endpoint (correct): `aggregation.v1.AggregationService/SendLogs` (ALREADY IMPLEMENTED)
- ConnectRPC adapter: Already correctly using AggregationService at `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- API handler: Already correctly implements AggregationService at `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`

### Architecture Notes
- Daemon uses domain model: `daemon/internal/platform/domain/watch.go` (WatchTarget, LogBatch)
- Shared domain model: `shared/domain/project.go` (Project with ID field)
- Local config read/write: Currently only in CLI, need to implement in daemon
- Pattern to follow: Similar to `cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go`

## Questions Resolved

| Question                                                                                      | Answer                                                                                                                                                    |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Where should project ID come from?                                                            | Local config file at `{projectPath}/.cops/config.json` which contains `{"id": "projectID"}`                                                              |
| Is the AggregationService endpoint already implemented?                                       | Yes, both daemon client and API handler already use AggregationService correctly. No protobuf changes needed.                                            |
| What should happen if local config file doesn't exist?                                        | Project was not registered via CLI. Options: (1) Skip sending logs for that project, (2) Use empty project ID, (3) Log warning. Recommend option 1 or 3. |
| Should daemon modify WatchTarget structure to include ProjectID?                              | Yes, enhancing WatchTarget with ProjectID field would simplify LogBatch creation and make the association explicit.                                      |
| Should daemon cache ProjectPath-to-ProjectID mapping or read config file on every flush?      | Cache the mapping when UpdateTargets is called. Re-read only when targets change (already triggered by config watcher).                                  |
| Is there a race condition between config file creation and daemon starting to watch a project?| Possibly. Daemon should handle missing config gracefully. ConfigWatcher publishes targets when config changes, so re-reading should sync state.          |
