# RBAC Service Implementation Plan

## Overview

This plan implements a Role-Based Access Control (RBAC) service that verifies whether authenticated users have permission to access specific projects. The service follows the hexagonal architecture pattern and will be placed under `api/internal/service/core/rbac/` as a cross-cutting service that can be injected into other services.

The core logic is:
1. A project belongs to an organization (via `OrganizationID` field)
2. A user can access a project if they are a member of the project's organization
3. The service provides a simple binary access check: `CanAccess(ctx, userID, projectID) (bool, error)`

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | No external packages needed | - | All required functionality is available via existing MongoDB driver and standard library |

## Step 1: Add OrganizationID Field to Project Domain Models

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`: Pointer vs value type rules for struct fields
- `/Users/jayce/team-attention/cops/shared/domain/project.go`: Current Project domain model
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Current Project MongoDB schema

### `/Users/jayce/team-attention/cops/shared/domain/project.go`

**Description**:
Add `OrganizationID` field to the `Project` struct. This field is required (value type) since all projects must belong to an organization.

```go
// Project represents a registered project for session tracking.
// Embeds ProjectAbstract for basic identification.
type Project struct {
	ProjectAbstract
	OrganizationID ID        `json:"organizationId"` // Organization that owns this project (required)
	IsGitProject   bool      `json:"gitProject"`     // true if git repo, false otherwise
	RegisteredAt   time.Time `json:"registeredAt"`   // When the project was registered
}
```

### `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`

**Description**:
Add `OrganizationID` field constant and update the MongoDB schema struct with proper BSON handling.

```go
const (
	ProjectIDField             = "_id"
	ProjectNameField           = "name"
	ProjectPathField           = "path"
	ProjectIsGitProjectField   = "isGitProject"
	ProjectRegisteredAtField   = "registeredAt"
	ProjectGitBranchField      = "git_branch"
	ProjectRemoteURLField      = "remoteUrl"
	ProjectOrganizationIDField = "organizationId" // New field constant
)

type Project struct {
	domain.Project `bson:",inline"`
	ID             bson.ObjectID `bson:"_id,omitempty"`
	OrganizationID bson.ObjectID `bson:"organizationId"` // New field (required)
}

// FromDomain converts domain.Project to MongoDB schema.
func (s *Project) FromDomain(d *domain.Project) {
	// Implementation outline:
	// 1. Check if d is nil, return early if so.
	// 2. Copy domain.Project to embedded field.
	// 3. Convert ID string to bson.ObjectID if not empty.
	// 4. Convert OrganizationID string to bson.ObjectID if not empty.
}

// ToDomain converts MongoDB schema to domain.Project.
func (s *Project) ToDomain() *domain.Project {
	// Implementation outline:
	// 1. Check if s is nil, return nil if so.
	// 2. Set domain ID from bson.ObjectID.
	// 3. Set domain OrganizationID from bson.ObjectID.
	// 4. Return pointer to embedded domain.Project.
}
```

---

## Step 2: Create RBAC Service Outbound Repository Ports

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter fundamentals
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/repository/user_repo_port.go`: Example repository port
- `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go`: Example repository port

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/project_repo_port.go`

**Description**:
Define the port interface for querying project data needed by RBAC service. Only includes the method required for access checks.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// ProjectRepositoryPort defines interface for project data access needed by RBAC.
type ProjectRepositoryPort interface {
	// GetByID retrieves a project by its ID.
	// Returns nil, nil if project not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, projectID string) (*domain.Project, error)
}
```

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go`

**Description**:
Define the port interface for querying organization membership.

```go
package repository

import "context"

// OrganizationMemberRepositoryPort defines interface for organization membership queries.
type OrganizationMemberRepositoryPort interface {
	// IsMember checks if a user is a member of an organization.
	// Returns true if membership exists (any role: admin or member).
	// Returns false if no membership found.
	// Returns false, error if database error occurs.
	IsMember(ctx context.Context, userID, organizationID string) (bool, error)
}
```

---

## Step 3: Create RBAC Service MongoDB Repository Adapters

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging patterns
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb/user_repo.go`: Example MongoDB adapter
- `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`: Example MongoDB adapter
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Project MongoDB schema
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization_member.go`: OrganizationMember MongoDB schema

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/project_repo.go`

**Description**:
Implement the ProjectRepositoryPort for MongoDB.

