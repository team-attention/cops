# Implementation Plan: Organization-based Access Control for CLI and Daemon

## Revision History

**Latest Revision**: Updated based on user feedback (Revision 2)
- **Use domain.Organization instead of UserOrganization**: Changed from `userv1.UserOrganization` to `domain.Organization` (shared domain type). Updated `user.proto` to use `domain.v1.Organization` in GetMeRes. Role field not needed for organization selection.
- **Add connectrpcschema helper package**: Created `shared/domain/connectrpcschema/` package to convert protobuf types (`domainv1.Organization`) to domain types (`domain.Organization`)
- **Rename LocalConfig.ID to ProjectID**: Changed field name from `ID` to `ProjectID` in both CLI and Daemon LocalConfig structs for clarity. This is a **BREAKING CHANGE** - the JSON field changes from "id" to "projectId", requiring regeneration of local config files

## Overview

This implementation adds Organization-based access control to the cops CLI and Daemon components. The API server already enforces organization-level RBAC on all endpoints, but the CLI does not currently send organization information when registering projects, and the Daemon does not include organization information when sending logs.

The implementation will:
1. Add a new User service to the CLI for fetching user organizations via `GetMe` API
2. Modify the tracking service to require authentication and organization selection during `cops add`
3. Add an organization picker TUI step to the add flow
4. Store `OrganizationID` in local config (`{project}/.cops/config.json`)
5. Send `OrganizationID` in the `RegisterProject` API call
6. Modify the Daemon to read `OrganizationID` from local config and include it in `SendLogs` API calls
7. Skip projects without `OrganizationID` in the Daemon (with warning log)

---

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | - | - | All required packages (bubbletea, bubbles, connectrpc) are already in the project |

---

## Step 0: Update user.proto to use domain.v1.Organization

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf conventions
- `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`: Existing proto file
- `/Users/jayce/team-attention/cops/idl/protobuf/domain/v1/domain.proto`: Domain proto with Organization

### `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto` (Modify)

**Description**:
Remove the `UserOrganization` message and use `domain.v1.Organization` in `GetMeRes`.

```protobuf
syntax = "proto3";

package user.v1;

import "domain/v1/domain.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/user/v1;userv1";

// Remove the UserOrganization message entirely

// GetMeReq is empty - user ID is extracted from JWT token.
message GetMeReq {}

// GetMeRes contains the authenticated user's data and organizations.
message GetMeRes {
  domain.v1.User user = 1;
  repeated domain.v1.Organization organizations = 2;  // Changed from UserOrganization
}

// UserService handles user-related operations.
service UserService {
  // GetMe returns the authenticated user's information and organizations.
  // Requires valid JWT token in Authorization header.
  rpc GetMe(GetMeReq) returns (GetMeRes);
}
```

**After modification, run:**
```bash
cd idl/protobuf && buf generate
```

---

## Step 0.5: Create connectrpcschema Helper Package

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`: General Go conventions
- `/Users/jayce/team-attention/cops/shared/domain/organization.go`: Domain Organization type

### `/Users/jayce/team-attention/cops/shared/domain/connectrpcschema/organization.go`

**Description**:
Create a helper package to convert protobuf `domainv1.Organization` to domain `domain.Organization`. This keeps conversion logic centralized and reusable.

```go
package connectrpcschema

import (
	"github.com/team-attention/cops/shared/domain"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
)

// OrganizationFromProto converts domainv1.Organization to domain.Organization.
// Members field is not populated as it's not needed for organization selection.
func OrganizationFromProto(pb *domainv1.Organization) *domain.Organization {
	// Implementation outline:
	// 1. If pb is nil, return nil.
	// 2. Return &domain.Organization with:
	//    a. ID: domain.ID(pb.Id)
	//    b. Name: pb.Name
	//    c. Slug: pb.Slug
	//    d. Members: nil (not needed for organization selection)
}

