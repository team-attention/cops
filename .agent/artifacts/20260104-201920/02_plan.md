# Implementation Plan: Organization Creation Flow

## Overview

This implementation adds an organization creation flow for new users who have zero organizations. When an authenticated user navigates to the dashboard with no organizations, they are redirected to `/organizations/new` where they must create an organization before accessing the dashboard. After successful creation, the new organization is set as selected in Zustand and the user is redirected to the dashboard.

The implementation follows hexagonal architecture for the backend and Feature Driven Development for the frontend.

## Package Changes

No external packages need to be added. All required dependencies are already present:
- Backend: ConnectRPC, MongoDB driver, fx
- Frontend: TanStack Router, TanStack Query (connect-query), Zustand, shadcn/ui

## Implementation Steps

### Step 1: Create Organization Proto Definition

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions
- `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`: Example of service definition
- `/Users/jayce/team-attention/cops/idl/protobuf/domain/v1/domain.proto`: Organization message definition

#### `/Users/jayce/team-attention/cops/idl/protobuf/organization/v1/organization.proto`

**Description**:
Create new proto file defining OrganizationService with CreateOrganization RPC.

```protobuf
syntax = "proto3";

package organization.v1;

import "domain/v1/domain.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1;organizationv1";

// CreateOrganizationReq contains the data needed to create a new organization.
message CreateOrganizationReq {
  // name is the display name for the organization (required)
  string name = 1;
  // slug is the URL-safe identifier for the organization (required)
  string slug = 2;
}

// CreateOrganizationRes contains the newly created organization.
message CreateOrganizationRes {
  // organization is the created organization with generated ID
  domain.v1.Organization organization = 1;
}

// OrganizationService handles organization management operations.
service OrganizationService {
  // CreateOrganization creates a new organization with the authenticated user as admin.
  // Requires valid JWT token in Authorization header.
  rpc CreateOrganization(CreateOrganizationReq) returns (CreateOrganizationRes);
}
```

**Post-step Action**:
Run `cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate` to generate Go and TypeScript code.

---

### Step 2: Create Organization Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository port conventions
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter pattern
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`: Existing organization repo port

#### `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/organization_repo_port.go`

**Description**:
Define repository port interface for organization creation operations.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// OrganizationRepositoryPort defines interface for organization persistence operations.
type OrganizationRepositoryPort interface {
	// Create persists a new organization to the database.
	// Returns the created organization with generated ID.
	// Returns error if slug already exists for the user or database error occurs.
	Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

	// ExistsSlugForUser checks if a slug already exists within the user's organizations.
	// Returns true if slug exists, false otherwise.
	// Returns error if database error occurs.
	ExistsSlugForUser(ctx context.Context, userID, slug string) (bool, error)
}
```

---

### Step 3: Implement MongoDB Organization Repository

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Adapter implementation
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger binding
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`: MongoDB repository example
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: MongoDB schema

#### `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb/organization_repo.go`

**Description**:
Implement MongoDB adapter for organization repository port.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoOrganizationRepository implements OrganizationRepositoryPort for MongoDB.
type MongoOrganizationRepository struct {
	logger  *slog.Logger
	orgColl *mongo.Collection
}

// NewMongoOrganizationRepository creates a new MongoDB organization repository.
func NewMongoOrganizationRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationRepository {
	// 1. Create logger with name "organization.repository.mongodb".
	// 2. Get organizations collection from db.
	// 3. Return initialized MongoOrganizationRepository.
}

// Create persists a new organization to the database.
func (r *MongoOrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	// 1. Create mongoschema.Organization and call FromDomain(org).
	// 2. Call orgColl.InsertOne with the schema.
	// 3. If error, log error and return nil, error.
	// 4. Get inserted ID from result and set it on schema.ID.
	// 5. Call schema.ToDomain() to convert back to domain model.
	// 6. Return the domain organization, nil.
}

// ExistsSlugForUser checks if a slug already exists within the user's organizations.
func (r *MongoOrganizationRepository) ExistsSlugForUser(ctx context.Context, userID, slug string) (bool, error) {
	// 1. Convert userID string to bson.ObjectID.
	// 2. If conversion fails, return false, error.
	// 3. Build filter to find organization where:
	//    a. slug matches the given slug
	//    b. members array contains an element with userId matching userObjectID
	// 4. Call orgColl.CountDocuments with the filter.
	// 5. If error, log error and return false, error.
	// 6. Return count > 0, nil.
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Create organization successfully | valid org with name, slug, members | *domain.Organization with ID, nil | Happy path |
| Create with invalid member userID | org with invalid member.UserID format | nil, error | ObjectID conversion error |
| MongoDB insert error | valid org, simulated DB error | nil, error | Database error handling |
| ExistsSlugForUser - slug exists | existing slug for user | true, nil | Slug found |
| ExistsSlugForUser - slug not exists | non-existing slug for user | false, nil | Slug not found |
| ExistsSlugForUser - invalid userID | invalid userID format | false, error | ObjectID conversion error |

---

### Step 4: Create Organization Service

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service structure
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger conventions
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`: Function parameter rules
- `/Users/jayce/team-attention/cops/api/internal/service/user/user_service.go`: Service example

