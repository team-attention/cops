# Requirements

## Request Summary

Fix the `cops add` command to properly handle non-Git directories. Currently, the command fails when registering non-Git projects because the API requires at least one search criterion (Git remote URL or existing project ID), but non-Git directories have neither. The solution must support hierarchical project tracking where subdirectories can be registered as separate projects, and the daemon must intelligently route session logs to the most specific matching project.

## Acceptance Criteria

- [ ] CLI successfully registers non-Git directories without requiring Git remote URLs
- [ ] When local config exists in the target directory, reuse the existing project ID
- [ ] When no local config exists, create a new project with a fresh ID
- [ ] CLI detects if any parent directory is already registered as a project
- [ ] When parent project exists, CLI prompts user: "This directory is already being tracked as a subdirectory of [parent project name] at [parent path]. Do you want to register it as a separate project?"
- [ ] User can choose to proceed with separate registration or cancel
- [ ] API accepts registration requests with only `name` and `isGitProject=false` (no remote URLs required)
- [ ] API creates new projects for non-Git directories without requiring search criteria
- [ ] MongoDB repository handles non-Git projects by creating new documents when no `existingID` is provided
- [ ] Local config is saved to `{project-path}/.cops/config.json` with the project ID
- [ ] Global config is updated with the new project entry
- [ ] Daemon uses hierarchical matching to find the most specific project for session logs
- [ ] Project matching priority in daemon: 1) Exact directory match 2) Parent directory match (most specific wins)
- [ ] No warnings or special messages are shown for non-Git projects (treat as normal)

## Scope

### In Scope

- **CLI Changes**:
  - Modify project registration flow to handle non-Git directories
  - Add parent project detection logic (walk up directory tree)
  - Add interactive prompt when parent project is detected
  - Support user confirmation/cancellation of separate project registration

- **API Changes**:
  - Modify `RegisterProject` validation to allow non-Git projects without remote URLs
  - Update MongoDB repository to create new projects when all search criteria are empty (for non-Git)
  - Ensure `name` and `isGitProject` fields are sufficient for non-Git project creation

- **Daemon Changes**:
  - Implement hierarchical project matching algorithm
  - When processing session logs, find the most specific registered project for the log file path
  - Use CIDR-like prefix matching (longest path prefix wins)
  - Handle cases where multiple projects could match (choose most specific)

- **Data Model**:
  - Modify schema if needed for cleaner implementation
  - For non-Git projects, `remote_url` field remains empty
  - Project identification for non-Git uses only the `existing_project_id` from local config

### Out of Scope

- Cross-machine synchronization of non-Git projects (not supported, accepted limitation)
- Migration of existing projects or configs
- Automatic detection of non-Git projects that should be merged
- User warnings about non-Git limitations (explicitly not wanted)
- Worktree discovery for non-Git projects (not applicable)

## Constraints

- **Local Config First**: Always check for local config before creating new projects
- **User Consent Required**: Must explicitly ask before creating separate project when parent exists
- **Hierarchical Matching**: Daemon must correctly route logs even with nested project registrations
- **Clean Implementation**: Prioritize clean, maintainable code over backward compatibility

## Additional Context

### Current Error

When running `cops add .` in a non-Git directory without local config:

```
level=ERROR msg="failed to register project" name=tracking.api.connectrpc error="unknown: no search criteria provided: at least one of configuredURL, actualURL, or existingID must be valid"
Error: internal: cannot register project: API unreachable and no existing local ID: unknown: no search criteria provided: at least one of configuredURL, actualURL, or existingID must be valid
```

### Current Code Behavior

1. **CLI** (`tracking_service.go` lines 100-106):
   - Sets `configuredURL` and `actualURL` to empty strings for non-Git projects
   - Only provides `existingProjectID` if local config exists

2. **API** (`project_repo.go` lines 54-57):
   - Validates that at least one of `configuredURL`, `actualURL`, or `existingID` is provided
   - Returns error if all are empty

3. **Daemon**:
   - Currently uses simple path matching (no hierarchical logic)
   - Does not handle nested project registrations

### Parent Project Detection Algorithm

When user runs `cops add /path/to/child`:

1. Walk up directory tree from `/path/to/child` to root
2. For each parent directory, check if it exists in global config's project list
3. If found, store the parent project info (name, path)
4. Present prompt to user with parent project details
5. If user confirms, proceed with separate registration
6. If user cancels, abort without making changes

### Daemon Matching Algorithm

When processing log file at `/path/to/child/logs/session.jsonl`:

1. Extract directory path: `/path/to/child/logs`
2. Get all registered projects from global config
3. Filter projects where log path starts with project path
4. Sort matches by path length (descending) - longest prefix first
5. Select first match (most specific project)
6. Route session records to that project ID

Example:
- Registered projects: `/home/user/repo` (Git), `/home/user/repo/subdir` (Non-Git)
- Log at `/home/user/repo/subdir/logs/session.jsonl`
- Both projects match, but `/home/user/repo/subdir` is more specific
- Route to `/home/user/repo/subdir` project

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| How to identify non-Git projects? | If local config exists, use existing project ID. Otherwise, create new project with fresh ID (no deduplication by path/name). |
| Should we deduplicate non-Git projects? | Only via local config. If no local config exists, always create new project. |
| Should we warn users about non-Git limitations? | No special warnings needed. |
| Do we need API schema changes? | Modify schema if needed for clean implementation. `remote_url` remains empty for non-Git projects. |
| What happens if parent directory is already registered? | Prompt user: "This directory is already being tracked as a subdirectory of [parent]. Do you want to register it as a separate project?" User must explicitly confirm. |
| How does daemon handle nested projects? | Use hierarchical matching - most specific (longest path prefix) project wins. |