// OrganizationsFromProto converts a slice of protobuf organizations to domain organizations.
func OrganizationsFromProto(pbs []*domainv1.Organization) []*domain.Organization {
	// Implementation outline:
	// 1. Create result slice with length len(pbs).
	// 2. For each pb in pbs:
	//    a. Call OrganizationFromProto(pb).
	//    b. Store in result slice.
	// 3. Return result.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid org | domainv1.Organization with all fields | domain.Organization with ID, Name, Slug | Happy path |
| Nil input | nil | nil | Nil handling |
| Empty slice | [] | [] | Empty slice |
| Multiple orgs | [org1, org2, org3] | [converted1, converted2, converted3] | Batch conversion |

---

## Step 1: Add User Service Port and Adapter (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter fundamentals
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/auth_port.go`: Existing API port pattern
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`: Existing ConnectRPC client pattern

### `/Users/jayce/team-attention/cops/cli/internal/service/user/outbound/api/user_port.go`

**Description**:
Create the User API port interface for fetching user information and organizations.

```go
package api

import (
	"context"

	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
)

// GetMeResult contains the result of GetMe API call.
type GetMeResult struct {
	UserID        string
	Organizations []*domainv1.Organization
}

// UserAPIPort defines the interface for user API operations.
type UserAPIPort interface {
	// GetMe fetches the authenticated user's information and organizations.
	// Requires valid access token.
	GetMe(ctx context.Context, accessToken string) (*GetMeResult, error)
}
```

---

### `/Users/jayce/team-attention/cops/cli/internal/service/user/outbound/api/connectrpc/user_client.go`

**Description**:
Implement the UserAPIPort using ConnectRPC. This client calls the UserService.GetMe RPC with the access token in the Authorization header.

```go
package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	userv1 "github.com/team-attention/cops/shared/gen/grpcstub/user/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect"
)

// UserAPIClient implements UserAPIPort using ConnectRPC.
type UserAPIClient struct {
	logger *slog.Logger
	client userv1connect.UserServiceClient
}

// NewUserAPIClient creates a new ConnectRPC user client.
func NewUserAPIClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *UserAPIClient {
	// Implementation outline:
	// 1. Create logger with "user.api.connectrpc" name.
	// 2. Create UserServiceClient using httpClient and cfg.API.URL.
	// 3. Return initialized UserAPIClient.
}

// GetMe fetches the authenticated user's information and organizations.
func (c *UserAPIClient) GetMe(ctx context.Context, accessToken string) (*api.GetMeResult, error) {
	// Implementation outline:
	// 1. Create GetMeReq (empty request).
	// 2. Create connect.NewRequest with the request.
	// 3. Set Authorization header to "Bearer " + accessToken.
	// 4. Call c.client.GetMe(ctx, req).
	// 5. If error, log and return error.
	// 6. Convert response to GetMeResult:
	//    a. Extract user ID from resp.Msg.User.Id.
	//    b. Use resp.Msg.Organizations directly (now []*domainv1.Organization).
	// 7. Return GetMeResult.
}

var _ api.UserAPIPort = (*UserAPIClient)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid token with orgs | Valid access token | GetMeResult with organizations | Happy path |
| Valid token no orgs | Valid access token, user has no orgs | GetMeResult with empty orgs slice | Empty orgs |
| Invalid token | Expired/invalid token | Error | Auth error handling |
| Network error | Network unreachable | Error | Network error handling |

---

## Step 2: Add User Service (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go`: Existing service pattern

### `/Users/jayce/team-attention/cops/cli/internal/service/user/user_service.go`

**Description**:
Create the User service that fetches user organizations using the auth service for token management.

```go
package user

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/cli/internal/service/auth"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/connectrpcschema"
)

// Service provides user operations.
type Service struct {
	logger    *slog.Logger
	apiClient api.UserAPIPort
	authSvc   *auth.Service
}

// NewService creates a new user service.
func NewService(l *slog.Logger, apiClient api.UserAPIPort, authSvc *auth.Service) *Service {
	// Implementation outline:
	// 1. Return &Service with logger bound to "user.service".
	// 2. Store apiClient and authSvc.
}

// GetMyOrganizations fetches the authenticated user's organizations.
// Returns domain.Organization (not protobuf types) for use in TUI and business logic.
func (s *Service) GetMyOrganizations(ctx context.Context) ([]*domain.Organization, error) {
	// Implementation outline:
	// 1. Call s.authSvc.GetAccessToken(ctx) to get valid access token.
	// 2. If error (not logged in or token refresh failed), return error.
	// 3. Call s.apiClient.GetMe(ctx, accessToken).
	// 4. If error, log and return error.
	// 5. Convert protobuf organizations to domain organizations:
	//    a. Call connectrpcschema.OrganizationsFromProto(result.Organizations).
	// 6. Return converted domain organizations.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Authenticated with orgs | User logged in with orgs | []*domain.Organization | Happy path |
| Authenticated no orgs | User logged in, no orgs | Empty slice | Empty orgs |
| Not authenticated | User not logged in | Error: "not logged in" | Auth check |
| Token refresh fails | Expired refresh token | Error | Token refresh error |
| API call fails | Network error | Error | API error handling |

---

## Step 3: Register User Module in DI Container (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-dig-container.md`: DI container patterns
- `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_auth.go`: Existing module pattern

### `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_user.go`

**Description**:
Create the user module to register UserAPIClient and UserService in the DI container.

```go
package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/user"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api/connectrpc"
)

// newUserModule registers all user-related providers.
func newUserModule(c *dig.Container) error {
	// Implementation outline:
	// 1. Provide connectrpc.NewUserAPIClient with dig.As(new(api.UserAPIPort)).
	// 2. Provide user.NewService.
	// 3. Return nil on success, error on failure.
}
```

---

### `/Users/jayce/team-attention/cops/cli/cmd/internal/container/container.go` (Modify)

**Description**:
Add user module registration to the container initialization.

```go
// In Run() function, add after newAuthModule:
if err := newUserModule(c); err != nil {
    return err
}
```

---

## Step 4: Extend LocalConfig with OrganizationID (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go`: Existing config structure

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go` (Modify)

**Description**:
Rename `ID` to `ProjectID` and add `OrganizationID` field to LocalConfig struct.

**BREAKING CHANGE**: This changes the JSON field name from "id" to "projectId". Existing local config files will need to be regenerated via `cops add`.

```go
// LocalConfig represents the {projectPath}/.cops/config.json structure.
type LocalConfig struct {
	ProjectID      domain.ID `json:"projectId"`
	OrganizationID string    `json:"organizationId,omitempty"`
}
```

**Additional Changes Required**:
All references to `LocalConfig.ID` must be updated to `LocalConfig.ProjectID`:
- In `cli/internal/service/tracking/tracking_service.go`: Update all reads/writes of `localConfig.ID`
- In `cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go`: Update JSON serialization (if any manual handling exists)

---

## Step 5: Extend RegisterProjectParams with OrganizationID (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`: Existing port structure

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go` (Modify)

**Description**:
Add OrganizationID field to RegisterProjectParams struct.

```go
// RegisterProjectParams contains parameters for registering a project with the API server.
type RegisterProjectParams struct {
	// ... existing fields ...

	// OrganizationID is the organization this project belongs to
	OrganizationID string
}
```

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` (Modify)

**Description**:
Send OrganizationID in the RegisterProject API call.

```go
// In RegisterProject method, modify the request creation:
req := connect.NewRequest(&projectv1.RegisterProjectReq{
	ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
	ActualRemoteUrl:     params.ActualRemoteURL,
	ExistingProjectId:   params.ExistingProjectID,
	Name:                params.Name,
	IsGitProject:        params.IsGitProject,
	OrganizationId:      params.OrganizationID, // Add this line
})
```

---

## Step 6: Extend AddProjectParams with OrganizationID (CLI Tracking Service)

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`: Existing service

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (Modify)

**Description**:
Add OrganizationID to AddProjectParams and pass it through to RegisterProject and LocalConfig.

```go
// AddProjectParams contains parameters for AddProject.
type AddProjectParams struct {
	Path           string
	Name           string
	NoGit          bool
	Sync           bool
	OrganizationID string // Add this field
}

// In AddProject method, modify the RegisterProject call:
result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
	ConfiguredRemoteURL: configuredURL,
	ActualRemoteURL:     actualURL,
	ExistingProjectID:   existingProjectID,
	Name:                name,
	IsGitProject:        isGitProject,
	OrganizationID:      params.OrganizationID, // Add this line
})

