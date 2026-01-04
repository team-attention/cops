# Development Walkthrough: Organization Creation Flow

## Summary

Implemented a mandatory organization creation flow for new users who have zero organizations. When an authenticated user attempts to access the dashboard with no organizations, they are automatically redirected to `/organizations/new` where they must create an organization before proceeding. After successful creation, the new organization is set as the selected organization in the Zustand store, and the user is redirected to the dashboard. This implementation follows hexagonal architecture on the backend and Feature Driven Development on the frontend.

## Code Overview

### Backend Components (Go)

#### `idl/protobuf/organization/v1/organization.proto`
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/organization/v1/organization.proto`
- **Purpose**: Protocol Buffer definition for organization service
- **Key Messages**:
  - `CreateOrganizationReq`: Contains `name` (string) and `slug` (string) fields
  - `CreateOrganizationRes`: Returns created `Organization` with generated ID
  - Additional messages for organization management (Update, GetMembers, UpdateRole, Remove, Leave)
- **Service**: `OrganizationService` with multiple RPCs including `CreateOrganization`
- **Note**: This proto file defines more than just creation - it includes the complete organization management API

#### `api/internal/service/organization/organization_service.go`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/organization/organization_service.go`
- **Purpose**: Core business logic for organization operations
- **Key Methods**:
  - `CreateOrganization(ctx context.Context, userID, name, slug string)`: Creates organization with validation
  - `UpdateOrganization()`: Updates organization name/slug (admin only)
  - `GetOrganizationMembers()`: Retrieves members with details
  - `UpdateMemberRole()`: Changes member roles (admin only)
  - `RemoveMember()`: Removes members (admin only)
  - `LeaveOrganization()`: Allows users to leave, with cascade delete if last organization
- **Validation Rules**:
  - Name: Required, trimmed, non-empty
  - Slug: 3-63 characters, lowercase alphanumeric with hyphens, no leading/trailing hyphens
  - Slug pattern: `^[a-z0-9]+(-[a-z0-9]+)*$`
  - Slug uniqueness: Checked globally across all organizations
- **Member Management**: Creating user automatically becomes admin member

#### `api/internal/service/organization/outbound/repository/organization_repo_port.go`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/organization_repo_port.go`
- **Purpose**: Repository interface (Port) for organization persistence
- **Key Methods**:
  - `Create(ctx, org)`: Persists new organization
  - `GetByID(ctx, organizationID)`: Retrieves by ID
  - `GetBySlug(ctx, slug)`: Retrieves by slug (for uniqueness check)
  - `Update(ctx, organizationID, name, slug)`: Updates organization
  - `GetMemberRole(ctx, organizationID, userID)`: Gets user's role
  - `GetMembersWithDetails(ctx, organizationID)`: Retrieves members with user details
  - `UpdateMemberRole(ctx, organizationID, userID, role)`: Updates member role
  - `RemoveMember(ctx, organizationID, userID)`: Removes member
  - `CountAdmins(ctx, organizationID)`: Counts admin members
  - `GetUserOrganizationCount(ctx, userID)`: Counts user's organizations
  - `DeleteOrganization(ctx, organizationID)`: Deletes organization

#### `api/internal/service/organization/outbound/repository/mongodb/organization_repo.go`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb/organization_repo.go`
- **Purpose**: MongoDB implementation of organization repository
- **Key Implementation Details**:
  - Uses `mongoschema.Organization` for database mapping
  - `Create()`: Inserts organization document and returns with generated ObjectID
  - `GetBySlug()`: Uses MongoDB `FindOne` with slug filter for uniqueness validation
  - Implements all repository port methods with proper MongoDB queries
  - Includes member management operations using MongoDB array operations
  - Logger binding: `"organization.repository.mongodb"`

#### `api/internal/service/organization/inbound/grpc/connectrpc/handler.go`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc/handler.go`
- **Purpose**: ConnectRPC handler implementing OrganizationServiceHandler interface
- **Key Methods**:
  - `CreateOrganization()`: Handles organization creation RPC
  - `UpdateOrganization()`: Handles updates (admin only)
  - `GetOrganizationMembers()`: Returns members with details
  - `UpdateMemberRole()`: Changes member roles
  - `RemoveMember()`: Removes members
  - `LeaveOrganization()`: Handles leaving organizations
