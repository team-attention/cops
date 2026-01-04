# Implementation Plan: Account Deletion Feature

## Overview

This plan implements a permanent account deletion feature for C-Ops. When a user deletes their account:

1. User must type 'DELETE' (case-sensitive) to confirm
2. All organizations where the user is the sole member are cascade-deleted (along with their projects and sessions)
3. For shared organizations, only the user's membership is removed
4. User profile and authentication accounts are permanently deleted
5. User is logged out (tokens invalidated client-side)

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | Dialog/Modal component needed for confirmation | `npx shadcn@latest add dialog` | shadcn dialog component is required for the deletion confirmation modal |

## Step 1: Add DeleteAccount RPC to Protobuf

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions (Req/Res suffix pattern)
- `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`: Existing UserService patterns

#### `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`

**Description**:
Add DeleteAccountReq message with confirmation phrase and DeleteAccountRes message. Add DeleteAccount RPC to UserService.

```protobuf
// DeleteAccountReq contains the confirmation phrase for account deletion.
// User must type 'DELETE' exactly to confirm the deletion.
message DeleteAccountReq {
  // confirmation_phrase must be exactly 'DELETE' (case-sensitive)
  string confirmation_phrase = 1;
}

// DeleteAccountRes contains the result of account deletion.
message DeleteAccountRes {
  // success indicates whether the account was deleted
  bool success = 1;
  // message provides additional context about the deletion result
  string message = 2;
}
```

Add to UserService:

```protobuf
// DeleteAccount permanently deletes the authenticated user's account.
// Requires valid JWT token in Authorization header.
// Performs cascade deletion for organizations where user is sole member.
rpc DeleteAccount(DeleteAccountReq) returns (DeleteAccountRes);
```

---

## Step 2: Generate Protobuf Code

**Files to Read**:
- `/Users/jayce/team-attention/cops/idl/protobuf/buf.gen.yaml`: Code generation configuration

**Description**:
Run buf generate to create Go and TypeScript code from the updated proto file.

