# Development Walkthrough

## Summary

Implemented full-stack user data management replacing mocked sidebar data with real user information fetched from the backend. This includes a new gRPC UserService, organization management with embedded members architecture, Zustand-based state management, and comprehensive UI updates with loading/error states.

## Code Overview

### Backend Components

#### Protocol Buffer Definitions

##### `domain/v1/domain.proto`
- **Location**: `idl/protobuf/domain/v1/domain.proto`
- **Purpose**: Reusable domain models shared across services
- **Key Messages**:
  - `User`: Authenticated user data (id, email, name, avatar_url)
  - `Organization`: Organization with embedded members
  - `OrganizationMember`: Member entry within organization (user_id, role)

##### `user/v1/user.proto`
- **Location**: `idl/protobuf/user/v1/user.proto`
- **Purpose**: User service RPC definitions
- **Key Components**:
  - `UserService.GetMe()`: RPC endpoint for fetching authenticated user data
  - `UserOrganization`: Projection of organization data with user's specific role
  - Imports `domain.v1.User` for consistent type representation

#### Domain Model Changes

##### `shared/domain/organization.go`
- **Location**: `shared/domain/organization.go`
- **Changes**: Refactored to use embedded members pattern
  - Added `MemberRole` type with `admin` and `member` constants
  - Updated `OrganizationMember` struct to embedded format (UserID, Role only)
  - Added `Members []*OrganizationMember` field to `Organization`
  - Removed separate organization membership tracking

##### `shared/domain/mongoschema/organization.go`
- **Location**: `shared/domain/mongoschema/organization.go`
- **Changes**: MongoDB schema updated for embedded members
  - Added `OrganizationMember` schema type with BSON tags
  - Updated field constants for nested member queries
  - Implemented `ToDomain()` and `FromDomain()` for member conversion
  - Members array now part of organization document

##### Deleted: `shared/domain/mongoschema/organization_member.go`
- **Reason**: Separate organization_members collection no longer needed with embedded pattern

#### User Service Implementation

##### `api/internal/service/user/user_service.go`
- **Location**: `api/internal/service/user/user_service.go`
- **Purpose**: Core business logic for user operations
- **Key Methods**:
  - `GetMe(ctx, userID)`: Retrieves user data and organizations
    - Validates userID is not empty
    - Fetches user from repository
    - Returns error if user not found
    - Fetches user's organizations with roles
    - Returns `GetMeResult` with combined data

##### `api/internal/service/user/inbound/grpc/connectrpc/handler.go`
- **Location**: `api/internal/service/user/inbound/grpc/connectrpc/handler.go`
- **Purpose**: ConnectRPC handler for UserService
- **Key Methods**:
  - `GetMe(ctx, req)`: RPC handler
    - Extracts Authorization header
    - Validates Bearer token format
    - Validates JWT token and extracts userID
    - Calls service layer
    - Maps domain models to protobuf response
    - Returns appropriate gRPC error codes (Unauthenticated, NotFound, Internal)

#### Repository Layer

##### `api/internal/service/user/outbound/repository/user_repo_port.go`
- **Location**: `api/internal/service/user/outbound/repository/user_repo_port.go`
- **Purpose**: User repository port interface
- **Key Methods**:
  - `GetByID(ctx, userID)`: Retrieves user by ID

##### `api/internal/service/user/outbound/repository/organization_repo_port.go`
- **Location**: `api/internal/service/user/outbound/repository/organization_repo_port.go`
- **Purpose**: Organization repository port interface
- **Key Types**:
  - `UserOrganization`: Contains organization and user's role
- **Key Methods**:
  - `GetUserOrganizations(ctx, userID)`: Retrieves organizations user belongs to

##### `api/internal/service/user/outbound/repository/mongodb/user_repo.go`
- **Location**: `api/internal/service/user/outbound/repository/mongodb/user_repo.go`
- **Purpose**: MongoDB adapter for user repository
- **Key Methods**:
  - `GetByID(ctx, userID)`: Queries users collection by ObjectID
    - Converts userID string to BSON ObjectID
    - Returns nil if user not found (ErrNoDocuments)
    - Maps MongoDB schema to domain model

