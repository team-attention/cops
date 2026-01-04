# Implementation Plan: Organization Settings

## Overview

This plan implements an Organization Settings section in the Settings page, allowing users to manage their currently selected organization. The feature includes:

- Display organization information (name, slug, user role)
- Edit organization details (admin only)
- Manage organization members (admin only) - view, change roles, remove
- Leave organization functionality with cascade delete for last organization

The implementation requires creating a new `OrganizationService` with 5 RPC methods on the backend, and a new `feature/organization/` directory on the frontend.

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | - | - | No new packages required. All functionality can be implemented with existing dependencies. |

---

## Step 1: Create Organization Protobuf Definition

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions
- `/Users/jayce/team-attention/cops/idl/protobuf/user/v1/user.proto`: Example service definition
- `/Users/jayce/team-attention/cops/idl/protobuf/domain/v1/domain.proto`: Existing domain types

### `/Users/jayce/team-attention/cops/idl/protobuf/organization/v1/organization.proto`

**Description**:
Create new OrganizationService with 5 RPC methods for organization management. Uses domain.proto types where applicable.

```protobuf
syntax = "proto3";

package organization.v1;

import "domain/v1/domain.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1;organizationv1";

// OrganizationMemberWithDetails represents a member with full user details.
// Used in GetOrganizationMembers response to avoid N+1 queries.
message OrganizationMemberWithDetails {
  string user_id = 1;
  string email = 2;
  string name = 3;
  string avatar_url = 4;
  string role = 5;  // "admin" or "member"
}

// UpdateOrganizationReq contains fields to update for an organization.
message UpdateOrganizationReq {
  string organization_id = 1;
  string name = 2;
  string slug = 3;
}

// UpdateOrganizationRes contains the updated organization.
message UpdateOrganizationRes {
  domain.v1.Organization organization = 1;
}

// GetOrganizationMembersReq requests members of an organization.
message GetOrganizationMembersReq {
  string organization_id = 1;
}

// GetOrganizationMembersRes contains the list of members with details.
message GetOrganizationMembersRes {
  repeated OrganizationMemberWithDetails members = 1;
}

// UpdateMemberRoleReq changes a member's role in an organization.
message UpdateMemberRoleReq {
  string organization_id = 1;
  string user_id = 2;
  string role = 3;  // "admin" or "member"
}

// UpdateMemberRoleRes confirms the role update.
message UpdateMemberRoleRes {
  bool success = 1;
}

// RemoveMemberReq removes a member from an organization.
message RemoveMemberReq {
  string organization_id = 1;
  string user_id = 2;
}

// RemoveMemberRes confirms the member removal.
message RemoveMemberRes {
  bool success = 1;
}

// LeaveOrganizationReq requests the current user to leave an organization.
message LeaveOrganizationReq {
  string organization_id = 1;
}

// LeaveOrganizationRes contains the result of leaving.
message LeaveOrganizationRes {
  bool success = 1;
  // is_last_organization indicates if this was the user's last organization (cascade deleted)
  bool is_last_organization = 2;
}

// OrganizationService handles organization management operations.
service OrganizationService {
  // UpdateOrganization updates an organization's name and/or slug.
  // Requires admin role in the organization.
  rpc UpdateOrganization(UpdateOrganizationReq) returns (UpdateOrganizationRes);

  // GetOrganizationMembers retrieves all members with their full user details.
  // Requires membership in the organization.
  rpc GetOrganizationMembers(GetOrganizationMembersReq) returns (GetOrganizationMembersRes);

  // UpdateMemberRole changes a member's role (admin/member).
  // Requires admin role. Cannot demote the last admin.
  rpc UpdateMemberRole(UpdateMemberRoleReq) returns (UpdateMemberRoleRes);

  // RemoveMember removes a user from the organization.
  // Requires admin role. Cannot remove the last admin.
  rpc RemoveMember(RemoveMemberReq) returns (RemoveMemberRes);

  // LeaveOrganization removes the current user from the organization.
  // If user is in only one organization, cascade deletes all data.
  rpc LeaveOrganization(LeaveOrganizationReq) returns (LeaveOrganizationRes);
}
```

---

## Step 2: Generate gRPC Stubs

**Files to Read**:
- `/Users/jayce/team-attention/cops/idl/protobuf/buf.gen.yaml`: Generation configuration

### Command

```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Generated Files** (auto-generated, do not edit):
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organization.pb.go`
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organizationv1connect/organization.connect.go`
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/organization/v1/organization_pb.ts`
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery.ts`

---

## Step 3: Create Organization Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port interface patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`: Existing organization repository

### `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/organization_repo_port.go`

**Description**:
Define repository interface for organization management operations.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// MemberWithDetails represents a member with full user information.
type MemberWithDetails struct {
	UserID    string
	Email     string
	Name      string
	AvatarURL string
	Role      domain.MemberRole
}

// OrganizationRepositoryPort defines interface for organization management operations.
type OrganizationRepositoryPort interface {
	// GetByID retrieves an organization by its ID.
	// Returns nil, nil if organization not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, organizationID string) (*domain.Organization, error)

	// GetBySlug retrieves an organization by its slug.
	// Returns nil, nil if organization not found.
	// Returns nil, error if database error occurs.
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)

	// Update updates an organization's name and slug.
	// Returns the updated organization.
	// Returns error if database error occurs.
	Update(ctx context.Context, organizationID, name, slug string) (*domain.Organization, error)

	// GetMembersWithDetails retrieves all members with their user details.
	// Performs join with users collection to get email, name, avatar.
	// Returns empty slice if no members.
	// Returns nil, error if database error occurs.
	GetMembersWithDetails(ctx context.Context, organizationID string) ([]*MemberWithDetails, error)

	// UpdateMemberRole updates a member's role in the organization.
	// Returns error if database error occurs.
	UpdateMemberRole(ctx context.Context, organizationID, userID string, role domain.MemberRole) error

	// RemoveMember removes a member from the organization.
	// Returns error if database error occurs.
	RemoveMember(ctx context.Context, organizationID, userID string) error

	// CountAdmins returns the number of admin members in an organization.
	// Returns count, nil on success.
	// Returns 0, error if database error occurs.
	CountAdmins(ctx context.Context, organizationID string) (int, error)

	// GetUserOrganizationCount returns the number of organizations a user belongs to.
	// Returns count, nil on success.
	// Returns 0, error if database error occurs.
	GetUserOrganizationCount(ctx context.Context, userID string) (int, error)

	// GetMemberRole retrieves a user's role in an organization.
	// Returns empty string, nil if user is not a member.
	// Returns role, nil on success.
	// Returns "", error if database error occurs.
	GetMemberRole(ctx context.Context, organizationID, userID string) (domain.MemberRole, error)

	// DeleteOrganization permanently deletes an organization.
	// Returns error if database error occurs.
	DeleteOrganization(ctx context.Context, organizationID string) error
}
```

---

## Step 4: Create Organization Repository MongoDB Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger binding patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`: Existing MongoDB patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: Organization schema