```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

This will generate:
- Go types: `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/user.pb.go`
- Go ConnectRPC: `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect/user.connect.go`
- TypeScript types: `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/user/v1/user_pb.ts`
- TypeScript ConnectQuery: `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/user/v1/user-UserService_connectquery.ts`

---

## Step 3: Extend User Repository Port with Delete Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/user_repo_port.go`: Existing interface

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/user_repo_port.go`

**Description**:
Add Delete method to UserRepositoryPort interface for permanent user deletion.

```go
// UserRepositoryPort defines interface for user data retrieval.
type UserRepositoryPort interface {
	// GetByID retrieves a user by their ID.
	// Returns nil, nil if user not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// Delete permanently removes a user by their ID.
	// Returns nil if user was deleted successfully.
	// Returns error if database error occurs.
	Delete(ctx context.Context, userID string) error
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Delete existing user | Valid userID | nil (success) | Happy path |
| Delete non-existent user | Non-existent userID | nil (idempotent) | Not found case |
| Invalid userID format | "invalid-id" | error | Validation branch |
| Database error | Valid userID with DB failure | error | Error handling |

---

## Step 4: Implement User Repository Delete Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository implementation patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/user_repo.go`: Existing implementation patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/user.go`: User collection constants

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/user_repo.go`

**Description**:
Add Delete method to MongoUserRepository for permanent user deletion.

```go
// Delete permanently removes a user by their ID.
func (r *MongoUserRepository) Delete(ctx context.Context, userID string) error {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID using bson.ObjectIDFromHex.
	// 2. If conversion fails, return error.
	// 3. Create filter with _id field.
	// 4. Execute DeleteOne on users collection.
	// 5. If error occurs, log error and return error.
	// 6. Return nil (success - idempotent operation, no check for deleted count).
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Delete existing user | Valid ObjectID hex | nil | Happy path |
| Delete non-existent user | Valid ObjectID hex | nil | Idempotent |
| Invalid ObjectID format | "not-a-valid-id" | error | Validation branch |
| Database connection error | Valid ID, DB down | error | Error handling |

---

## Step 5: Extend Organization Repository Port with Deletion Methods

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Port interface patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`: Existing interface
- `/Users/jayce/team-attention/cops/shared/domain/organization.go`: Organization domain model

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`

**Description**:
Add methods for organization membership management and deletion.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserOrganization represents a user's membership in an organization.
// Contains organization data plus the user's specific role.
type UserOrganization struct {
	Organization *domain.Organization
	Role         domain.MemberRole
}

// OrganizationWithMemberCount represents an organization with its member count.
// Used to determine if cascade deletion is needed.
type OrganizationWithMemberCount struct {
	Organization *domain.Organization
	MemberCount  int
}

// OrganizationRepositoryPort defines interface for organization queries.
type OrganizationRepositoryPort interface {
	// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
	// Queries organizations collection filtering by embedded members.userId.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizations(ctx context.Context, userID string) ([]*UserOrganization, error)

	// GetUserOrganizationsWithMemberCount retrieves all organizations a user belongs to with member counts.
	// Used to determine which organizations need cascade deletion (sole member) vs membership removal.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizationsWithMemberCount(ctx context.Context, userID string) ([]*OrganizationWithMemberCount, error)

	// RemoveUserFromOrganization removes a user from an organization's members array.
	// Returns nil if successful or if user was not a member.
	// Returns error if database error occurs.
	RemoveUserFromOrganization(ctx context.Context, organizationID, userID string) error

	// DeleteOrganization permanently deletes an organization by ID.
	// Returns nil if successful or if organization did not exist.
	// Returns error if database error occurs.
	DeleteOrganization(ctx context.Context, organizationID string) error
}
```

**Test Scenarios for GetUserOrganizationsWithMemberCount**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| User is sole member of one org | userID with 1-member org | [{Org, 1}] | Sole member case |
| User in shared org | userID with multi-member org | [{Org, 3}] | Shared org case |
| User in multiple orgs | userID with mix | Multiple results | Mixed case |
| User not in any org | userID with no orgs | Empty slice | No orgs case |
| Invalid userID | "invalid" | error | Validation branch |

**Test Scenarios for RemoveUserFromOrganization**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Remove existing member | Valid IDs | nil | Happy path |
| User not in org | Valid IDs, not member | nil | Idempotent |
| Invalid organizationID | "invalid", valid userID | error | Validation |
| Invalid userID | Valid orgID, "invalid" | error | Validation |

**Test Scenarios for DeleteOrganization**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Delete existing org | Valid orgID | nil | Happy path |
| Delete non-existent org | Non-existent orgID | nil | Idempotent |
| Invalid organizationID | "invalid" | error | Validation |

---

## Step 6: Implement Organization Repository Methods

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository implementation patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`: Existing implementation
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: Organization collection constants

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`

**Description**:
Add GetUserOrganizationsWithMemberCount, RemoveUserFromOrganization, and DeleteOrganization methods.

```go
// GetUserOrganizationsWithMemberCount retrieves all organizations a user belongs to with member counts.
func (r *MongoOrganizationRepository) GetUserOrganizationsWithMemberCount(ctx context.Context, userID string) ([]*repository.OrganizationWithMemberCount, error) {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID.
	// 2. If conversion fails, return nil, error.
	// 3. Build filter to find organizations where user is a member using $elemMatch.
	// 4. Execute Find query on organizations collection.
	// 5. If error, log and return nil, error.
	// 6. Iterate cursor, decode each result to mongoschema.Organization.
	// 7. For each organization:
	//    a. Convert to domain.Organization using ToDomain().
	//    b. Count members in Members slice.
	//    c. Create OrganizationWithMemberCount with org and count.
	// 8. Return slice of OrganizationWithMemberCount, nil.
}

// RemoveUserFromOrganization removes a user from an organization's members array.
func (r *MongoOrganizationRepository) RemoveUserFromOrganization(ctx context.Context, organizationID, userID string) error {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Convert userID string to bson.ObjectID.
	// 4. If conversion fails, return error.
	// 5. Create filter with organization _id.
	// 6. Create update using $pull to remove member with matching userId from members array.
	// 7. Execute UpdateOne on organizations collection.
	// 8. If error occurs, log error and return error.
	// 9. Return nil (success - idempotent operation).
}

// DeleteOrganization permanently deletes an organization by ID.
func (r *MongoOrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Create filter with _id field.
	// 4. Execute DeleteOne on organizations collection.
	// 5. If error occurs, log error and return error.
	// 6. Return nil (success - idempotent operation).
}
```

---

## Step 7: Create Cascade Delete Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Port interface patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Project collection constants
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Session collection constants

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/cascade_delete_repo_port.go`

**Description**:
Create new port for cascade deletion of projects and sessions by organization.

```go
package repository

import "context"

// CascadeDeleteRepositoryPort defines interface for cascade deletion operations.
// Used during account deletion to clean up related data.
type CascadeDeleteRepositoryPort interface {
	// DeleteProjectsByOrganization permanently deletes all projects for an organization.
	// Returns nil if successful or if no projects existed.
	// Returns error if database error occurs.
	DeleteProjectsByOrganization(ctx context.Context, organizationID string) error

	// DeleteSessionRecordsByOrganization permanently deletes all session records for projects in an organization.
	// First queries projects to get project IDs, then deletes records matching those project IDs.
	// Returns nil if successful or if no records existed.
	// Returns error if database error occurs.
	DeleteSessionRecordsByOrganization(ctx context.Context, organizationID string) error
}
```

**Test Scenarios for DeleteProjectsByOrganization**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Delete projects for org with projects | Valid orgID | nil | Happy path |
| No projects in org | Valid orgID | nil | Empty case |
| Invalid organizationID | "invalid" | error | Validation |

**Test Scenarios for DeleteSessionRecordsByOrganization**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Delete records for org with sessions | Valid orgID | nil | Happy path |
| No projects in org | Valid orgID | nil | No projects case |
| Projects but no sessions | Valid orgID | nil | No sessions case |
| Invalid organizationID | "invalid" | error | Validation |

---

## Step 8: Implement Cascade Delete Repository

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository implementation patterns
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Example of multi-collection queries
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Project field constants
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Session record field constants

#### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/cascade_delete_repo.go`

**Description**:
Create MongoDB implementation for cascade deletion operations.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoCascadeDeleteRepository implements CascadeDeleteRepositoryPort for MongoDB.
type MongoCascadeDeleteRepository struct {
	logger             *slog.Logger
	projectsColl       *mongo.Collection
	sessionRecordsColl *mongo.Collection
}

// NewMongoCascadeDeleteRepository creates a new MongoDB cascade delete repository.
func NewMongoCascadeDeleteRepository(l *slog.Logger, db *mongo.Database) *MongoCascadeDeleteRepository {
	// Implementation outline:
	// 1. Return &MongoCascadeDeleteRepository with:
	//    - logger bound with name "user.repository.mongodb.cascade_delete"
	//    - projectsColl from db.Collection(mongoschema.ProjectCollectionName)
	//    - sessionRecordsColl from db.Collection(mongoschema.RecordCollectionName)
}

// DeleteProjectsByOrganization permanently deletes all projects for an organization.
func (r *MongoCascadeDeleteRepository) DeleteProjectsByOrganization(ctx context.Context, organizationID string) error {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Create filter with organizationId field matching the ObjectID.
	// 4. Execute DeleteMany on projects collection.
	// 5. If error occurs, log error with organizationID and return error.
	// 6. Log info with deleted count.
	// 7. Return nil.
}

// DeleteSessionRecordsByOrganization permanently deletes all session records for projects in an organization.
func (r *MongoCascadeDeleteRepository) DeleteSessionRecordsByOrganization(ctx context.Context, organizationID string) error {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Query projects collection to get all project IDs for this organization.
	//    a. Create filter with organizationId field.
	//    b. Use Find with projection for _id only.
	//    c. Collect project IDs into slice of bson.ObjectID.
	// 4. If no projects found, return nil (nothing to delete).
	// 5. Create filter for session records with projectId in the collected project IDs.
	// 6. Execute DeleteMany on sessionRecords collection.
	// 7. If error occurs, log error with organizationID and return error.
	// 8. Log info with deleted count.
	// 9. Return nil.
}

// Interface verification
var _ repository.CascadeDeleteRepositoryPort = (*MongoCascadeDeleteRepository)(nil)
```

---

## Step 9: Implement DeleteAccount Service Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service implementation patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/user_service.go`: Existing service implementation
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging patterns

#### `/Users/jayce/team-attention/cops/api/internal/service/user/user_service.go`

**Description**:
Add DeleteAccount method and update Service struct to include new repository dependencies.

```go
const (
	// DeleteConfirmationPhrase is the required confirmation phrase for account deletion.
	DeleteConfirmationPhrase = "DELETE"
)

// DeleteAccountResult contains the result of account deletion.
type DeleteAccountResult struct {
	Success bool
	Message string
}

// Service implements user business logic.
type Service struct {
	logger            *slog.Logger
	userRepo          repository.UserRepositoryPort
	orgRepo           repository.OrganizationRepositoryPort
	cascadeDeleteRepo repository.CascadeDeleteRepositoryPort
}

// NewService creates a new user service.
func NewService(
	l *slog.Logger,
	userRepo repository.UserRepositoryPort,
	orgRepo repository.OrganizationRepositoryPort,
	cascadeDeleteRepo repository.CascadeDeleteRepositoryPort,
) *Service {
	// Implementation outline:
	// 1. Return &Service with:
	//    - logger bound with name "user.service"
	//    - userRepo, orgRepo, cascadeDeleteRepo assigned
}

// DeleteAccount permanently deletes the authenticated user's account and related data.
func (s *Service) DeleteAccount(ctx context.Context, userID, confirmationPhrase string) (*DeleteAccountResult, error) {
	// Implementation outline:
	// 1. Validate confirmationPhrase equals DeleteConfirmationPhrase.
	//    a. If not, log warn with userID and return error "confirmation phrase must be 'DELETE'".
	// 2. Validate userID is not empty.
	//    a. If empty, return error "userID is required".
	// 3. Call userRepo.GetByID to verify user exists.
	//    a. If error, log error and return nil, error.
	//    b. If user is nil, log info and return nil, error "user not found".
	// 4. Call orgRepo.GetUserOrganizationsWithMemberCount to get all user's organizations with member counts.
	//    a. If error, log error and return nil, error.
	// 5. Iterate through organizations:
	//    a. If MemberCount == 1 (user is sole member):
	//       i. Call cascadeDeleteRepo.DeleteSessionRecordsByOrganization.
	//          - If error, log error and return nil, error.
	//       ii. Call cascadeDeleteRepo.DeleteProjectsByOrganization.
	//          - If error, log error and return nil, error.
	//       iii. Call orgRepo.DeleteOrganization.
	//          - If error, log error and return nil, error.
	//       iv. Log info about cascade deletion with orgID.
	//    b. If MemberCount > 1 (shared organization):
	//       i. Call orgRepo.RemoveUserFromOrganization.
	//          - If error, log error and return nil, error.
	//       ii. Log info about membership removal with orgID.
	// 6. Call userRepo.Delete to delete user profile.
	//    a. If error, log error and return nil, error.
	// 7. Log info with userID about successful account deletion.
	// 8. Return &DeleteAccountResult{Success: true, Message: "Account deleted successfully"}, nil.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid deletion with confirmation | userID, "DELETE" | Success | Happy path |
| Wrong confirmation phrase | userID, "delete" | error | Case-sensitive check |
| Wrong confirmation phrase | userID, "DELETEE" | error | Exact match check |
| Empty confirmation phrase | userID, "" | error | Empty check |
| Empty userID | "", "DELETE" | error | Validation |
| User not found | non-existent ID, "DELETE" | error | User check |
| User is sole member of org | userID (sole), "DELETE" | Cascade delete | Sole member branch |
| User in shared org | userID (shared), "DELETE" | Remove membership | Shared org branch |
| User in multiple orgs (mixed) | userID, "DELETE" | Both operations | Mixed case |
| Cascade delete failure | userID, "DELETE" | error | Error handling |

---

## Step 10: Implement DeleteAccount Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`: Existing handler implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
Add DeleteAccount RPC handler method.

```go
// DeleteAccount permanently deletes the authenticated user's account.
func (h *UserGRPCHandler) DeleteAccount(
	ctx context.Context,
	req *connect.Request[userv1.DeleteAccountReq],
) (*connect.Response[userv1.DeleteAccountRes], error) {
	// Implementation outline:
	// 1. Extract Authorization header from request.
	// 2. Validate header exists and has "Bearer " prefix.
	//    a. If not, log warn and return connect.CodeUnauthenticated error.
	// 3. Extract token string by trimming "Bearer " prefix.
	// 4. Create jwtutil.Config from h.cfg.JWT fields.
	// 5. Call jwtutil.ValidateAccessToken to get userID.
	//    a. If error, log warn and return connect.CodeUnauthenticated error.
	// 6. Get confirmation phrase from req.Msg.ConfirmationPhrase.
	// 7. Call h.svc.DeleteAccount with userID and confirmationPhrase.
	//    a. If error contains "confirmation phrase", return connect.CodeInvalidArgument.
	//    b. If error contains "user not found", return connect.CodeNotFound.
	//    c. If other error, log error and return connect.CodeInternal.
	// 8. Create response with result.Success and result.Message.
	// 9. Return connect.NewResponse with the response.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid deletion | Valid token, "DELETE" | Success response | Happy path |
| Missing auth header | No header | CodeUnauthenticated | Auth validation |
| Invalid token | Invalid JWT | CodeUnauthenticated | Token validation |
| Wrong confirmation | Valid token, "wrong" | CodeInvalidArgument | Confirmation check |
| User not found | Valid token (deleted user) | CodeNotFound | User check |
| Internal error | Valid token, DB failure | CodeInternal | Error handling |

---

## Step 11: Update User Module Container Registration

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: DI container patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`: Existing module registration

#### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`

**Description**:
Add CascadeDeleteRepository registration to user module.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/user"
	"github.com/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/user/outbound/repository/mongodb"
)

func newUserModule() fx.Option {
	return fx.Module("user",
		// User repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoUserRepository,
				fx.As(new(repository.UserRepositoryPort)),
			),
		),

		// Organization repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationRepository,
				fx.As(new(repository.OrganizationRepositoryPort)),
			),
		),

		// Cascade delete repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoCascadeDeleteRepository,
				fx.As(new(repository.CascadeDeleteRepositoryPort)),
			),
		),

		// Service
		fx.Provide(user.NewService),

		// ConnectRPC handler (private - requires auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewUserGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
