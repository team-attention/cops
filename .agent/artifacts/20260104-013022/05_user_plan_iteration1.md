# Implementation Plan: Fix UserOrganization Type Reference

## Overview

The API server's user gRPC handler currently references `userv1.UserOrganization`, which was removed when the protobuf schema was updated to use `domain.v1.Organization`. This causes a build error. The handler needs to be updated to map `repository.UserOrganization` (which contains both organization data and user role) to the new `domainv1.Organization` protobuf type.

**Key Challenge**: The old `userv1.UserOrganization` proto type included a `Role` field, but `domainv1.Organization` does not have a direct role field. Instead, it has a `Members` array of type `[]*OrganizationMember`, where each member has `user_id` and `role` fields. Based on the review document (Step 0 notes), the role field is "not needed for organization selection", so we will not populate the `Members` field in the response.

## Implementation Steps

### Step 1: Update Handler Type Mapping

**Files to Read**:
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler implementation patterns
- `.agent/rules/go/go-struct.md`: Go struct pointer/value type rules
- `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`

**Description**:
Update the `GetMe` method to map `repository.UserOrganization` to `domainv1.Organization` instead of the removed `userv1.UserOrganization` type.

**Changes**:

**Line 107**: Change type declaration from `userv1.UserOrganization` to `domainv1.Organization`

```go
// Before:
var protoOrgs []*userv1.UserOrganization

// After:
var protoOrgs []*domainv1.Organization
```

**Lines 110-114**: Update struct instantiation to use `domainv1.Organization` with appropriate fields

```go
// Before:
protoOrgs = append(protoOrgs, &userv1.UserOrganization{
    Id:   string(userOrg.Organization.ID),
    Name: userOrg.Organization.Name,
    Role: string(userOrg.Role),
})

// After:
protoOrgs = append(protoOrgs, &domainv1.Organization{
    Id:   string(userOrg.Organization.ID),
    Name: userOrg.Organization.Name,
    Slug: userOrg.Organization.Slug,
    // Note: Members field intentionally not populated for GetMe response
})
```

**Comment Update (Line 106)**: Update comment to reflect new type

```go
// Before:
// b. Map each UserOrganization to userv1.UserOrganization

// After:
// b. Map each UserOrganization to domainv1.Organization
```

**Implementation Details**:

1. **Import**: No import changes needed - `domainv1` is already imported at line 15
2. **Type Reference**: Change `userv1.UserOrganization` to `domainv1.Organization`
3. **Field Mapping**:
   - `Id`: Map from `userOrg.Organization.ID` (convert to string)
   - `Name`: Map from `userOrg.Organization.Name`
   - `Slug`: Map from `userOrg.Organization.Slug` (NEW - was not in old type)
   - `Members`: Do not populate (empty slice by default)
4. **Role Field**: The `userOrg.Role` value exists in `repository.UserOrganization` but is intentionally not included in the response, as per the review document's note that "Role field is not needed for organization selection"

**Test Scenarios**:

| Scenario | Input | Expected Output | Notes |
| :------- | :---- | :-------------- | :---- |
| User with organizations | `repository.UserOrganization` with valid Organization and Role | `domainv1.Organization` with Id, Name, Slug populated | Members field empty |
| Organization is nil | `repository.UserOrganization` with nil Organization | Organization skipped (not added to array) | Existing nil check preserved |
| Empty organizations array | Empty `result.Organizations` slice | Empty `protoOrgs` array | No iterations, empty result |
| Multiple organizations | Multiple `repository.UserOrganization` entries | Multiple `domainv1.Organization` entries | All organizations mapped |

## Quality Checklist

- [x] Concrete type signatures provided (`domainv1.Organization`)
- [x] No "or" statements - single approach specified
- [x] Test scenarios cover all branches (happy path, nil check, empty array)
- [x] Field mappings explicitly defined
- [x] Intentional omissions documented (Members field, Role value)
- [x] Import statements verified (no changes needed)
- [x] Follows ConnectRPC handler patterns from rules