// In AddProject method, modify the LocalConfig creation:
localConfig := &config.LocalConfig{
	ProjectID:      projectID, // Renamed from ID
	OrganizationID: params.OrganizationID, // Add this line
}

// In AddProject method, update the existing local config read:
// Change: existingProjectID = localConfig.ID.String()
// To:     existingProjectID = localConfig.ProjectID.String()
```

---

## Step 7: Add Organization Selection TUI Step (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`: Existing TUI model
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`: Existing TUI views
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`: Existing TUI updates

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go` (Modify)

**Description**:
Add organization selection step to the TUI flow. Insert the step after stepParentDetection (before stepGitSelection).

```go
// TUI step constants - add new step
const (
	stepParentDetection = iota
	stepOrgSelection    // New step for organization selection
	stepGitSelection
	stepNameInput
	stepSyncSelection
	stepCompleted
)

// Add to addTUIResult struct:
type addTUIResult struct {
	ProjectPath    string
	ProjectName    string
	IsGitProject   bool
	SyncPastLogs   bool
	Cancelled      bool
	OrganizationID string // Add this field
}

// Add to addModel struct:
type addModel struct {
	// ... existing fields ...

	// Organization selection
	organizations   []*domain.Organization // List of user's organizations
	orgCursor       int                    // Cursor for organization selection
	selectedOrgID   string                 // Selected organization ID
	selectedOrgName string                 // Selected organization name (for display)
	userSvc         *user.Service          // Reference to user service for fetching orgs
	authSvc         *auth.Service          // Reference to auth service for auth check

	// ... existing fields ...
}

