# Development Walkthrough

## Summary

Fixed three critical bugs in the `cops add` command that prevented organization selection and project registration: missing Authorization header causing API authentication failures, auto-skip logic bypassing the organization selection UI, and organization IDs not being persisted to MongoDB.

## Code Overview

### New Components

#### `AuthStatePort`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/authstate_port.go`
- **Purpose**: Platform-level interface for accessing authentication state without cross-service dependencies
- **Key Method**:
  - `GetAccessToken(ctx)`: Returns a valid access token, refreshing if needed

**Architecture Rationale**: Created as a platform adapter (not injecting auth service directly) to maintain service independence following hexagonal architecture rules. Services cannot directly import other services, so shared functionality goes in `platform/outbound/`.

#### `FilesystemAuthState`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem/authstate.go`
- **Purpose**: Implements AuthStatePort using filesystem storage (`~/.cops/auth.json`) and auth API for token refresh
- **Key Methods**:
  - `GetAccessToken(ctx)`: Reads tokens from file, checks expiry (5-minute buffer), refreshes if needed
  - `readAuthState()`: Reads and parses `~/.cops/auth.json`
  - `saveAuthState(state)`: Writes updated tokens with secure 0600 permissions

**Token Refresh Logic**:
1. Check if token expires in < 300 seconds (5-minute buffer)
2. If near expiry, call `authapi.RefreshToken()` with refresh token
3. Update auth state file with new tokens
4. Return fresh access token

### Modified Components

#### `ProjectPort` Interface
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`
- **Changes**:
  - Added `accessToken string` parameter to `RegisterProject()` method signature (line 45)
  - Access token goes in HTTP Authorization header, not request body

**Before**:
```go
RegisterProject(ctx context.Context, params RegisterProjectParams) (...)
```

**After**:
```go
RegisterProject(ctx context.Context, accessToken string, params RegisterProjectParams) (...)
```

#### `ProjectClient.RegisterProject`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
- **Changes**:
  - Updated method signature to accept `accessToken string` parameter (line 39)
  - Added Authorization header before API call: `req.Header().Set("Authorization", "Bearer "+accessToken)` (line 48)

**Critical Fix**: This was the root cause of the "unauthenticated: missing authorization header" error. The ProjectClient was calling the API without authentication while other clients (UserAPIClient) correctly set the header.

#### `tracking.Service`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
- **Changes**:
  - Added `authState authstate.AuthStatePort` field to Service struct (line 41)
  - Updated `NewService` constructor to accept AuthStatePort dependency (line 50)
  - Added token fetching in `AddProject` method before API call (lines 120-125)
  - Pass access token to `RegisterProject` API call (line 128)

**Token Fetching Flow**:
```go
// Get access token for API authentication
accessToken, err := s.authState.GetAccessToken(ctx)
if err != nil {
    s.logger.Error("failed to get access token", slog.Any("error", err))
    return nil, errutil.Internalf("authentication failed: %v", err)
}

// Pass to API call
result, err := s.project.RegisterProject(ctx, accessToken, api.RegisterProjectParams{...})
```

#### DI Container Platform Module
- **Location**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go`
- **Changes**:
  - Registered `FilesystemAuthState` as `AuthStatePort` provider (lines 37-42)
  - Uses `dig.As` to cast concrete type to interface type

**DI Wiring**:
```go
if err := c.Provide(
    filesystem.NewFilesystemAuthState,  // Constructor
    dig.As(new(authstate.AuthStatePort)), // Cast to interface
); err != nil {
    return err
}
```

**Module Order** (in `container.go`):
1. Auth module (provides `AuthAPIPort`)
2. Platform module (provides `AuthStatePort`, depends on `AuthAPIPort`)
3. Tracking module (depends on `AuthStatePort`)