```

---

## Step 12: Install shadcn Dialog Component

**Description**:
Install the shadcn dialog component needed for the confirmation modal.

```bash
cd /Users/jayce/team-attention/cops/web && npx shadcn@latest add dialog
```

This generates: `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/dialog.tsx`

---

## Step 13: Create useDeleteAccount Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: React hook patterns
- `/Users/jayce/team-attention/cops/web/src/feature/auth/hook/use-approve-device.ts`: Example mutation hook
- `/Users/jayce/team-attention/cops/web/src/feature/user/hook/use-get-me.ts`: Example query hook

#### `/Users/jayce/team-attention/cops/web/src/feature/user/hook/use-delete-account.ts`

**Description**:
Create a mutation hook for the DeleteAccount RPC.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { deleteAccount } from '@/gen/grpcstub/user/v1/user-UserService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useDeleteAccount provides a mutation hook for deleting the current user's account.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useDeleteAccount = () => {
  return useMutation(deleteAccount, { transport })
}
```

---

## Step 14: Create DeleteAccountDialog Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Component patterns
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript/React rules
- `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx`: Example component with mutation

#### `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx`

**Description**:
Create a dialog component for account deletion confirmation.

```typescript
import { useState, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, Loader2 } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useDeleteAccount } from '../hook/use-delete-account'
import { useAuth } from '@/shared/hook/use-auth'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/gen/shadcn/ui/dialog'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

// DeleteAccountDialogState represents the dialog's internal state.
type DeleteAccountDialogState =
  | { status: 'idle' }
  | { status: 'confirming' }
  | { status: 'error'; message: string }

interface DeleteAccountDialogProps {
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

export const DeleteAccountDialog = ({ trigger }: DeleteAccountDialogProps) => {
  // Implementation outline:
  // 1. useState for:
  //    - isOpen: boolean (dialog open state)
  //    - confirmationInput: string (user's typed confirmation)
  //    - state: DeleteAccountDialogState (idle, confirming, error)
  // 2. Get mutation from useDeleteAccount()
  // 3. Get logout from useAuth()
  // 4. Get reset from useUserStore()
  // 5. Get navigate from useNavigate()
  //
  // 6. Create handleConfirmationChange callback:
  //    a. Update confirmationInput state
  //    b. Clear any error state (set to idle)
  //
  // 7. Create handleDelete async callback:
  //    a. Set state to confirming
  //    b. Try:
  //       i. Call mutation.mutateAsync({ confirmationPhrase: confirmationInput })
  //       ii. Call logout() to clear tokens
  //       iii. Call reset() to clear user store
  //       iv. Navigate to '/' (home page)
  //    c. Catch (error):
  //       i. Map error.code to user-friendly message:
  //          - Code.InvalidArgument: "Please type 'DELETE' exactly to confirm"
  //          - Code.Unauthenticated: "Session expired. Please log in again."
  //          - Default: error.message or "An error occurred"
  //       ii. Set state to error with message
  //
  // 8. Create isConfirmationValid computed:
  //    a. Return confirmationInput === 'DELETE'
  //
  // 9. Create handleOpenChange callback:
  //    a. If closing (open is false), reset state to idle and clear confirmationInput
  //    b. Update isOpen state
  //
  // 10. Return Dialog with:
  //     - open={isOpen}
  //     - onOpenChange={handleOpenChange}
  //     - DialogTrigger with asChild wrapping {trigger}
  //     - DialogContent containing:
  //       a. DialogHeader with warning icon and title "Delete Account"
  //       b. DialogDescription explaining the permanent action and cascade deletion
  //       c. Warning alert with list of what will be deleted:
  //          - All your personal data and authentication accounts
  //          - Organizations where you are the sole member (with projects and sessions)
  //          - Your membership in shared organizations
  //       d. Input field for typing 'DELETE'
  //       e. Error alert (shown when state.status === 'error')
  //       f. DialogFooter with:
  //          - Cancel button (type="button", variant="outline")
  //          - Delete button (variant="destructive", disabled when invalid or confirming)
}
```

