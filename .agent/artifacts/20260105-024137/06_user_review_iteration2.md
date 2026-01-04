# Review Result

**Status**: Changes Required

## Request Summary

User reported that the CLI organization selection UI is working, BUT the organization ID is NOT being saved to the database. The data flow investigation revealed that `OrganizationID` is being correctly passed through most of the chain (TUI → service → API → repository), but the MongoDB repository implementation is NOT including the `organizationId` field when inserting new project documents.

## Acceptance Criteria

- [ ] MongoDB repository must save `OrganizationID` when creating new projects
- [ ] MongoDB repository must return `OrganizationID` when reading existing projects
- [ ] Verify OrganizationID is persisted correctly in database after CLI project registration

## Scope

### In Scope
- Fix MongoDB repository to save and return OrganizationID
- Ensure data flow from TUI to DB is complete

### Out of Scope
- Any other refactoring or improvements not related to OrganizationID persistence
- UI/UX changes to organization selection

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` | 112-117 | N/A - Bug Fix | `createProject()` does not save `OrganizationID` to MongoDB document. The `params.OrganizationID` is available but not included in `newDoc` | Add `mongoschema.ProjectOrganizationIDField` to the bson.M document at line 112-117 with the converted ObjectID from `params.OrganizationID` |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` | 130-135 | N/A - Bug Fix | `createProject()` does not return `OrganizationID` in result | Add `OrganizationID: params.OrganizationID` to the `FindOrCreateResult` struct at line 130-135 |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` | 139-146 | N/A - Bug Fix | `docToResult()` does not extract `OrganizationID` from MongoDB document | Add OrganizationID extraction from `doc[mongoschema.ProjectOrganizationIDField]` and convert to hex string in the result |

## Data Flow Analysis

Complete trace of OrganizationID through the system:

### ✅ Working Sections

1. **TUI Selection** (`add_tui_update.go:258`)
   - `m.result.OrganizationID = m.selectedOrgID` ✅ CORRECT

2. **Command Handler** (`add.go:72`)
   - `params.OrganizationID = result.OrganizationID` ✅ CORRECT

3. **Service Layer** (`tracking_service.go:134`)
   - `OrganizationID: params.OrganizationID` passed to API ✅ CORRECT

4. **API Client** (`project_client.go:46`)
   - `OrganizationId: params.OrganizationID` sent in gRPC request ✅ CORRECT

5. **API Service** (`project_service.go:42`)
   - `OrganizationID: params.OrganizationID` passed to repository ✅ CORRECT

### ❌ Broken Section

6. **MongoDB Repository** (`project_repo.go:112-135`)
   - `createProject()` builds `newDoc` WITHOUT `organizationId` field ❌ **MISSING**
   - `createProject()` returns result WITHOUT `OrganizationID` field ❌ **MISSING**
   - `docToResult()` does not extract `organizationId` from document ❌ **MISSING**

## Detailed Fix Requirements

### Fix 1: Save OrganizationID to Database

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Function**: `createProject()`

**Current Code** (lines 112-117):
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
}
```

**Required Fix**:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
}

// Add organizationId field if provided
if params.OrganizationID != "" {
    if orgID, err := bson.ObjectIDFromHex(params.OrganizationID); err == nil {
        newDoc[mongoschema.ProjectOrganizationIDField] = orgID
    }
}
```

### Fix 2: Return OrganizationID from createProject()

**Current Code** (lines 130-135):
```go
return &repository.FindOrCreateResult{
    ProjectID:    newID,
    IsNew:        true,
    Name:         params.Name,
    IsGitProject: params.IsGitProject,
}, nil
```

**Required Fix**:
```go
return &repository.FindOrCreateResult{
    ProjectID:      newID,
    IsNew:          true,
    Name:           params.Name,
    IsGitProject:   params.IsGitProject,
    OrganizationID: params.OrganizationID,
}, nil
```

### Fix 3: Extract OrganizationID in docToResult()

**Current Code** (lines 139-146):
```go
func (r *MongoProjectRepository) docToResult(doc bson.M) *repository.FindOrCreateResult {
    return &repository.FindOrCreateResult{
        ProjectID:    doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex(),
        IsNew:        false,
        Name:         doc[mongoschema.ProjectNameField].(string),
        IsGitProject: doc[mongoschema.ProjectIsGitProjectField].(bool),
    }
}
```

**Required Fix**:
```go
func (r *MongoProjectRepository) docToResult(doc bson.M) *repository.FindOrCreateResult {
    result := &repository.FindOrCreateResult{
        ProjectID:    doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex(),
        IsNew:        false,
        Name:         doc[mongoschema.ProjectNameField].(string),
        IsGitProject: doc[mongoschema.ProjectIsGitProjectField].(bool),
    }

    // Extract OrganizationID if present
    if orgID, ok := doc[mongoschema.ProjectOrganizationIDField].(bson.ObjectID); ok {
        result.OrganizationID = orgID.Hex()
    }

    return result
}
```

## Additional Context

- The MongoDB schema already defines `ProjectOrganizationIDField = "organizationId"` in `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go:20`
- The `FindOrCreateResult` struct already includes `OrganizationID string` field in `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go:21`
- The entire data flow is working correctly EXCEPT for the MongoDB repository implementation
- This is a critical bug that prevents multi-tenant organization isolation from working

## Verification Steps

After implementing the fixes:

1. Run `cops add .` in a test directory
2. Select an organization from the TUI
3. Complete the project registration flow
4. Check MongoDB directly to verify `organizationId` field is present in the project document
5. Verify the API logs show the correct organizationID in the registration response

## Rules References

This is a bug fix, not a rule violation. However, the following rules apply:
- [`.agent/rules/go/go-backend.md`](/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md) - General backend coding standards
- [`.agent/rules/go/go-outbound.md`](/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md) - Outbound adapter patterns
