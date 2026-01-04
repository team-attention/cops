# Requirements: Organization-based Access Control for CLI and Daemon

## Request Summary

Implement Organization-based access control in the cops CLI and Daemon components. The API server already enforces organization-level RBAC on all endpoints (Dashboard and Aggregation services). However, the CLI does not currently send organization information when registering projects, and the Daemon does not include organization information when sending logs. This causes API calls to fail due to missing required organization context.

This implementation will enable:
1. Users to select which Organization to add a project to during `cops add`
2. Authentication verification before project registration
3. Daemon to include Organization information when sending logs to the API server

## Acceptance Criteria

- [ ] User can authenticate via `cops login` command using Device Flow authentication
- [ ] Authentication tokens are stored securely in `~/.cops/auth.json` with 0600 permissions
- [ ] `cops add` fails with clear error message if user is not authenticated
- [ ] `cops add` fetches user's Organization list via UserService.GetMe API
- [ ] If user has only one organization, auto-select it without prompting
- [ ] If user has multiple organizations, show interactive TUI picker for selection
- [ ] Selected OrganizationID is stored in local config (`{project}/.cops/config.json`)
- [ ] CLI sends OrganizationID when calling RegisterProject API
- [ ] Daemon retrieves OrganizationID from local config for each project
- [ ] Daemon skips projects without OrganizationID and logs a warning
- [ ] Daemon includes OrganizationID when sending logs via SendLogs API
- [ ] Token refresh mechanism automatically renews expired access tokens
- [ ] Clear error messages when user has no organizations

## Scope

### In Scope
- `cops login` command implementation using Device Flow (auth.proto)
- Token storage in `~/.cops/auth.json` with secure file permissions (0600)
- Token refresh mechanism to handle expired access tokens
- Fetching user's Organization list via UserService.GetMe API
- Interactive TUI picker for organization selection (when multiple orgs exist)
- Auto-selection of organization (when user has exactly one org)
- Storing OrganizationID in local config (`{project}/.cops/config.json`)
- CLI: Adding OrganizationID field to RegisterProjectParams
- CLI: Sending OrganizationID in RegisterProject API call
- CLI: Authentication check before `cops add` (fail with error if not authenticated)
- Daemon: Reading OrganizationID from local config for each project
- Daemon: Adding OrganizationID field to LogBatch domain model
- Daemon: Sending OrganizationID in SendLogs API call
- Daemon: Graceful handling of projects without OrganizationID (skip with warning)

### Out of Scope
- Creating new authentication methods (use existing Device Flow)
- Web-based authentication flow (CLI only)
- Organization management features (creating, updating, deleting organizations)
- Multi-organization support for a single project
- Migration of existing projects without OrganizationID (users must re-register manually)
- Updating existing projects with OrganizationID via `cops add` (out of scope for this iteration)
- Changes to API server RBAC implementation (already complete)
- Creating new Organization list API (UserService.GetMe already provides this)

## Constraints

- Must use existing authentication service (auth.proto: DeviceCode, DevicePoll, RefreshToken RPCs)
- Must use existing UserService.GetMe API to fetch organizations (user.proto)
- Must follow existing protobuf conventions (organization_id field already exists in RegisterProjectReq and LogBatch)
- Must store authentication tokens in `~/.cops/auth.json` with 0600 file permissions
- Must store OrganizationID in local config (`{project}/.cops/config.json`)
- Daemon must gracefully handle projects without OrganizationID (skip sending logs and log warning)
- TUI picker must follow existing interactive picker patterns used in `cops add`
- `cops add` must fail with clear error if user is not authenticated (no auto-prompt for login)

## Additional Context

### Existing API Support
- `idl/protobuf/auth/v1/auth.proto`: Device Flow authentication already implemented
  - DeviceCode RPC: Initiates device flow
  - DevicePoll RPC: Polls for authentication completion
  - RefreshToken RPC: Refreshes expired access tokens
- `idl/protobuf/user/v1/user.proto`: UserService.GetMe API returns user info and organizations
  - GetMe RPC: Returns authenticated user's data and list of UserOrganization
  - UserOrganization: Contains id, name, and role (admin/member)
  - Requires valid JWT token in Authorization header