#### `/Users/jayce/team-attention/cops/api/internal/service/organization/organization_service.go`

**Description**:
Implement organization service with CreateOrganization business logic.

```go
package organization

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

const (
	// MaxNameLength is the maximum length for organization name.
	MaxNameLength = 100
	// MaxSlugLength is the maximum length for organization slug.
	MaxSlugLength = 50
)

// slugRegex validates that slug contains only lowercase alphanumeric characters, hyphens, and underscores.
var slugRegex = regexp.MustCompile(`^[a-z0-9-_]+$`)

// CreateOrganizationParams contains parameters for CreateOrganization.
type CreateOrganizationParams struct {
	UserID string
	Name   string
	Slug   string
}

// Service implements organization business logic.
type Service struct {
	logger  *slog.Logger
	orgRepo repository.OrganizationRepositoryPort
}

// NewService creates a new organization service.
func NewService(l *slog.Logger, orgRepo repository.OrganizationRepositoryPort) *Service {
	// 1. Create logger with name "organization.service".
	// 2. Return initialized Service.
}

// CreateOrganization creates a new organization with the user as admin.
func (s *Service) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*domain.Organization, error) {
	// 1. Validate params.UserID is not empty.
	//    a. If empty, return nil, fmt.Errorf("userID is required").

	// 2. Validate params.Name:
	//    a. Trim whitespace using strings.TrimSpace.
	//    b. Check trimmed name is not empty.
	//       - If empty, return nil, fmt.Errorf("name is required").
	//    c. Check length <= MaxNameLength.
	//       - If exceeds, return nil, fmt.Errorf("name must be at most %d characters", MaxNameLength).

	// 3. Validate params.Slug:
	//    a. Check not empty.
	//       - If empty, return nil, fmt.Errorf("slug is required").
	//    b. Check length <= MaxSlugLength.
	//       - If exceeds, return nil, fmt.Errorf("slug must be at most %d characters", MaxSlugLength).
	//    c. Check matches slugRegex.
	//       - If not, return nil, fmt.Errorf("slug must contain only lowercase letters, numbers, hyphens, and underscores").

	// 4. Check slug uniqueness by calling s.orgRepo.ExistsSlugForUser(ctx, params.UserID, params.Slug).
	//    a. If error, log error and return nil, error.
	//    b. If exists == true, return nil, fmt.Errorf("slug already exists").

	// 5. Create domain.Organization:
	//    a. Set Name to trimmed params.Name.
	//    b. Set Slug to params.Slug.
	//    c. Set Members to single OrganizationMember with:
	//       - UserID: domain.ID(params.UserID)
	//       - Role: domain.MemberRoleAdmin

	// 6. Call s.orgRepo.Create(ctx, org) to persist.
	//    a. If error, log error and return nil, error.

	// 7. Log info with created organization ID and user ID.

	// 8. Return created organization, nil.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid creation | valid userID, name, slug | *domain.Organization, nil | Happy path |
| Empty userID | userID="" | nil, error "userID is required" | UserID validation |
| Empty name | name="" | nil, error "name is required" | Name empty validation |
| Whitespace-only name | name="   " | nil, error "name is required" | Name trim validation |
| Name too long | name with 101 chars | nil, error "name must be at most..." | Name length validation |
| Empty slug | slug="" | nil, error "slug is required" | Slug empty validation |
| Slug too long | slug with 51 chars | nil, error "slug must be at most..." | Slug length validation |
| Invalid slug chars | slug="My Org!" | nil, error "slug must contain only..." | Slug regex validation |
| Uppercase slug | slug="MyOrg" | nil, error "slug must contain only..." | Slug case validation |
| Slug already exists | existing slug | nil, error "slug already exists" | Slug uniqueness check |
| Repository error | valid params, repo error | nil, error | Repository error handling |

---

### Step 5: Create Organization ConnectRPC Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: Handler conventions
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md`: Inbound structure
- `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`: Handler example