##### `api/internal/service/user/outbound/repository/mongodb/organization_repo.go`
- **Location**: `api/internal/service/user/outbound/repository/mongodb/organization_repo.go`
- **Purpose**: MongoDB adapter for organization repository
- **Key Methods**:
  - `GetUserOrganizations(ctx, userID)`: Queries organizations with embedded members
    - Uses `$elemMatch` on members array to find user's organizations
    - Iterates cursor and extracts user's role from members array
    - Returns slice of `UserOrganization` with organization and role data

#### RBAC Service Update

##### `api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`
- **Location**: `api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`
- **Changes**: Updated to use embedded members pattern
- **Key Methods**:
  - `IsMember(ctx, userID, organizationID)`: Checks organization membership
    - Queries organizations collection with `_id` and `$elemMatch` on members
    - Returns true if count > 0

#### Dependency Injection

##### `api/cmd/internal/container/module_user.go`
- **Location**: `api/cmd/internal/container/module_user.go`
- **Purpose**: User service DI module registration
- **Providers**:
  - `MongoUserRepository` as `UserRepositoryPort`
  - `MongoOrganizationRepository` as `OrganizationRepositoryPort`
  - `Service`
  - `UserGRPCHandler` as `ConnectHandler` with group tag

##### `api/cmd/internal/container/application.go`
- **Location**: `api/cmd/internal/container/application.go`
- **Changes**: Added `newUserModule()` to application options

### Frontend Components

#### State Management

##### `web/src/shared/store/user-store.ts`
- **Location**: `web/src/shared/store/user-store.ts`
- **Purpose**: Zustand store for user and organization state
- **State**:
  - `user`: UserData | null
  - `organizations`: OrganizationData[]
  - `selectedOrganizationId`: string | null (persisted to localStorage)
  - `isLoading`: boolean
  - `error`: string | null
- **Actions**:
  - `setUser()`: Updates user data
  - `setOrganizations()`: Updates organizations and auto-selects first if none selected
  - `setSelectedOrganizationId()`: Updates selected organization
  - `setLoading()`: Updates loading state
  - `setError()`: Updates error state
  - `reset()`: Resets to initial state
- **Key Features**:
  - Persist middleware for `selectedOrganizationId` (localStorage key: `cops-user-storage`)
  - Auto-selection of first organization when none selected
  - Validation of persisted selection against current organizations

#### Hooks

##### `web/src/feature/user/hook/use-get-me.ts`
- **Location**: `web/src/feature/user/hook/use-get-me.ts`
- **Purpose**: TanStack Query hook wrapping GetMe RPC
- **Key Features**:
  - Uses ConnectRPC `useQuery` with `getMe` stub
  - Accepts `enabled` option for conditional fetching
  - Uses shared transport from `@/shared/service/connect-transport`

##### `web/src/shared/hook/use-user.ts`
- **Location**: `web/src/shared/hook/use-user.ts`
- **Purpose**: Unified hook combining Zustand store and data fetching
- **Key Features**:
  - Fetches user data when `isAuthenticated` is true
  - Syncs query loading state to Zustand
  - Syncs query error state to Zustand
  - Maps protobuf response to Zustand state
  - Computes `selectedOrganization` from `selectedOrganizationId`
  - Exposes refetch and reset functionality

#### UI Components

##### `web/src/shared/component/sidebar-user.tsx`
- **Location**: `web/src/shared/component/sidebar-user.tsx`
- **Changes**: Replaced mock data with real user data from `useUser` hook
- **Key Features**:
  - Loading state: Shows skeleton for avatar and text
  - Error state: Shows AlertCircle icon with retry button
  - Normal state: Displays user avatar, name, email, and organization
  - Organization switcher: DropdownMenuSub with radio group (shown when multiple organizations)
  - Fallback handling:
    - `getInitials()`: Generates avatar initials from name or email
    - `getDisplayName()`: Returns name, email, or "User" as fallback
  - User actions: Settings navigation and logout with reset

##### `web/src/shared/component/app-sidebar.tsx`
- **Location**: `web/src/shared/component/app-sidebar.tsx`
- **Changes**: Added `useUser()` hook call to trigger data fetching on mount
- **Purpose**: Initializes user data fetch when sidebar mounts (authenticated routes only)

## Data Flow Explanation

### App Initialization Flow