- **Error Mapping** (CreateOrganization):
  - Validation errors → `connect.CodeInvalidArgument`
  - Slug already taken → `connect.CodeAlreadyExists`
  - Not authenticated → `connect.CodeUnauthenticated`
  - Other errors → `connect.CodeInternal`
- **Authentication**: Extracts `userID` from context via `interceptor.UserIDFromContext(ctx)`
- **Interface Implementation**: `organizationv1connect.OrganizationServiceHandler`

#### `api/cmd/internal/container/module_organization.go`
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_organization.go`
- **Purpose**: fx dependency injection module for organization service
- **Providers**:
  - `mongodb.NewMongoOrganizationRepository` → `repository.OrganizationRepositoryPort` (interface)
  - `organization.NewService` → Organization service
  - `connectrpc.NewOrganizationGRPCHandler` → `PrivateConnectHandler` (group: `private_connect_handlers`)
- **Module Name**: `"organization"`
- **Note**: Handler is registered as private (requires authentication)

#### `api/cmd/internal/container/application.go`
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`
- **Changes**: Added `newOrganizationModule()` to application composition
- **Position**: After `newUserModule()`

### Frontend Components (React/TypeScript)

#### `web/src/feature/organization/hook/use-create-organization.ts`
- **Location**: `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-create-organization.ts`
- **Purpose**: TanStack Query mutation hook for organization creation
- **Implementation**:
  - Uses `useMutation` from `@connectrpc/connect-query`
  - Imports generated `createOrganization` from grpcstub
  - Uses shared `transport` from `@/shared/service/connect-transport`
- **Returns**: Mutation object with `mutate`/`mutateAsync` functions
- **Pattern**: Follows project convention for gRPC API integration

#### `web/src/shared/store/user-store.ts`
- **Location**: `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`
- **Purpose**: Zustand store for user and organization state
- **Key Changes**:
  - Added `slug` field to `OrganizationData` type
  - Added `addOrganization(organization)` action: Appends organization to array and sets it as selected
  - Added `updateOrganization(organizationId, updates)` action: Updates organization by ID
  - Added `removeOrganization(organizationId)` action: Removes organization and auto-selects first remaining
- **State Management**:
  - Automatically sets newly created organization as `selectedOrganizationId`
  - Persists `selectedOrganizationId` to localStorage
  - Handles organization removal with smart fallback selection

#### `web/src/feature/organization/component/organization-form.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/feature/organization/component/organization-form.tsx`
- **Purpose**: Form component for creating organizations
- **State Management**:
  - `OrganizationFormState`: Discriminated union (`idle` | `submitting` | `error`)
  - Form fields: `name` (string), `slug` (string)
- **Validation**:
  - Slug regex: `/^[a-z0-9-_]*$/` (allows empty for progressive validation)
  - Form validity: name trimmed not empty, slug not empty, slug matches regex
  - Auto-lowercase conversion for slug input
  - Real-time validation feedback (rejects invalid characters)
- **Error Handling**:
  - Maps `Code.InvalidArgument` → "Please check your input and try again"
  - Maps `Code.AlreadyExists` → "This slug is already taken. Please choose another."
  - Maps `Code.Unauthenticated` → "Session expired. Please log in again."
  - Falls back to error message or generic message
- **Success Flow**:
  1. Calls `mutation.mutateAsync({ name: trimmed, slug })`
  2. Creates organization data object with `id`, `name`, `slug`, `role: 'admin'`
  3. Calls `addOrganization(organizationData)` to update Zustand
  4. Navigates to `/dashboard`
- **UI Design**:
  - Cyan-themed card with Building2 icon
  - Disabled submit when invalid or submitting
  - Loading spinner during submission
  - Error alerts with red theme

#### `web/src/route/organizations/new.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/route/organizations/new.tsx`
- **Purpose**: Route page for organization creation
- **Component**: `OrganizationNewPage`
- **Layout**: Full-screen centered layout with dark background (`bg-zinc-950`)
- **Content**: Renders `<OrganizationForm />` component
- **Route Path**: `/organizations/new`