#### `/Users/jayce/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc/handler.go`

**Description**:
Create ConnectRPC handler struct and constructor.

```go
package connectrpc

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/organization"
	"github.com/team-attention/cops/shared/gen/grpcstub/organization/v1/organizationv1connect"
)

// OrganizationGRPCHandler handles gRPC requests for organization service.
type OrganizationGRPCHandler struct {
	svc    *organization.Service
	logger *slog.Logger
}

// NewOrganizationGRPCHandler creates a new organization gRPC handler.
func NewOrganizationGRPCHandler(l *slog.Logger, svc *organization.Service) *OrganizationGRPCHandler {
	// 1. Create logger with name "organization.grpc.connectrpc".
	// 2. Return initialized OrganizationGRPCHandler.
}

// GetHandler implements ConnectHandler interface.
func (h *OrganizationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// 1. Return organizationv1connect.NewOrganizationServiceHandler(h, opts...).
}

// Interface verification
var _ organizationv1connect.OrganizationServiceHandler = (*OrganizationGRPCHandler)(nil)
```

#### `/Users/jayce/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc/organization.go`

**Description**:
Implement CreateOrganization RPC method.

```go
package connectrpc

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/organization"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	organizationv1 "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1"
)

// CreateOrganization creates a new organization with the authenticated user as admin.
func (h *OrganizationGRPCHandler) CreateOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.CreateOrganizationReq],
) (*connect.Response[organizationv1.CreateOrganizationRes], error) {
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	//    a. If empty, log warn and return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated")).

	// 2. Create organization.CreateOrganizationParams with:
	//    a. UserID from context
	//    b. Name from req.Msg.Name
	//    c. Slug from req.Msg.Slug

	// 3. Call h.svc.CreateOrganization(ctx, params).
	//    a. If error:
	//       i. If error contains "is required", return connect.NewError(connect.CodeInvalidArgument, err).
	//       ii. If error contains "must be at most", return connect.NewError(connect.CodeInvalidArgument, err).
	//       iii. If error contains "must contain only", return connect.NewError(connect.CodeInvalidArgument, err).
	//       iv. If error contains "already exists", return connect.NewError(connect.CodeAlreadyExists, err).
	//       v. Otherwise, log error and return connect.NewError(connect.CodeInternal, err).

	// 4. Convert result to protobuf response:
	//    a. Create protoMembers slice by iterating result.Members:
	//       - For each member, create domainv1.OrganizationMember with UserId and Role
	//    b. Create domainv1.Organization with:
	//       - Id: string(result.ID)
	//       - Name: result.Name
	//       - Slug: result.Slug
	//       - Members: protoMembers

	// 5. Create organizationv1.CreateOrganizationRes with Organization field.

	// 6. Return connect.NewResponse with the response.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid creation | valid name, slug, authenticated | CreateOrganizationRes, nil | Happy path |
| Unauthenticated | no userID in context | nil, CodeUnauthenticated | Auth check |
| Empty name | name="" | nil, CodeInvalidArgument | "is required" error mapping |
| Invalid slug format | slug="Bad Slug" | nil, CodeInvalidArgument | "must contain only" error mapping |
| Slug already exists | existing slug | nil, CodeAlreadyExists | "already exists" error mapping |
| Internal error | service internal error | nil, CodeInternal | Default error handling |

---

### Step 6: Register Organization Module in Container

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`: Module example
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/register_connectrpc.go`: Handler registration

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_organization.go`

