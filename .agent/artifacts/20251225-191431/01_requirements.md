# Requirements

## Request Summary

Add two CLI features to improve project management: (1) a `cops remove` command to unregister projects by removing them from global config and deleting local config, and (2) modify `cops add` to coordinate with the API server to obtain ProjectIDs instead of generating arbitrary UUIDs locally. The `add` command will implement intelligent duplicate detection based on git repository information and local config data.

## Acceptance Criteria

### Feature 1: `cops remove` Command

- [ ] **AC1.1**: Running `cops remove <project-path>` removes the project entry from `~/.cops/config.json`
- [ ] **AC1.2**: The command deletes the `.cops/` directory in the specified project folder (if it exists)
- [ ] **AC1.3**: User receives a confirmation prompt before removal (y/n)
- [ ] **AC1.4**: If project directory doesn't exist, command skips deletion gracefully without error
- [ ] **AC1.5**: If project is not registered in global config, command skips removal gracefully without error
- [ ] **AC1.6**: Command does NOT send delete request to API server
- [ ] **AC1.7**: Success message displayed after successful removal
- [ ] **AC1.8**: Appropriate error messages shown for failure cases

### Feature 2: API-Coordinated ProjectID

- [ ] **AC2.1**: `cops add` command collects comprehensive project information before contacting API
- [ ] **AC2.2**: Command detects if directory is a git project (worktree or main branch)
- [ ] **AC2.3**: For git worktrees, command identifies the main/default branch directory
- [ ] **AC2.4**: For git projects, command extracts both configured remote URL and actual remote URL
- [ ] **AC2.5**: Command sends collected information (git remote URLs, local config if exists) to API server endpoint
- [ ] **AC2.6**: API server checks for existing project based on git remote URL and/or local config
- [ ] **AC2.7**: If project exists on server, synchronize information and use returned ProjectID
- [ ] **AC2.8**: If project doesn't exist on server, create new project and use returned ProjectID
- [ ] **AC2.9**: ProjectID from API server is saved to both global and local config
- [ ] **AC2.10**: If API server is unreachable, command fails with clear error message (no fallback to UUID)
- [ ] **AC2.11**: Command handles duplicate `cops add` attempts gracefully (synchronizes instead of errors)

## Scope

### In Scope

#### Feature 1: `cops remove`
- Remove project entry from global config (`~/.cops/config.json`)
- Delete local project config directory (`.cops/` in project folder)
- User confirmation prompt before removal
- Graceful handling when project directory or config doesn't exist
- Local-only operation (no API server communication)

#### Feature 2: `cops add` API Integration
- Git project detection (worktree vs main branch vs non-git)
- Git worktree main branch directory resolution
- Git remote URL extraction (both configured and actual)
- Local config information collection
- API server communication to check for existing projects
- API server communication to create new projects
- ProjectID retrieval and storage in configs
- Synchronization of existing project information
- Error handling for API server unavailability

### Out of Scope

- Migration of existing UUID-based projects to API-managed ProjectIDs
- Offline mode or UUID fallback when API is unreachable
- Bulk remove operations (removing multiple projects at once)
- `cops remove` with interactive project selection
- Daemon restart or notification mechanisms
- API server endpoint implementation (separate task)
- Git repository validation or authentication
- Project activation/deactivation on API server
- Removal of project data from API server database

## Constraints

- **No Migration**: Existing projects with UUID-based IDs are not automatically migrated
- **API Dependency**: `cops add` requires API server connectivity (no offline mode)
- **Git Dependency**: Git remote URL detection requires `git` CLI to be installed
- **Path-based**: `cops remove` only accepts project paths, not ProjectIDs
- **Single Project**: Both commands operate on one project at a time
- **Daemon Independence**: Commands do not directly communicate with daemon (daemon watches config file changes)
- **Communication Protocol**: This project uses protobuf IDL definitions and ConnectRPC for service communication (not REST)

## Additional Context

### Daemon Behavior
- Daemon watches `~/.cops/config.json` for changes
- When config changes, daemon should automatically pick up additions/removals
- No explicit daemon restart required after `cops add` or `cops remove`

### Config File Structure
Current structure (to be maintained):
```json
{
  "projects": [
    {
      "id": "project-id-from-api",
      "path": "/absolute/path/to/project",
      "name": "project-name",
      "gitInfo": { ... }
    }
  ]
}
```