### `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb/organization_repo.go`

**Description**:
Implement OrganizationRepositoryPort for MongoDB with aggregation pipeline for member details.

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
	logger    *slog.Logger
	orgColl   *mongo.Collection
	usersColl *mongo.Collection
}

// NewMongoOrganizationRepository creates a new MongoDB organization repository.
func NewMongoOrganizationRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationRepository {
	// Implementation outline:
	// 1. Return &MongoOrganizationRepository with:
	//    - logger bound with name "organization.repository.mongodb"
	//    - orgColl from db.Collection(mongoschema.OrganizationCollectionName)
	//    - usersColl from db.Collection(mongoschema.UserCollectionName)
}

// GetByID retrieves an organization by its ID.
func (r *MongoOrganizationRepository) GetByID(ctx context.Context, organizationID string) (*domain.Organization, error) {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return nil, error.
	// 3. Create filter with _id field.
	// 4. Execute FindOne query on organizations collection.
	// 5. If mongo.ErrNoDocuments, return nil, nil.
	// 6. If other error, log and return nil, error.
	// 7. Convert mongoschema.Organization to domain.Organization using ToDomain().
	// 8. Return organization, nil.
}

// GetBySlug retrieves an organization by its slug.
func (r *MongoOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	// Implementation outline:
	// 1. Create filter with slug field.
	// 2. Execute FindOne query on organizations collection.
	// 3. If mongo.ErrNoDocuments, return nil, nil.
	// 4. If other error, log and return nil, error.
	// 5. Convert mongoschema.Organization to domain.Organization using ToDomain().
	// 6. Return organization, nil.
}

// Update updates an organization's name and slug.
func (r *MongoOrganizationRepository) Update(ctx context.Context, organizationID, name, slug string) (*domain.Organization, error) {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return nil, error.
	// 3. Create filter with _id field.
	// 4. Create update with $set for name and slug fields.
	// 5. Execute FindOneAndUpdate with ReturnDocument: After option.
	// 6. If mongo.ErrNoDocuments, return nil, error "organization not found".
	// 7. If other error, log and return nil, error.
	// 8. Convert result to domain.Organization using ToDomain().
	// 9. Return updated organization, nil.
}

// GetMembersWithDetails retrieves all members with their user details.
func (r *MongoOrganizationRepository) GetMembersWithDetails(ctx context.Context, organizationID string) ([]*repository.MemberWithDetails, error) {
	// Implementation outline:
	// 1. Convert organizationID string to bson.ObjectID.
	// 2. If conversion fails, return nil, error.
	// 3. Build aggregation pipeline:
	//    a. $match: filter by organization _id
	//    b. $unwind: expand members array
	//    c. $lookup: join with users collection on members.userId = users._id
	//    d. $unwind: expand joined user (preserveNullAndEmptyArrays: true)
	//    e. $project: select userId, role, email, name, profileImageUrl from user
	// 4. Execute Aggregate on organizations collection.
	// 5. Iterate cursor, decode each result to MemberWithDetails struct.
	// 6. Return slice of MemberWithDetails, nil.
}

// UpdateMemberRole updates a member's role in the organization.
func (r *MongoOrganizationRepository) UpdateMemberRole(ctx context.Context, organizationID, userID string, role domain.MemberRole) error {
	// Implementation outline:
	// 1. Convert organizationID and userID to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Create filter with organization _id AND members.userId.
	// 4. Create update with $set for members.$.role (positional operator).
	// 5. Execute UpdateOne on organizations collection.
	// 6. If error, log and return error.
	// 7. Return nil.
}

// RemoveMember removes a member from the organization.
func (r *MongoOrganizationRepository) RemoveMember(ctx context.Context, organizationID, userID string) error {
	// Implementation outline:
	// 1. Convert organizationID and userID to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Create filter with organization _id.
	// 4. Create update with $pull to remove member from members array.
	// 5. Execute UpdateOne on organizations collection.
	// 6. If error, log and return error.
	// 7. Return nil.
}

// CountAdmins returns the number of admin members in an organization.
func (r *MongoOrganizationRepository) CountAdmins(ctx context.Context, organizationID string) (int, error) {
	// Implementation outline:
	// 1. Convert organizationID to bson.ObjectID.
	// 2. If conversion fails, return 0, error.
	// 3. Build aggregation pipeline:
	//    a. $match: filter by organization _id
	//    b. $unwind: expand members array
	//    c. $match: filter by members.role == "admin"
	//    d. $count: count admin members
	// 4. Execute Aggregate on organizations collection.
	// 5. Decode result to get count.
	// 6. Return count, nil.
}

// GetUserOrganizationCount returns the number of organizations a user belongs to.
func (r *MongoOrganizationRepository) GetUserOrganizationCount(ctx context.Context, userID string) (int, error) {
	// Implementation outline:
	// 1. Convert userID to bson.ObjectID.
	// 2. If conversion fails, return 0, error.
	// 3. Create filter with members.userId using $elemMatch.
	// 4. Execute CountDocuments on organizations collection.
	// 5. Return count, nil.
}