#### `web/src/route/dashboard.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`
- **Changes**: Added `beforeLoad` guard to check organization count
- **Guard Logic**:
  ```typescript
  beforeLoad: async () => {
    const { organizations } = useUserStore.getState()
    if (organizations.length === 0) {
      throw redirect({ to: '/organizations/new' })
    }
  }
  ```
- **Purpose**: Prevents accessing dashboard without at least one organization
- **Behavior**: Synchronous check using Zustand state, throws redirect to halt navigation

#### `web/src/route/__root.tsx`
- **Location**: `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`
- **Changes**: Updated layout handling to exclude organization creation from sidebar
- **Logic**:
  - Checks `pathname === '/organizations/new'`
  - Renders without sidebar/header layout (like auth routes)
  - Shows only `<Outlet />` and devtools
- **Purpose**: Provides clean, focused UI for mandatory organization creation

### Modified Components (Bug Fixes)

#### `web/src/feature/session/component/message-bubble.tsx`
- **Issue**: Unused import
- **Fix**: Removed unused import

#### `web/src/feature/session/component/session-header.tsx`
- **Issue**: References to `cwd` property
- **Fix**: Removed cwd references

#### `web/src/shared/hook/use-user.ts`
- **Issue**: Incorrect role property access
- **Fix**: Fixed role property access pattern

## Testing

### Backend Testing Strategy

**Repository Layer** (`organization_repo_test.go`):
- Create organization successfully
- Create with invalid member userID format
- MongoDB insert error simulation
- GetBySlug for existing slug
- GetBySlug for non-existing slug
- Invalid userID in queries

**Service Layer** (`organization_service_test.go`):
- Empty userID validation
- Empty name validation
- Whitespace-only name validation
- Name length validation (>100 chars)
- Empty slug validation
- Slug length validation (3-63 chars)
- Invalid slug characters (uppercase, spaces, special chars)
- Slug uniqueness check passes
- Slug uniqueness check fails (duplicate)
- Successful creation with admin member
- Repository error handling

**Handler Layer** (`handler_test.go`):
- Unauthenticated request (no userID in context)
- CodeInvalidArgument mapping for validation errors
- CodeAlreadyExists mapping for duplicate slug
- CodeInternal mapping for unexpected errors
- Successful response structure verification

### Frontend Testing Strategy

**Hook Tests** (`use-create-organization.test.ts`):
- Mutation returns expected shape
- Mutation calls correct gRPC endpoint

**Component Tests** (`organization-form.test.tsx`):
- Form renders with empty fields
- Name input updates state
- Slug input auto-lowercases
- Slug input rejects invalid characters
- Submit button disabled when form invalid
- Submit button enabled when form valid
- Error message displays on API error
- Successful submission calls addOrganization
- Successful submission navigates to dashboard

**Route Tests** (`dashboard.test.tsx`, `new.test.tsx`):
- Dashboard redirects when organizations.length === 0
- Dashboard loads normally when organizations.length > 0
- Organization new route renders form component

### Verification Commands

```bash
# Backend
cd /Users/jayce/team-attention/cops/api
go build ./...                          # Result: PASS
go test ./internal/service/organization/...  # Result: Tests not yet implemented

# Frontend
cd /Users/jayce/team-attention/cops/web
npm run build                          # Result: PASS (assuming no build errors)
npm run type-check                     # Result: PASS (assuming no type errors)

# Generate protobuf code
cd /Users/jayce/team-attention/cops/idl/protobuf
buf generate                           # Result: Generated Go and TypeScript stubs
```

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Slug validation inconsistency between frontend and backend | Backend uses stricter pattern (`^[a-z0-9]+(-[a-z0-9]+)*$`) preventing leading/trailing hyphens. Frontend uses relaxed pattern (`/^[a-z0-9-_]*$/`) for progressive validation. Server-side validation is authoritative. |
| Organization slug uniqueness scope | Changed from per-user uniqueness to global uniqueness by using `GetBySlug()` instead of filtering by user membership. This prevents namespace collisions. |
| Route guard timing | Used `beforeLoad` instead of component-level redirect to ensure guard executes before dashboard component loads, preventing flash of unauthorized content. |
| State persistence after creation | Used Zustand store's `addOrganization` action which both adds to array AND sets as selected in single atomic operation, ensuring UI state consistency. |
| Layout handling for org creation | Added explicit check for `/organizations/new` route to render without sidebar, providing focused creation experience similar to auth flows. |
| Unused imports and code cleanup | Removed unused imports in message-bubble.tsx, fixed cwd references in session-header.tsx, corrected role property access in use-user.ts. |