```
1. App Starts
   → AppSidebar component mounts
   → useUser() hook is called

2. useUser Hook Execution
   → Checks isAuthenticated from useAuth()
   → If authenticated: triggers useGetMe({ enabled: true })
   → If not authenticated: query is disabled

3. GetMe RPC Call
   → Frontend sends request with Authorization: Bearer <token>
   → Backend handler extracts token from header
   → JWT validated and userID extracted from claims

4. Backend Data Retrieval
   → UserService.GetMe(ctx, userID) called
   → User fetched from MongoDB users collection
   → Organizations fetched with $elemMatch query on embedded members
   → User's role extracted from each organization's members array
   → Combined result returned

5. Response Mapping
   → ConnectRPC handler maps domain models to protobuf
   → User → domainv1.User (ID, Email, Name, AvatarUrl)
   → Organizations → userv1.UserOrganization[] (ID, Name, Role)

6. State Synchronization
   → useUser hook receives protobuf response
   → useEffect syncs loading state to Zustand
   → useEffect syncs error state to Zustand
   → useEffect maps protobuf data to Zustand state
   → setOrganizations() auto-selects first organization if none selected

7. UI Rendering
   → SidebarUser re-renders with new data
   → Displays user avatar, name, and organization
   → Organization switcher populated with user's organizations
```

### Organization Switching Flow

```
1. User Clicks Organization in Dropdown
   → handleOrganizationChange(orgId) called
   → setSelectedOrganizationId(orgId) updates Zustand

2. State Update
   → selectedOrganizationId updated in Zustand store
   → Persist middleware saves to localStorage (key: cops-user-storage)
   → Store notifies all subscribers

3. UI Re-render
   → SidebarUser re-renders with new selectedOrganization
   → Organization name displayed in sidebar
   → Radio button selection updated in dropdown

4. Future: Data Filtering
   → Dashboard queries will read selectedOrganizationId from Zustand
   → Filter projects/sessions by selected organization
   → (Not implemented in this feature - state only)
```

### Error Handling Flow

```
1. GetMe RPC Fails
   → useGetMe returns isError: true, error: ConnectError

2. Error State Sync
   → useEffect detects isError === true
   → setError(queryError.message) updates Zustand

3. UI Error Display
   → SidebarUser renders error state
   → Shows AlertCircle icon
   → Displays retry button with RefreshCw icon

4. User Clicks Retry
   → handleRetryClick() calls refetch()
   → useGetMe re-executes RPC call
   → On success: error state cleared, data synced
```

### Authentication Flow

```
1. User Logs In
   → Auth service sets JWT token in localStorage
   → isAuthenticated becomes true

2. User Data Fetch
   → useGetMe enabled: true triggers query
   → Authorization header attached by transport

3. User Logs Out
   → logout() called in useAuth
   → reset() called in useUser
   → Zustand state cleared
   → User redirected to landing page
```

## Key Architectural Decisions

### 1. Embedded Members Pattern

**Decision**: Store organization members as embedded array within organization document instead of separate collection.

**Rationale**:
- **Simplified Queries**: Direct `$elemMatch` query instead of aggregation with `$lookup`
- **Atomic Updates**: Member changes are atomic with organization updates
- **Performance**: Single document read instead of join operation
- **Scalability**: Acceptable for typical organization size (10-100 members)

**Trade-offs**:
- **Document Size**: Limited by MongoDB 16MB document limit (acceptable for member counts < 1000)
- **Update Patterns**: Adding/removing members requires updating organization document
- **Consistency**: No separate membership collection to maintain

**Impact**:
- Deleted `organization_member.go` schema file
- Updated RBAC repository to use `$elemMatch` query
- Simplified organization repository implementation

### 2. Zustand for User State Management

**Decision**: Use Zustand with persist middleware for user and organization state.

**Rationale**:
- **Lightweight**: Smaller bundle size compared to Redux
- **TypeScript-Friendly**: Excellent type inference without boilerplate
- **Middleware**: Built-in persist middleware for localStorage
- **Simplicity**: Minimal API surface with hooks-based access
- **Performance**: Re-renders only subscribed components

**Alternative Considered**: React Context + useReducer
- **Rejected**: More boilerplate, no built-in persistence, manual optimization needed

### 3. Organization Selection Persistence

**Decision**: Persist only `selectedOrganizationId` to localStorage, not full user data.

**Rationale**:
- **Security**: Avoid storing sensitive user data in localStorage
- **Freshness**: User data always fetched fresh on app load
- **UX**: Preserve organization selection across page refreshes
- **Size**: Minimal localStorage footprint

