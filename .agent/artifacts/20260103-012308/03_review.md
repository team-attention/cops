# Review Result

**Status**: Changes Required

## Request Summary

Code review identified rule violations that need to be addressed. The implementation does not fully follow project standards defined in `.agent/rules/`. Please address the violations listed below.

## Acceptance Criteria

- [ ] Remove `CreatedAt` field from `shared/domain/device_code.go` domain model (or provide justification in code comment)
- [ ] Add `bson:"-"` tag to `UserID` field in domain model to exclude it from BSON serialization

## Scope

### In Scope
- Fix identified rule violations
- Ensure all changes follow applicable rules

### Out of Scope
- Any other refactoring or improvements not related to rule violations
- Feature additions beyond fixing violations

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
| ---- | ---- | ---- | ----- | ------------- |
| `shared/domain/device_code.go` | 13 | `go/go-platform-domain.md` | `CreatedAt` field is metadata unrelated to business logic - rules say "Avoid adding fields that are metadata unrelated to business logic (e.g., `CreatedAt`, `UpdatedAt`)" | Remove `CreatedAt` field from DeviceCode struct OR add a comment explaining why it is necessary for business logic |
| `shared/domain/device_code.go` | 10 | `go/go-platform-domain.md` | `UserID` field is an ID field and should use `bson:"-"` tag. Per rules: "ALWAYS exclude ID fields with `bson:\"-\"`" since ID fields are handled in MongoSchema layer | Change from `UserID *ID \`json:"userId,omitempty" bson:"userId,omitempty"\`` to `UserID *ID \`json:"userId,omitempty" bson:"-"\`` |

## Additional Context

- Requirements document: `.agent/artifacts/20260103-012308/01_requirements.md`
- Plan document: `.agent/artifacts/20260103-012308/02_plan.md`
- Review triggered by changes to 18 files (7 new, 11 modified)

## Files Reviewed

### New Files
- `shared/domain/device_code.go`
- `shared/domain/mongoschema/device_code.go`
- `api/internal/service/auth/outbound/repository/device_code_repo_port.go`
- `api/internal/service/auth/outbound/repository/mongodb/device_code_repo.go`
- `api/internal/service/auth/devicecode/devicecode.go`
- `doc/mongodb-indexes.md`

### Modified Files
- `api/internal/service/auth/auth_service.go`
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`
- `api/cmd/internal/container/module_auth.go`
- `api/internal/platform/setup/config/config.go`
- `cli/internal/service/auth/inbound/cli/cobra/login.go`
- `idl/protobuf/auth/v1/auth.proto`
- `shared/gen/grpcstub/auth/v1/auth.pb.go`
- `shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go`
- `web/src/gen/grpcstub/auth/v1/auth-AuthService_connectquery.ts`
- `web/src/gen/grpcstub/auth/v1/auth_pb.ts`

## Rules Applied

- `.agent/rules/common.md`
- `.agent/rules/workflow.md`
- `.agent/rules/go/go-struct.md`
- `.agent/rules/go/go-platform-domain.md`
- `.agent/rules/go/go-platform-domain-mongoschema.md`
- `.agent/rules/go/go-outbound.md`
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`
- `.agent/rules/go/go-service.md`
- `.agent/rules/idl/protobuf.md`

## Detailed Analysis

### Domain Model (`shared/domain/device_code.go`)

**Compliant:**
- Uses `domain.ID` type for entity identifiers
- Uses pointer type for optional `UserID` field with `omitempty` tag
- Uses singular entity name (DeviceCode)
- Uses camelCase for JSON field names
- ID field correctly uses `json:"-" bson:"-"`
- All comments are in English

**Violations:**
1. **`CreatedAt` field** - According to `go-platform-domain.md`: "Avoid adding fields that are metadata unrelated to business logic (e.g., `CreatedAt`, `UpdatedAt`)". Unless there is a specific business requirement for this field beyond metadata purposes, it should be removed.

2. **`UserID` field BSON tag** - According to `go-platform-domain.md`: "ALWAYS exclude ID fields with `bson:\"-\"`" because ID fields are handled in the MongoSchema layer. The `UserID` field should have `bson:"-"` tag.

### MongoSchema (`shared/domain/mongoschema/device_code.go`)

**Fully Compliant:**
- Follows mandatory file structure order (imports, collection const, field consts, struct, FromDomain, ToDomain)
- Uses `{Entity}CollectionName` naming pattern
- Defines field constants for all fields using `{Entity}{FieldName}Field` pattern
- Embeds domain model with `bson:",inline"`
- Overrides ID fields as `bson.ObjectID`
- FromDomain correctly validates empty strings before conversion
- FromDomain uses implicit error ignore (`_`)
- ToDomain correctly converts ObjectIDs to hex and wraps in `domain.ID()`
- No factory functions created (only FromDomain and ToDomain methods)

### Repository Port (`api/internal/service/auth/outbound/repository/device_code_repo_port.go`)