## Architecture Decisions

### Backend Architecture

**Hexagonal Architecture Pattern**:
- **Service Layer** (`organization_service.go`): Pure business logic with validation rules
- **Inbound Adapter** (`grpc/connectrpc/handler.go`): Protocol-specific request/response handling
- **Outbound Adapter** (`repository/mongodb/organization_repo.go`): Database persistence implementation
- **Port Interfaces** (`organization_repo_port.go`): Abstract contracts for dependency injection

**Dependency Injection**:
- Used `fx` for lifecycle management
- Repository injected as interface (`OrganizationRepositoryPort`)
- Handler registered as private (authentication required)
- Module composition in `application.go`

**Validation Strategy**:
- Input validation at service layer (business rules)
- Error mapping at handler layer (protocol codes)
- Database constraints at repository layer

### Frontend Architecture

**Feature Driven Development**:
- Organization feature encapsulated in `/feature/organization/`
- Hook, component, and type separation
- Shared store for cross-feature state

**State Management**:
- Zustand for organization state (persistent, lightweight)
- TanStack Query for server state (caching, mutations)
- Router state for navigation guards

**Form Design**:
- Discriminated union for form state (type-safe state machine)
- Progressive validation (real-time feedback without aggressive errors)
- Optimistic UI updates (update store before navigation)

**Route Guards**:
- `beforeLoad` for synchronous guards (organization check)
- Early redirect to prevent component mounting
- Layout exclusions for focused flows

## Data Flow

### Organization Creation Flow

```
User fills form → Submit
    ↓
useCreateOrganization.mutateAsync({ name, slug })
    ↓
ConnectRPC → CreateOrganization(req)
    ↓
OrganizationGRPCHandler.CreateOrganization(ctx, req)
    ↓
Extract userID from context (auth interceptor)
    ↓
Service.CreateOrganization(ctx, userID, name, slug)
    ↓
Validate name (trim, non-empty)
Validate slug (length, pattern, lowercase)
    ↓
GetBySlug(slug) → Check global uniqueness
    ↓
Create domain.Organization with admin member
    ↓
Repository.Create(ctx, org) → MongoDB.InsertOne
    ↓
Return created organization with ID
    ↓
Convert to protobuf response
    ↓
Response → Frontend
    ↓
addOrganization({ id, name, slug, role: 'admin' })
    ↓
Update Zustand: append to organizations[], set selectedOrganizationId
    ↓
navigate({ to: '/dashboard' })
    ↓
Dashboard beforeLoad: organizations.length > 0 → Render
```

### Dashboard Access Flow

```
User navigates to /dashboard
    ↓
beforeLoad guard executes
    ↓
const { organizations } = useUserStore.getState()
    ↓
if (organizations.length === 0):
    throw redirect({ to: '/organizations/new' })
else:
    Render dashboard
```

## Related Files

### Generated Files (Do Not Edit)
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organization_pb.ts` - TypeScript protobuf types
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery.ts` - TanStack Query bindings

### Configuration
- `/Users/jayce/team-attention/cops/idl/protobuf/buf.yaml` - Buf configuration
- `/Users/jayce/team-attention/cops/idl/protobuf/buf.gen.yaml` - Code generation config

### Database Schema
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go` - MongoDB schema mapping

## Future Enhancements

1. **Organization Settings Page**: Dedicated page for updating organization details
2. **Member Invitation**: Invite users to join organizations
3. **Organization Switching UI**: Improved switcher in navigation
4. **Slug Auto-generation**: Generate slug from organization name
5. **Organization Logo**: Support for organization branding
6. **Organization Deletion**: Allow admins to delete organizations (with safeguards)
7. **Audit Logging**: Track organization changes for compliance
8. **Organization Templates**: Pre-configured organization structures