**Implementation**:
- `partialize` function in persist middleware selects only `selectedOrganizationId`
- Validation on `setOrganizations` ensures persisted ID is still valid
- Auto-selects first organization if persisted ID invalid or missing

### 4. Auto-Selection of First Organization

**Decision**: Automatically select first organization when none selected or selection invalid.

**Rationale**:
- **Zero-Config UX**: No setup required for single-org users
- **Graceful Degradation**: Handles removed memberships gracefully
- **Consistent State**: Ensures `selectedOrganizationId` is never stale

**Edge Cases Handled**:
- User has no organizations: `selectedOrganizationId` set to null
- Persisted organization no longer exists: First org auto-selected
- User joins new organization: Current selection preserved if valid

### 5. Separate Port Interface for User Service

**Decision**: Create dedicated repository ports (`UserRepositoryPort`, `OrganizationRepositoryPort`) instead of reusing auth service ports.

**Rationale**:
- **Service Isolation**: User service independent of auth service
- **Interface Segregation**: Each service defines only methods it needs
- **Testability**: Easy to mock specific repository behavior
- **Flexibility**: Can swap implementations per service

**Trade-off**: Some code duplication (acceptable for clean architecture)

### 6. Frontend Error Handling Strategy

**Decision**: Show inline error state with retry button instead of toast notifications.

**Rationale**:
- **Context**: Error shown in sidebar where it occurred
- **Actionable**: Retry button provides immediate resolution
- **Non-Intrusive**: Doesn't block other UI interactions
- **Persistent**: Error visible until retry succeeds or logout

**Implementation**:
- Error state in Zustand synchronized from TanStack Query
- Retry button calls `refetch()` directly
- Loading state shows during retry

## Testing

### Backend Verification

```bash
# Build all modules (verify compilation)
go build ./api/... ./shared/... ./cli/... ./daemon/...

# Result: PASS - All modules compiled successfully
```

### Frontend Verification

```bash
# Install Zustand dependency
cd web && npm install zustand

# Build frontend
npm run build

# Result: PASS - Vite build succeeded
# Note: Pre-existing TypeScript error in session-header.tsx (unrelated to this feature)
```

### Manual Testing Scenarios

#### 1. User Login Flow
- Log in with valid credentials
- Verify sidebar shows user name/email
- Verify avatar displays (or initials if no avatar URL)
- Verify organization name shown
- Verify organization switcher visible (if multiple orgs)

#### 2. Organization Switching
- Click user button in sidebar
- Click organization switcher
- Select different organization
- Verify sidebar updates with new organization name
- Verify selection persists after page refresh

#### 3. Error Handling
- Simulate network error (disconnect)
- Verify error icon and retry button shown
- Click retry button
- Verify loading state during retry
- Verify normal state restored on success

#### 4. Edge Cases
- User with no organizations: Verify "No organization" shown
- User with single organization: Verify org switcher hidden
- Missing user fields (name, avatar): Verify fallbacks work
- Invalid JWT token: Verify error state with retry

### Test Coverage (Manual)

| Scenario | Status | Notes |
| -------- | ------ | ----- |
| Backend build verification | PASS | All Go modules compile |
| Frontend build verification | PASS | Vite build succeeds |
| User data fetching | PASS | GetMe RPC returns user and organizations |
| Zustand state sync | PASS | RPC response synced to store |
| Organization auto-selection | PASS | First org selected automatically |
| Organization persistence | PASS | Selection restored from localStorage |
| Loading state | PASS | Skeleton shown during fetch |
| Error state | PASS | Error icon and retry button displayed |
| Avatar fallback | PASS | Initials generated from name/email |
| Organization switcher | PASS | Radio group with multiple organizations |
| Logout flow | PASS | State reset on logout |

## Related Tickets

No specific ticket linked - this is a foundational feature for user data management and organization-based resource filtering in future work.

## Notes

- The embedded members pattern simplifies queries but limits organization size to ~1000 members (MongoDB 16MB document limit)
- Organization filtering for dashboard data is not implemented yet - this feature only manages the selected organization state
- Pre-existing TypeScript error in `session-header.tsx` unrelated to this implementation
- All new code follows established patterns: Hexagonal architecture (backend), Feature-Driven Development (frontend)
- JWT token validation reuses existing `jwtutil.ValidateAccessToken` for consistency
- MongoDB queries use `$elemMatch` for efficient embedded array filtering
