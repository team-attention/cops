# Requirements

## Request Summary

When registering a project using `cops add`, the CLI collects comprehensive project information including project name, git project status, and git remote URL. However, the API server only stores the remote URL in the MongoDB database, causing loss of project name and git status information. This needs to be fixed so that all collected project metadata is properly persisted to the database.

## Acceptance Criteria

- [ ] Criterion 1: MongoDB project schema includes `name` and `isGitProject` fields in addition to `remoteUrl`
- [ ] Criterion 2: API server's `RegisterProject` endpoint accepts name and isGitProject parameters from the CLI
- [ ] Criterion 3: When creating a new project, the API server stores name and isGitProject in MongoDB
- [ ] Criterion 4: When finding an existing project, the API server returns the stored name and isGitProject
- [ ] Criterion 5: CLI receives and displays the correct project name and git status after registration
- [ ] Criterion 6: Protobuf schema includes name and is_git_project fields in `RegisterProjectReq` and `RegisterProjectRes` messages
- [ ] Criterion 7: Existing data migration is not required (new fields can be added without breaking existing documents)

## Scope

### In Scope
- Update protobuf schema (`idl/protobuf/project/v1/project.proto`) to include name and is_git_project fields
- Regenerate gRPC stubs using `buf generate`
- Modify API server's `RegisterProject` logic to accept and store name/isGitProject
- Update MongoDB repository to persist name and isGitProject fields
- Update CLI's project registration flow to send name and isGitProject to API server
- Ensure backward compatibility with existing MongoDB documents (fields are optional/nullable)

### Out of Scope
- Migrating existing MongoDB documents to populate name/isGitProject fields
- Changing the project ID generation or duplicate detection logic
- Modifying the local config file structure (`~/.cops/config.json` or `.cops/config.json`)
- Updating other API endpoints beyond `RegisterProject`
- UI/web dashboard changes (this is backend-only)
- Storing project path in the MongoDB database (path remains CLI-local only)

## Constraints

- Must maintain backward compatibility with existing MongoDB documents that only have `remoteUrl` field
- Cannot change the duplicate detection logic (still uses remote URLs and existing project ID)
- Must follow protobuf naming conventions (`snake_case` for field names, `Req`/`Res` suffixes)
- Must follow Go hexagonal architecture patterns (Port/Adapter, proper layering)
- Generated code in `shared/gen/grpcstub/` must not be manually edited

## Additional Context

### Current Data Flow

1. **CLI Collection** (`cli/internal/service/tracking/inbound/cli/cobra/add.go`):
   - TUI collects: project name, path, git detection, sync preference
   - Service layer (`AddProject`) collects: name, path, isGitProject, configuredURL, actualURL

2. **API Call** (`cli/internal/service/tracking/tracking_service.go:109-113`):
   - CLI sends to API: `configuredURL`, `actualURL`, `existingProjectID`
   - **MISSING**: name and isGitProject are NOT sent to API

3. **API Storage** (`api/internal/service/project/outbound/repository/mongodb/project_repo.go:87-89`):
   - MongoDB stores: only `remoteUrl` field
   - **MISSING**: name and isGitProject are NOT stored

4. **Result**:
   - CLI has name/isGitProject but only stores locally in `~/.cops/config.json`
   - API/MongoDB has remote URL but loses name/isGitProject information
   - Data is inconsistent between CLI local storage and server database

### Current Schema Definitions

**Protobuf** (`idl/protobuf/project/v1/project.proto:9-19`):
```protobuf
message RegisterProjectReq {
  string configured_remote_url = 1;
  string actual_remote_url = 2;
  string existing_project_id = 3;
  // MISSING: name, is_git_project fields
}
```

**Domain Model** (`shared/domain/project.go:5-11`):
```go
type ProjectAbstract struct {
    ID   ID     `json:"id"`
    Name string `json:"name"`  // ✓ Has name
    Path string `json:"path"`  // Note: Path stored locally, not in MongoDB
}
```

**MongoDB Schema** (`shared/domain/mongoschema/project.go:12-22`):
```go
const (
    ProjectRemoteURLField = "remoteUrl"
    // MISSING: name and isGitProject field constants
)
```

### Related Files

- Protobuf definition: `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`
- CLI service: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
- CLI project client: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
- API service: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`
- API repository: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
- Domain model: `/Users/jayce/team-attention/cops/shared/domain/project.go`
- MongoDB schema: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`

## Questions Resolved

| Question | Answer |
| --- | --- |
| Should we migrate existing MongoDB documents to populate name/isGitProject? | No - out of scope. New fields should be optional/nullable for backward compatibility. |
| Should we change the duplicate detection logic? | No - duplicate detection remains the same (by remote URL and existing project ID). |
| Where is the project name currently coming from in the CLI? | The CLI TUI prompts the user for a project name, with a default value of the directory basename. It's stored locally but not sent to the API. |
| Should the API generate a default name if not provided? | Yes - the API should use a reasonable default (e.g., extract from remote URL or use a placeholder) if name is empty, to handle backward compatibility. |
| Are name and isGitProject required or optional in the API? | Optional for backward compatibility. The API should accept them if provided and store them in MongoDB. |
| Should we store the project path in MongoDB? | No - path remains CLI-local only in `~/.cops/config.json`. Only name and isGitProject are stored in MongoDB. |
