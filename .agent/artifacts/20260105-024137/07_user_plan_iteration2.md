# Implementation Plan: Fix OrganizationID Persistence in MongoDB Repository

## Overview

This plan addresses a critical bug where the `organizationId` field is not being persisted to MongoDB when creating new projects. The entire data flow from CLI TUI to API service layer is working correctly, but the MongoDB repository implementation fails to:

1. Save the `organizationId` field when inserting new project documents
2. Return the `OrganizationID` in the result after creating a project
3. Extract the `organizationId` when reading existing project documents

This bug prevents multi-tenant organization isolation from functioning correctly.

## Package Changes

None required. All necessary packages and schema definitions already exist:
- `go.mongodb.org/mongo-driver/v2/bson` - Already imported
- `mongoschema.ProjectOrganizationIDField` - Already defined as `"organizationId"`

## Step 1: Add OrganizationID Field to createProject() Document

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository adapter patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: MongoDB schema field constants

### `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Description**:
Modify the `createProject()` function to include the `organizationId` field in the MongoDB document when inserting new projects. The field should only be added when a valid OrganizationID is provided in params.

```go
// createProject inserts a new project document.
func (r *MongoProjectRepository) createProject(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
	// Implementation outline:
	// 1. Determine remoteURL from params (prefer ConfiguredURL, fallback to ActualURL)
	// 2. Build newDoc bson.M with required fields:
	//    - ProjectRemoteURLField: remoteURL
	//    - ProjectNameField: params.Name
	//    - ProjectIsGitProjectField: params.IsGitProject
	//    - ProjectRegisteredAtField: time.Now()
	// 3. If params.OrganizationID is not empty:
	//    a. Convert params.OrganizationID to bson.ObjectID using bson.ObjectIDFromHex()
	//    b. If conversion succeeds, add ProjectOrganizationIDField to newDoc
	//    c. If conversion fails, log warning but continue (don't fail the operation)
	// 4. Insert newDoc into collection
	// 5. On error, log and return error
	// 6. Extract newID from result.InsertedID
	// 7. Log success with projectID and name
	// 8. Return FindOrCreateResult with:
	//    - ProjectID: newID
	//    - IsNew: true
	//    - Name: params.Name
	//    - IsGitProject: params.IsGitProject
	//    - OrganizationID: params.OrganizationID
}
```

**Specific Changes to Lines 112-117**:

Current code:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
}
```

Replace with:
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

## Step 2: Return OrganizationID in createProject() Result

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go`: FindOrCreateResult struct definition

### `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Description**:
Modify the return statement in `createProject()` to include the `OrganizationID` field from the input params.

**Specific Changes to Lines 130-135**:

Current code:
```go
return &repository.FindOrCreateResult{
    ProjectID:    newID,
    IsNew:        true,
    Name:         params.Name,
    IsGitProject: params.IsGitProject,
}, nil
```

Replace with:
```go
return &repository.FindOrCreateResult{
    ProjectID:      newID,
    IsNew:          true,
    Name:           params.Name,
    IsGitProject:   params.IsGitProject,
    OrganizationID: params.OrganizationID,
}, nil
```

## Step 3: Extract OrganizationID in docToResult()

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: MongoDB schema field constants

### `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Description**:
Modify the `docToResult()` function to extract the `organizationId` field from MongoDB documents when reading existing projects. The field extraction must handle the case where the field may not exist (for legacy documents).

```go
// docToResult converts a MongoDB document to FindOrCreateResult.
func (r *MongoProjectRepository) docToResult(doc bson.M) *repository.FindOrCreateResult {
	// Implementation outline:
	// 1. Create result struct with required fields:
	//    - ProjectID: extract from doc[ProjectIDField], cast to bson.ObjectID, convert to Hex()
	//    - IsNew: false (since this is reading an existing document)
	//    - Name: extract from doc[ProjectNameField], cast to string
	//    - IsGitProject: extract from doc[ProjectIsGitProjectField], cast to bool
	// 2. Extract OrganizationID if present in document:
	//    a. Get value from doc[ProjectOrganizationIDField]
	//    b. Type assert to bson.ObjectID
	//    c. If type assertion succeeds, convert to Hex() and assign to result.OrganizationID
	//    d. If field is missing or wrong type, leave OrganizationID as empty string
	// 3. Return result
}
```

**Specific Changes to Lines 139-146**:

Current code:
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

Replace with:
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

## Test Scenarios

### createProject() - OrganizationID Persistence

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid OrganizationID provided | `params.OrganizationID = "507f1f77bcf86cd799439011"` | Document contains `organizationId` field with ObjectID value | Happy path with org |
| Empty OrganizationID | `params.OrganizationID = ""` | Document does NOT contain `organizationId` field | No org provided |
| Invalid OrganizationID format | `params.OrganizationID = "invalid-hex"` | Document does NOT contain `organizationId` field, no error | Invalid hex handling |

### createProject() - Result OrganizationID Return

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid OrganizationID provided | `params.OrganizationID = "507f1f77bcf86cd799439011"` | Result.OrganizationID = "507f1f77bcf86cd799439011" | Happy path |
| Empty OrganizationID | `params.OrganizationID = ""` | Result.OrganizationID = "" | No org case |

### docToResult() - OrganizationID Extraction

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Document has organizationId field | `doc["organizationId"] = bson.ObjectID{...}` | Result.OrganizationID contains hex string | Happy path |
| Document missing organizationId field | `doc` without `organizationId` key | Result.OrganizationID = "" | Legacy document |
| organizationId field has wrong type | `doc["organizationId"] = "string value"` | Result.OrganizationID = "" | Type mismatch handling |

## Verification Steps

After implementing the fixes:

1. Run `cops add .` in a test directory
2. Select an organization from the TUI
3. Complete the project registration flow
4. Check MongoDB directly to verify `organizationId` field is present in the project document:
   ```javascript
   db.projects.find().pretty()
   ```
5. Verify the API logs show the correct organizationID in the registration response
6. Run the existing project again with `cops add .` to verify `docToResult()` correctly extracts the organizationId

## Summary of Changes

| File | Function | Change |
|:-----|:---------|:-------|
| `project_repo.go` | `createProject()` | Add `organizationId` to newDoc bson.M when params.OrganizationID is valid |
| `project_repo.go` | `createProject()` | Add `OrganizationID: params.OrganizationID` to return struct |
| `project_repo.go` | `docToResult()` | Extract `organizationId` from doc and add to result struct |