```go
package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoProjectRepository implements ProjectRepositoryPort for MongoDB.
type MongoProjectRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
}

// NewMongoProjectRepository creates a new MongoDB project repository for RBAC.
func NewMongoProjectRepository(l *slog.Logger, db *mongo.Database) *MongoProjectRepository {
	return &MongoProjectRepository{
		logger:       l.With(slog.String("name", "rbac.repository.mongodb.project")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// GetByID retrieves a project by its ID.
func (r *MongoProjectRepository) GetByID(ctx context.Context, projectID string) (*domain.Project, error) {
	// Implementation outline:
	// 1. Convert projectID string to bson.ObjectID.
	//    - If conversion fails, return nil, error.
	// 2. Build filter with _id field.
	// 3. Execute FindOne query.
	// 4. If mongo.ErrNoDocuments, return nil, nil.
	// 5. If other error, log error and return nil, error.
	// 6. Convert mongoschema.Project to domain.Project using ToDomain().
	// 7. Return project, nil.
}

// Interface verification
var _ repository.ProjectRepositoryPort = (*MongoProjectRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid project ID exists | `"507f1f77bcf86cd799439011"` | `(*domain.Project, nil)` | Happy path |
| Valid project ID not found | `"507f1f77bcf86cd799439012"` | `(nil, nil)` | Not found branch |
| Invalid ObjectID format | `"invalid-id"` | `(nil, error)` | ObjectID parsing error |
| Database error | `"507f1f77bcf86cd799439011"` (with DB failure) | `(nil, error)` | Database error handling |

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go`

**Description**:
Implement the OrganizationMemberRepositoryPort for MongoDB.

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
	logger      *slog.Logger
	membersColl *mongo.Collection
}

// NewMongoOrganizationMemberRepository creates a new MongoDB organization member repository.
func NewMongoOrganizationMemberRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationMemberRepository {
	return &MongoOrganizationMemberRepository{
		logger:      l.With(slog.String("name", "rbac.repository.mongodb.organization_member")),
		membersColl: db.Collection(mongoschema.OrganizationMemberCollectionName),
	}
}

// IsMember checks if a user is a member of an organization.
func (r *MongoOrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	// Implementation outline:
	// 1. Convert userID string to bson.ObjectID.
	//    - If conversion fails, return false, error.
	// 2. Convert organizationID string to bson.ObjectID.
	//    - If conversion fails, return false, error.
	// 3. Build filter with userId and organizationId fields.
	// 4. Execute CountDocuments query with limit 1.
	// 5. If error, log error and return false, error.
	// 6. Return count > 0, nil.
}

