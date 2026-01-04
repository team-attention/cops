# Development Walkthrough

## Summary
Implemented Organization-based access control for CLI and Daemon components. Users must now select an organization when adding projects via `cops add`, and the Daemon includes organization information when sending logs to the API server. This enables the API server's existing organization-level RBAC to function correctly.

## Code Overview

### Proto Schema Changes

#### `idl/protobuf/user/v1/user.proto`
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`
- **Purpose**: Standardize organization data type across services
- **Changes**:
  - Removed `UserOrganization` message (contained redundant organization fields)
  - Updated `GetMeRes.organizations` to use `domain.v1.Organization` from shared domain proto
  - Simplified schema by reusing domain types instead of duplicating structures

**Generated Code**: `shared/gen/grpcstub/user/v1/user.pb.go` (auto-regenerated via `buf generate`)

---

### CLI - New Components

#### User Service - Port and Adapter

**`cli/internal/service/user/outbound/api/user_port.go`**
- **Purpose**: Define interface for user API operations
- **Key Types**:
  - `GetMeResult`: Contains user ID and list of organizations (protobuf types)
- **Interface**:
  - `GetMe(ctx, accessToken)`: Fetches authenticated user's organizations from API

**`cli/internal/service/user/outbound/api/connectrpc/user_client.go`**
- **Purpose**: ConnectRPC implementation of UserAPIPort
- **Key Methods**:
  - `GetMe()`: Calls UserService.GetMe RPC with JWT token in Authorization header
  - Returns organizations as `[]*domainv1.Organization` (protobuf types)

**`cli/internal/service/user/user_service.go`**
- **Purpose**: Business logic for fetching user organizations
- **Key Methods**:
  - `GetMyOrganizations(ctx)`: Returns `[]*domain.Organization` (domain types for TUI)
  - Uses `auth.Service.GetAccessToken()` to obtain valid token (auto-refreshes if expired)
  - Calls `UserAPIPort.GetMe()` and returns domain-typed organizations

**Dependencies**:
- User service depends on Auth service for token management
- Uses existing authentication infrastructure (no login prompts in add flow)

---

#### Organization Selection TUI

**`cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`**
- **Purpose**: Add organization selection step to project registration flow
- **Changes**:
  - Added new step: `stepOrgSelection` (between parent detection and git selection)
  - Added fields to `addModel`:
    - `organizations`: List of user's organizations (domain types)
    - `orgCursor`: Cursor position for organization picker
    - `selectedOrgID`, `selectedOrgName`: Selected organization
    - `authSvc`, `userSvc`: Service dependencies
  - Added to `addTUIResult`:
    - `OrganizationID`: Selected organization to pass to service layer
  - **Auto-selection**: If user has exactly one organization, it's selected automatically without showing picker
  - **Fetching**: Organizations fetched via `userSvc.GetMyOrganizations()` at TUI initialization

**`cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`**
- **Purpose**: Handle organization selection interactions
- **New Message Types**:
  - `orgFetchMsg`: Result of organization fetch (success or error)
  - `authErrorMsg`: Authentication failures
- **Key Update Logic**:
  - `stepOrgSelection`: Handle up/down/enter keys for organization picker
  - Auto-skip picker if single organization
  - Move to `stepParentDetection` after organization selected

**`cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`**
- **Purpose**: Render organization selection UI
- **New View**:
  - `viewOrgSelection()`: Shows list of organizations with cursor
  - Displays organization names only (role not needed for selection)
  - Help text: "up/down: navigate | enter: select | ctrl+c: cancel"

---

#### CLI Handler Updates

**`cli/internal/service/tracking/inbound/cli/cobra/handler.go`**
- **Changes**:
  - Added `authSvc *auth.Service` field
  - Added `userSvc *user.Service` field
  - Updated constructor to accept both services
- **Purpose**: Provide auth and user services to TUI

**`cli/internal/service/tracking/inbound/cli/cobra/add.go`**
- **Changes**:
  - Added authentication check: `if !h.authSvc.IsLoggedIn()` returns error
  - Error message: "not authenticated. Run 'cops login' first"
  - Pass `authSvc` and `userSvc` to `runAddTUI()`
  - Pass `result.OrganizationID` to `AddProjectParams`
- **Purpose**: Enforce authentication before project registration

---

### CLI - Modified Components

#### LocalConfig Schema (BREAKING CHANGE)

**`cli/internal/service/tracking/outbound/config/config_port.go`**
- **Changes**:
  - Renamed `ID` → `ProjectID` (JSON field: `"id"` → `"projectId"`)
  - Added `OrganizationID string` field (JSON: `"organizationId,omitempty"`)
- **Impact**: Existing `.cops/config.json` files must be regenerated via `cops add`

#### RegisterProject API Parameters

**`cli/internal/service/tracking/outbound/api/project_port.go`**
- **Changes**: Added `OrganizationID string` field to `RegisterProjectParams`

**`cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`**
- **Changes**: Set `organization_id` field in `RegisterProjectReq` protobuf message

#### Tracking Service

**`cli/internal/service/tracking/tracking_service.go`**
- **Changes**:
  - Added `OrganizationID string` to `AddProjectParams`
  - Updated `AddProject()` method:
    - Changed `localConfig.ID` → `localConfig.ProjectID` (all references)
    - Pass `params.OrganizationID` to `RegisterProject` API call
    - Store `params.OrganizationID` in `LocalConfig`
- **Purpose**: Thread organization ID through registration flow

#### DI Container

**`cli/cmd/internal/container/container.go`**
- **Changes**: Added `newUserModule(c)` registration
- **Purpose**: Wire user service into DI container

**`cli/cmd/internal/container/module_user.go`** (NEW)
- **Purpose**: Register user service components in DI container
- **Providers**:
  - `connectrpc.NewUserAPIClient` as `api.UserAPIPort`
  - `user.NewService`

---

### Daemon - Modified Components

#### LocalConfig Schema (BREAKING CHANGE)

**`daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go`**
- **Changes**: Same as CLI (must match)
  - Renamed `ID` → `ProjectID` (JSON field: `"id"` → `"projectId"`)
  - Added `OrganizationID string` field (JSON: `"organizationId,omitempty"`)

#### WatchTarget and LogBatch Domain Models

**`daemon/internal/platform/domain/watch.go`**
- **Changes**:
  - Added `OrganizationID string` to `WatchTarget` struct
  - Added `OrganizationID string` to `LogBatch` struct
- **Purpose**: Track organization ID for each project and include in log batches

#### ConfigWatcher Service

**`daemon/internal/service/configwatcher/configwatcher_service.go`**
- **Changes**:
  - Renamed `loadProjectID()` → `loadLocalConfig()` (returns full config, not just ID)
  - Updated `buildWatchTargets()`:
    - Read full `LocalConfig` for each project
    - **Skip projects without OrganizationID** with warning:
      ```
      "skipping project without organization ID (run 'cops add' to re-register)"
      ```
    - Changed `localCfg.ID` → `localCfg.ProjectID` (all references)
    - Include `localCfg.OrganizationID` in `WatchTarget`
    - Apply same logic to worktrees (skip if no OrganizationID)
- **Purpose**: Gracefully handle legacy projects without organization ID

#### LogWatcher Service

**`daemon/internal/service/logwatcher/log_service.go`**
- **Changes**:
  - Added `projectIDToOrgID map[shareddomain.ID]string` field to track mappings
  - Updated `UpdateTargets()`:
    - Build `projectIDToOrgID` mapping from `WatchTarget.OrganizationID`
    - Update internal mapping when targets change
  - Updated `flushProjectLines()`:
    - Look up `orgID := s.projectIDToOrgID[projectID]`
    - Include `OrganizationID` in `LogBatch`
- **Purpose**: Include organization ID when sending logs to API

#### API Client

**`daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`**
- **Changes**: Set `organization_id` field in `LogBatch` protobuf message
- **Purpose**: Send organization ID to API server for RBAC enforcement

---

### API Server - Fix for Protobuf Schema Change

#### User Service Handler

**`api/internal/service/user/inbound/grpc/connectrpc/handler.go`**
- **Changes**: Updated `GetMe()` RPC implementation
  - Changed `var protoOrgs []*userv1.UserOrganization` → `[]*domainv1.Organization`
  - Updated mapping to include `Slug` field (new in domain.v1.Organization)
  - Removed `Role` field from response (not needed for organization selection)
  - Members field intentionally not populated (empty by default)
- **Purpose**: Fix build error caused by removal of `userv1.UserOrganization` type

---

## How the New Features Work

### CLI Organization Selection Flow

1. **User runs `cops add`**
   - CLI checks if authenticated via `authSvc.IsLoggedIn()`
   - If not authenticated, returns error: "Run 'cops login' first"

2. **TUI Initialization**
   - Fetches organizations via `userSvc.GetMyOrganizations()`
   - User service calls `auth.Service.GetAccessToken()` (auto-refreshes token if needed)
   - User service calls `UserAPIPort.GetMe()` with access token
   - Converts protobuf organizations to domain organizations

3. **Organization Selection**
   - If user has **1 organization**: Auto-selected, proceeds to parent detection
   - If user has **multiple organizations**: Shows TUI picker
   - User navigates with up/down, selects with enter
   - Selected organization ID stored in `addTUIResult.OrganizationID`

4. **Project Registration**
   - Organization ID passed to `tracking.Service.AddProject()`
   - Included in `RegisterProjectParams` for API call
   - Stored in local config at `{projectPath}/.cops/config.json`

### Daemon OrganizationID Handling

1. **Config Watcher Service**
   - Reads global config (`~/.cops/config.json`) for project list
   - For each project:
     - Reads local config (`{projectPath}/.cops/config.json`)
     - If `OrganizationID` is missing, **skips project** with warning log
     - If OrganizationID exists, creates `WatchTarget` with `ProjectID` and `OrganizationID`

2. **Log Watcher Service**
   - Receives `WatchTarget` list from Config Watcher
   - Builds `projectIDToOrgID` mapping for quick lookup
   - When flushing logs for a project:
     - Looks up organization ID via `projectIDToOrgID[projectID]`
     - Creates `LogBatch` with `ProjectID` and `OrganizationID`
     - Sends to API server

3. **API Server RBAC**
   - API server receives `LogBatch` with `organization_id`
   - Enforces organization-level RBAC on all endpoints
   - Rejects logs from unauthorized organizations

---

## Breaking Changes

### LocalConfig Field Rename: `ID` → `ProjectID`

**Impact**: JSON field changes from `"id"` to `"projectId"`

**Before** (`{projectPath}/.cops/config.json`):
```json
{
  "id": "01JGXYZ..."
}
```

**After**:
```json
{
  "projectId": "01JGXYZ...",
  "organizationId": "01JGABC..."
}
```

**Migration Required**:
- Existing local config files will fail to parse (ID field not found)
- Users must re-register projects via `cops add {directory}`
- Daemon gracefully skips projects without OrganizationID and logs warning

**Files Changed**:
- `cli/internal/service/tracking/outbound/config/config_port.go` (line 12)
- `daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go` (line 9)

**Code References Updated**:
- All `localConfig.ID` changed to `localConfig.ProjectID`
- CLI: `tracking_service.go` (lines 105, 143)
- Daemon: `configwatcher_service.go` (lines 126, 165)

---

## Testing

### Build Verification

**Commands Run**:
```bash
cd idl/protobuf && buf generate  # Generate protobuf code
go build ./cli/... ./daemon/... ./api/...  # Build all modules
```

**Result**: ✅ All modules built successfully

### Manual Testing Scenarios

#### CLI - New User Registration Flow

1. **Unauthenticated User**:
   ```bash
   cops add .
   # Expected: Error "not authenticated. Run 'cops login' first"
   ```

2. **Authenticated User - Single Organization**:
   ```bash
   cops login
   cops add .
   # Expected: Organization auto-selected, no picker shown
   # Expected: Local config created with organizationId
   ```

3. **Authenticated User - Multiple Organizations**:
   ```bash
   cops login
   cops add .
   # Expected: TUI picker shown with organization list
   # Expected: Can navigate with up/down, select with enter
   # Expected: Local config created with selected organizationId
   ```

4. **Local Config Structure**:
   ```bash
   cat .cops/config.json
   # Expected output:
   # {
   #   "projectId": "01JGXYZ...",
   #   "organizationId": "01JGABC..."
   # }
   ```

#### Daemon - OrganizationID Handling

1. **Legacy Project (No OrganizationID)**:
   - Daemon logs: `"skipping project without organization ID (run 'cops add' to re-register)"`
   - Project not watched, no logs sent

2. **Valid Project (With OrganizationID)**:
   - Daemon creates watch target with `ProjectID` and `OrganizationID`
   - LogBatch sent to API includes `organization_id` field
   - API server enforces RBAC based on organization

3. **Worktree Projects**:
   - Each worktree has own local config
   - Daemon skips worktrees without OrganizationID
   - Valid worktrees send logs with correct organization ID

#### API Server - UserService.GetMe

1. **GetMe RPC**:
   - Request: JWT token in Authorization header
   - Response: User data + list of organizations (domain.v1.Organization)
   - Organizations include: `id`, `name`, `slug` (members empty)

---

## Key Files Changed

| Component | File | Change Type |
|-----------|------|-------------|
| **Proto** | `idl/protobuf/user/v1/user.proto` | Modified (removed UserOrganization) |
| **CLI - User Service** | `cli/internal/service/user/user_service.go` | New |
| **CLI - User API Port** | `cli/internal/service/user/outbound/api/user_port.go` | New |
| **CLI - User API Client** | `cli/internal/service/user/outbound/api/connectrpc/user_client.go` | New |
| **CLI - DI Container** | `cli/cmd/internal/container/module_user.go` | New |
| **CLI - TUI** | `cli/internal/service/tracking/inbound/cli/cobra/add_tui.go` | Modified (org selection step) |
| **CLI - TUI Update** | `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go` | Modified (org selection logic) |
| **CLI - TUI View** | `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go` | Modified (org picker view) |
| **CLI - Add Command** | `cli/internal/service/tracking/inbound/cli/cobra/add.go` | Modified (auth check) |
| **CLI - Handler** | `cli/internal/service/tracking/inbound/cli/cobra/handler.go` | Modified (service dependencies) |
| **CLI - LocalConfig** | `cli/internal/service/tracking/outbound/config/config_port.go` | Modified (BREAKING: ID→ProjectID) |
| **CLI - API Port** | `cli/internal/service/tracking/outbound/api/project_port.go` | Modified (OrganizationID param) |
| **CLI - API Client** | `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` | Modified (send OrganizationID) |
| **CLI - Service** | `cli/internal/service/tracking/tracking_service.go` | Modified (OrganizationID flow) |
| **Daemon - LocalConfig** | `daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go` | Modified (BREAKING: ID→ProjectID) |
| **Daemon - Domain** | `daemon/internal/platform/domain/watch.go` | Modified (OrganizationID fields) |
| **Daemon - ConfigWatcher** | `daemon/internal/service/configwatcher/configwatcher_service.go` | Modified (skip without OrgID) |
| **Daemon - LogWatcher** | `daemon/internal/service/logwatcher/log_service.go` | Modified (track OrganizationID) |
| **Daemon - API Client** | `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` | Modified (send OrganizationID) |
| **API - User Handler** | `api/internal/service/user/inbound/grpc/connectrpc/handler.go` | Modified (fix proto type) |

---

## Related Tickets

- Implementation based on artifacts in `.agent/artifacts/20260104-013022/`
- Requirements: `01_clarify.md`
- Plan: `02_plan.md`
- Review: `03_review.md`, `04_user_review_iteration1.md`
- Iteration Plan: `05_user_plan_iteration1.md`
