# Implementation Plan: Full-Stack User Data and Zustand State Management

## Overview

This implementation adds real user data display in the dashboard sidebar, replacing the current mocked "Code Operator" data. The feature involves:

1. **Backend**: A new `user.v1.UserService` gRPC service with `GetMe` RPC endpoint that extracts user ID from JWT token, fetches user data and organizations from MongoDB, and returns the combined result.

2. **Frontend**: Zustand state management for user data and selected organization, with hooks for data fetching and UI updates to the `SidebarUser` component.

The data flow is:
```
App Start (Authenticated)
  -> useUser hook calls GetMe RPC
  -> Backend extracts userID from JWT
  -> Backend queries MongoDB (users + organizations with embedded members)
  -> Response stored in Zustand
  -> SidebarUser renders real data
```

---

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | Frontend state management for user data and selected organization | `zustand` | Lightweight, TypeScript-friendly state management with persist middleware for selected organization |

---

## Step 1a: Create Domain Proto Definition

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions and patterns
- `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto`: Example proto file structure

### `idl/protobuf/domain/v1/domain.proto`

**Description**:
Define reusable domain models (`User`, `Organization`, `OrganizationMember`) in a separate module so they can be imported by other services. This enables consistent domain representation across the API.

```protobuf
syntax = "proto3";

package domain.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1;domainv1";

// User represents an authenticated user in the system.
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  string avatar_url = 4;
}

// OrganizationMember represents a user's membership within an organization.
message OrganizationMember {
  string user_id = 1;
  string role = 2;  // "admin" or "member"
}

// Organization represents a group that owns projects.
// Members are embedded within the organization document.
message Organization {
  string id = 1;
  string name = 2;
  string slug = 3;
  repeated OrganizationMember members = 4;
}
```

---

## Step 1b: Create User Service Proto Definition

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions and patterns
- `/Users/jayce/team-attention/cops/idl/protobuf/domain/v1/domain.proto`: Domain models (created in Step 1a)

### `idl/protobuf/user/v1/user.proto`

**Description**:
Define the `UserService` with `GetMe` RPC endpoint. Import domain models from `domain/v1/domain.proto`. The response uses `domain.v1.User` for user data and a local `UserOrganization` message for the user's organizations with their role (since we don't need to return all members of each organization).

```protobuf
syntax = "proto3";

package user.v1;

import "domain/v1/domain.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/user/v1;userv1";

// UserOrganization represents a user's membership in an organization.
// This is a projection that includes the user's role without exposing all members.
message UserOrganization {
  string id = 1;
  string name = 2;
  string role = 3;  // "admin" or "member"
}

// GetMeReq is empty - user ID is extracted from JWT token.
message GetMeReq {}

// GetMeRes contains the authenticated user's data and organizations.
message GetMeRes {
  domain.v1.User user = 1;
  repeated UserOrganization organizations = 2;
}

// UserService handles user-related operations.
service UserService {
  // GetMe returns the authenticated user's information and organizations.
  // Requires valid JWT token in Authorization header.
  rpc GetMe(GetMeReq) returns (GetMeRes);
}
```

After creating both proto files, run:
```bash
cd idl/protobuf && buf generate
```

---

## Step 2: Update Domain Organization Model (Embedded Members)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-domain.md`: Domain model patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`: Struct pointer vs value rules
- `/Users/jayce/team-attention/cops/shared/domain/organization.go`: Current organization domain model

### `shared/domain/organization.go`

**Description**:
Update the `Organization` domain model to include an embedded `Members` slice. Refactor `OrganizationMember` struct to represent an embedded member entry (with `UserID` and `Role` fields only, removing the separate ID and OrganizationID fields).

```go
package domain

// MemberRole represents the role of a user within an organization.
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// OrganizationMember represents a member entry within an organization.
type OrganizationMember struct {
	UserID ID         `json:"userId" bson:"userId"`
	Role   MemberRole `json:"role" bson:"role"`
}

// Organization represents a group that owns projects.
// Members are embedded as an array within the organization document.
type Organization struct {
	ID      ID                    `json:"id" bson:"-"`
	Name    string                `json:"name" bson:"name"`
	Slug    string                `json:"slug" bson:"slug"`
	Members []*OrganizationMember `json:"members" bson:"members"`
}
```

---