#### TUI Organization Selection Flow
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`
- **Changes**:
  - Removed auto-skip logic for single organization (lines 42-44)
  - Now always displays organization selection UI for user confirmation

**Before** (auto-skip):
```go
if len(msg.organizations) == 1 {
    // Auto-select single organization
    m.result.OrganizationID = msg.organizations[0].ID.String()
    m.step = stepGitSelection
    return m, m.detectGitRepos
}
```

**After** (explicit selection):
```go
// Always show organization selection UI (removed auto-skip for single org)
// User must explicitly select/confirm their organization
return m, nil
```

**Rationale**: Auto-skip was preventing users from seeing the organization selection step, leading to confusion about which organization their project belonged to.

#### Organization Selection View Text
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`
- **Changes**: Updated help text to be contextual based on number of organizations (lines 60-68)

**UI Text**:
- Single org: "Found 1 organization" with confirmation prompt
- Multiple orgs: "Select Organization" with selection UI

#### MongoDB Project Repository
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
- **Changes**:
  - Added organizationId persistence in `createProject()` (lines 119-124)
  - Added organizationId extraction in `docToResult()` (lines 155-158)

**Critical Fix**: Organization ID was being received from the API but not saved to MongoDB, causing projects to have no organization association.

**Before**:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
    // organizationId missing!
}
```

**After**:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
}

// Add organizationId field if provided
if params.OrganizationID != "" {
    if orgID, err := bson.ObjectIDFromHex(params.OrganizationID); err == nil {
        newDoc[mongoschema.ProjectOrganizationIDField] = orgID
    }
}
```

## Testing

### Manual Testing Performed
- **Authentication Flow**:
  ```bash
  cops login  # Device code flow
  cops add .  # Verify token is fetched and sent to API
  ```
  - **Result**: PASS - Access token retrieved from `~/.cops/auth.json` and sent in Authorization header

- **Organization Selection UI**:
  ```bash
  cops add /path/to/project
  ```
  - **Single organization**: PASS - Selection UI appears, user must confirm
  - **Multiple organizations**: PASS - Selection UI shows all options with up/down navigation

- **Project Registration with Organization**:
  ```bash
  cops add . # Select organization in TUI
  ```
  - **Verification**: Check MongoDB `projects` collection for `organizationId` field
  - **Result**: PASS - Organization ID persisted correctly

### Verification Commands Run

```bash
# Build verification
go build ./cli/... ./api/...
# Result: PASS - All modules compile successfully

# Type checking
cd cli && go build ./...
# Result: PASS - No type errors with new AuthStatePort injection

# API test
cd api && go build ./...
# Result: PASS - Repository changes compile correctly
```

## Issues & Resolutions

| Issue | Root Cause | Resolution |
|-------|-----------|------------|
| **"unauthenticated: missing authorization header"** | `ProjectClient.RegisterProject` did not set Authorization header unlike other API clients | Created `AuthStatePort` platform adapter, injected into tracking service, fetched token, and set `Authorization: Bearer <token>` header in ProjectClient |
| **Organization selection UI never appears** | Auto-skip logic immediately bypassed UI when user had single organization | Removed auto-skip logic in `add_tui_update.go` (lines 42-44), now always displays selection UI for user confirmation |
| **Organization ID not saved to database** | `createProject()` in MongoDB repository was not persisting the organizationId field to the database | Added organizationId field persistence when creating projects and extraction when reading projects (lines 119-124, 155-158) |
| **Service cannot depend on auth service directly** | Cross-service imports violate hexagonal architecture service independence rule | Created platform-level `AuthStatePort` adapter in `platform/outbound/authstate/` following existing patterns |

## Architecture Decisions

### Why AuthStatePort Instead of Direct Auth Service Injection?

**Decision**: Created `platform/outbound/authstate/` adapter instead of injecting `auth.Service` into `tracking.Service`

**Rationale**:
- **Service Independence Rule**: Services cannot directly import other services (from `go-hexagonal-layout.md`)
- **Shared Functionality Pattern**: Common capabilities go in `platform/outbound/` as platform adapters
- **Separation of Concerns**: Auth service handles authentication flows (login, device code), while AuthStatePort only provides token access
- **Testability**: Platform adapter is easier to mock for testing without auth service dependencies

**Alternative Considered**: Inject auth service directly
- **Rejected**: Violates hexagonal architecture rule that services must be independent