// GetMemberRole retrieves a user's role in an organization.
func (r *MongoOrganizationRepository) GetMemberRole(ctx context.Context, organizationID, userID string) (domain.MemberRole, error) {
	// Implementation outline:
	// 1. Convert organizationID and userID to bson.ObjectID.
	// 2. If conversion fails, return "", error.
	// 3. Create filter with organization _id AND members.userId.
	// 4. Project only the matching member element using $elemMatch.
	// 5. Execute FindOne with projection.
	// 6. If mongo.ErrNoDocuments, return "", nil (not a member).
	// 7. Extract role from the matched member element.
	// 8. Return role, nil.
}

// DeleteOrganization permanently deletes an organization.
func (r *MongoOrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	// Implementation outline:
	// 1. Convert organizationID to bson.ObjectID.
	// 2. If conversion fails, return error.
	// 3. Create filter with _id field.
	// 4. Execute DeleteOne on organizations collection.
	// 5. If error, log and return error.
	// 6. Return nil.
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| GetByID - valid ID | valid organizationID | organization, nil | Happy path |
| GetByID - not found | non-existent ID | nil, nil | Not found |
| GetByID - invalid ID | "invalid" | nil, error | Invalid ObjectID |
| GetBySlug - valid slug | "my-org" | organization, nil | Happy path |
| GetBySlug - not found | "nonexistent" | nil, nil | Not found |
| Update - success | valid ID, name, slug | updated org, nil | Happy path |
| Update - not found | non-existent ID | nil, error | Not found |
| GetMembersWithDetails - success | valid ID | []MemberWithDetails, nil | Happy path |
| GetMembersWithDetails - empty | org with no members | [], nil | Empty result |
| UpdateMemberRole - success | valid IDs, "admin" | nil | Happy path |
| RemoveMember - success | valid IDs | nil | Happy path |
| CountAdmins - multiple admins | org with 2 admins | 2, nil | Multiple admins |
| CountAdmins - single admin | org with 1 admin | 1, nil | Single admin |
| GetUserOrganizationCount - multiple | user in 3 orgs | 3, nil | Multiple orgs |
| GetUserOrganizationCount - single | user in 1 org | 1, nil | Single org |
| GetMemberRole - admin | admin user | "admin", nil | Admin role |
| GetMemberRole - member | member user | "member", nil | Member role |
| GetMemberRole - not member | non-member | "", nil | Not a member |

---

## Step 5: Create Organization Service

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/user_service.go`: Example service implementation
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/cascade_delete_repo_port.go`: Cascade delete interface

### `/Users/jayce/team-attention/cops/api/internal/service/organization/organization_service.go`

**Description**:
Implement organization business logic with RBAC checks and cascade delete support.

```go
package organization

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

const (
	// SlugMinLength is the minimum length for organization slug.
	SlugMinLength = 3
	// SlugMaxLength is the maximum length for organization slug.
	SlugMaxLength = 63
)

// SlugPattern validates organization slug format.
// Lowercase alphanumeric with hyphens, no leading/trailing hyphens.
var SlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// UpdateOrganizationResult contains the updated organization.
type UpdateOrganizationResult struct {
	Organization *domain.Organization
}

// GetOrganizationMembersResult contains members with details.
type GetOrganizationMembersResult struct {
	Members []*repository.MemberWithDetails
}

// LeaveOrganizationResult contains the result of leaving.
type LeaveOrganizationResult struct {
	Success            bool
	IsLastOrganization bool
}

// Service implements organization business logic.
type Service struct {
	logger            *slog.Logger
	orgRepo           repository.OrganizationRepositoryPort
	cascadeDeleteRepo userrepo.CascadeDeleteRepositoryPort
}

// NewService creates a new organization service.
func NewService(
	l *slog.Logger,
	orgRepo repository.OrganizationRepositoryPort,
	cascadeDeleteRepo userrepo.CascadeDeleteRepositoryPort,
) *Service {
	// Implementation outline:
	// 1. Return &Service with:
	//    - logger bound with name "organization.service"
	//    - orgRepo
	//    - cascadeDeleteRepo
}

// UpdateOrganization updates an organization's name and slug.
// Requires admin role.
func (s *Service) UpdateOrganization(ctx context.Context, userID, organizationID, name, slug string) (*UpdateOrganizationResult, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	//    a. If empty, return nil, error "userID is required".
	// 2. Validate organizationID is not empty.
	//    a. If empty, return nil, error "organizationID is required".
	// 3. Check user's role in organization using orgRepo.GetMemberRole.
	//    a. If error, log and return nil, error.
	//    b. If role is empty (not a member), return nil, error "user is not a member of this organization".
	//    c. If role is not "admin", return nil, error "admin role required".
	// 4. Validate name is not empty.
	//    a. If empty, return nil, error "name is required".
	// 5. Validate slug format:
	//    a. Trim and lowercase slug.
	//    b. Check length >= SlugMinLength and <= SlugMaxLength.
	//    c. Check slug matches SlugPattern.
	//    d. If invalid, return nil, error with specific validation message.
	// 6. Check if slug is already taken by another organization.
	//    a. Call orgRepo.GetBySlug.
	//    b. If found and ID != organizationID, return nil, error "slug is already taken".
	// 7. Call orgRepo.Update to update organization.
	//    a. If error, log and return nil, error.
	// 8. Log successful update with organizationID.
	// 9. Return &UpdateOrganizationResult{Organization: updated}, nil.
}

// GetOrganizationMembers retrieves members with their details.
// Requires membership in the organization.
func (s *Service) GetOrganizationMembers(ctx context.Context, userID, organizationID string) (*GetOrganizationMembersResult, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	//    a. If empty, return nil, error "userID is required".
	// 2. Validate organizationID is not empty.
	//    a. If empty, return nil, error "organizationID is required".
	// 3. Check user's role in organization using orgRepo.GetMemberRole.
	//    a. If error, log and return nil, error.
	//    b. If role is empty (not a member), return nil, error "user is not a member of this organization".
	// 4. Call orgRepo.GetMembersWithDetails.
	//    a. If error, log and return nil, error.
	// 5. Log successful retrieval with organizationID and member count.
	// 6. Return &GetOrganizationMembersResult{Members: members}, nil.
}

// UpdateMemberRole changes a member's role.
// Requires admin role. Cannot demote the last admin.
func (s *Service) UpdateMemberRole(ctx context.Context, userID, organizationID, targetUserID, newRole string) error {
	// Implementation outline:
	// 1. Validate userID, organizationID, targetUserID are not empty.
	// 2. Validate newRole is either "admin" or "member".
	//    a. If invalid, return error "role must be 'admin' or 'member'".
	// 3. Check requesting user's role in organization.
	//    a. If not admin, return error "admin role required".
	// 4. Get target user's current role.
	//    a. If not a member, return error "target user is not a member".
	// 5. If demoting from admin to member:
	//    a. Call orgRepo.CountAdmins.
	//    b. If count == 1, return error "cannot demote the last admin".
	// 6. Call orgRepo.UpdateMemberRole.
	//    a. If error, log and return error.
	// 7. Log successful role update with organizationID, targetUserID, newRole.
	// 8. Return nil.
}

// RemoveMember removes a member from the organization.
// Requires admin role. Cannot remove the last admin.
func (s *Service) RemoveMember(ctx context.Context, userID, organizationID, targetUserID string) error {
	// Implementation outline:
	// 1. Validate userID, organizationID, targetUserID are not empty.
	// 2. Check requesting user's role in organization.
	//    a. If not admin, return error "admin role required".
	// 3. Get target user's current role.
	//    a. If not a member, return error "target user is not a member".
	// 4. If target user is admin:
	//    a. Call orgRepo.CountAdmins.
	//    b. If count == 1, return error "cannot remove the last admin".
	// 5. Call orgRepo.RemoveMember.
	//    a. If error, log and return error.
	// 6. Log successful removal with organizationID, targetUserID.
	// 7. Return nil.
}

// LeaveOrganization removes the current user from the organization.
// If this is the user's last organization, cascade deletes all data.
func (s *Service) LeaveOrganization(ctx context.Context, userID, organizationID string) (*LeaveOrganizationResult, error) {
	// Implementation outline:
	// 1. Validate userID is not empty.
	// 2. Validate organizationID is not empty.
	// 3. Get user's role in organization.
	//    a. If not a member, return nil, error "user is not a member of this organization".
	// 4. Get organization to check member count.
	//    a. If error, log and return nil, error.
	//    b. If not found, return nil, error "organization not found".
	// 5. If user is admin:
	//    a. Call orgRepo.CountAdmins.
	//    b. If count == 1 AND organization has more than 1 member, return nil, error "cannot leave as the sole admin with other members".
	// 6. Check if this is user's last organization.
	//    a. Call orgRepo.GetUserOrganizationCount.
	//    b. isLastOrganization = count == 1.
	// 7. If organization has only this user as member (sole member):
	//    a. Call cascadeDeleteRepo.DeleteSessionRecordsByOrganization.
	//    b. Call cascadeDeleteRepo.DeleteProjectsByOrganization.
	//    c. Call orgRepo.DeleteOrganization.
	//    d. Log cascade deletion.
	// 8. Else (shared organization):
	//    a. Call orgRepo.RemoveMember.
	//    b. Log membership removal.
	// 9. Log successful leave with organizationID, isLastOrganization.
	// 10. Return &LeaveOrganizationResult{Success: true, IsLastOrganization: isLastOrganization}, nil.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| UpdateOrganization - success | admin user, valid data | updated org, nil | Happy path |
| UpdateOrganization - not admin | member user | nil, error | Admin check |
| UpdateOrganization - not member | non-member user | nil, error | Membership check |
| UpdateOrganization - invalid slug | slug with spaces | nil, error | Slug validation |
| UpdateOrganization - slug taken | existing slug | nil, error | Uniqueness check |
| GetOrganizationMembers - success | member user | members, nil | Happy path |
| GetOrganizationMembers - not member | non-member user | nil, error | Membership check |
| UpdateMemberRole - success | admin changes member | nil | Happy path |
| UpdateMemberRole - demote last admin | demote only admin | error | Last admin check |
| UpdateMemberRole - not admin | member tries to change | error | Admin check |
| RemoveMember - success | admin removes member | nil | Happy path |
| RemoveMember - remove last admin | remove only admin | error | Last admin check |
| RemoveMember - admin removes self | admin removes self (other admins exist) | nil | Self-removal |
| LeaveOrganization - shared org | user in 2+ orgs | success, isLast=false | Shared org |
| LeaveOrganization - last org | user in 1 org, sole member | success, isLast=true | Last org cascade |
| LeaveOrganization - sole admin with members | sole admin, other members | error | Sole admin check |
| LeaveOrganization - not member | non-member | error | Membership check |

---

## Step 6: Create Organization gRPC Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler patterns
- `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`: Example handler

### `/Users/jayce/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc/handler.go`

**Description**:
Implement ConnectRPC handler for OrganizationService.

```go
package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/organization"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	organizationv1 "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/organization/v1/organizationv1connect"
)