## Step 3: Update MongoDB Organization Schema (Embedded Members)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-domain-mongoschema.md`: MongoDB schema patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: Current organization schema

### `shared/domain/mongoschema/organization.go`

**Description**:
Update the MongoDB schema to support embedded members array within the organization document. Add field constants for member queries.

```go
package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	OrganizationCollectionName = "organizations"
)

const (
	OrganizationIDField      = "_id"
	OrganizationNameField    = "name"
	OrganizationSlugField    = "slug"
	OrganizationMembersField = "members"
)

// OrganizationMember field constants for nested queries.
const (
	OrganizationMemberUserIDField = "userId"
	OrganizationMemberRoleField   = "role"
)

// OrganizationMember represents a member entry within an organization document.
type OrganizationMember struct {
	UserID bson.ObjectID     `bson:"userId"`
	Role   domain.MemberRole `bson:"role"`
}

// ToDomain converts schema OrganizationMember to domain OrganizationMember.
func (s *OrganizationMember) ToDomain() *domain.OrganizationMember {
	if s == nil {
		return nil
	}
	return &domain.OrganizationMember{
		UserID: domain.ID(s.UserID.Hex()),
		Role:   s.Role,
	}
}

// FromDomain converts domain OrganizationMember to schema OrganizationMember.
func (s *OrganizationMember) FromDomain(d *domain.OrganizationMember) {
	if d == nil {
		return
	}
	if d.UserID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(string(d.UserID))
	}
	s.Role = d.Role
}

type Organization struct {
	ID      bson.ObjectID         `bson:"_id,omitempty"`
	Name    string                `bson:"name"`
	Slug    string                `bson:"slug"`
	Members []*OrganizationMember `bson:"members"`
}

func (s *Organization) FromDomain(d *domain.Organization) {
	if d == nil {
		return
	}

	s.Name = d.Name
	s.Slug = d.Slug

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	// Convert organization members
	if d.Members != nil {
		s.Members = make([]*OrganizationMember, len(d.Members))
		for i, m := range d.Members {
			s.Members[i] = &OrganizationMember{}
			s.Members[i].FromDomain(m)
		}
	}
}

func (s *Organization) ToDomain() *domain.Organization {
	if s == nil {
		return nil
	}

	org := &domain.Organization{
		ID:   domain.ID(s.ID.Hex()),
		Name: s.Name,
		Slug: s.Slug,
	}

	// Convert organization members
	if s.Members != nil {
		org.Members = make([]*domain.OrganizationMember, len(s.Members))
		for i, m := range s.Members {
			org.Members[i] = m.ToDomain()
		}
	}

	return org
}
```

---

## Step 4: Delete Organization Member Schema (No Longer Needed)

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization_member.go`: Current schema to be deleted

### Delete `shared/domain/mongoschema/organization_member.go`

**Description**:
Delete this file as organization members are now embedded within the organization document. The separate `organization_members` collection is no longer used.

---

## Step 5: Create User Service Outbound Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter pattern
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/repository/user_repo_port.go`: Existing user repository port

### `api/internal/service/user/outbound/repository/user_repo_port.go`

**Description**:
Define port interface for user repository. Reuse the existing user domain model.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserRepositoryPort defines interface for user data retrieval.
type UserRepositoryPort interface {
	// GetByID retrieves a user by their ID.
	// Returns nil, nil if user not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, userID string) (*domain.User, error)
}
```

---

## Step 6: Create User Service Outbound Repository MongoDB Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter implementation
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger binding patterns
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb/user_repo.go`: Existing user repository implementation

### `api/internal/service/user/outbound/repository/mongodb/user_repo.go`

**Description**:
MongoDB adapter implementing `UserRepositoryPort`. Uses existing `mongoschema.User` for data mapping.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoUserRepository implements UserRepositoryPort for MongoDB.
type MongoUserRepository struct {
	logger    *slog.Logger
	usersColl *mongo.Collection
}

// NewMongoUserRepository creates a new MongoDB user repository.
func NewMongoUserRepository(l *slog.Logger, db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		logger:    l.With(slog.String("name", "user.repository.mongodb.user")),
		usersColl: db.Collection(mongoschema.UserCollectionName),
	}
}