**Fully Compliant:**
- Interface named `DeviceCodeRepositoryPort` following `{Domain}{Category}Port` pattern
- Methods have clear documentation
- Uses context.Context as first parameter
- Returns domain types, not database-specific types

### Repository Implementation (`api/internal/service/auth/outbound/repository/mongodb/device_code_repo.go`)

**Fully Compliant:**
- Struct named `MongoDeviceCodeRepository` following `{Technology}{Domain}{Category}` pattern
- Constructor `NewMongoDeviceCodeRepository` follows naming convention
- Logger correctly bound with `l.With(slog.String("name", "auth.repository.mongodb.device_code"))`
- Accepts pre-initialized `*mongo.Database` from platform/setup (not config)
- Interface verification present: `var _ repository.DeviceCodeRepositoryPort = (*MongoDeviceCodeRepository)(nil)`
- Uses mongoschema field constants for queries
- Error handling follows patterns (log errors, return wrapped errors)

### Device Code Utility (`api/internal/service/auth/devicecode/devicecode.go`)

**Fully Compliant:**
- Package location appropriate (auth service-specific utility)
- Uses `crypto/rand` for secure random generation
- All comments in English
- Clear function documentation

### Auth Service (`api/internal/service/auth/auth_service.go`)

**Fully Compliant:**
- Logger bound correctly in constructor
- Dependencies injected through constructor
- Service methods follow pattern (validate, business logic, return)
- Error handling includes logging with context
- Uses parameter struct pattern for `DeviceCodeApproveParams`

### ConnectRPC Handler (`api/internal/service/auth/inbound/grpc/connectrpc/handler.go`)

**Fully Compliant:**
- Package name `connectrpc`
- Logger name `"auth.grpc.connectrpc"`
- Interface verification present
- Handler struct follows `{Domain}GRPCHandler` pattern
- JWT validation done directly in handler (ConnectRPC handlers don't use Fiber middleware)
- Error mapping to appropriate connect error codes

### Module Registration (`api/cmd/internal/container/module_auth.go`)

**Fully Compliant:**
- Uses `fx.Annotate` with `fx.As` for interface type conversion
- Group pattern correctly applied
- Device code repository registered with proper interface casting

### Configuration (`api/internal/platform/setup/config/config.go`)

**Fully Compliant:**
- New config struct `DeviceCodeConfig` added
- Uses `env` tags for environment variable binding
- Provides sensible defaults (`envDefault`)
- Required field marked with `required` tag

### Protobuf (`idl/protobuf/auth/v1/auth.proto`)

**Fully Compliant:**
- Uses `Req`/`Res` suffix naming (not `Request`/`Response`)
- Uses snake_case for field names
- Service and RPC follow naming conventions
- Comments provided for new messages and RPC

### CLI Login (`cli/internal/service/auth/inbound/cli/cobra/login.go`)

**Fully Compliant:**
- Minimal change as planned
- Display message updated appropriately
- No new functionality added beyond display changes

### MongoDB Index Documentation (`doc/mongodb-indexes.md`)

**Fully Compliant:**
- Documents TTL index for automatic expiration
- Documents unique index on userCode
- Includes explanatory notes

## Code Quality Notes (Non-Blocking)

The following observations are noted but do not require changes:

1. **JWT Config Duplication**: The `jwtutil.Config` is constructed multiple times in `auth_service.go` (lines 88-93, 133-138, 262-267, 276-281). Consider extracting to a helper method for DRY compliance. This is a code quality suggestion, not a rule violation.

2. **Error Message Consistency**: Error messages use string matching in the handler (e.g., `strings.Contains(errMsg, "not found")`). Consider using custom error types for more robust error handling. This is a code quality suggestion, not a rule violation.

3. **Generated Code**: The web/shared generated files (`auth.pb.go`, `auth_pb.ts`, etc.) are correctly regenerated from protobuf definitions.

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](.agent/rules/common.md) - Common rules for all files
- [`.agent/rules/workflow.md`](.agent/rules/workflow.md) - Workflow rules
- [`.agent/rules/go/go-struct.md`](.agent/rules/go/go-struct.md) - Go struct pointer/value type rules
- [`.agent/rules/go/go-platform-domain.md`](.agent/rules/go/go-platform-domain.md) - Domain model guidelines
- [`.agent/rules/go/go-platform-domain-mongoschema.md`](.agent/rules/go/go-platform-domain-mongoschema.md) - MongoSchema guidelines
- [`.agent/rules/go/go-outbound.md`](.agent/rules/go/go-outbound.md) - Outbound adapter guidelines
- [`.agent/rules/go/go-inbound-grpc-connectrpc.md`](.agent/rules/go/go-inbound-grpc-connectrpc.md) - ConnectRPC handler guidelines
- [`.agent/rules/go/go-service.md`](.agent/rules/go/go-service.md) - Service implementation guidelines
- [`.agent/rules/idl/protobuf.md`](.agent/rules/idl/protobuf.md) - Protobuf conventions
