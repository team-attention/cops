# Requirements: Organization Creation Flow for New Users

## Request Summary

Implement an organization creation flow for users who have zero organizations when they try to access the dashboard. When a user completes authentication and attempts to navigate to the dashboard, the system should check if they have any organizations. If they have none, redirect them to an organization creation page where they must create an organization before proceeding to the dashboard. After creating an organization, set it as the currently selected organization in Zustand and redirect back to the dashboard.

## Acceptance Criteria

### Frontend (Web)
- [ ] Create `/organizations/new` route with organization creation form
- [ ] Form includes Name field (required) and Slug field (required)
- [ ] Implement organization check in dashboard route that redirects to `/organizations/new` if user has 0 organizations
- [ ] After successful organization creation, set the newly created organization as selected in Zustand store
- [ ] Redirect to `/dashboard` after successful organization creation
- [ ] Handle organization creation errors with user-friendly error messages
- [ ] Prevent skipping organization creation (no "Skip" button or bypass)

### Backend (API)
- [ ] Create `organization/v1/organization.proto` with `CreateOrganization` RPC
- [ ] Define `CreateOrganizationReq` message with `name` and `slug` fields (both required)
- [ ] Define `CreateOrganizationRes` message returning created organization details
- [ ] Implement OrganizationService in `api/internal/service/organization/`
- [ ] Add organization repository method `Create` to persist organizations
- [ ] Validate organization name and slug uniqueness per user
- [ ] Set creating user as organization admin member
- [ ] Generate protobuf code using `buf generate`

### State Management
- [ ] Update Zustand store to handle setting selected organization after creation
- [ ] Ensure `setOrganizations` properly updates when new organization is created
- [ ] Verify `selectedOrganizationId` is set to newly created organization ID

## Scope

### In Scope
- Organization creation API endpoint (gRPC/ConnectRPC)
- Frontend organization creation form page at `/organizations/new`
- Dashboard route guard to check organization count
- Zustand state update after organization creation
- Automatic redirect flow: Login → Dashboard → (if no orgs) → Create Org → Dashboard
- Form validation for name and slug fields
- Error handling for creation failures

### Out of Scope
- Organization editing/deletion functionality
- Organization settings page
- Inviting other members to organizations
- Organization switching UI improvements beyond existing functionality
- Multiple organization creation (users create one at a time)
- Organization onboarding tutorial or welcome screens

## Constraints

### Technical Constraints
- Must follow existing protobuf conventions in `idl/protobuf/`
- Must follow hexagonal architecture pattern in `api/internal/service/organization/`
- Must use ConnectRPC for API communication
- Must use TanStack Router for routing
- Organization creation is mandatory - no skip option allowed
- Slug must be URL-safe and unique within the user's organizations

### Business Constraints
- Users cannot access dashboard without at least one organization
- User who creates organization becomes admin automatically
- Organization name and slug are both required fields

## Additional Context

### Current System State

**Existing Infrastructure:**
- User authentication flow: Login → Auth Callback → Dashboard
- Zustand store (`user-store.ts`) manages organizations and selectedOrganizationId
- `useGetMe` hook fetches user data including organizations array
- Organization repository exists with read/delete operations but no create operation
- User service handles GetMe which returns user + organizations

**Data Models:**
```protobuf
// domain/v1/domain.proto
message Organization {
  string id = 1;
  string name = 2;
  string slug = 3;
  repeated OrganizationMember members = 4;
}

message OrganizationMember {
  string user_id = 1;
  string role = 2;  // "admin" or "member"
}
```

**Zustand Store Behavior:**
- Auto-selects first organization if none selected
- Persists `selectedOrganizationId` to localStorage
- When organizations are set, validates current selection or picks first org

### API Implementation Requirements

**New Proto Service:**
```protobuf
// idl/protobuf/organization/v1/organization.proto
syntax = "proto3";

package organization.v1;

import "domain/v1/domain.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1;organizationv1";

message CreateOrganizationReq {
  string name = 1;  // Required, organization display name
  string slug = 2;  // Required, URL-safe identifier
}

message CreateOrganizationRes {
  domain.v1.Organization organization = 1;
}

service OrganizationService {
  rpc CreateOrganization(CreateOrganizationReq) returns (CreateOrganizationRes);
}
```

**Backend Service Structure:**
```
api/internal/service/organization/
├── organization_service.go          # Service with CreateOrganization method
├── inbound/
│   └── grpc/
│       └── connectrpc/
│           ├── handler.go           # ConnectRPC handler struct + constructor
│           └── organization.go      # RPC method implementations
└── outbound/
    └── repository/
        ├── organization_repo_port.go  # Add Create method to existing port
        └── mongodb/
            └── organization_repo.go   # Implement Create method
```