// GetByID retrieves a user by their ID.
func (r *MongoUserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID using bson.ObjectIDFromHex.
	// 2. If conversion fails, return nil, error.
	// 3. Create filter with _id field.
	// 4. Execute FindOne query on users collection.
	// 5. If mongo.ErrNoDocuments, return nil, nil.
	// 6. If other error, log and return nil, error.
	// 7. Convert mongoschema.User to domain.User using ToDomain().
	// 8. Return user, nil.
}

// Interface verification
var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid user ID exists | `"507f1f77bcf86cd799439011"` | `*domain.User, nil` | Happy path |
| User not found | `"507f1f77bcf86cd799439012"` | `nil, nil` | ErrNoDocuments branch |
| Invalid ObjectID format | `"invalid-id"` | `nil, error` | ObjectID conversion error |
| Database connection error | valid ID with DB down | `nil, error` | Database error branch |

---

## Step 7: Create Organization Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter pattern

### `api/internal/service/user/outbound/repository/organization_repo_port.go`

**Description**:
Define port interface for querying user's organizations with their roles. Uses embedded members approach.

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

// OrganizationRepositoryPort defines interface for organization queries.
type OrganizationRepositoryPort interface {
	// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
	// Queries organizations collection filtering by embedded members.userId.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizations(ctx context.Context, userID string) ([]*UserOrganization, error)
}
```

---

## Step 8: Create Organization Repository MongoDB Adapter (Embedded Members)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter implementation
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger binding patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: Updated organization schema with embedded members

### `api/internal/service/user/outbound/repository/mongodb/organization_repo.go`

**Description**:
MongoDB adapter implementing `OrganizationRepositoryPort`. Queries organizations collection directly using `$elemMatch` on the embedded members array to find organizations where the user is a member.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
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
	return &MongoOrganizationRepository{
		logger:  l.With(slog.String("name", "user.repository.mongodb.organization")),
		orgColl: db.Collection(mongoschema.OrganizationCollectionName),
	}
}

// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
func (r *MongoOrganizationRepository) GetUserOrganizations(ctx context.Context, userID string) ([]*repository.UserOrganization, error) {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID.
	// 2. If conversion fails, return nil, error.
	// 3. Build filter to find organizations where user is a member:
	//    filter := bson.M{
	//        "members": bson.M{
	//            "$elemMatch": bson.M{
	//                "userId": userObjectID,
	//            },
	//        },
	//    }
	// 4. Execute Find query on organizations collection.
	// 5. Iterate cursor, decode each result to mongoschema.Organization.
	// 6. For each organization:
	//    a. Convert to domain.Organization using ToDomain().
	//    b. Find the user's membership entry in Members slice.
	//    c. Extract the user's role from that entry.
	//    d. Create repository.UserOrganization with org and role.
	// 7. Return slice of UserOrganization, nil.
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| User with multiple organizations | valid userID | `[]*UserOrganization` with 2+ items | Happy path |
| User with single organization | valid userID | `[]*UserOrganization` with 1 item | Single org |
| User with no organizations | valid userID | empty slice `[]*UserOrganization{}` | No memberships |
| Invalid userID format | `"invalid"` | `nil, error` | ObjectID error |
| Database error | valid ID with DB down | `nil, error` | Database error |

---

## Step 9: Create User Service

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service structure patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging patterns
- `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go`: Example service implementation

### `api/internal/service/user/user_service.go`

**Description**:
Core user service implementing `GetMe` business logic. Orchestrates user and organization data retrieval.

```go
package user

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// GetMeResult contains the user data and their organizations.
type GetMeResult struct {
	User          *domain.User
	Organizations []*repository.UserOrganization
}

// Service implements user business logic.
type Service struct {
	logger   *slog.Logger
	userRepo repository.UserRepositoryPort
	orgRepo  repository.OrganizationRepositoryPort
}

// NewService creates a new user service.
func NewService(
	l *slog.Logger,
	userRepo repository.UserRepositoryPort,
	orgRepo repository.OrganizationRepositoryPort,
) *Service {
	return &Service{
		logger:   l.With(slog.String("name", "user.service")),
		userRepo: userRepo,
		orgRepo:  orgRepo,
	}
}