### API Server Requirements
- **Action Required**: Verify if project registration service exists in API server
- API server must provide capability to register projects and return ProjectIDs
- API server must support duplicate project detection based on git remote URLs and existing project IDs
- API server must support both creating new projects and synchronizing existing projects

### Related Files to Review
- `cli/internal/commands/add.go` - Current `cops add` implementation
- `cli/internal/config/` - Config file read/write logic
- `shared/domain/project.go` - Project domain model
- `idl/protobuf/` - Service definitions for API communication
- API server implementation files - To verify/implement registration service

## Information Requirements

### Feature 1: `cops remove` Command

**What information is needed**:
- Project path (from user input)
- Current global config state
- Existence of local project config directory

**What should be displayed to user**:
- Project information before removal (path, name)
- Confirmation prompt
- Success or error messages

**Expected behaviors**:
- Remove project from global config
- Delete local config directory
- Handle missing/invalid projects gracefully
- Require user confirmation before removal

### Feature 2: `cops add` API Integration

**What information must be collected**:

1. **Existing Project Information**:
   - Check if project already has local config with ProjectID

2. **Git Repository Information**:
   - Whether directory is git-managed
   - Whether it's a git worktree or main branch
   - For worktrees: location of main branch directory
   - For git projects: git remote URLs (both configured and actual, as they may differ)

3. **Project Metadata**:
   - Absolute path to project directory
   - Project name (derived from directory)
   - Git repository classification (worktree/main/non-git)

**What must be communicated to API server**:
- Existing project ID (if available)
- Project absolute path
- Project name
- Git repository information (remote URLs, worktree status, main branch path)

**What must be received from API server**:
- ProjectID (either existing or newly created)
- Synchronized project metadata

**Expected API server behaviors**:
- Check if project already exists (match by git remote URLs or existing project ID)
- If exists: return existing ProjectID and synchronized metadata
- If doesn't exist: create new project and return ProjectID
- Prioritize git remote URL for duplicate detection

**Expected CLI behaviors**:
- Fail with clear error if API server is unreachable
- Save received ProjectID to both global and local configs
- Handle both new project creation and existing project synchronization
- Continue as non-git project if git information unavailable

## Questions Resolved

| Question | Answer |
|----------|--------|
| Should `cops remove` send delete request to API server? | No, only local config cleanup (global + local) |
| What if API server is unreachable during removal? | Not applicable - removal is local-only |
| What if API server is unreachable during `cops add`? | Fail with error, no UUID fallback |
| How to detect duplicate projects? | By git remote URL or existing local ProjectID |
| What happens if project already registered in `cops add`? | Synchronize with server and update local configs |
| Should we support migration of UUID projects? | No, out of scope for this task |
| Command interface for removal? | Path-based: `cops remove <project-path>` |
| Need confirmation before removal? | Yes, y/n prompt |
| What project information to send to API? | Absolute path, name, git info (remotes, worktree status, main branch path), local ProjectID if exists |

## User Stories

### Story 1: Remove Unused Project
**As a** developer
**I want to** unregister a project I'm no longer tracking
**So that** it no longer appears in my registered projects list and stops being monitored

**Acceptance**: Running `cops remove /path/to/old-project` with confirmation removes it from global config and deletes its `.cops/` directory

### Story 2: Add New Project with Server Coordination
**As a** developer
**I want to** register a new project with a server-assigned ID
**So that** my project has a consistent ID across my team and is properly tracked in the central system

**Acceptance**: Running `cops add /path/to/project` contacts API server, receives ProjectID, and saves it to both configs

### Story 3: Re-add Existing Project
**As a** developer
**I want to** run `cops add` on a project that was previously registered
**So that** my local config is synchronized with the server without creating duplicates

**Acceptance**: Running `cops add` on an already-registered project (same git remote) returns existing ProjectID and synchronizes metadata

### Story 4: Add Git Worktree
**As a** developer working with git worktrees
**I want to** register a worktree directory
**So that** the system correctly associates it with the main repository and tracks it properly

**Acceptance**: Running `cops add` on a worktree directory detects it as a worktree, finds the main branch directory, and sends this information to the API server
