# Implementation Plan

## Overview

Add `name` and `isGitProject` fields to the project registration flow, enabling the CLI to send project name and git status to the API server, which will store them in MongoDB and return them in the response.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| N/A     | N/A     | N/A         | No new packages required - all dependencies already exist |

## Architecture Decisions

### Decision 1: Protobuf Field Naming

**Choice**: Use snake_case for protobuf fields (`name`, `is_git_project`)
**Rationale**: Per `protobuf.md` rules, all field names must use snake_case. The protobuf compiler will automatically convert these to PascalCase in generated Go code (`Name`, `IsGitProject`).

### Decision 2: Repository Interface Params Struct

**Choice**: Introduce `FindOrCreateParams` struct for the repository interface instead of individual parameters
**Rationale**: Per `go-backend.md` rules, functions with more than 3 parameters should use a params struct. With the addition of `name` and `isGitProject`, the `FindOrCreate` function will have 5 parameters (configuredURL, actualURL, existingID, name, isGitProject), exceeding the 3-parameter limit.

### Decision 3: Server-Side Name Handling

**Choice**: Server accepts and stores the name provided by CLI without modification or default generation
**Rationale**: The CLI always provides a name (with directory basename as default), so the server doesn't need fallback logic. This keeps the server implementation clean and simple.

### Decision 4: No Backward Compatibility Required

**Choice**: Treat `name` and `isGitProject` as required fields in the implementation
**Rationale**: The system has never been deployed in production, so there are no existing MongoDB documents to support. This allows for a cleaner implementation without optional field handling.

## Implementation Steps