**Description**:
Create fx module for organization service.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/organization"
	"github.com/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb"
)

func newOrganizationModule() fx.Option {
	return fx.Module("organization",
		// Organization repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationRepository,
				fx.As(new(repository.OrganizationRepositoryPort)),
			),
		),

		// Service
		fx.Provide(organization.NewService),

		// ConnectRPC handler (private - requires auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewOrganizationGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
```

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`

**Description**:
Add organization module to application.

```go
// In Run() function, add after newUserModule():
newOrganizationModule(),
```

---

### Step 7: Create Frontend Organization Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook patterns
- `/Users/jayce/team-attention/cops/web/src/feature/user/hook/use-delete-account.ts`: Mutation hook example
- `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`: Transport config

#### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-create-organization.ts`

**Description**:
Create TanStack Query mutation hook for organization creation.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { createOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useCreateOrganization provides a mutation hook for creating a new organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useCreateOrganization = () => {
  return useMutation(createOrganization, { transport })
}
```

---

### Step 8: Update Zustand Store with addOrganization Action

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`: Current store implementation

#### `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`

**Description**:
Add `addOrganization` action to handle adding newly created organization.

```typescript
// Add to UserStoreActions interface (after setError):
addOrganization: (organization: OrganizationData) => void

// Add to store implementation inside persist() set callbacks (after setError):
addOrganization: (organization) =>
  set((state) => ({
    organizations: [...state.organizations, organization],
    // Set newly created organization as selected
    selectedOrganizationId: organization.id,
  })),
```

---

### Step 9: Create Organization Form Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Component conventions
- `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx`: Form component example

#### `/Users/jayce/team-attention/cops/web/src/feature/organization/component/organization-form.tsx`

**Description**:
Create form component for organization creation with name and slug fields.

```typescript
import { useState, useCallback, useMemo } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Building2, Loader2 } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useCreateOrganization } from '../hook/use-create-organization'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// OrganizationFormState represents the form's internal state.
type OrganizationFormState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'error'; message: string }

// slugRegex validates slug format (lowercase alphanumeric, hyphens, underscores)
const slugRegex = /^[a-z0-9-_]*$/

export const OrganizationForm = () => {
  // 1. useState for:
  //    - name: string (initial: '')
  //    - slug: string (initial: '')
  //    - state: OrganizationFormState (initial: { status: 'idle' })

  // 2. Get mutation from useCreateOrganization()

  // 3. Get addOrganization from useUserStore((s) => s.addOrganization)

  // 4. Get navigate from useNavigate()

  // 5. Create handleNameChange callback using useCallback:
  //    a. Get value from e.target.value
  //    b. Update name state with value
  //    c. If state.status === 'error', set state to { status: 'idle' }

  // 6. Create handleSlugChange callback using useCallback:
  //    a. Get value from e.target.value
  //    b. Convert value to lowercase
  //    c. If lowercase value matches slugRegex OR value is empty string:
  //       i. Update slug state with lowercase value
  //    d. If state.status === 'error', set state to { status: 'idle' }

  // 7. Create isFormValid computed using useMemo:
  //    a. Return true if ALL conditions met:
  //       i. name.trim().length > 0
  //       ii. slug.length > 0
  //       iii. slugRegex.test(slug)

  // 8. Create handleSubmit async callback using useCallback:
  //    a. Call e.preventDefault()
  //    b. If !isFormValid, return early
  //    c. Set state to { status: 'submitting' }
  //    d. Try block:
  //       i. Call response = await mutation.mutateAsync({ name: name.trim(), slug })
  //       ii. If response.organization exists:
  //           - Create organizationData object with:
  //             * id: response.organization.id
  //             * name: response.organization.name
  //             * role: 'admin' as const
  //           - Call addOrganization(organizationData)
  //           - Call navigate({ to: '/dashboard' })
  //       iii. Else:
  //           - Set state to { status: 'error', message: 'Failed to create organization' }
  //    e. Catch block (error):
  //       i. Cast error as { code?: Code; message?: string }
  //       ii. Initialize errorMessage = 'An error occurred'
  //       iii. Map error.code to message:
  //            - Code.InvalidArgument: 'Please check your input and try again'
  //            - Code.AlreadyExists: 'This slug is already taken. Please choose another.'
  //            - Code.Unauthenticated: 'Session expired. Please log in again.'
  //            - else if error.message exists: use error.message
  //       iv. Set state to { status: 'error', message: errorMessage }

  // 9. Return JSX:
  //    <Card className="w-full max-w-md border-zinc-800 bg-zinc-900">
  //      <CardHeader>
  //        <div className="flex items-center gap-3">
  //          <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/10 p-2">
  //            <Building2 className="h-5 w-5 text-cyan-400" />
  //          </div>
  //          <div>
  //            <CardTitle className="text-zinc-100">Create Organization</CardTitle>
  //            <CardDescription className="text-zinc-500">
  //              Create your first organization to get started
  //            </CardDescription>
  //          </div>
  //        </div>
  //      </CardHeader>
  //      <CardContent>
  //        <form onSubmit={handleSubmit} className="space-y-4">
  //          {/* Name field */}
  //          <div className="space-y-2">
  //            <label htmlFor="name" className="text-sm font-medium text-zinc-300">
  //              Organization Name
  //            </label>
  //            <Input
  //              id="name"
  //              value={name}
  //              onChange={handleNameChange}
  //              placeholder="My Organization"
  //              className="bg-zinc-800/50"
  //            />
  //          </div>
  //
  //          {/* Slug field */}
  //          <div className="space-y-2">
  //            <label htmlFor="slug" className="text-sm font-medium text-zinc-300">
  //              URL Slug
  //            </label>
  //            <Input
  //              id="slug"
  //              value={slug}
  //              onChange={handleSlugChange}
  //              placeholder="my-organization"
  //              className="bg-zinc-800/50 font-mono"
  //            />
  //            <p className="text-xs text-zinc-600">
  //              Lowercase letters, numbers, hyphens, and underscores only
  //            </p>
  //          </div>
  //
  //          {/* Error alert */}
  //          {state.status === 'error' && (
  //            <Alert className="border-red-900/50 bg-red-950/30">
  //              <AlertDescription className="text-red-200">
  //                {state.message}
  //              </AlertDescription>
  //            </Alert>
  //          )}
  //
  //          {/* Submit button */}
  //          <Button
  //            type="submit"
  //            disabled={!isFormValid || state.status === 'submitting'}
  //            className="w-full bg-cyan-600 text-white hover:bg-cyan-500"
  //          >
  //            {state.status === 'submitting' ? (
  //              <>
  //                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
  //                Creating...
  //              </>
  //            ) : (
  //              'Create Organization'
  //            )}
  //          </Button>
  //        </form>
  //      </CardContent>
  //    </Card>
}
```

---

### Step 10: Create Organization New Route

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`: Route example
- `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`: Layout handling

#### `/Users/jayce/team-attention/cops/web/src/route/organizations/new.tsx`

**Description**:
Create route for organization creation page.

```typescript
import { createFileRoute } from '@tanstack/react-router'
import { OrganizationForm } from '@/feature/organization/component/organization-form'

export const Route = createFileRoute('/organizations/new')({
  component: OrganizationNewPage,
})

// OrganizationNewPage displays the organization creation form.
function OrganizationNewPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950">
      <div className="w-full max-w-md px-4">
        <OrganizationForm />
      </div>
    </div>
  )
}
```

---

### Step 11: Add Organization Route Guard to Dashboard

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`: Current dashboard route
- `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`: Zustand store