// Interface verification
var _ repository.OrganizationMemberRepositoryPort = (*MongoOrganizationMemberRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| User is member of org | `userID="u1", orgID="o1"` (membership exists) | `(true, nil)` | Happy path - member found |
| User is not member of org | `userID="u1", orgID="o2"` (no membership) | `(false, nil)` | Not member branch |
| Invalid userID format | `userID="invalid", orgID="o1"` | `(false, error)` | UserID parsing error |
| Invalid organizationID format | `userID="u1", orgID="invalid"` | `(false, error)` | OrgID parsing error |
| Database error | Valid IDs with DB failure | `(false, error)` | Database error handling |

---

## Step 4: Create RBAC Service Implementation

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service implementation patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`: Function parameter rules
- `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go`: Example service implementation
- `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`: Example service implementation

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go`

**Description**:
Implement the RBAC service with the CanAccess method for checking user access to projects.

```go
package rbac

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// Service implements RBAC business logic.
type Service struct {
	logger      *slog.Logger
	projectRepo repository.ProjectRepositoryPort
	memberRepo  repository.OrganizationMemberRepositoryPort
}

// NewService creates a new RBAC service.
func NewService(
	l *slog.Logger,
	projectRepo repository.ProjectRepositoryPort,
	memberRepo repository.OrganizationMemberRepositoryPort,
) *Service {
	return &Service{
		logger:      l.With(slog.String("name", "rbac.service")),
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

// CanAccess checks if a user can access a project.
// Returns true if user is a member of the project's organization.
// Returns false, nil if access is denied (project not found, not a member).
// Returns false, error if a system error occurs.
func (s *Service) CanAccess(ctx context.Context, userID, projectID string) (bool, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	//    - If empty, log warning and return false, error("userID is required").
	// 2. Validate projectID is not empty.
	//    - If empty, log warning and return false, error("projectID is required").
	// 3. Query project by projectID using projectRepo.GetByID().
	//    - If error, log error and return false, error.
	//    - If project is nil, log info "project not found" and return false, nil.
	// 4. Query membership using memberRepo.IsMember(userID, string(project.OrganizationID)).
	//    - If error, log error and return false, error.
	// 5. If isMember is false, log info "access denied: user is not member of organization".
	// 6. If isMember is true, log debug "access granted".
	// 7. Return isMember, nil.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| User is member of project's org | `userID="u1", projectID="p1"` (member of org) | `(true, nil)` | Happy path - access granted |
| User is not member of project's org | `userID="u1", projectID="p1"` (not member) | `(false, nil)` | Access denied - not member |
| Project not found | `userID="u1", projectID="p-invalid"` | `(false, nil)` | Project not found branch |
| Empty userID | `userID="", projectID="p1"` | `(false, error)` | Validation - empty userID |
| Empty projectID | `userID="u1", projectID=""` | `(false, error)` | Validation - empty projectID |
| Project query fails | `userID="u1", projectID="p1"` (DB error) | `(false, error)` | Project repo error |
| Membership query fails | `userID="u1", projectID="p1"` (DB error on member check) | `(false, error)` | Member repo error |

---

## Step 5: Create Mock Repository Implementations

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/mock/api_client_mock.go`: Example mock implementation pattern
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/project_repo_port.go`: Interface to mock
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go`: Interface to mock

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/project_repo_mock.go`

**Description**:
Create mock implementation of ProjectRepositoryPort for testing. Follows the pattern from daemon with function injection and call tracking.

```go
package mock

import (
	"context"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// ProjectRepository implements repository.ProjectRepositoryPort for testing.
type ProjectRepository struct {
	// GetByIDFunc is the behavior to execute when GetByID is called.
	GetByIDFunc func(ctx context.Context, projectID string) (*domain.Project, error)
	// CallCount tracks the number of GetByID calls.
	CallCount int
	// Projects records all projectIDs queried.
	ProjectIDs []string
}

// GetByID implements repository.ProjectRepositoryPort.
func (m *ProjectRepository) GetByID(ctx context.Context, projectID string) (*domain.Project, error) {
	// Implementation outline:
	// 1. Increment CallCount.
	// 2. If GetByIDFunc is set, call it and get result.
	// 3. Record projectID in ProjectIDs slice.
	// 4. Return the result from GetByIDFunc (or nil, nil if not set).
}

// Compile-time interface verification.
var _ repository.ProjectRepositoryPort = (*ProjectRepository)(nil)
```

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/organization_member_repo_mock.go`

**Description**:
Create mock implementation of OrganizationMemberRepositoryPort for testing.

```go
package mock

import (
	"context"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// MembershipQuery represents a single IsMember query.
type MembershipQuery struct {
	UserID         string
	OrganizationID string
}

// OrganizationMemberRepository implements repository.OrganizationMemberRepositoryPort for testing.
type OrganizationMemberRepository struct {
	// IsMemberFunc is the behavior to execute when IsMember is called.
	IsMemberFunc func(ctx context.Context, userID, organizationID string) (bool, error)
	// CallCount tracks the number of IsMember calls.
	CallCount int
	// Queries records all membership queries.
	Queries []MembershipQuery
}

// IsMember implements repository.OrganizationMemberRepositoryPort.
func (m *OrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	// Implementation outline:
	// 1. Increment CallCount.
	// 2. If IsMemberFunc is set, call it and get result.
	// 3. Record query in Queries slice (append MembershipQuery{userID, organizationID}).
	// 4. Return the result from IsMemberFunc (or false, nil if not set).
}

// Compile-time interface verification.
var _ repository.OrganizationMemberRepositoryPort = (*OrganizationMemberRepository)(nil)
```

---

## Step 6: Create RBAC Service Unit Tests

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`: Example test file using Ginkgo
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/connectrpc_suite_test.go`: Example test suite setup
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/project_repo_mock.go`: Mock project repository
- `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/organization_member_repo_mock.go`: Mock member repository

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_suite_test.go`

**Description**:
Set up the Ginkgo test suite for the RBAC service.

```go
package rbac_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRBAC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Service Suite")
}
```

### `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service_test.go`

**Description**:
Unit tests for the RBAC service using mock implementations from the mock package.

```go
package rbac_test

import (
	"context"
	"errors"
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock"
	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("RBAC Service", func() {
	var (
		logger      *slog.Logger
		projectRepo *mock.ProjectRepository
		memberRepo  *mock.OrganizationMemberRepository
		svc         *rbac.Service
		ctx         context.Context
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		projectRepo = &mock.ProjectRepository{}
		memberRepo = &mock.OrganizationMemberRepository{}
		svc = rbac.NewService(logger, projectRepo, memberRepo)
		ctx = context.Background()
	})

	Describe("CanAccess", func() {
		Context("when user is member of project's organization", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return true, nil
				}
			})

			It("should return true, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeTrue())
			})
		})

		Context("when user is not member of project's organization", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, nil
				}
			})

			It("should return false, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when project not found", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return nil, nil
				}
			})

			It("should return false, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when userID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("userID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when projectID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("projectID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when project query fails", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return nil, errors.New("database error")
				}
			})

			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when membership query fails", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, errors.New("database error")
				}
			})

			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})
	})
})
```

---

## Step 7: Create RBAC fx Module Registration

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: fx DI patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`: Example module registration
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`: Example module registration

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_rbac.go`

**Description**:
Create the fx module for RBAC service registration. Since RBAC is a core service without inbound handlers, it only provides the service and its repository adapters.

```go
package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb"
)

func newRBACModule() fx.Option {
	return fx.Module("rbac",
		// Project repository for RBAC
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoProjectRepository,
				fx.As(new(repository.ProjectRepositoryPort)),
			),
		),

		// Organization member repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationMemberRepository,
				fx.As(new(repository.OrganizationMemberRepositoryPort)),
			),
		),

		// RBAC Service
		fx.Provide(rbac.NewService),
	)
}
```

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`

