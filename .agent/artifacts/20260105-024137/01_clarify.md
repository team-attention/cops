# Requirements

## Request Summary

The `cops add .` command has organization selection implemented in the TUI, but the organization selection step never appears, and users encounter an authentication error: "unauthenticated: missing authorization header". Investigation reveals TWO CRITICAL BUGS that prevent the feature from working. This task involves fixing the missing authorization header in the ProjectClient API call and ensuring the TUI properly transitions to the organization selection step.

## ROOT CAUSE ANALYSIS

### Error Message
```
Error: internal: cannot register project: API unreachable and no existing local ID: unauthenticated: missing authorization header
```

### Bug 1: Missing Authorization Header in ProjectClient (CRITICAL)

**File**: `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`

**Problem**: The `RegisterProject` method does NOT set the Authorization header, while other API clients do.

**Current Code** (Lines 39-49):
```go
func (c *ProjectClient) RegisterProject(ctx context.Context, params api.RegisterProjectParams) (*api.RegisterProjectResult, error) {
    req := connect.NewRequest(&projectv1.RegisterProjectReq{
        ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
        ActualRemoteUrl:     params.ActualRemoteURL,
        ExistingProjectId:   params.ExistingProjectID,
        Name:                params.Name,
        IsGitProject:        params.IsGitProject,
        OrganizationId:      params.OrganizationID,
    })
    // BUG: Missing authorization header!
    resp, err := c.client.RegisterProject(ctx, req)
    // ...
}
```

**Comparison with Working Code** (`user_client.go` lines 46-50):
```go
func (c *UserAPIClient) GetMe(ctx context.Context, accessToken string) (*api.GetMeResult, error) {
    req := connect.NewRequest(&userv1.GetMeReq{})
    req.Header().Set("Authorization", "Bearer "+accessToken) // ✅ CORRECT
    resp, err := c.client.GetMe(ctx, req)
    // ...
}
```

**Impact**:
- API server rejects the request with "missing authorization header"
- User never sees organization selection because the error occurs AFTER org selection
- Misleading error message suggests API is unreachable when it's actually an auth issue

**Solution Required**:
1. Add `accessToken` parameter to `RegisterProjectParams` struct
2. Update `ProjectPort` interface signature to accept access token
3. Set Authorization header in `RegisterProject` method
4. Update `tracking.Service.AddProject` to fetch and pass access token

### Bug 2: TUI Flow Transition Issue (POTENTIAL)

**File**: `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`

**Observation**: User reports organization selection step NEVER appears.

**Current TUI Flow**:
1. `stepParentDetection` - Checks for parent projects
2. `stepOrgSelection` - Fetches and displays organizations
3. `stepGitSelection` - Detects git repositories
4. `stepNameInput` - Project name input
5. `stepSyncSelection` - Sync past logs selection

**Hypothesis**: The parent detection step may be blocking transition to organization selection.

**Code Analysis** (Lines 15-27):
```go
case parentDetectionMsg:
    if msg.err != nil {
        m.err = msg.err
        return m, tea.Quit  // Error terminates TUI
    }
    m.parentProject = msg.parent
    if msg.parent == nil {
        // No parent found, proceed to org selection
        m.step = stepOrgSelection
        return m, m.fetchOrganizations
    }
    // Parent found, stay on stepParentDetection for user confirmation
    return m, nil
```

**Problem**: If parent project is found, TUI stays on `stepParentDetection` waiting for user input. However, the error occurs BEFORE user can proceed, suggesting the parent detection might be failing or the API call happens too early.

**Additional Investigation Needed**:
- Check if `fetchOrganizations` is being called correctly
- Verify error handling doesn't quit TUI prematurely
- Ensure organization API error doesn't happen before TUI displays

### Technical Stack
- **CLI Framework**: Cobra + Bubble Tea TUI
- **Auth Service**: `cli/internal/service/auth/auth_service.go`
  - Method: `GetAccessToken(ctx) (string, error)`
  - Handles token refresh automatically
  - Reads from `~/.cops/auth.json`
- **User Service**: `cli/internal/service/user/user_service.go`
  - Method: `GetMyOrganizations(ctx) ([]*domain.Organization, error)`
  - Uses auth service to get access token
  - Sets Authorization header correctly
- **Tracking Service**: `cli/internal/service/tracking/tracking_service.go`
  - Method: `AddProject(ctx, params) (*domain.Project, error)`
  - Calls `project.RegisterProject()` WITHOUT access token