**Component Props Interface**:

```typescript
interface DeleteAccountDialogProps {
  trigger: React.ReactNode
}
```

---

## Step 15: Update Settings Page with Danger Zone

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Page component patterns
- `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`: Existing settings page
- `/Users/jayce/team-attention/cops/web/src/feature/auth/component/device-approval.tsx`: Card styling patterns

#### `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`

**Description**:
Replace placeholder content with Danger Zone section containing delete account functionality.

```typescript
import { createFileRoute } from '@tanstack/react-router'
import { Settings, Trash2 } from 'lucide-react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Button } from '@/gen/shadcn/ui/button'
import { DeleteAccountDialog } from '@/feature/user/component/delete-account-dialog'

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
})

// SettingsPage displays account settings including the danger zone.
function SettingsPage() {
  // Implementation outline:
  // 1. Return page layout with:
  //    a. Header section with Settings icon and title "Account Settings"
  //    b. Danger Zone section:
  //       i. Card with red/destructive border styling (border-red-900/50)
  //       ii. CardHeader with:
  //           - Trash2 icon (text-red-400)
  //           - CardTitle "Danger Zone"
  //           - CardDescription explaining permanent actions
  //       iii. CardContent with:
  //           - Delete Account subsection:
  //             * Heading "Delete Account"
  //             * Description text explaining consequences
  //             * DeleteAccountDialog with Button trigger (variant="destructive")
  //    c. Footer with version number
}
```