#### `/Users/jayce/team-attention/cops/web/src/route/dashboard.tsx`

**Description**:
Add beforeLoad hook to check organization count and redirect if zero.

```typescript
import { createFileRoute, redirect } from '@tanstack/react-router'
// ... existing imports
import { useUserStore } from '@/shared/store/user-store'

export const Route = createFileRoute('/dashboard')({
  beforeLoad: async () => {
    // 1. Get organizations array from useUserStore.getState().organizations
    const { organizations } = useUserStore.getState()
    // 2. If organizations.length === 0:
    //    a. Throw redirect({ to: '/organizations/new' })
    if (organizations.length === 0) {
      throw redirect({ to: '/organizations/new' })
    }
  },
  component: DashboardPage,
})

// ... rest of file unchanged
```

---

### Step 12: Update Root Layout for Organization Routes

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`: Current root layout

#### `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`

**Description**:
Update RootComponent to handle organization routes without sidebar (like auth routes).

```typescript
// Update RootComponent function:
function RootComponent() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const isAuthRoute = pathname.startsWith('/auth')
  const isOrganizationNewRoute = pathname === '/organizations/new'

  // Auth routes and organization creation render without sidebar/header layout
  if (isAuthRoute || isOrganizationNewRoute) {
    return (
      <>
        <Outlet />
        <TanStackDevtools
          config={{
            position: 'bottom-right',
          }}
          plugins={[
            {
              name: 'Tanstack Router',
              render: <TanStackRouterDevtoolsPanel />,
            },
            TanStackQueryDevtools,
          ]}
        />
      </>
    )
  }

  // ... rest of existing code for sidebar layout unchanged
}
```

---

## Implementation Order and Dependencies

```
Step 1: Proto Definition
    |
    v