// Modify newAddModel to accept auth and user services:
func newAddModel(
	dir string,
	noGitFlag bool,
	service *tracking.Service,
	authSvc *auth.Service,
	userSvc *user.Service,
) addModel {
	// Implementation outline:
	// 1. Create text input for name.
	// 2. Initialize model with all fields.
	// 3. Store authSvc and userSvc references.
	// 4. Return model.
}

// Add message type for org fetching:
type orgFetchMsg struct {
	organizations []*domain.Organization
	err           error
}

// Add command to fetch organizations:
func (m addModel) fetchOrganizations() tea.Msg {
	// Implementation outline:
	// 1. Create context with timeout.
	// 2. Call m.userSvc.GetMyOrganizations(ctx).
	// 3. Return orgFetchMsg with result or error.
}

// Modify Init to check auth first:
func (m addModel) Init() tea.Cmd {
	// Implementation outline:
	// 1. Check if authenticated via m.authSvc.IsLoggedIn().
	// 2. If not authenticated, return error message.
	// 3. Otherwise, fetch organizations as first step.
}

// Modify runAddTUI to accept new parameters:
func runAddTUI(
	dir string,
	noGitFlag bool,
	service *tracking.Service,
	authSvc *auth.Service,
	userSvc *user.Service,
) (*addTUIResult, error) {
	// Implementation outline:
	// 1. Check if terminal is interactive.
	// 2. Create model with new parameters.
	// 3. Run tea program.
	// 4. Return result.
}
```

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go` (Modify)

**Description**:
Add organization selection update logic and auth error handling.

