# Requirements

## Request Summary

Add an Organization Settings section to the existing Settings page (`/settings`). Currently, the Settings page only includes the account deletion feature in the "Danger Zone". The new Organization Settings section should allow users to manage the currently selected organization, including editing organization details (name and slug), managing members, and leaving the organization.

## Acceptance Criteria

### Organization Information Display
- [ ] Display currently selected organization's name and slug
- [ ] Show current user's role in the organization (admin or member)
- [ ] Show list of organization members with their roles

### Edit Organization Details (Admin Only)
- [ ] Provide UI to edit organization name
- [ ] Provide UI to edit organization slug
- [ ] Validate slug format (e.g., alphanumeric, hyphens, lowercase)
- [ ] Save changes via API call
- [ ] Update zustand store (user-store) after successful edit

### Member Management (Admin Only)
- [ ] Display list of current members with full details (name, email, avatar)
- [ ] Ability to change member roles (member ↔ admin)
- [ ] Ability to remove members from organization
- [ ] Prevent removing the last admin
- [ ] Allow admins to remove themselves if other admins exist

### Leave Organization
- [ ] Allow any member to leave the organization
- [ ] Prevent leaving if user is the sole member
- [ ] Show confirmation dialog before leaving
- [ ] When leaving last organization, CASCADE DELETE all related data (projects, sessions, logs)
- [ ] When leaving last organization, redirect to "Create New Organization" page
- [ ] Update zustand store after leaving

### UI/UX Requirements
- [ ] Follow existing Settings page design patterns (Card components, consistent styling)
- [ ] Disable admin-only features for non-admin users
- [ ] Show appropriate error messages for failed operations
- [ ] Loading states for async operations

## Scope

### In Scope
- **Frontend**: Organization settings UI components in Settings page
- **Frontend**: Integration with zustand user-store for organization data
- **Frontend**: API hooks for organization CRUD operations
- **Frontend**: Member management UI (view members with full details, change roles, remove members)
- **Frontend**: Leave organization functionality with cascade delete for last org
- **Frontend**: Edit organization name and slug
- **Backend**: Create new `organization.proto` with OrganizationService
- **Backend**: Implement organization service with repository pattern
- **Backend**: APIs for: UpdateOrganization, GetOrganizationMembers, UpdateMemberRole, RemoveMember, LeaveOrganization

### Out of Scope
- Creating new organizations (separate feature, will be implemented later)
- Deleting entire organizations (may be added later to Danger Zone)
- Invitation system for adding new members (explicitly excluded - too complex for this iteration)
- Organization-level permissions beyond admin/member roles
- Project transfer between organizations
- Organization billing or subscription features

## Constraints

### Technical Constraints
- Must use existing zustand `user-store` for organization state management
- Must follow Feature Driven Development (FDD) architecture
- Generated gRPC stubs must come from `@/gen/grpcstub/` (regenerated from protobuf)
- Must use shadcn/ui components for UI
- Follow existing patterns from `delete-account-dialog.tsx`

### Design Constraints
- Must match existing Settings page styling and layout
- Must use Card components similar to "Danger Zone"
- Must maintain dark theme consistency
- Icons from lucide-react

### Data Model Constraints
- Organization structure defined in `domain.proto`:
  - `id: string`
  - `name: string`
  - `slug: string`
  - `members: OrganizationMember[]` (with `user_id` and `role`)
- User roles: "admin" or "member"
- Organization data managed in `user-store.ts` zustand store

## Additional Context

### Existing Implementation References
- Settings page: `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`
- User store: `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`
- Delete account dialog: `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx`
- Domain protobuf: `/Users/jayce/team-attention/cops/idl/protobuf/domain/v1/domain.proto`

### Current Organization State Management
The `user-store.ts` currently has:
- `organizations: Array<OrganizationData>` - List of user's organizations
- `selectedOrganizationId: string | null` - Currently selected organization
- `setOrganizations()` - Update organizations list
- `setSelectedOrganizationId()` - Change selected organization

### Backend API Investigation Results

**Status**: Backend APIs **DO NOT EXIST** - need to be created.

**Findings**:
- No `organization.proto` exists in `/idl/protobuf/` (only domain.proto with Organization message)
- No OrganizationService exists in `/api/internal/service/`
- Existing organization-related code is limited to:
  - `user.Service.GetMe()` - returns user's organizations (read-only)
  - `rbac.Service.CanAccessOrganization()` - checks membership
  - `organization_repo_port.go` - has basic methods (GetUserOrganizations, RemoveUserFromOrganization, DeleteOrganization)

**Required New APIs** (must be implemented):
1. **UpdateOrganization** - Edit name/slug (needs protobuf + service + handler)
2. **GetOrganizationMembers** - Get members with full user details (needs protobuf + service + handler)
3. **UpdateMemberRole** - Change member role (needs protobuf + service + handler)
4. **RemoveMember** - Remove a member from organization (needs protobuf + service + handler)
5. **LeaveOrganization** - Current user leaves, with cascade delete for last org (needs protobuf + service + handler)

**Implementation Strategy**:
- Create `/idl/protobuf/organization/v1/organization.proto` with OrganizationService
- Create `/api/internal/service/organization/` service following hexagonal architecture
- Extend existing repository methods or create new ones as needed
- Generate gRPC stubs and ConnectRPC handlers

### Member Details API Requirements

**Question**: How to get member details (name, email, avatar)?

**Investigation Result**:
- Domain model: `User` has `Email`, `Name`, `ProfileImageURL` (avatar)
- Organization members only store `UserID` and `Role`
- Need to fetch user details by ID for each member

**Solution**: Create enriched member response in `GetOrganizationMembers` API:
- Returns `OrganizationMemberWithDetails` (includes embedded User data)
- Avoids N+1 queries from frontend
- Backend performs efficient batch user lookup

### Cascade Delete Implementation

**Context**: When a user leaves their last organization, all related data must be deleted.

**Existing Infrastructure**:
- `CascadeDeleteRepositoryPort` already exists in `/api/internal/service/user/outbound/repository/`
- Used by `DeleteAccount` feature
- Implementation: `/api/internal/service/user/outbound/repository/mongodb/cascade_delete_repo.go`

**Strategy**:
- Reuse existing `CascadeDeleteRepositoryPort` for `LeaveOrganization` when user has only one organization
- Delete: Projects, Sessions, Session logs owned by the organization
- After cascade delete completes, delete the organization itself
- Frontend redirects to "Create New Organization" page after successful deletion

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Do the backend APIs already exist for organization CRUD operations? | **NO** - Backend APIs do NOT exist. Must create: `organization.proto`, `OrganizationService`, and all 5 RPC methods listed above. |
| Should we create a new feature directory `feature/organization/` or extend `feature/user/`? | Create new **`feature/organization/`** directory for better separation of concerns. |
| For adding new members, should we implement an invitation system now, or is that out of scope? | **OUT OF SCOPE** - Explicitly exclude invitation system. Too complex for this iteration. |
| What should happen when a user leaves their last organization? | **CASCADE DELETE** all related data (projects, sessions, logs), then **redirect to "Create New Organization" page**. |
| Should organization slug validation happen on frontend, backend, or both? | **Both** - Frontend for UX (immediate feedback), Backend for security (final validation). Rules: lowercase alphanumeric + hyphens, no spaces, 3-63 characters. |
| Can admins remove themselves from the organization if there are other admins? | **YES** - Admins can remove themselves if at least one other admin exists. Prevent removing last admin. |
| Should we show member details (name, email, avatar) or just user IDs? | **Full details** - Show name, email, and avatar. API should return enriched member data with user details embedded. |