// GetMe retrieves the authenticated user's information and organizations.
func (s *Service) GetMe(ctx context.Context, userID string) (*GetMeResult, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	//    - If empty, log warning and return error.
	// 2. Call userRepo.GetByID to fetch user data.
	//    - If error, log error and return nil, error.
	//    - If user is nil (not found), log info and return error "user not found".
	// 3. Call orgRepo.GetUserOrganizations to fetch user's organizations.
	//    - If error, log error and return nil, error.
	// 4. Log successful retrieval with user ID and organization count.
	// 5. Return GetMeResult with user and organizations.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid user with organizations | valid userID | `*GetMeResult` with user and orgs | Happy path |
| Valid user without organizations | valid userID | `*GetMeResult` with user, empty orgs | No orgs |
| Empty userID | `""` | `nil, error` | Validation error |
| User not found | non-existent userID | `nil, error "user not found"` | User not found |
| User repo error | valid ID, DB error | `nil, error` | User repo error |
| Org repo error | valid ID, DB error on orgs | `nil, error` | Org repo error |

---

## Step 10: Create User Service ConnectRPC Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md`: Inbound adapter structure
- `/Users/jayce/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: Example handler with JWT extraction

### `api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
ConnectRPC handler for UserService. Extracts userID from JWT token in Authorization header and calls the service. Maps service result to protobuf response using imported `domain.v1.User` type.

```go
package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/service/user"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	userv1 "github.com/team-attention/cops/shared/gen/grpcstub/user/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect"
)

// UserGRPCHandler handles gRPC requests for user service.
type UserGRPCHandler struct {
	svc    *user.Service
	logger *slog.Logger
	cfg    *config.Config
}

// NewUserGRPCHandler creates a new user gRPC handler.
func NewUserGRPCHandler(l *slog.Logger, svc *user.Service, cfg *config.Config) *UserGRPCHandler {
	return &UserGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "user.grpc.connectrpc")),
		cfg:    cfg,
	}
}

// GetHandler implements ConnectHandler interface.
func (h *UserGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return userv1connect.NewUserServiceHandler(h, opts...)
}

// GetMe returns the authenticated user's information and organizations.
func (h *UserGRPCHandler) GetMe(
	ctx context.Context,
	req *connect.Request[userv1.GetMeReq],
) (*connect.Response[userv1.GetMeRes], error) {
	// Implementation outline:
	// 1. Extract Authorization header from request.
	// 2. Validate header exists and has "Bearer " prefix.
	//    - If invalid, return connect.CodeUnauthenticated error.
	// 3. Extract token string by trimming "Bearer " prefix.
	// 4. Create jwtutil.Config from h.cfg.JWT fields.
	// 5. Call jwtutil.ValidateAccessToken to get userID.
	//    - If error, log warning and return connect.CodeUnauthenticated error.
	// 6. Call h.svc.GetMe with userID.
	//    - If error contains "user not found", return connect.CodeNotFound.
	//    - If other error, log error and return connect.CodeInternal.
	// 7. Convert service result to protobuf response:
	//    a. Map domain.User to domainv1.User:
	//       - ID, Email, Name, AvatarUrl (ProfileImageURL -> AvatarUrl)
	//    b. Map each UserOrganization to userv1.UserOrganization:
	//       - Organization.ID, Organization.Name, Role
	// 8. Return connect.NewResponse with the response.
}

// Interface verification
var _ userv1connect.UserServiceHandler = (*UserGRPCHandler)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid JWT token | valid Bearer token | `*GetMeRes` with user data | Happy path |
| Missing Authorization header | no header | `CodeUnauthenticated` | Missing header |
| Invalid header format | "Basic token" | `CodeUnauthenticated` | Invalid format |
| Expired/invalid JWT | invalid token | `CodeUnauthenticated` | Token validation error |
| User not found | valid JWT, no user | `CodeNotFound` | User not found |
| Service error | valid JWT, DB error | `CodeInternal` | Service error |

---

## Step 11: Register User Service in DI Container

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`: Example module registration

### `api/cmd/internal/container/module_user.go`

**Description**:
Register user service components in the fx DI container.

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

		// Service
		fx.Provide(user.NewService),

		// ConnectRPC handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewUserGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
```

### `api/cmd/internal/container/application.go`

**Description**:
Add `newUserModule()` to the application's fx.Options.

```go
// Add to existing New() function's fx.Options:
newUserModule(),
```

---

## Step 12: Update RBAC Repository to Use Embedded Members

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`: Current implementation
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go`: Port interface

### `api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`

**Description**:
Update to query organizations collection with embedded members instead of the now-deleted organization_members collection.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoOrganizationMemberRepository implements OrganizationMemberRepositoryPort for MongoDB.
type MongoOrganizationMemberRepository struct {
	logger  *slog.Logger
	orgColl *mongo.Collection
}

// NewMongoOrganizationMemberRepository creates a new MongoDB organization member repository.
func NewMongoOrganizationMemberRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationMemberRepository {
	return &MongoOrganizationMemberRepository{
		logger:  l.With(slog.String("name", "rbac.repository.mongodb.organization_member")),
		orgColl: db.Collection(mongoschema.OrganizationCollectionName),
	}
}

// IsMember checks if a user is a member of an organization.
func (r *MongoOrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID.
	// 2. If conversion fails, return false, error.
	// 3. Convert organizationID string to bson.ObjectID.
	// 4. If conversion fails, return false, error.
	// 5. Build filter to find organization with specific ID and user in members:
	//    filter := bson.M{
	//        "_id": orgObjectID,
	//        "members": bson.M{
	//            "$elemMatch": bson.M{
	//                "userId": userObjectID,
	//            },
	//        },
	//    }
	// 6. Execute CountDocuments query on organizations collection.
	// 7. If error, log and return false, error.
	// 8. Return count > 0, nil.
}

// Interface verification
var _ repository.OrganizationMemberRepositoryPort = (*MongoOrganizationMemberRepository)(nil)
```

---

## Step 13: Install Zustand Package

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/common.md`: Dependency installation rules

**Command**:
```bash
cd web && npm install zustand
```

---

## Step 14: Create Zustand User Store

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Frontend directory structure
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript and React rules

### `web/src/shared/store/user-store.ts`

**Description**:
Zustand store for managing user state and selected organization. Uses persist middleware for selected organization.

```typescript
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// UserData represents the authenticated user's information.
interface UserData {
  id: string
  email: string
  name: string
  avatarUrl: string
}

// OrganizationData represents a user's organization membership.
interface OrganizationData {
  id: string
  name: string
  role: 'admin' | 'member'
}

// UserStoreState defines the state shape.
interface UserStoreState {
  user: UserData | null
  organizations: OrganizationData[]
  selectedOrganizationId: string | null
  isLoading: boolean
  error: string | null
}

// UserStoreActions defines the available actions.
interface UserStoreActions {
  setUser: (user: UserData | null) => void
  setOrganizations: (organizations: OrganizationData[]) => void
  setSelectedOrganizationId: (id: string | null) => void
  setLoading: (isLoading: boolean) => void
  setError: (error: string | null) => void
  reset: () => void
}

type UserStore = UserStoreState & UserStoreActions

const initialState: UserStoreState = {
  user: null,
  organizations: [],
  selectedOrganizationId: null,
  isLoading: false,
  error: null,
}

export const useUserStore = create<UserStore>()(
  persist(
    (set) => ({
      ...initialState,

      setUser: (user) => set({ user }),

      setOrganizations: (organizations) =>
        set((state) => ({
          organizations,
          // Auto-select first organization if none selected or current selection invalid
          selectedOrganizationId:
            state.selectedOrganizationId &&
            organizations.some((org) => org.id === state.selectedOrganizationId)
              ? state.selectedOrganizationId
              : organizations.length > 0
                ? organizations[0].id
                : null,
        })),

      setSelectedOrganizationId: (selectedOrganizationId) =>
        set({ selectedOrganizationId }),

      setLoading: (isLoading) => set({ isLoading }),

      setError: (error) => set({ error }),

      reset: () => set(initialState),
    }),
    {
      name: 'cops-user-storage',
      partialize: (state) => ({
        selectedOrganizationId: state.selectedOrganizationId,
      }),
    }
  )
)
```

---

## Step 15: Create useGetMe Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Hook patterns for gRPC
- `/Users/jayce/team-attention/cops/web/src/feature/dashboard/hook/use-get-overview.ts`: Example hook

### `web/src/feature/user/hook/use-get-me.ts`

**Description**:
Hook wrapping the GetMe RPC call using TanStack Query.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { getMe } from '@/gen/grpcstub/user/v1/user-UserService_connectquery'
import { transport } from '@/shared/service/connect-transport'

export const useGetMe = (options?: { enabled?: boolean }) => {
  return useQuery(getMe, {}, { transport, ...options })
}
```

---

## Step 16: Create useUser Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Shared hook patterns
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-auth.ts`: Example shared hook

### `web/src/shared/hook/use-user.ts`

**Description**:
Hook that combines Zustand store access with data fetching. Syncs RPC response to Zustand store.

```typescript
import { useEffect } from 'react'
import { useUserStore } from '@/shared/store/user-store'
import { useGetMe } from '@/feature/user/hook/use-get-me'
import { useAuth } from '@/shared/hook/use-auth'

// useUser provides access to user state and handles data synchronization.
// Automatically fetches user data when authenticated and syncs to Zustand.
export const useUser = () => {
  const { isAuthenticated } = useAuth()

  const {
    user,
    organizations,
    selectedOrganizationId,
    isLoading,
    error,
    setUser,
    setOrganizations,
    setLoading,
    setError,
    setSelectedOrganizationId,
    reset,
  } = useUserStore()

  const {
    data,
    isLoading: isQueryLoading,
    isError,
    error: queryError,
    refetch,
  } = useGetMe({ enabled: isAuthenticated })

  // Sync query state to Zustand store
  useEffect(() => {
    setLoading(isQueryLoading)
  }, [isQueryLoading, setLoading])

  useEffect(() => {
    if (isError && queryError) {
      setError(queryError.message)
    } else {
      setError(null)
    }
  }, [isError, queryError, setError])

  useEffect(() => {
    if (data) {
      // Map protobuf User to UserData
      if (data.user) {
        setUser({
          id: data.user.id,
          email: data.user.email,
          name: data.user.name,
          avatarUrl: data.user.avatarUrl,
        })
      }

      // Map protobuf Organizations to OrganizationData[]
      const orgs = data.organizations.map((org) => ({
        id: org.id,
        name: org.name,
        role: org.role as 'admin' | 'member',
      }))
      setOrganizations(orgs)
    }
  }, [data, setUser, setOrganizations])

  // Get selected organization object
  const selectedOrganization = organizations.find(
    (org) => org.id === selectedOrganizationId
  )

  return {
    user,
    organizations,
    selectedOrganization,
    selectedOrganizationId,
    isLoading,
    error,
    setSelectedOrganizationId,
    refetch,
    reset,
  }
}
```

---

## Step 17: Update SidebarUser Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/component/sidebar-user.tsx`: Current component
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: Component patterns

### `web/src/shared/component/sidebar-user.tsx`

**Description**:
Update to use real user data from useUser hook. Add organization switcher, loading state, and error handling.

```tsx
import { useNavigate } from '@tanstack/react-router'
import { Settings, LogOut, ChevronDown, Building2, RefreshCw, AlertCircle } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/gen/shadcn/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from '@/gen/shadcn/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/gen/shadcn/ui/sidebar'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { useAuth } from '@/shared/hook/use-auth'
import { useUser } from '@/shared/hook/use-user'

// getInitials extracts initials from name or email for avatar fallback.
const getInitials = (name: string | undefined, email: string | undefined): string => {
  // Implementation outline:
  // 1. If name exists and has length > 0:
  //    a. Split name by spaces.
  //    b. Take first character of first word and first character of last word.
  //    c. Return uppercase initials (max 2 characters).
  // 2. If email exists and has length > 0:
  //    a. Take first 2 characters of email before @ symbol.
  //    b. Return uppercase.
  // 3. Return "U" as default.
}

// getDisplayName returns the best available display name.
const getDisplayName = (name: string | undefined, email: string | undefined): string => {
  // Implementation outline:
  // 1. If name exists and has length > 0, return name.
  // 2. If email exists and has length > 0, return email.
  // 3. Return "User" as default.
}

// SidebarUser displays user info with dropdown menu for Settings and Logout.
// Dropdown appears above the button since user button is at bottom of sidebar.
export const SidebarUser = () => {
  const navigate = useNavigate()
  const { logout } = useAuth()
  const {
    user,
    organizations,
    selectedOrganization,
    selectedOrganizationId,
    isLoading,
    error,
    setSelectedOrganizationId,
    refetch,
    reset,
  } = useUser()

  const handleSettingsClick = () => {
    navigate({ to: '/settings' })
  }

  const handleLogoutClick = () => {
    reset()
    logout()
    navigate({ to: '/' })
  }

  const handleRetryClick = () => {
    refetch()
  }

  const handleOrganizationChange = (orgId: string) => {
    setSelectedOrganizationId(orgId)
  }

  // Render loading state
  // Implementation: Show Skeleton components for avatar and text

  // Render error state
  // Implementation: Show AlertCircle icon with retry button

  // Render normal state with user data
  // Implementation:
  // 1. Display Avatar with AvatarImage (if avatarUrl exists) and AvatarFallback (initials)
  // 2. Display getDisplayName() for user name
  // 3. Display selectedOrganization?.name or "No organization" as secondary text
  // 4. DropdownMenu contains:
  //    a. User email display (if available)
  //    b. Organization switcher submenu (if organizations.length > 1)
  //    c. Settings menu item
  //    d. Separator
  //    e. Logout menu item
}
```

**UI States to Handle**:

1. **Loading State**: Show skeleton animation for avatar and text areas
2. **Error State**: Show error icon with retry button, hide organization info
3. **Normal State**: Show user avatar, name, and organization with dropdown
4. **No Organizations**: Show "No organization" in secondary text, hide org switcher

---

## Step 18: Add DropdownMenu Subcomponents (if needed)

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/shadcn/ui/dropdown-menu.tsx`: Check if submenu components exist

**Command** (if submenu components are missing):
```bash
cd web && npx shadcn@latest add dropdown-menu
```

This will update the dropdown-menu component with the latest version including SubMenu components.

---

## Step 19: Initialize User Data Fetch on App Startup

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/__root.tsx`: Root component
- `/Users/jayce/team-attention/cops/web/src/shared/component/app-sidebar.tsx`: AppSidebar component

### Option A: Add to AppSidebar component

**Description**:
Initialize user data fetching when the sidebar mounts (only for authenticated routes).

```tsx
// In app-sidebar.tsx, add at the top of the component:
import { useUser } from '@/shared/hook/use-user'

export const AppSidebar = () => {
  // This triggers the useGetMe query when authenticated
  useUser()

  // ... rest of component
}
```

This approach works because:
1. AppSidebar only renders for non-auth routes (per `__root.tsx` logic)
2. useUser hook has `enabled: isAuthenticated` which prevents fetching when not authenticated
3. Data is fetched once and synced to Zustand store

---

## Quality Checklist

- [x] Every function has a concrete signature (not "something like X")
- [x] Detailed algorithm explanation included as comments in the body of every function
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Execute Agent
- [x] All packages are selected (zustand)
- [x] Execution order is clear and dependencies are explicit

---

## Implementation Order Summary

1. **Step 1a**: Create domain proto file `idl/protobuf/domain/v1/domain.proto`
2. **Step 1b**: Create user proto file `idl/protobuf/user/v1/user.proto` (imports domain), run `buf generate`
3. **Step 2**: Update `shared/domain/organization.go` with embedded members
4. **Step 3**: Update `shared/domain/mongoschema/organization.go` with embedded members schema
5. **Step 4**: Delete `shared/domain/mongoschema/organization_member.go`
6. **Step 5-8**: Create backend repository ports and adapters for user service
7. **Step 9**: Create user service
8. **Step 10**: Create ConnectRPC handler
9. **Step 11**: Register in DI container, update `application.go`
10. **Step 12**: Update RBAC repository to use embedded members
11. **Step 13**: Install zustand package
12. **Step 14**: Create Zustand user store
13. **Step 15**: Create useGetMe hook
14. **Step 16**: Create useUser hook
15. **Step 17**: Update SidebarUser component
16. **Step 18**: Ensure dropdown submenu components available
17. **Step 19**: Initialize user data fetch in AppSidebar

---

## Key Changes from Original Plan

1. **Domain Proto Module**: Created `idl/protobuf/domain/v1/domain.proto` with reusable `User`, `Organization`, and `OrganizationMember` messages that can be imported by other services.

2. **Embedded Members**: Organization members are now embedded within the Organization document instead of using a separate `organization_members` collection:
   - Added `OrganizationMember` struct to domain and mongoschema
   - Organization now has a `Members []*OrganizationMember` field
   - MongoDB queries use `$elemMatch` on the embedded members array
   - Deleted the separate `organization_member.go` mongoschema file

3. **Simplified Queries**: No more aggregation pipelines with `$lookup` - direct queries on the organizations collection with member filtering.

4. **Updated RBAC Repository**: The existing RBAC service's organization member repository also needs updating to work with embedded members.