// OrganizationGRPCHandler handles gRPC requests for organization service.
type OrganizationGRPCHandler struct {
	svc    *organization.Service
	logger *slog.Logger
}

// NewOrganizationGRPCHandler creates a new organization gRPC handler.
func NewOrganizationGRPCHandler(l *slog.Logger, svc *organization.Service) *OrganizationGRPCHandler {
	// Implementation outline:
	// 1. Return &OrganizationGRPCHandler with:
	//    - svc
	//    - logger bound with name "organization.grpc.connectrpc"
}

// GetHandler implements ConnectHandler interface.
func (h *OrganizationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	// Implementation outline:
	// 1. Return organizationv1connect.NewOrganizationServiceHandler(h, opts...)
}

// UpdateOrganization updates an organization's name and slug.
func (h *OrganizationGRPCHandler) UpdateOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.UpdateOrganizationReq],
) (*connect.Response[organizationv1.UpdateOrganizationRes], error) {
	// Implementation outline:
	// 1. Extract userID from context using interceptor.UserIDFromContext.
	//    a. If empty, return connect.CodeUnauthenticated error.
	// 2. Get organizationID, name, slug from req.Msg.
	// 3. Call h.svc.UpdateOrganization(ctx, userID, organizationID, name, slug).
	//    a. If error contains "admin role required", return connect.CodePermissionDenied.
	//    b. If error contains "not a member", return connect.CodePermissionDenied.
	//    c. If error contains validation messages, return connect.CodeInvalidArgument.
	//    d. If other error, return connect.CodeInternal.
	// 4. Convert result.Organization to domainv1.Organization protobuf.
	// 5. Return connect.NewResponse with UpdateOrganizationRes.
}