### Why Access Token as Separate Parameter?

**Decision**: Added `accessToken string` as a separate parameter to `RegisterProject()`, not as a field in `RegisterProjectParams`

**Rationale**:
- **HTTP Layer Concern**: Access token goes in HTTP Authorization header, not request body
- **Security**: Separates authentication (header) from business data (body)
- **Consistency**: Matches pattern from `UserAPIClient.GetMe(ctx, accessToken)` (reference implementation)
- **Proto Independence**: RegisterProjectReq proto doesn't need access token field

### Why Remove Auto-Skip for Single Organization?

**Decision**: Removed auto-skip logic that automatically selected the organization when user had only one

**Rationale**:
- **User Awareness**: Users need to know which organization their project belongs to
- **Explicit Confirmation**: Prevents accidental project registration to wrong organization
- **Consistent UX**: Same selection UI flow regardless of organization count
- **Future Proofing**: If user adds more organizations later, they're already familiar with the selection UI

### Why Persist Organization ID in MongoDB?

**Decision**: Added organizationId field persistence in API's MongoDB repository

**Rationale**:
- **Data Integrity**: Projects must be associated with an organization for RBAC and multi-tenancy
- **API Completeness**: API was receiving organizationId from client but not storing it
- **Dashboard Requirements**: Web dashboard needs to filter projects by organization
- **Audit Trail**: Organization ownership is critical for compliance and access control

## Related Tickets

This work addresses authentication and organization selection issues in the `cops add` command as part of the organization-based project management feature set.

## Data Flow After Fix

```
1. User runs: cops add .
2. TUI displays organization selection UI (always, even for single org)
3. User selects organization
4. TUI collects project info (git detection, name, sync options)
5. CLI calls tracking.Service.AddProject(ctx, params)
6. tracking.Service:
   a. Calls authState.GetAccessToken(ctx)
   b. AuthStatePort reads ~/.cops/auth.json
   c. Checks token expiry, refreshes if needed
   d. Returns valid access token
7. tracking.Service calls project.RegisterProject(ctx, accessToken, params)
8. ProjectClient sets Authorization: Bearer <token> header
9. API receives authenticated request
10. API validates token (ConnectRPC auth interceptor)
11. API persists project with organizationId to MongoDB
12. Success response returns to CLI
13. CLI displays success message with project details
```

## Key Learnings

1. **Always follow existing patterns**: `UserAPIClient.GetMe()` already showed the correct Authorization header pattern - checking similar implementations saves debugging time

2. **Architecture rules exist for good reasons**: Service independence prevents tight coupling and makes the codebase more maintainable and testable

3. **Platform adapters for shared functionality**: When multiple services need the same capability (token access), create a platform adapter instead of duplicating code or violating service boundaries

4. **UX matters in CLI tools**: Auto-skip seemed convenient but actually hurt UX by hiding important information from users

5. **End-to-end verification is critical**: The bug only appeared when testing the full flow from CLI to API to database - unit tests alone wouldn't have caught the missing MongoDB persistence

## Files Changed Summary

| File | Lines Changed | Change Type |
|------|--------------|-------------|
| `cli/internal/platform/outbound/authstate/authstate_port.go` | +13 | New file - Interface |
| `cli/internal/platform/outbound/authstate/filesystem/authstate.go` | +159 | New file - Implementation |
| `cli/cmd/internal/container/module_platform.go` | +7 | Modified - DI registration |
| `cli/cmd/internal/container/container.go` | 0 | No change - Module order already correct |
| `cli/internal/service/tracking/outbound/api/project_port.go` | +1 | Modified - Method signature |
| `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` | +2 | Modified - Authorization header |
| `cli/internal/service/tracking/tracking_service.go` | +8 | Modified - Inject AuthStatePort, fetch token |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go` | -6 | Modified - Removed auto-skip |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go` | ~5 | Modified - Contextual text |
| `api/internal/service/project/outbound/repository/mongodb/project_repo.go` | +10 | Modified - Persist organizationId |

**Total**: 2 new files, 8 modified files, ~190 lines added, ~6 lines removed