- **Project API Client**: `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
  - Missing authorization header in `RegisterProject` method

## Acceptance Criteria

### Bug 1: Authorization Header Fix
- [ ] Add `AccessToken` field to `RegisterProjectParams` struct in `project_port.go`
- [ ] Update `ProjectPort.RegisterProject` interface to accept access token via params
- [ ] Update `ProjectClient.RegisterProject` implementation to set Authorization header
- [ ] Inject `auth.Service` into `tracking.Service` via constructor
- [ ] Update `tracking.Service.AddProject` to call `authSvc.GetAccessToken(ctx)`
- [ ] Pass access token to `project.RegisterProject()` call
- [ ] Verify API call succeeds with proper authentication

### Bug 2: TUI Flow Verification
- [ ] Verify organization selection UI appears when no parent project exists
- [ ] Verify organization selection UI appears after parent project confirmation
- [ ] Verify single organization auto-selection works correctly
- [ ] Verify multiple organization selection UI displays all options
- [ ] Ensure proper error handling when GetMyOrganizations API call fails
- [ ] Ensure proper error handling when user has no organizations
- [ ] Add debug logging to track TUI step transitions

### Integration Testing
- [ ] Test complete flow: login -> add project (no parent) -> verify org selection appears
- [ ] Test complete flow: login -> add project (with parent) -> confirm -> verify org selection appears
- [ ] Test with single organization: verify auto-selection and project creation
- [ ] Test with multiple organizations: verify selection UI and project creation
- [ ] Verify selected organization ID is correctly saved to local project config
- [ ] Verify selected organization ID is correctly sent to API RegisterProject endpoint
- [ ] Verify project registration succeeds end-to-end

## Scope

### In Scope
- **FIX BUG 1**: Add authorization header to ProjectClient.RegisterProject
- **FIX BUG 2**: Verify and fix TUI flow transitions to organization selection
- Add access token parameter to ProjectPort interface and implementation
- Inject auth service into tracking service
- Update tracking service to fetch and pass access token
- Verify organization selection UI appears correctly
- Add debug logging for TUI step transitions
- Test complete end-to-end flow with authentication
- Error handling improvements for authentication failures

### Out of Scope
- Creating new organization management features
- Modifying organization API endpoints or proto definitions (already support auth)
- Adding organization creation flow to CLI
- Implementing organization switching after project creation
- Changing TUI step order or adding new steps
- Dashboard or web UI changes
- Daemon or collector changes
- Implementing new authentication mechanisms (device flow already works)

## Constraints

- Must maintain backward compatibility with existing local config format
- Must use existing TUI framework (Bubble Tea)
- Must follow hexagonal architecture patterns
- Must use existing user service and API client
- Should not require API changes (organization endpoints already exist)
- Must handle offline mode gracefully (when API is unreachable)

## Additional Context

### Files Requiring Changes

#### Critical - Bug 1 Fix (Authorization Header)
1. **`cli/internal/service/tracking/outbound/api/project_port.go`**
   - Add `AccessToken string` field to `RegisterProjectParams` struct

2. **`cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`**
   - Update `RegisterProject` method to set Authorization header
   - Pattern: `req.Header().Set("Authorization", "Bearer "+params.AccessToken)`

3. **`cli/internal/service/tracking/tracking_service.go`**
   - Add `authSvc *auth.Service` field to Service struct
   - Update `NewService` constructor to accept auth service
   - Update `AddProject` method to fetch access token
   - Pass access token in `RegisterProjectParams`

4. **`cli/cmd/internal/container/module_tracking.go`** (or wherever DI is configured)
   - Inject `auth.Service` into `tracking.NewService` constructor

#### Investigation - Bug 2 (TUI Flow)
1. **`cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`**
   - Add debug logging to track step transitions
   - Verify organization fetch is triggered correctly
   - Check error handling doesn't prematurely quit TUI

### File Locations Reference
- **Auth Service**: `cli/internal/service/auth/auth_service.go`
- **User Service**: `cli/internal/service/user/user_service.go`
- **User API Client**: `cli/internal/service/user/outbound/api/connectrpc/user_client.go` (REFERENCE - shows correct auth pattern)
- **TUI Model**: `cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`
- **TUI View**: `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`
- **User Proto**: `idl/protobuf/user/v1/user.proto`
- **Project Proto**: `idl/protobuf/project/v1/project.proto`

### Code Patterns to Follow

#### Authorization Header Pattern (from user_client.go)
```go
func (c *UserAPIClient) GetMe(ctx context.Context, accessToken string) (*api.GetMeResult, error) {
    req := connect.NewRequest(&userv1.GetMeReq{})
    req.Header().Set("Authorization", "Bearer "+accessToken) // ✅ Follow this pattern
    resp, err := c.client.GetMe(ctx, req)
    // ...
}
```

#### Token Fetching Pattern (from user_service.go)
```go
func (s *Service) GetMyOrganizations(ctx context.Context) ([]*domain.Organization, error) {
    accessToken, err := s.authSvc.GetAccessToken(ctx) // ✅ Follow this pattern
    if err != nil {
        s.logger.Error("failed to get access token", slog.Any("error", err))
        return nil, err
    }

    result, err := s.apiClient.GetMe(ctx, accessToken)
    // ...
}
```

### Architecture Notes
- Hexagonal pattern: inbound/outbound adapters
- DI Container: Uses `go.uber.org/fx` for dependency injection
- TUI Pattern: Bubble Tea with multi-step flow and async commands
- Auth: JWT tokens stored in `~/.cops/auth.json`, refreshed automatically

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Is organization selection already implemented? | Yes, organization selection is fully implemented in the TUI flow at stepOrgSelection. The UI code exists and is correct. |
| What specific error is occurring? | "internal: cannot register project: API unreachable and no existing local ID: unauthenticated: missing authorization header" - The ProjectClient is missing the Authorization header. |
| Why doesn't the organization selection UI appear? | The error occurs AFTER organization selection (during project registration), so the TUI never reaches the org selection step. Once auth is fixed, org selection should appear. |
| Does the API support organization selection? | Yes, both UserService.GetMe returns organizations (with auth) and ProjectService.RegisterProject accepts organization_id (but currently fails due to missing auth). |
| Where is the organization ID stored? | Organization ID is stored in local config (.cops/config.json) and sent to API server during project registration. |
| How does single organization selection work? | When user has only one organization, it's automatically selected without showing the selection UI (add_tui_update.go lines 39-46). This code is correct. |
| What happens if user has no organizations? | Current code shows error "no organizations found. Please create an organization first" and quits TUI (add_tui_update.go lines 35-38). This behavior is correct. |
| Why does the error say "API unreachable"? | Misleading error message. The API is reachable but returns 401 Unauthenticated. The tracking service interprets any API error as "unreachable" and falls back to local ID, which doesn't exist. |
| How do other API clients handle authentication? | UserAPIClient.GetMe correctly accepts accessToken parameter and sets Authorization header. ProjectClient should follow the same pattern. |
| Does the tracking service have access to auth service? | Currently NO - auth service is not injected into tracking service. This needs to be added via DI container. |

## Implementation Summary

### The Real Problem

The organization selection UI is **already implemented and correct**. The user never sees it because the command fails with an authentication error BEFORE reaching that step. The error occurs at a later stage (project registration API call) but prevents the TUI from displaying earlier steps.

### Root Cause

The `ProjectClient.RegisterProject` method is missing the Authorization header, causing the API to reject the request with 401 Unauthenticated. This happens because:

1. `tracking.Service` doesn't have access to `auth.Service` (not injected)
2. `ProjectPort` interface doesn't accept access token parameter
3. `ProjectClient.RegisterProject` doesn't set Authorization header
4. Error propagates back before TUI can complete, giving misleading "API unreachable" message

### The Fix (4 Steps)

1. **Add auth service to tracking service** - Inject via DI container
2. **Update ProjectPort interface** - Add AccessToken to RegisterProjectParams
3. **Fetch token in AddProject** - Call authSvc.GetAccessToken(ctx)
4. **Set Authorization header** - Follow UserAPIClient pattern

### Why Organization Selection Doesn't Appear

The TUI flow is:
1. Parent detection ✅ (works)
2. Organization selection ❌ (never reached because error occurs later)
3. Git selection ❌ (never reached)
4. Name input ❌ (never reached)
5. Sync selection ❌ (never reached)
6. **Project registration API call** ❌ (fails with auth error HERE)

The authentication error at step 6 prevents steps 2-5 from completing, even though those steps would work fine if they could execute.

### Expected Behavior After Fix

1. Parent detection ✅
2. Organization selection ✅ (will appear and work correctly)
3. Git selection ✅
4. Name input ✅
5. Sync selection ✅
6. Project registration API call ✅ (will succeed with auth header)
7. Success message displayed ✅