// GetOrganizationMembers retrieves members with their details.
func (h *OrganizationGRPCHandler) GetOrganizationMembers(
	ctx context.Context,
	req *connect.Request[organizationv1.GetOrganizationMembersReq],
) (*connect.Response[organizationv1.GetOrganizationMembersRes], error) {
	// Implementation outline:
	// 1. Extract userID from context.
	//    a. If empty, return connect.CodeUnauthenticated error.
	// 2. Get organizationID from req.Msg.
	// 3. Call h.svc.GetOrganizationMembers(ctx, userID, organizationID).
	//    a. If error contains "not a member", return connect.CodePermissionDenied.
	//    b. If other error, return connect.CodeInternal.
	// 4. Convert result.Members to []*organizationv1.OrganizationMemberWithDetails.
	// 5. Return connect.NewResponse with GetOrganizationMembersRes.
}

// UpdateMemberRole changes a member's role.
func (h *OrganizationGRPCHandler) UpdateMemberRole(
	ctx context.Context,
	req *connect.Request[organizationv1.UpdateMemberRoleReq],
) (*connect.Response[organizationv1.UpdateMemberRoleRes], error) {
	// Implementation outline:
	// 1. Extract userID from context.
	//    a. If empty, return connect.CodeUnauthenticated error.
	// 2. Get organizationID, targetUserID (user_id), role from req.Msg.
	// 3. Call h.svc.UpdateMemberRole(ctx, userID, organizationID, targetUserID, role).
	//    a. If error contains "admin role required", return connect.CodePermissionDenied.
	//    b. If error contains "last admin", return connect.CodeFailedPrecondition.
	//    c. If error contains validation messages, return connect.CodeInvalidArgument.
	//    d. If other error, return connect.CodeInternal.
	// 4. Return connect.NewResponse with UpdateMemberRoleRes{Success: true}.
}

// RemoveMember removes a member from the organization.
func (h *OrganizationGRPCHandler) RemoveMember(
	ctx context.Context,
	req *connect.Request[organizationv1.RemoveMemberReq],
) (*connect.Response[organizationv1.RemoveMemberRes], error) {
	// Implementation outline:
	// 1. Extract userID from context.
	//    a. If empty, return connect.CodeUnauthenticated error.
	// 2. Get organizationID, targetUserID from req.Msg.
	// 3. Call h.svc.RemoveMember(ctx, userID, organizationID, targetUserID).
	//    a. If error contains "admin role required", return connect.CodePermissionDenied.
	//    b. If error contains "last admin", return connect.CodeFailedPrecondition.
	//    c. If other error, return connect.CodeInternal.
	// 4. Return connect.NewResponse with RemoveMemberRes{Success: true}.
}

// LeaveOrganization removes the current user from the organization.
func (h *OrganizationGRPCHandler) LeaveOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.LeaveOrganizationReq],
) (*connect.Response[organizationv1.LeaveOrganizationRes], error) {
	// Implementation outline:
	// 1. Extract userID from context.
	//    a. If empty, return connect.CodeUnauthenticated error.
	// 2. Get organizationID from req.Msg.
	// 3. Call h.svc.LeaveOrganization(ctx, userID, organizationID).
	//    a. If error contains "sole admin", return connect.CodeFailedPrecondition.
	//    b. If error contains "not a member", return connect.CodePermissionDenied.
	//    c. If other error, return connect.CodeInternal.
	// 4. Return connect.NewResponse with LeaveOrganizationRes{
	//       Success: result.Success,
	//       IsLastOrganization: result.IsLastOrganization,
	//    }.
}

// Interface verification
var _ organizationv1connect.OrganizationServiceHandler = (*OrganizationGRPCHandler)(nil)
```

---

## Step 7: Create Organization Module for DI Container

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container patterns
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`: Example module

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_organization.go`

**Description**:
Register organization service and handler with fx container.

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
	// Implementation outline:
	// 1. Return fx.Module("organization",
	//    a. fx.Provide organization repository with fx.Annotate:
	//       - mongodb.NewMongoOrganizationRepository
	//       - fx.As(new(repository.OrganizationRepositoryPort))
	//    b. fx.Provide organization service:
	//       - organization.NewService
	//    c. fx.Provide ConnectRPC handler with fx.Annotate:
	//       - connectrpc.NewOrganizationGRPCHandler
	//       - fx.As(new(PrivateConnectHandler))
	//       - fx.ResultTags(`group:"private_connect_handlers"`)
	// )
}
```

### Modify `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`

**Description**:
Add organization module to application.

```go
// Add to Run() function:
// newOrganizationModule(),
```

---

## Step 8: Create Frontend Organization Hooks

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: FDD architecture rules
- `/Users/jayce/team-attention/cops/web/src/feature/user/hook/use-delete-account.ts`: Example hook

### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-update-organization.ts`