```go
// Add message type for auth error:
type authErrorMsg struct {
	err error
}

// Modify Update function to handle new messages:
func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case authErrorMsg:
		// Implementation outline:
		// 1. Set m.err to msg.err.
		// 2. Return m, tea.Quit.

	case orgFetchMsg:
		// Implementation outline:
		// 1. If msg.err != nil, set m.err and quit.
		// 2. Store organizations in m.organizations.
		// 3. If len(organizations) == 0, set error "no organizations found" and quit.
		// 4. If len(organizations) == 1, auto-select it:
		//    a. Set m.selectedOrgID and m.selectedOrgName.
		//    b. Set m.result.OrganizationID.
		//    c. Move to stepParentDetection.
		//    d. Return m, m.detectParentProject.
		// 5. If multiple orgs, stay on stepOrgSelection for user selection.
		// 6. Return m, nil.

	case parentDetectionMsg:
		// ... existing logic, ensure it triggers after org selection ...

	case tea.KeyMsg:
		switch m.step {
		case stepOrgSelection:
			return m.updateOrgSelection(msg)
		// ... existing cases ...
		}
	}
	return m, nil
}

// Add organization selection update handler:
func (m addModel) updateOrgSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Implementation outline:
	// 1. Handle "ctrl+c", "esc": cancel and quit.
	// 2. Handle "up", "k": decrement m.orgCursor (min 0).
	// 3. Handle "down", "j": increment m.orgCursor (max len(m.organizations)-1).
	// 4. Handle "enter":
	//    a. Set m.selectedOrgID = m.organizations[m.orgCursor].ID.
	//    b. Set m.selectedOrgName = m.organizations[m.orgCursor].Name.
	//    c. Set m.result.OrganizationID = m.selectedOrgID.
	//    d. Move to stepParentDetection.
	//    e. Return m, m.detectParentProject.
	// 5. Return m, nil.
}
```

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go` (Modify)

**Description**:
Add organization selection view.

```go
// Modify View function to handle stepOrgSelection:
func (m addModel) View() string {
	// ... existing error handling ...

	switch m.step {
	case stepOrgSelection:
		b.WriteString(m.viewOrgSelection())
	// ... existing cases ...
	}
	return b.String()
}

// Add organization selection view:
func (m addModel) viewOrgSelection() string {
	// Implementation outline:
	// 1. If len(m.organizations) == 0, return "Fetching organizations..."
	// 2. Build title "Select Organization".
	// 3. Add description "Choose which organization to add this project to:".
	// 4. Render each organization with cursor indicator:
	//    a. Show cursor "> " if selected.
	//    b. Show org name only (e.g., "Acme Corp") - role not needed.
	// 5. Add help text "up/down: navigate | enter: select | ctrl+c: cancel".
	// 6. Return rendered string.
}
```

---

## Step 8: Modify Add Command Handler (CLI)

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go`: Existing handler
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`: Existing add command

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go` (Modify)

**Description**:
Add auth and user service dependencies to the handler.

```go
// TrackingCLIHandler handles CLI commands for tracking.
type TrackingCLIHandler struct {
	logger  *slog.Logger
	svc     *tracking.Service
	authSvc *auth.Service    // Add this field
	userSvc *user.Service    // Add this field
}

// NewTrackingCLIHandler creates a new CLI handler.
func NewTrackingCLIHandler(
	l *slog.Logger,
	svc *tracking.Service,
	authSvc *auth.Service,
	userSvc *user.Service,
) *TrackingCLIHandler {
	// Implementation outline:
	// 1. Return handler with all dependencies stored.
}
```

---

### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` (Modify)

**Description**:
Update the add command to check authentication and pass organization to TUI.

```go
// In NewAddCommand, modify the RunE function:
RunE: func(cmd *cobra.Command, args []string) error {
	// Implementation outline:
	// 1. Check authentication first:
	//    if !h.authSvc.IsLoggedIn() {
	//        return fmt.Errorf("not authenticated. Run 'cops login' first")
	//    }
	// 2. Determine directory (from args or current working directory).
	// 3. Call runAddTUI with auth and user services.
	// 4. If cancelled, return nil.
	// 5. Call h.svc.AddProject with OrganizationID from result.
	// 6. Print success message.
	// 7. Return nil.
}
```

---

## Step 9: Extend LocalConfig with OrganizationID (Daemon)

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go`: Existing port

### `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go` (Modify)

**Description**:
Rename `ID` to `ProjectID` and add `OrganizationID` field to LocalConfig struct.

**BREAKING CHANGE**: This changes the JSON field name from "id" to "projectId". Must match CLI LocalConfig structure.

```go
// LocalConfig represents the {projectPath}/.cops/config.json structure.
type LocalConfig struct {
	ProjectID      domain.ID `json:"projectId"`
	OrganizationID string    `json:"organizationId,omitempty"`
}
```

**Additional Changes Required**:
All references to `LocalConfig.ID` must be updated to `LocalConfig.ProjectID`:
- In `daemon/internal/service/configwatcher/configwatcher_service.go`: Update all reads of `localCfg.ID`

---

## Step 10: Extend WatchTarget with OrganizationID (Daemon)

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`: Existing domain