- `idl/protobuf/project/v1/project.proto`: RegisterProjectReq has organization_id field (line 28)
- `idl/protobuf/aggregation/v1/aggregation.proto`: LogBatch has organization_id field (line 166)
- `idl/protobuf/dashboard/v1/dashboard.proto`: All Dashboard RPCs require organization_id

### Configuration Files
- Global config: `~/.cops/config.json` (contains list of registered projects)
- Local config: `{projectPath}/.cops/config.json` (currently contains project ID only, will add organization_id)
- Auth file: `~/.cops/auth.json` (existing - stores authentication tokens with 0600 permissions)

### Code Locations Requiring Changes

#### CLI - New Components
- `cli/cmd/login.go`: New command for `cops login`
- `cli/internal/service/auth/`: Existing service - extend for authentication operations
  - Auth service: Device Flow implementation (DeviceCode, DevicePoll)
  - Auth repository: Token storage/retrieval from `~/.cops/auth.json` (already exists)
- `cli/internal/service/user/`: New service for user operations
  - User service: Fetch user info and organizations via GetMe API

#### CLI - Existing Components to Modify
- `cli/internal/service/tracking/tracking_service.go`: AddProject method (line 60)
  - Add authentication check (fail if not authenticated)
  - Fetch organizations via User service
  - Show TUI picker if multiple orgs, auto-select if only one org
  - Pass selected OrganizationID to RegisterProject
- `cli/internal/service/tracking/outbound/api/project_port.go`: RegisterProjectParams struct (line 10)
  - Add OrganizationID field
- `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`: RegisterProject implementation
  - Set organization_id in RegisterProjectReq
- `cli/internal/service/tracking/outbound/config/config_port.go`: LocalConfig struct (line 11)
  - Add OrganizationID field

#### Daemon - Existing Components to Modify
- `daemon/internal/platform/domain/watch.go`: LogBatch struct (line 36)
  - Add OrganizationID field
- `daemon/internal/service/logwatcher/log_service.go`: LogBatch creation (line 236)
  - Read OrganizationID from local config
  - Skip projects without OrganizationID (log warning)
- `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`: SendLogs implementation
  - Set organization_id in LogBatch protobuf message

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| **Organization Selection UX**: How should the user select an organization during `cops add`? Should it be an interactive TUI picker, a CLI flag (--org-id), or both? | **Interactive TUI picker** - Follow the existing pattern used in `cops add` command for interactive selection. |
| **Authentication Flow**: Should `cops add` automatically prompt for authentication if the user is not logged in, or should it fail with an error instructing the user to run a separate `cops login` command first? | **Fail with error** - Require user to run `cops login` first. Do not auto-prompt for authentication. |
| **Token Storage Location**: Where should authentication tokens be stored? Should we create a new file like `~/.cops/credentials.json` or add it to the existing `~/.cops/config.json`? What file permissions should be set for security? | **Existing file: `~/.cops/auth.json`** - Use existing auth file convention (already used by CLI and Daemon). File permissions are **0600** (read/write for owner only). |
| **OrganizationID Storage**: Where should the OrganizationID be stored for each project? Options: (a) In local config (`.cops/config.json` in each project directory), (b) In global config (`~/.cops/config.json`), or (c) Both? | **Local config** - Store OrganizationID in `{project}/.cops/config.json` alongside the existing project ID. |
| **Organization List API**: Does an API endpoint exist to fetch the user's Organization list? If not, should we create a new protobuf service or extend an existing one? | **API exists!** - Use `UserService.GetMe` from `idl/protobuf/user/v1/user.proto`. It returns `repeated UserOrganization organizations` which includes id, name, and role. |
| **Existing Projects**: How should we handle projects that were registered before OrganizationID was required? Should `cops add` be able to update existing projects with an OrganizationID? | **Out of scope** - Do not implement migration for existing projects in this iteration. Users must manually re-register projects. |
| **Daemon Behavior**: When the Daemon encounters a project without OrganizationID in configuration, should it: (a) Skip sending logs and log a warning, (b) Fail and stop the daemon, or (c) Attempt to send without OrganizationID and let the API reject it? | **Skip with warning (Option A)** - Daemon should skip projects without OrganizationID, log a warning message, and continue processing other projects gracefully. |
| **Default Organization**: If a user belongs to only one organization, should we automatically select it during `cops add` without prompting? | **Auto-select** - If user has exactly one organization, automatically select it without showing the TUI picker. |