**Description**:
Mutation hook for updating organization details.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { updateOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useUpdateOrganization provides a mutation hook for updating an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useUpdateOrganization = () => {
  return useMutation(updateOrganization, { transport })
}
```

### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-get-organization-members.ts`

**Description**:
Query hook for fetching organization members with details.

```typescript
import { useQuery } from '@connectrpc/connect-query'
import { getOrganizationMembers } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

interface UseGetOrganizationMembersInput {
  organizationId: string
}

// useGetOrganizationMembers provides a query hook for fetching organization members.
// Returns a TanStack Query object with data, isLoading, error states.
export const useGetOrganizationMembers = (input: UseGetOrganizationMembersInput) => {
  return useQuery(getOrganizationMembers, input, { transport })
}
```

### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-update-member-role.ts`

**Description**:
Mutation hook for changing member role.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { updateMemberRole } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useUpdateMemberRole provides a mutation hook for changing a member's role.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useUpdateMemberRole = () => {
  return useMutation(updateMemberRole, { transport })
}
```

### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-remove-member.ts`

**Description**:
Mutation hook for removing a member.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { removeMember } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useRemoveMember provides a mutation hook for removing a member from an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useRemoveMember = () => {
  return useMutation(removeMember, { transport })
}
```

### `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-leave-organization.ts`

**Description**:
Mutation hook for leaving an organization.

```typescript
import { useMutation } from '@connectrpc/connect-query'
import { leaveOrganization } from '@/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery'
import { transport } from '@/shared/service/connect-transport'

// useLeaveOrganization provides a mutation hook for leaving an organization.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useLeaveOrganization = () => {
  return useMutation(leaveOrganization, { transport })
}
```

---

## Step 9: Create Organization Settings Types

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript rules

### `/Users/jayce/team-attention/cops/web/src/feature/organization/type/member.ts`

**Description**:
TypeScript types for organization member management.

```typescript
// MemberWithDetails represents a member with full user information.
// Used in the members list display.
export interface MemberWithDetails {
  userId: string
  email: string
  name: string
  avatarUrl: string
  role: 'admin' | 'member'
}

// EditOrganizationFormData represents the form state for editing organization.
export interface EditOrganizationFormData {
  name: string
  slug: string
}

// SlugValidationResult represents slug validation state.
export interface SlugValidationResult {
  isValid: boolean
  errorMessage: string | null
}
```

---

## Step 10: Create Edit Organization Dialog Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx`: Example dialog pattern

### `/Users/jayce/team-attention/cops/web/src/feature/organization/component/edit-organization-dialog.tsx`

**Description**:
Dialog for editing organization name and slug with validation.

```typescript
import { useState, useCallback, useEffect } from 'react'
import { Loader2, Pencil } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useUpdateOrganization } from '../hook/use-update-organization'
import { useUserStore } from '@/shared/store/user-store'
import { Button } from '@/gen/shadcn/ui/button'
import { Input } from '@/gen/shadcn/ui/input'
import { Label } from '@/gen/shadcn/ui/label'
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

import type { EditOrganizationFormData, SlugValidationResult } from '../type/member'

// EditOrganizationDialogState represents the dialog's internal state.
type EditOrganizationDialogState =
  | { status: 'idle' }
  | { status: 'submitting' }
  | { status: 'error'; message: string }

interface EditOrganizationDialogProps {
  // organizationId is the ID of the organization to edit
  organizationId: string
  // currentName is the current organization name
  currentName: string
  // currentSlug is the current organization slug
  currentSlug: string
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

// validateSlug validates the slug format.
// Returns validation result with isValid and errorMessage.
const validateSlug = (slug: string): SlugValidationResult => {
  // Implementation outline:
  // 1. Trim and lowercase slug.
  // 2. Check if empty -> error "Slug is required".
  // 3. Check if length < 3 -> error "Slug must be at least 3 characters".
  // 4. Check if length > 63 -> error "Slug must be at most 63 characters".
  // 5. Check if matches /^[a-z0-9]+(-[a-z0-9]+)*$/ -> error "Slug must contain only lowercase letters, numbers, and hyphens".
  // 6. Return { isValid: true, errorMessage: null } if all checks pass.
}

export const EditOrganizationDialog = ({
  organizationId,
  currentName,
  currentSlug,
  trigger,
}: EditOrganizationDialogProps) => {
  // Implementation outline:
  // 1. useState for:
  //    - isOpen: boolean (dialog open state)
  //    - formData: EditOrganizationFormData (name, slug)
  //    - state: EditOrganizationDialogState (idle, submitting, error)
  //    - slugValidation: SlugValidationResult
  // 2. Get mutation from useUpdateOrganization().
  // 3. Get setOrganizations from useUserStore().
  // 4. useEffect to reset form when dialog opens with current values.
  // 5. Create handleNameChange callback:
  //    a. Update formData.name.
  //    b. Clear error state.
  // 6. Create handleSlugChange callback:
  //    a. Update formData.slug (lowercase, trim).
  //    b. Validate slug and update slugValidation state.
  //    c. Clear error state.
  // 7. Create handleSubmit async callback:
  //    a. Validate slug one more time.
  //    b. If invalid, set error state and return.
  //    c. Set state to submitting.
  //    d. Try:
  //       i. Call mutation.mutateAsync({ organizationId, name, slug }).
  //       ii. Update zustand store with new organization data.
  //       iii. Close dialog.
  //    e. Catch (error):
  //       i. Map error.code to user-friendly message:
  //          - Code.PermissionDenied: "You don't have permission to edit this organization"
  //          - Code.InvalidArgument: error.message
  //          - Default: "An error occurred"
  //       ii. Set state to error with message.
  // 8. Create isFormValid computed:
  //    a. Return formData.name.trim() !== '' && slugValidation.isValid.
  // 9. Create hasChanges computed:
  //    a. Return formData.name !== currentName || formData.slug !== currentSlug.
  // 10. Return Dialog with form for editing name and slug.
}
```

---

## Step 11: Create Member List Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx`: Example table pattern

### `/Users/jayce/team-attention/cops/web/src/feature/organization/component/member-list.tsx`

**Description**:
Component displaying organization members with role management controls.