### `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` (Modify)

**Description**:
Add OrganizationID field to WatchTarget and LogBatch structs.

```go
// WatchTarget represents a directory to watch for Claude Code logs.
type WatchTarget struct {
	ProjectPath    string               // Original project path from GlobalConfig
	ClaudeDir      string               // ~/.claude/projects/{encoded-path}
	Type           WatchTargetType      // Type of watch target
	ProjectID      shareddomain.ID      // Project ID from local config
	OrganizationID string               // Organization ID from local config
}

// LogBatch contains raw JSONL lines for API transmission.
type LogBatch struct {
	Lines          []string        // Raw JSONL lines (unparsed)
	ProjectID      shareddomain.ID // Project ID (for aggregation API)
	OrganizationID string          // Organization ID (for aggregation API)
}
```

---

## Step 11: Modify ConfigWatcher Service to Read OrganizationID (Daemon)

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go`: Existing service

### `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` (Modify)

**Description**:
Modify buildWatchTargets to skip projects without OrganizationID and include OrganizationID in WatchTarget.

```go
// Modify loadProjectID to return both ID and OrganizationID:
// Rename to loadLocalConfig and return full LocalConfig
func (s *Service) loadLocalConfig(projectPath string) (*localconfig.LocalConfig, error) {
	// Implementation outline:
	// 1. Call s.localConfigPort.LoadLocalConfig(projectPath).
	// 2. Return result or error.
}

// Modify buildWatchTargets to skip projects without OrganizationID:
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
	// Implementation outline (modify existing logic):
	// For each project:
	// 1. Load local config using loadLocalConfig.
	// 2. If error (no local config), skip with existing warning.
	// 3. NEW: If localCfg.OrganizationID == "", skip with warning:
	//    s.logger.Warn("skipping project without organization ID (run 'cops add' to re-register)",
	//        slog.String("path", project.Path),
	//    )
	//    continue
	// 4. Create WatchTarget with ProjectID (renamed from ID) and OrganizationID from localCfg:
	//    targets = append(targets, domain.WatchTarget{
	//        ProjectPath:    project.Path,
	//        ClaudeDir:      pathutil.GetClaudeProjectDir(project.Path),
	//        Type:           domain.WatchTargetRoot,
	//        ProjectID:      localCfg.ProjectID,  // Changed from localCfg.ID
	//        OrganizationID: localCfg.OrganizationID,
	//    })
	//
	// Same logic for worktrees:
	// 1. Load local config.
	// 2. If no OrganizationID, skip with warning.
	// 3. Use localCfg.ProjectID (not localCfg.ID) in WatchTarget.
	// 4. Include OrganizationID in WatchTarget.
}
```

---

## Step 12: Modify LogWatcher Service to Include OrganizationID (Daemon)

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`: Existing service

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` (Modify)

**Description**:
Add OrganizationID tracking and include it in LogBatch creation.

```go
// Add new field to Service struct to track OrganizationID mappings:
type Service struct {
	// ... existing fields ...
	projectIDToOrgID map[shareddomain.ID]string // ProjectID -> OrganizationID mapping
}

// Modify NewService to initialize new map:
func NewService(...) *Service {
	return &Service{
		// ... existing initialization ...
		projectIDToOrgID: make(map[shareddomain.ID]string),
	}
}

// Modify UpdateTargets to store OrganizationID mappings:
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
	// Implementation outline (add to existing logic):
	// Build new projectIDToOrgID mapping from targets.
	// For each target:
	//   newProjectIDToOrgID[target.ProjectID] = target.OrganizationID
	// After the loop, update s.projectIDToOrgID = newProjectIDToOrgID
}