**Service Method Signature:**
```go
func (s *Service) CreateOrganization(ctx context.Context, userID, name, slug string) (*domain.Organization, error)
```

**Repository Method to Add:**
```go
// OrganizationRepositoryPort interface - add this method
Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)
```

### Frontend Implementation Requirements

**Route Structure:**
```
web/src/route/
├── organizations/
│   └── new.tsx              # Organization creation page
└── dashboard.tsx            # Add beforeLoad check
```

**Component Structure:**
```
web/src/feature/organization/
├── component/
│   └── organization-form.tsx        # Form component
└── hook/
    └── use-create-organization.ts   # TanStack Query mutation hook
```

**React Query Hook:**
```typescript
// use-create-organization.ts
interface CreateOrganizationParams {
  name: string
  slug: string
}

export const useCreateOrganization = () => {
  return useMutation({
    mutationFn: async (params: CreateOrganizationParams) => {
      // Call CreateOrganization RPC
    },
    onSuccess: (data) => {
      // Update Zustand store with new organization
      // Set as selected organization
    },
  })
}
```

**Dashboard Route Guard:**
```typescript
// dashboard.tsx
export const Route = createFileRoute('/dashboard')({
  beforeLoad: async ({ context }) => {
    const { organizations } = useUserStore.getState()
    if (organizations.length === 0) {
      throw redirect({ to: '/organizations/new' })
    }
  },
  component: DashboardPage,
})
```

### Dependencies

**Backend:**
- Existing: MongoDB organization collection
- Existing: User authentication via JWT
- New: OrganizationService protobuf definition
- New: CreateOrganization repository method

**Frontend:**
- Existing: TanStack Router for routing
- Existing: TanStack Query for API calls
- Existing: Zustand for state management
- Existing: ConnectRPC client setup
- New: Organization creation form component
- New: Organization creation hook

### User Flow Diagram

```
┌─────────────┐
│   Login     │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  Auth Callback      │
│  (store tokens)     │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────────┐
│  Navigate to /dashboard │
└──────┬──────────────────┘
       │
       ▼
┌──────────────────────────────┐
│  beforeLoad: Check org count │
└──────┬───────────────────┬───┘
       │                   │
  orgs > 0            orgs === 0
       │                   │
       ▼                   ▼
┌─────────────┐   ┌──────────────────────┐
│  Dashboard  │   │  /organizations/new  │
└─────────────┘   └──────┬───────────────┘
                         │
                         ▼
                  ┌──────────────────┐
                  │  Fill Name/Slug  │
                  └──────┬───────────┘
                         │
                         ▼
                  ┌──────────────────┐
                  │  Submit Form     │
                  └──────┬───────────┘
                         │
                         ▼
                  ┌──────────────────────┐
                  │  Call CreateOrg API  │
                  └──────┬───────────────┘
                         │
                   Success │ Error
                         ├──────────┐
                         │          ▼
                         │    ┌───────────┐
                         │    │ Show Error│
                         │    └───────────┘
                         ▼
                  ┌──────────────────────┐
                  │  Update Zustand:     │
                  │  - Add to orgs array │
                  │  - Set as selected   │
                  └──────┬───────────────┘
                         │
                         ▼
                  ┌──────────────────┐
                  │  Redirect to     │
                  │  /dashboard      │
                  └──────────────────┘
```

### Validation Rules

**Name Field:**
- Required
- Minimum length: 1 character
- Maximum length: 100 characters
- Cannot be empty or whitespace only

**Slug Field:**
- Required
- Must be URL-safe (alphanumeric, hyphens, underscores only)
- Minimum length: 1 character
- Maximum length: 50 characters
- Must be unique within user's organizations
- Suggested pattern: `^[a-z0-9-_]+$`

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Does organization creation API exist? | **No** - Needs to be created. Repository has read/delete operations only. |
| Organization creation route path? | `/organizations/new` |
| Required form fields? | Name (display name) + Slug (URL-safe identifier), both required |
| Can users skip organization creation? | **No** - Organization is mandatory for dashboard access |
| Where should org count check happen? | Dashboard route's `beforeLoad` hook (redirect if 0 organizations) |
| What happens after creation? | Set created org as selected in Zustand, then redirect to `/dashboard` |
| Backend service location? | New service: `api/internal/service/organization/` following hexagonal architecture |
| Proto file location? | `idl/protobuf/organization/v1/organization.proto` (new file) |
| Initial organization role? | Creating user becomes "admin" automatically |
| Slug validation? | URL-safe characters only, unique within user's organizations |