```typescript
import { useState, useCallback } from 'react'
import { Loader2, Shield, User, MoreHorizontal, UserMinus, ShieldCheck, ShieldOff } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useGetOrganizationMembers } from '../hook/use-get-organization-members'
import { useUpdateMemberRole } from '../hook/use-update-member-role'
import { useRemoveMember } from '../hook/use-remove-member'
import { useUserStore } from '@/shared/store/user-store'
import { Avatar, AvatarFallback, AvatarImage } from '@/gen/shadcn/ui/avatar'
import { Button } from '@/gen/shadcn/ui/button'
import { Badge } from '@/gen/shadcn/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/gen/shadcn/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/gen/shadcn/ui/alert-dialog'
import { Skeleton } from '@/gen/shadcn/ui/skeleton'
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert'

import type { MemberWithDetails } from '../type/member'

interface MemberListProps {
  // organizationId is the ID of the organization
  organizationId: string
  // isAdmin indicates if the current user has admin role
  isAdmin: boolean
  // currentUserId is the ID of the current user
  currentUserId: string
}

export const MemberList = ({ organizationId, isAdmin, currentUserId }: MemberListProps) => {
  // Implementation outline:
  // 1. useState for:
  //    - confirmDialog: { type: 'remove' | 'changeRole', member: MemberWithDetails, newRole?: string } | null
  //    - actionLoading: boolean
  //    - error: string | null
  // 2. Get members query from useGetOrganizationMembers({ organizationId }).
  // 3. Get mutations from useUpdateMemberRole() and useRemoveMember().
  // 4. Create handleRoleChange async callback:
  //    a. Set actionLoading true.
  //    b. Try:
  //       i. Call updateMemberRole.mutateAsync({ organizationId, userId, role }).
  //       ii. Refetch members query.
  //       iii. Close confirm dialog.
  //    c. Catch (error):
  //       i. Map error.code to user-friendly message.
  //       ii. Set error state.
  //    d. Set actionLoading false.
  // 5. Create handleRemoveMember async callback:
  //    a. Set actionLoading true.
  //    b. Try:
  //       i. Call removeMember.mutateAsync({ organizationId, userId }).
  //       ii. Refetch members query.
  //       iii. Close confirm dialog.
  //    c. Catch (error):
  //       i. Map error.code to user-friendly message.
  //       ii. Set error state.
  //    d. Set actionLoading false.
  // 6. If members query is loading, return skeleton loaders.
  // 7. If members query has error, return error alert.
  // 8. Render members list:
  //    a. For each member, display:
  //       i. Avatar with fallback.
  //       ii. Name and email.
  //       iii. Role badge (admin or member).
  //       iv. If isAdmin, show dropdown menu with:
  //          - Change role option (toggle between admin/member).
  //          - Remove member option.
  //          - Both disabled if member is current user and sole admin.
  // 9. Render confirmation AlertDialog for role change and remove actions.
}
```

---

## Step 12: Create Leave Organization Dialog Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/user/component/delete-account-dialog.tsx`: Example destructive dialog

### `/Users/jayce/team-attention/cops/web/src/feature/organization/component/leave-organization-dialog.tsx`

**Description**:
Dialog for leaving an organization with cascade delete warning.

```typescript
import { useState, useCallback } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { AlertTriangle, Loader2, LogOut } from 'lucide-react'
import { Code } from '@connectrpc/connect'
import { useLeaveOrganization } from '../hook/use-leave-organization'
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

// LeaveOrganizationDialogState represents the dialog's internal state.
type LeaveOrganizationDialogState =
  | { status: 'idle' }
  | { status: 'confirming' }
  | { status: 'error'; message: string }

interface LeaveOrganizationDialogProps {
  // organizationId is the ID of the organization to leave
  organizationId: string
  // organizationName is the name of the organization (for display)
  organizationName: string
  // isLastOrganization indicates if this is the user's last organization
  isLastOrganization: boolean
  // isSoleMember indicates if the user is the only member
  isSoleMember: boolean
  // trigger is the element that opens the dialog when clicked
  trigger: React.ReactNode
}

export const LeaveOrganizationDialog = ({
  organizationId,
  organizationName,
  isLastOrganization,
  isSoleMember,
  trigger,
}: LeaveOrganizationDialogProps) => {
  // Implementation outline:
  // 1. useState for:
  //    - isOpen: boolean (dialog open state)
  //    - confirmationInput: string (for last org, must type 'LEAVE')
  //    - state: LeaveOrganizationDialogState (idle, confirming, error)
  // 2. Get mutation from useLeaveOrganization().
  // 3. Get organizations, setOrganizations, setSelectedOrganizationId from useUserStore().
  // 4. Get navigate from useNavigate().
  // 5. Create handleLeave async callback:
  //    a. Set state to confirming.
  //    b. Try:
  //       i. Call mutation.mutateAsync({ organizationId }).
  //       ii. If isLastOrganization:
  //          - Clear user store organizations.
  //          - Navigate to '/create-organization' (or appropriate route).
  //       iii. Else:
  //          - Update organizations in store (remove this org).
  //          - Select first remaining organization.
  //       iv. Close dialog.
  //    c. Catch (error):
  //       i. Map error.code to user-friendly message:
  //          - Code.FailedPrecondition: "You cannot leave as the sole admin with other members"
  //          - Code.PermissionDenied: "You are not a member of this organization"
  //          - Default: error.message or "An error occurred"
  //       ii. Set state to error with message.
  // 6. Create isConfirmationRequired computed:
  //    a. Return isLastOrganization && isSoleMember.
  // 7. Create isConfirmationValid computed:
  //    a. If isConfirmationRequired, return confirmationInput === 'LEAVE'.
  //    b. Else return true.
  // 8. Return Dialog with:
  //    a. Warning about leaving organization.
  //    b. If isLastOrganization && isSoleMember:
  //       i. Show cascade delete warning (all projects, sessions will be deleted).
  //       ii. Require typing 'LEAVE' to confirm.
  //    c. If !isLastOrganization:
  //       i. Show simple leave confirmation.
  //    d. Leave button with loading state.
}
```

---