**Layout Structure**:

```
SettingsPage
+-- Header
|   +-- Settings Icon
|   +-- Title: "Account Settings"
|   +-- Subtitle: "Manage your account"
|
+-- Danger Zone Card (border-red-900/50 bg-red-950/10)
|   +-- CardHeader
|   |   +-- Trash2 Icon (text-red-400)
|   |   +-- CardTitle: "Danger Zone"
|   |   +-- CardDescription
|   |
|   +-- CardContent
|       +-- Delete Account Section
|           +-- Heading: "Delete Account"
|           +-- Description paragraph
|           +-- DeleteAccountDialog
|               +-- Button (trigger): "Delete Account"
|
+-- Footer
    +-- Version: "C-Ops v0.1.0"
```

---

## Step 16: Create User Feature Directory Structure

**Description**:
Ensure the user feature directory has the proper structure.

Final structure after implementation:

```
web/src/feature/user/
+-- component/
|   +-- delete-account-dialog.tsx
+-- hook/
    +-- use-get-me.ts (existing)
    +-- use-delete-account.ts
```

---

## Implementation Order Summary

1. **Step 1**: Update `user.proto` with DeleteAccount RPC
2. **Step 2**: Run `buf generate` to generate code
3. **Step 3**: Extend `UserRepositoryPort` with Delete method
4. **Step 4**: Implement `MongoUserRepository.Delete`
5. **Step 5**: Extend `OrganizationRepositoryPort` with new methods
6. **Step 6**: Implement organization repository methods
7. **Step 7**: Create `CascadeDeleteRepositoryPort`
8. **Step 8**: Implement `MongoCascadeDeleteRepository`
9. **Step 9**: Implement `Service.DeleteAccount`
10. **Step 10**: Implement `UserGRPCHandler.DeleteAccount`
11. **Step 11**: Update `module_user.go` container registration
12. **Step 12**: Install shadcn dialog component
13. **Step 13**: Create `useDeleteAccount` hook
14. **Step 14**: Create `DeleteAccountDialog` component
15. **Step 15**: Update Settings page with Danger Zone