[Run: cd idl/protobuf && buf generate]
    |
    +------------------+
    |                  |
    v                  v
Step 2: Repo Port    Step 7: Frontend Hook (after buf generate for TS stubs)
    |
    v
Step 3: MongoDB Repo
    |
    v
Step 4: Organization Service
    |
    v
Step 5: ConnectRPC Handler
    |
    v
Step 6: Container Registration
    |
    +------------------+------------------+
    |                  |                  |
    v                  v                  v
Step 8: Zustand     Step 10: Route    Step 12: Root Layout
    |
    v
Step 9: Form Component
    |
    v
Step 11: Dashboard Guard
```

**Critical Dependencies**:
1. Proto definition (Step 1) must complete before any backend implementation
2. `buf generate` must run after Step 1 to generate both Go and TypeScript stubs
3. Repository port (Step 2) must exist before MongoDB implementation (Step 3)
4. Service (Step 4) depends on repository port (Step 2)
5. Handler (Step 5) depends on service (Step 4) and generated proto stubs
6. Frontend hook (Step 7) requires generated TypeScript stubs from `buf generate`
7. Form component (Step 9) depends on hook (Step 7) and Zustand update (Step 8)
8. Dashboard guard (Step 11) depends on route (Step 10) existing

## Testing Considerations

### Backend Tests

1. **Repository Tests** (`organization_repo_test.go`):
   - Test Create with valid organization
   - Test Create with MongoDB error
   - Test ExistsSlugForUser with existing slug
   - Test ExistsSlugForUser with non-existing slug
   - Test ExistsSlugForUser with invalid userID format
   - Use MongoDB testcontainer or mock

2. **Service Tests** (`organization_service_test.go`):
   - Test all validation scenarios (empty fields, invalid slug format, slug too long)
   - Test slug uniqueness check passes
   - Test slug uniqueness check fails
   - Test successful creation with admin member
   - Mock repository using interface

3. **Handler Tests** (`handler_test.go`):
   - Test authentication check (no userID in context)
   - Test CodeInvalidArgument mapping for validation errors
   - Test CodeAlreadyExists mapping for duplicate slug
   - Test CodeInternal mapping for unexpected errors
   - Test successful response structure with members

### Frontend Tests

1. **Hook Tests**:
   - Test mutation function returns expected shape

2. **Component Tests** (`organization-form.test.tsx`):
   - Test form renders with empty fields
   - Test name input updates state
   - Test slug input auto-lowercases
   - Test slug input rejects invalid characters
   - Test submit button disabled when form invalid
   - Test submit button enabled when form valid
   - Test error message displays on API error
   - Test successful submission calls addOrganization and navigates

3. **Route Tests**:
   - Test dashboard redirects when organizations.length === 0
   - Test dashboard loads normally when organizations.length > 0
   - Test organization/new route renders form component

### Integration Tests

1. **End-to-end flow**:
   - Login with new user (0 organizations)
   - Verify redirect to /organizations/new
   - Fill form with valid data
   - Submit and verify redirect to /dashboard
   - Verify organization appears in sidebar