### Step 1: Update Protobuf Schema

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto` (modify)

**Changes**:

Add `name` (field 4) and `is_git_project` (field 5) to `RegisterProjectReq`:

```protobuf
// RegisterProjectReq contains parameters for registering a project with the API server.
// The server performs duplicate detection using remote URLs and existing project ID.
message RegisterProjectReq {
  // configured_remote_url is from git config (git config --get remote.origin.url)
  string configured_remote_url = 1;

  // actual_remote_url is from git ls-remote (git ls-remote --get-url origin)
  // This may differ from configured URL if the GitHub repo was renamed
  string actual_remote_url = 2;

  // existing_project_id is optional - from local config if available
  // Used as fallback for finding existing projects
  string existing_project_id = 3;

  // name is the human-readable project name
  // If empty, the server will generate a default from the remote URL
  string name = 4;

  // is_git_project indicates whether this is a git repository
  bool is_git_project = 5;
}
```

Add `name` (field 3) and `is_git_project` (field 4) to `RegisterProjectRes`:

```protobuf
// RegisterProjectRes contains the result of project registration.
message RegisterProjectRes {
  // project_id is the MongoDB ObjectID hex representation
  string project_id = 1;

  // is_new indicates whether a new project was created (true) or existing found (false)
  bool is_new = 2;

  // name is the project name (either provided or generated)
  string name = 3;

  // is_git_project indicates whether this is a git repository
  bool is_git_project = 4;
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Valid proto compilation | Updated proto file | Generated Go code compiles | Happy path |

### Step 2: Regenerate gRPC Stubs

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/project/v1/project.pb.go` (auto-generated)
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/project/v1/projectv1connect/project.connect.go` (auto-generated)

**Command**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Successful generation | buf generate | No errors, files updated | Happy path |
| Generated code has new fields | Inspect generated code | `Name` and `IsGitProject` fields present | Verification |

### Step 3: Update API Repository Port

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go` (modify)

**Functions**:

```go
// FindOrCreateParams contains parameters for FindOrCreate operation.
type FindOrCreateParams struct {
	ConfiguredURL string
	ActualURL     string
	ExistingID    string
	Name          string
	IsGitProject  bool
}

// FindOrCreateResult contains the result of find-or-create operation.
type FindOrCreateResult struct {
	ProjectID    string
	IsNew        bool
	Name         string
	IsGitProject bool
}

// ProjectRepositoryPort defines the interface for project data persistence.
type ProjectRepositoryPort interface {
	// FindOrCreate finds existing project or creates new one.
	// Search order:
	// 1. By remote URL (either configured or actual)
	// 2. By existing project ID (if provided)
	// 3. Create new if not found
	FindOrCreate(ctx context.Context, params FindOrCreateParams) (*FindOrCreateResult, error)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Interface compiles | Updated interface | No compilation errors | Happy path |

### Step 4: Update MongoDB Repository

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` (modify)

**Functions**:

```go
// FindOrCreate finds existing project or creates new one.
// Search order:
// 1. By remote URL (either configured or actual)
// 2. By existing project ID (if provided)
// 3. Create new if not found
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
	// Implementation outline:
	// 1. Build $or conditions array with all search criteria (configuredURL, actualURL, existingID)
	// 2. Execute findOne with $or filter
	// 3. If found, return existing project with stored name and isGitProject
	// 4. If not found, create new document with name, isGitProject, and remoteURL (use values from params as-is)
	// 5. Return result with all fields populated
}
```

**Detailed Changes for FindOrCreate**:

Replace the entire `FindOrCreate` function with:

```go
// FindOrCreate finds existing project or creates new one.
// Search order:
// 1. By remote URL (either configured or actual)
// 2. By existing project ID (if provided)
// 3. Create new if not found
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
	// Build $or conditions array with all search criteria
	conditions := []bson.M{}

	// Add remote URL conditions
	if params.ConfiguredURL != "" {
		conditions = append(conditions, bson.M{mongoschema.ProjectRemoteURLField: params.ConfiguredURL})
	}
	if params.ActualURL != "" && params.ActualURL != params.ConfiguredURL {
		conditions = append(conditions, bson.M{mongoschema.ProjectRemoteURLField: params.ActualURL})
	}

	// Add existing ID condition if valid
	if params.ExistingID != "" {
		if objectID, err := bson.ObjectIDFromHex(params.ExistingID); err == nil {
			conditions = append(conditions, bson.M{mongoschema.ProjectIDField: objectID})
		}
	}

	// Validate at least one condition exists
	if len(conditions) == 0 {
		return nil, fmt.Errorf("no search criteria provided: at least one of configuredURL, actualURL, or existingID must be valid")
	}

	// Execute single findOne with $or filter
	filter := bson.M{"$or": conditions}
	var doc bson.M
	err := r.projectsColl.FindOne(ctx, filter).Decode(&doc)

	// If found, return existing project
	if err == nil {
		projectID := doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex()
		name := doc[mongoschema.ProjectNameField].(string)
		isGitProject := doc[mongoschema.ProjectIsGitProjectField].(bool)

		r.logger.Info("found existing project",
			slog.String("projectID", projectID))
		return &repository.FindOrCreateResult{
			ProjectID:    projectID,
			IsNew:        false,
			Name:         name,
			IsGitProject: isGitProject,
		}, nil
	}

	// If error is not "not found", return error
	if err != mongo.ErrNoDocuments {
		r.logger.Error("failed to find project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	// Not found, create new document
	// Prefer configured URL, fallback to actual URL
	remoteURL := params.ConfiguredURL
	if remoteURL == "" {
		remoteURL = params.ActualURL
	}

	newDoc := bson.M{
		mongoschema.ProjectRemoteURLField:    remoteURL,
		mongoschema.ProjectNameField:         params.Name,
		mongoschema.ProjectIsGitProjectField: params.IsGitProject,
	}

	result, err := r.projectsColl.InsertOne(ctx, newDoc)
	if err != nil {
		r.logger.Error("failed to create project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	newID := result.InsertedID.(bson.ObjectID).Hex()
	r.logger.Info("created new project",
		slog.String("projectID", newID),
		slog.String("name", params.Name),
		slog.String("remoteURL", remoteURL),
		slog.Bool("isGitProject", params.IsGitProject))

	return &repository.FindOrCreateResult{
		ProjectID:    newID,
		IsNew:        true,
		Name:         params.Name,
		IsGitProject: params.IsGitProject,
	}, nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Create new git project | `{Name: "my-project", IsGitProject: true, ConfiguredURL: "..."}` | Document created with name and isGitProject=true | New project creation |
| Create non-git project | `{Name: "my-project", IsGitProject: false, ConfiguredURL: ""}` | Document with isGitProject=false | Non-git project |
| Find existing project | Existing doc has name/isGitProject | Returns stored name and isGitProject | Found existing |

### Step 5: Update API Service

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go` (modify)

**Changes**:

1. Add `Name` and `IsGitProject` fields to `RegisterProjectParams`:

```go
// RegisterProjectParams contains parameters for registering a project.
type RegisterProjectParams struct {
	ConfiguredRemoteURL string
	ActualRemoteURL     string
	ExistingProjectID   string
	Name                string
	IsGitProject        bool
}
```

2. Update `RegisterProject` method to pass new fields to repository:

```go
// RegisterProject registers a project or returns existing project ID if already registered.
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
	result, err := s.repo.FindOrCreate(ctx, repository.FindOrCreateParams{
		ConfiguredURL: params.ConfiguredRemoteURL,
		ActualURL:     params.ActualRemoteURL,
		ExistingID:    params.ExistingProjectID,
		Name:          params.Name,
		IsGitProject:  params.IsGitProject,
	})
	if err != nil {
		s.logger.Error("failed to register project", slog.Any("error", err))
		return nil, err
	}

	s.logger.Info("project registered",
		slog.String("projectID", result.ProjectID),
		slog.Bool("isNew", result.IsNew),
		slog.String("name", result.Name),
		slog.Bool("isGitProject", result.IsGitProject))

	return result, nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Register with all fields | `{Name: "test", IsGitProject: true, ...}` | Success with name/isGitProject | Happy path |
| Repository error | Repository returns error | Error returned | Error handling |

### Step 6: Update API gRPC Handler

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go` (modify)

**Changes**:

1. Update params construction in `RegisterProject` to include new fields:

```go
	// Parse request
	msg := req.Msg
	params := projectservice.RegisterProjectParams{
		ConfiguredRemoteURL: msg.GetConfiguredRemoteUrl(),
		ActualRemoteURL:     msg.GetActualRemoteUrl(),
		ExistingProjectID:   msg.GetExistingProjectId(),
		Name:                msg.GetName(),
		IsGitProject:        msg.GetIsGitProject(),
	}
```

2. Update response construction to include new fields:

```go
	// Build response
	res := &projectv1.RegisterProjectRes{
		ProjectId:    result.ProjectID,
		IsNew:        result.IsNew,
		Name:         result.Name,
		IsGitProject: result.IsGitProject,
	}

	return connect.NewResponse(res), nil
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Request with name | `{Name: "test", IsGitProject: true}` | Response with same values | Happy path |
| Service error | Service returns error | gRPC error returned | Error handling |

### Step 7: Update CLI Project Port

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go` (modify)

**Changes**:

1. Add `Name` and `IsGitProject` fields to `RegisterProjectParams`:

```go
// RegisterProjectParams contains parameters for registering a project with the API server.
type RegisterProjectParams struct {
	// ConfiguredRemoteURL is from git config (git config --get remote.origin.url)
	ConfiguredRemoteURL string

	// ActualRemoteURL is from git ls-remote (git ls-remote --get-url origin)
	// This may differ from configured URL if the GitHub repo was renamed
	ActualRemoteURL string

	// ExistingProjectID is optional - from local config if available
	// Used as fallback for finding existing projects
	ExistingProjectID string

	// Name is the human-readable project name
	Name string

	// IsGitProject indicates whether this is a git repository
	IsGitProject bool
}
```

2. Add `Name` and `IsGitProject` fields to `RegisterProjectResult`:

```go
// RegisterProjectResult contains the result of project registration.
type RegisterProjectResult struct {
	ProjectID    domain.ID
	IsNew        bool
	Name         string
	IsGitProject bool
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Struct compiles | Updated struct | No compilation errors | Happy path |

### Step 8: Update CLI ConnectRPC Client

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` (modify)

**Changes**:

1. Update request construction in `RegisterProject`:

```go
	req := connect.NewRequest(&projectv1.RegisterProjectReq{
		ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
		ActualRemoteUrl:     params.ActualRemoteURL,
		ExistingProjectId:   params.ExistingProjectID,
		Name:                params.Name,
		IsGitProject:        params.IsGitProject,
	})
```

2. Update result construction:

```go
	result := &api.RegisterProjectResult{
		ProjectID:    domain.ID(resp.Msg.ProjectId),
		IsNew:        resp.Msg.IsNew,
		Name:         resp.Msg.Name,
		IsGitProject: resp.Msg.IsGitProject,
	}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Send request with name | `{Name: "test", IsGitProject: true}` | Request sent with fields | Happy path |
| Receive response with name | Server returns name | Result contains name | Response parsing |
| API error | Server returns error | Error returned | Error handling |

### Step 9: Update CLI Tracking Service

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (modify)

**Changes**:

Update the API call (around line 109-113) to include `Name` and `IsGitProject`:

```go
	// Always call API to register project
	result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
		ConfiguredRemoteURL: configuredURL,
		ActualRemoteURL:     actualURL,
		ExistingProjectID:   existingProjectID,
		Name:                name,
		IsGitProject:        isGitProject,
	})
```

Note: The `name` variable is already defined at line 82 and `isGitProject` is defined at line 61 in the existing code.

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Git project with name | Git repo with custom name | API called with name and isGitProject=true | Happy path |
| Non-git project | Directory without git | API called with isGitProject=false | Non-git |
| API success | API returns successfully | Project created locally | Success flow |
| API failure with existing ID | API fails but local ID exists | Uses local ID | Fallback |

## Execution Order

1. **Step 1**: Update Protobuf Schema (no dependencies)
2. **Step 2**: Regenerate gRPC Stubs (depends on Step 1)
3. **Step 3**: Update API Repository Port (depends on Step 2 for type references)
4. **Step 4**: Update MongoDB Repository (depends on Step 3)
5. **Step 5**: Update API Service (depends on Step 3, Step 4)
6. **Step 6**: Update API gRPC Handler (depends on Step 2, Step 5)
7. **Step 7**: Update CLI Project Port (depends on Step 2 for type references)
8. **Step 8**: Update CLI ConnectRPC Client (depends on Step 2, Step 7)
9. **Step 9**: Update CLI Tracking Service (depends on Step 7, Step 8)

## Notes for Execute Agent

- **MongoDB field constants already exist**: `mongoschema.ProjectNameField` and `mongoschema.ProjectIsGitProjectField` are already defined in `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go` - no need to add them
- **No domain model changes needed**: The `domain.Project` struct already has `Name` and `IsGitProject` fields
- **Clean implementation**: The system has never been deployed, so implement cleanly without backward compatibility workarounds
- **Build verification**: Run `go build ./...` from the project root after each step to verify compilation
- **Regenerate stubs carefully**: After Step 2, verify the generated code has the new `Name` and `IsGitProject` fields in both request and response messages before proceeding
- **Order matters for testing**: Steps 1-6 (API side) can be tested independently with API unit tests. Steps 7-9 (CLI side) require the API to be updated first for end-to-end testing
- **Line references**: In Step 9, `name` is the local variable from line 82 and `isGitProject` is from line 61 of tracking_service.go - these already exist and just need to be passed to the API