---

## File Summary

### Files to Create

| File | Purpose |
| :--- | :------ |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/cascade_delete_repo_port.go` | Port interface for cascade deletion |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/cascade_delete_repo.go` | MongoDB implementation for cascade deletion |
| `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx` | Confirmation dialog component |
| `/Users/jayce/team-attention/cops/web/src/feature/user/hook/use-delete-account.ts` | Mutation hook for delete account |

### Files to Modify

| File | Changes |
| :--- | :------ |
| `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto` | Add DeleteAccountReq, DeleteAccountRes, DeleteAccount RPC |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/user_repo_port.go` | Add Delete method |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/user_repo.go` | Implement Delete method |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go` | Add new types and methods |
| `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go` | Implement new methods |
| `/Users/jayce/team-attention/cops/api/internal/service/user/user_service.go` | Add DeleteAccount method, update constructor |
| `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go` | Add DeleteAccount handler |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go` | Add cascade delete repo registration |
| `/Users/jayce/team-attention/cops/web/src/route/settings.tsx` | Replace placeholder with Danger Zone |

### Files to Generate (via commands)

| File | Generated By |
| :--- | :----------- |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/user.pb.go` | `buf generate` |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect/user.connect.go` | `buf generate` |
| `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/user/v1/user_pb.ts` | `buf generate` |
| `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/user/v1/user-UserService_connectquery.ts` | `buf generate` |
| `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/dialog.tsx` | `npx shadcn@latest add dialog` |