## Step 13: Create Organization Settings Section Component

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`: Existing settings page structure

### `/Users/jayce/team-attention/cops/web/src/feature/organization/component/organization-settings-section.tsx`

**Description**:
Main organization settings section to be added to Settings page.

```typescript
import { Building2, Users, Crown, User } from 'lucide-react'
import { useUserStore } from '@/shared/store/user-store'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/gen/shadcn/ui/card'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Button } from '@/gen/shadcn/ui/button'
import { Separator } from '@/gen/shadcn/ui/separator'
import { EditOrganizationDialog } from './edit-organization-dialog'
import { MemberList } from './member-list'
import { LeaveOrganizationDialog } from './leave-organization-dialog'

export const OrganizationSettingsSection = () => {
  // Implementation outline:
  // 1. Get user, organizations, selectedOrganizationId from useUserStore().
  // 2. Find current organization from organizations array.
  // 3. If no organization selected, return null or empty state.
  // 4. Determine if current user is admin:
  //    a. Find user's role in currentOrg.role (from zustand store).
  // 5. Calculate isLastOrganization = organizations.length === 1.
  // 6. Calculate isSoleMember (need to check from API or store).
  //    Note: This might require fetching members data.
  // 7. Return Card with:
  //    a. CardHeader with Building2 icon and "Organization Settings" title.
  //    b. CardContent with:
  //       i. Organization Info Section:
  //          - Display name, slug.
  //          - Display user's role badge (admin or member).
  //          - If admin, show Edit button triggering EditOrganizationDialog.
  //       ii. Separator.
  //       iii. Members Section:
  //          - "Members" heading.
  //          - MemberList component.
  //       iv. Separator.
  //       v. Leave Organization Section:
  //          - Warning text about leaving.
  //          - LeaveOrganizationDialog with Leave button trigger.
}
```

---

## Step 14: Update Settings Page

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`: Current settings page

### Modify `/Users/jayce/team-attention/cops/web/src/route/settings.tsx`

**Description**:
Add OrganizationSettingsSection above the Danger Zone.

```typescript
// Add import:
import { OrganizationSettingsSection } from '@/feature/organization/component/organization-settings-section'

// In SettingsPage component, add before Danger Zone Card:
{/* Organization Settings */}
<OrganizationSettingsSection />

{/* Spacer */}
<div className="h-8" />

{/* Danger Zone */}
// ... existing Danger Zone code
```

---

## Step 15: Update User Store for Organization Updates

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`: Current store

### Modify `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts`

**Description**:
Add action to update a specific organization's details.

```typescript
// Add to UserStoreActions interface:
updateOrganization: (organizationId: string, updates: Partial<OrganizationData>) => void
removeOrganization: (organizationId: string) => void

// Add implementations in create():
updateOrganization: (organizationId, updates) =>
  set((state) => ({
    organizations: state.organizations.map((org) =>
      org.id === organizationId ? { ...org, ...updates } : org
    ),
  })),

removeOrganization: (organizationId) =>
  set((state) => {
    const newOrganizations = state.organizations.filter(
      (org) => org.id !== organizationId
    )
    return {
      organizations: newOrganizations,
      // Auto-select first remaining organization if current was removed
      selectedOrganizationId:
        state.selectedOrganizationId === organizationId
          ? newOrganizations.length > 0
            ? newOrganizations[0].id
            : null
          : state.selectedOrganizationId,
    }
  }),
```

---

## Summary of Files

### Backend Files to Create

| Path | Description |
| :--- | :---------- |
| `/Users/jayce/team-attention/cops/idl/protobuf/organization/v1/organization.proto` | Organization service protobuf definition |
| `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/organization_repo_port.go` | Repository port interface |
| `/Users/jayce/team-attention/cops/api/internal/service/organization/outbound/repository/mongodb/organization_repo.go` | MongoDB repository implementation |
| `/Users/jayce/team-attention/cops/api/internal/service/organization/organization_service.go` | Organization service business logic |
| `/Users/jayce/team-attention/cops/api/internal/service/organization/inbound/grpc/connectrpc/handler.go` | ConnectRPC handler |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_organization.go` | DI container module |

### Backend Files to Modify

| Path | Description |
| :--- | :---------- |
| `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go` | Add organization module |

### Frontend Files to Create

| Path | Description |
| :--- | :---------- |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-update-organization.ts` | Update organization mutation hook |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-get-organization-members.ts` | Get members query hook |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-update-member-role.ts` | Update role mutation hook |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-remove-member.ts` | Remove member mutation hook |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/hook/use-leave-organization.ts` | Leave organization mutation hook |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/type/member.ts` | TypeScript types |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/component/edit-organization-dialog.tsx` | Edit organization dialog |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/component/member-list.tsx` | Members list component |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/component/leave-organization-dialog.tsx` | Leave organization dialog |
| `/Users/jayce/team-attention/cops/web/src/feature/organization/component/organization-settings-section.tsx` | Main settings section |

### Frontend Files to Modify

| Path | Description |
| :--- | :---------- |
| `/Users/jayce/team-attention/cops/web/src/route/settings.tsx` | Add organization settings section |
| `/Users/jayce/team-attention/cops/web/src/shared/store/user-store.ts` | Add organization update actions |

### Auto-Generated Files (after `buf generate`)

| Path | Description |
| :--- | :---------- |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organization.pb.go` | Go protobuf types |
| `/Users/jayce/team-attention/cops/shared/gen/grpcstub/organization/v1/organizationv1connect/organization.connect.go` | Go ConnectRPC handlers |
| `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/organization/v1/organization_pb.ts` | TypeScript protobuf types |
| `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/organization/v1/organization-OrganizationService_connectquery.ts` | TypeScript TanStack Query hooks |

---

## Execution Order

1. **Step 1**: Create organization.proto
2. **Step 2**: Generate gRPC stubs (`buf generate`)
3. **Step 3**: Create repository port interface
4. **Step 4**: Create MongoDB repository implementation
5. **Step 5**: Create organization service
6. **Step 6**: Create ConnectRPC handler
7. **Step 7**: Create DI container module and update application.go
8. **Steps 8-15**: Frontend implementation (can be done in parallel after Step 7)
