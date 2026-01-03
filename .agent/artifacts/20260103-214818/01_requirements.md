# Requirements

## Request Summary

Implement real user data display in the dashboard sidebar user button, replacing the current mocked data ("Code Operator"). This requires a full-stack implementation including: (1) Backend gRPC service to fetch user information and organizations from JWT tokens, (2) Zustand state management for user data and selected organization, (3) Frontend UI updates to display real user information with organization switching capability.

## Acceptance Criteria

- [ ] Backend gRPC service `user.v1.UserService` created with `GetMe` RPC endpoint
- [ ] User information fetched from JWT token (userID extracted from token claims)
- [ ] User data stored in MongoDB with optional fields (name, email, avatar URL)
- [ ] Organization data retrieved with user information (ID, name, user role)
- [ ] Zustand store installed and configured for global state management
- [ ] User store created to manage current user data and selected organization
- [ ] Sidebar user button displays real user information (name, email, avatar)
- [ ] Missing user data handles gracefully (show fallback initials, default avatar)
- [ ] Organization switcher UI implemented in user dropdown menu
- [ ] Selected organization persisted to Zustand store
- [ ] Error states handled with retry UI (error message + retry button)
- [ ] Loading states shown while fetching user data
- [ ] User data automatically fetched on app startup after authentication

## Scope

### In Scope

**Backend Implementation:**
- Create `idl/protobuf/user/v1/user.proto` with GetMe RPC definition
- Implement `api/internal/service/user/user_service.go` to handle GetMe requests
- Extract userID from JWT token using existing auth middleware
- Query MongoDB user collection for user information
- Query MongoDB organization collection for user's organizations
- Return user data with list of organizations and current organization

**Frontend Implementation:**
- Install zustand package for state management
- Create `/web/src/shared/store/user-store.ts` for user and organization state
- Create hook `/web/src/shared/hook/use-user.ts` to access user store
- Update `SidebarUser` component to display real user data from Zustand
- Add organization switcher dropdown to user menu
- Implement user data fetching on app initialization (after authentication)
- Handle optional user fields (name, email, avatar) gracefully
- Show loading skeleton while fetching user data
- Show error message with retry button on fetch failure

**UI Updates:**
- Display user name (or fallback to email, or "User" if both missing)
- Display user email in dropdown menu
- Display avatar image URL (or generated initials if missing)
- Add organization list dropdown to user menu
- Highlight currently selected organization
- Show loading state in user button
- Show error state with retry action

### Out of Scope

- User profile editing functionality (name, email, avatar upload)
- Organization creation or management features
- User settings page implementation
- Role-based access control UI
- Invitation or team member management
- Multi-factor authentication
- User profile page with detailed information
- Organization switching affecting data filtering (implement state only, filtering in future work)

## Constraints

- Must use existing authentication middleware to extract userID from JWT
- Must follow existing MongoDB schema patterns (similar to project/session collections)
- Must use ConnectRPC for gRPC implementation (consistent with other services)
- Must use existing TanStack Query for data fetching on frontend
- Must handle optional fields without breaking UI (user documents may be incomplete)
- Avatar should support both image URLs and fallback to initials
- Organization switching should update Zustand state immediately (no backend call needed)

## Additional Context

### Current Architecture
- Backend: Go with ConnectRPC and MongoDB
- Frontend: React with TanStack Router and TanStack Query
- Authentication: JWT tokens stored in localStorage
- API structure: `api/internal/service/{service}` pattern
- Protobuf definitions: `idl/protobuf/{service}/v1/{service}.proto`

### Related Files
- Current user button: `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-user.tsx`
- Auth hook: `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`
- Auth middleware: `/Users/jayce/team-attention/cops/api/internal/platform/middleware/auth.go`
- JWT utilities: `/Users/jayce/team-attention/cops/api/internal/platform/util/jwtutil/jwtutil.go`

### Data Flow
```
App Start (Authenticated)
  → useUser hook calls GetMe RPC
  → Backend extracts userID from JWT
  → Backend queries MongoDB (users + organizations)
  → Response stored in Zustand
  → SidebarUser renders real data

User Clicks Org Switcher
  → Zustand store updated with new selected org
  → UI re-renders with new organization context
  → (Future: Dashboard data filtered by selected org)
```

### MongoDB Collections Expected
- `users`: Store user information (name, email, avatarUrl, createdAt, etc.)
- `organizations`: Store organization data (name, members with roles)
- `user_organizations`: Junction table for user-organization relationships (if needed)

### Zustand Store Structure
```typescript
{
  user: {
    id: string;
    name?: string;
    email?: string;
    avatarUrl?: string;
  } | null;
  organizations: Array<{
    id: string;
    name: string;
    role: 'owner' | 'admin' | 'member';
  }>;
  selectedOrganizationId: string | null;
}
```

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should we create a new backend gRPC service to fetch user information? | Yes, implement full-stack: Backend gRPC service + Frontend Zustand state management |
| What user information should be displayed and stored? | Basic information only: name, email, avatar URL (all optional fields) |
| Should users be able to switch between multiple organizations? | Yes, include organization switching functionality with list of organizations |
| Should I also implement the backend gRPC service? | Yes, full-stack implementation required |
| What should happen if fetching user data fails? | Show error message with retry button to allow user to retry fetching |
| What should be shown while loading user data? | Show loading state/skeleton in user button |
| When should user data be fetched? | On app startup after authentication is confirmed |
| How to handle missing user data fields? | Handle gracefully with fallbacks (show initials for missing avatar, fallback name to email or "User") |