// Modify flushProjectLines to include OrganizationID in LogBatch:
func (s *Service) flushProjectLines(ctx context.Context, projectID shareddomain.ID, lines []string) error {
	// In the existing logic, when creating domain.LogBatch:
	// 1. Look up OrganizationID: orgID := s.projectIDToOrgID[projectID]
	// 2. Include in LogBatch:
	batch := domain.LogBatch{
		Lines:          currentBatch,
		ProjectID:      projectID,
		OrganizationID: orgID, // Add this
	}
}
```

---

## Step 13: Modify API Client to Send OrganizationID (Daemon)

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`: Existing client

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` (Modify)

**Description**:
Include OrganizationID in the SendLogs API call.

```go
// Modify SendLogs to include OrganizationID:
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			OrganizationId: batch.OrganizationID, // Add this line
			ProjectId:      batch.ProjectID.String(),
			Jsonl:          batch.Lines,
		},
	}
	// ... rest of existing logic ...
}
```

---

## Summary of File Changes

### Protobuf - Modified Files
| File | Change |
| :--- | :----- |
| `idl/protobuf/user/v1/user.proto` | Remove UserOrganization, use domain.v1.Organization in GetMeRes |

### Shared - New Files
| File | Description |
| :--- | :---------- |
| `shared/domain/connectrpcschema/organization.go` | Helper to convert domainv1.Organization to domain.Organization |

### CLI - New Files
| File | Description |
| :--- | :---------- |
| `cli/internal/service/user/outbound/api/user_port.go` | User API port interface (uses domainv1.Organization) |
| `cli/internal/service/user/outbound/api/connectrpc/user_client.go` | ConnectRPC implementation |
| `cli/internal/service/user/user_service.go` | User service (returns domain.Organization) |
| `cli/cmd/internal/container/module_user.go` | User module for DI |

### CLI - Modified Files
| File | Change |
| :--- | :----- |
| `cli/cmd/internal/container/container.go` | Add user module registration |
| `cli/internal/service/tracking/outbound/config/config_port.go` | **BREAKING**: Rename `ID` to `ProjectID`, add `OrganizationID` |
| `cli/internal/service/tracking/outbound/api/project_port.go` | Add OrganizationID to RegisterProjectParams |
| `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` | Send OrganizationID in request |
| `cli/internal/service/tracking/tracking_service.go` | Add OrganizationID to AddProjectParams, update LocalConfig field references |
| `cli/internal/service/tracking/inbound/cli/cobra/handler.go` | Add auth and user service dependencies |
| `cli/internal/service/tracking/inbound/cli/cobra/add.go` | Check auth, pass services to TUI |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui.go` | Add org selection step using domain.Organization |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go` | Handle org selection messages |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go` | Render org selection view |

### Daemon - Modified Files
| File | Change |
| :--- | :----- |
| `daemon/internal/service/configwatcher/outbound/localconfig/localconfig_port.go` | **BREAKING**: Rename `ID` to `ProjectID`, add `OrganizationID` |
| `daemon/internal/platform/domain/watch.go` | Add OrganizationID to WatchTarget and LogBatch |
| `daemon/internal/service/configwatcher/configwatcher_service.go` | Skip projects without OrganizationID, update LocalConfig field references |
| `daemon/internal/service/logwatcher/log_service.go` | Track OrganizationID mapping, include in LogBatch |
| `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` | Send OrganizationID in request |

---

## Execution Order

1. **Step 0**: Update user.proto to use domain.v1.Organization (Protobuf)
   - Run `buf generate` after modification
2. **Step 0.5**: Create connectrpcschema helper package (Shared)
3. **Step 1-2**: Create User service port, adapter, and service (CLI)
4. **Step 3**: Register User module in DI container (CLI)
5. **Step 4-6**: Extend config and params with OrganizationID (CLI)
6. **Step 7-8**: Add organization selection TUI and update handler (CLI)
7. **Step 9-10**: Extend domain models with OrganizationID (Daemon)
8. **Step 11-13**: Modify services and clients to use OrganizationID (Daemon)

Dependencies:
- Step 0 must complete before Step 1 (generates domainv1.Organization in protobuf)
- Step 0.5 must complete before Step 2 (service needs the helper)
- Steps 1-2 must complete before Step 3
- Step 3 must complete before Step 8 (handler needs user service)
- Steps 4-6 can run in parallel
- Steps 9-10 must complete before Steps 11-13
