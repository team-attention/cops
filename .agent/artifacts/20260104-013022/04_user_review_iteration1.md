# Review Result

**Status**: Changes Required

## Request Summary

Code review identified a build error caused by the protobuf schema change. The `user.proto` was updated to use `domain.v1.Organization` instead of `userv1.UserOrganization`, but the API server's user handler still references the old (now removed) `userv1.UserOrganization` type. This causes the API server to fail to compile.

## Acceptance Criteria

- [ ] Update handler to use `domainv1.Organization` instead of `userv1.UserOrganization`
- [ ] Map `repository.UserOrganization.Organization` fields to `domainv1.Organization` fields
- [ ] Include user's role from `repository.UserOrganization.Role` in the response (via `domainv1.OrganizationMember`)

## Scope

### In Scope
- Fix the type reference error in `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`
- Map the service result to the new protobuf schema

### Out of Scope
- Any other refactoring or improvements not related to fixing the build error
- Changes to protobuf definitions (already correctly updated)

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
| ---- | ---- | ---- | ----- | ------------- |
| `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go` | 107 | N/A (Build Error) | `userv1.UserOrganization` type no longer exists - it was removed from `user.proto` when the schema was updated to use `domain.v1.Organization` | Change `var protoOrgs []*userv1.UserOrganization` to `var protoOrgs []*domainv1.Organization` |
| `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go` | 110 | N/A (Build Error) | Creating instance of non-existent type `userv1.UserOrganization` | Change `&userv1.UserOrganization{...}` to `&domainv1.Organization{...}` with fields: `Id`, `Name`, `Slug` (from `userOrg.Organization`) |

## Detailed Fix

The handler needs to be updated to convert `repository.UserOrganization` to `domainv1.Organization`.

**Current code (lines 106-116):**
```go
// b. Map each UserOrganization to userv1.UserOrganization
var protoOrgs []*userv1.UserOrganization
for _, userOrg := range result.Organizations {
    if userOrg.Organization != nil {
        protoOrgs = append(protoOrgs, &userv1.UserOrganization{
            Id:   string(userOrg.Organization.ID),
            Name: userOrg.Organization.Name,
            Role: string(userOrg.Role),
        })
    }
}
```

**Corrected code:**
```go
// b. Map each UserOrganization to domainv1.Organization
var protoOrgs []*domainv1.Organization
for _, userOrg := range result.Organizations {
    if userOrg.Organization != nil {
        protoOrgs = append(protoOrgs, &domainv1.Organization{
            Id:   string(userOrg.Organization.ID),
            Name: userOrg.Organization.Name,
            Slug: userOrg.Organization.Slug,
            // Note: Members field is not populated as it's not needed for GetMe response
        })
    }
}
```

**Note on Role field**: The old `userv1.UserOrganization` had a `Role` field, but `domainv1.Organization` does not have this field directly - it has a `Members` field instead. According to the plan document (Step 0), the Role field is "not needed for organization selection". If the role needs to be exposed, a different approach would be required (such as a separate field in GetMeRes or populating the Members field).

## Additional Context

- Requirements document: `/Users/jayce/team-attention/cops/.agent/artifacts/20260104-013022/01_clarify.md`
- Plan document: `/Users/jayce/team-attention/cops/.agent/artifacts/20260104-013022/02_plan.md`
- Review triggered by API build error

## Files Reviewed

- `/Users/jayce/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc/handler.go`

## Rules Applied

- `/Users/jayce/team-attention/cops/.agent/rules/common.md`
- `/Users/jayce/team-attention/cops/.agent/rules/workflow.md`
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md`

## Supporting Evidence

### Proto schema (user.proto) now uses domain.v1.Organization:
From `/Users/jayce/team-attention/cops/shared/gen/grpcstub/user/v1/user.pb.go` (line 66):
```go
Organizations []*v1.Organization `protobuf:"bytes,2,rep,name=organizations,proto3" json:"organizations,omitempty"`
```

### domainv1.Organization struct (from domain.pb.go lines 148-156):
```go
type Organization struct {
    Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
    Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
    Slug          string                 `protobuf:"bytes,3,opt,name=slug,proto3" json:"slug,omitempty"`
    Members       []*OrganizationMember  `protobuf:"bytes,4,rep,name=members,proto3" json:"members,omitempty"`
    // ...
}
```

### Service returns repository.UserOrganization (from user_service.go):
```go
type GetMeResult struct {
    User          *domain.User
    Organizations []*repository.UserOrganization
}
```

### repository.UserOrganization contains both Organization and Role (from organization_repo_port.go):
```go
type UserOrganization struct {
    Organization *domain.Organization
    Role         domain.MemberRole
}
```