**Description**:
Add the RBAC module to the application's fx composition. Insert `newRBACModule()` in the fx.New() call.

```go
// In the fx.New() call, add newRBACModule() to the list of modules:
func Run() {
	fx.New(
		// Modules
		newPlatformModule(),
		newHealthModule(),
		newAuthModule(),
		newAggregationModule(),
		newDashboardModule(),
		newProjectModule(),
		newRBACModule(), // Add this line

		// Registrations (invoked for side effects)
		fx.Invoke(registerConnectRPCServer),

		// Lifecycle timeouts
		fx.StartTimeout(30*time.Second),
		fx.StopTimeout(30*time.Second),
	).Run()
}
```

---

## Summary of Files to Create/Modify

### New Files (10 files)

| File Path | Description |
| :-------- | :---------- |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service.go` | RBAC service implementation |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_suite_test.go` | Ginkgo test suite setup |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/rbac_service_test.go` | Unit tests for RBAC service |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/project_repo_port.go` | Project repository port interface |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/organization_member_repo_port.go` | Org member repository port interface |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/project_repo.go` | MongoDB project repository adapter |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mongodb/organization_member_repo.go` | MongoDB org member repository adapter |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/project_repo_mock.go` | Mock project repository for testing |
| `/Users/jayce/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock/organization_member_repo_mock.go` | Mock member repository for testing |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_rbac.go` | fx module registration |

### Modified Files (3 files)

| File Path | Description |
| :-------- | :---------- |
| `/Users/jayce/team-attention/cops/shared/domain/project.go` | Add OrganizationID field to Project struct |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go` | Add OrganizationID field and BSON handling |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go` | Add newRBACModule() to fx composition |

## Execution Order

1. **Step 1**: Modify domain models (`shared/domain/project.go`, `shared/domain/mongoschema/project.go`)
2. **Step 2**: Create repository port interfaces (`outbound/repository/*.go`)
3. **Step 3**: Create MongoDB repository adapters (`outbound/repository/mongodb/*.go`)
4. **Step 4**: Create RBAC service implementation (`rbac_service.go`)
5. **Step 5**: Create mock repository implementations (`outbound/repository/mock/*.go`)
6. **Step 6**: Create unit tests (`rbac_suite_test.go`, `rbac_service_test.go`)
7. **Step 7**: Create fx module and register in application (`module_rbac.go`, `application.go`)
