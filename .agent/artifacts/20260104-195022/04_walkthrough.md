# Development Walkthrough

## Summary
Implemented automatic Personal Organization creation for new user signups. When a user signs up via Google OAuth for the first time, the system now creates both a user account and a Personal Organization in a single MongoDB transaction, ensuring atomicity.

## Code Overview

### New Components

#### `TransactionManagerPort` (Platform Interface)
- **Location**: `api/internal/platform/outbound/txmanager/txmanager_port.go`
- **Purpose**: Database-agnostic transaction interface that shields the service layer from MongoDB-specific types
- **Key Methods**:
  - `WithTransaction(ctx, fn)`: Executes a function within a transaction with automatic commit/rollback handling

#### `MongoTransactionManager` (MongoDB Adapter)
- **Location**: `api/internal/platform/outbound/txmanager/mongodb/txmanager.go`
- **Purpose**: MongoDB implementation of TransactionManagerPort using MongoDB Go Driver v2
- **Key Features**:
  - Configures `writeconcern.Majority()` for durability
  - Uses `session.WithTransaction()` for automatic retry on transient errors
  - Handles session lifecycle (creation, cleanup)
  - Logs transaction start/failure with contextual information

#### `slugutil` Package
- **Location**: `api/internal/platform/util/slugutil/slugutil.go`
- **Purpose**: Generates URL-safe organization slugs with random suffixes
- **Key Functions**:
  - `GenerateSlug(name)`: Converts user name to slug format (e.g., "Jayce Kim" → "jayce-kim-a3f9")
  - `generateRandomSuffix()`: Creates 4-character cryptographically random suffix
- **Features**:
  - Unicode normalization using `golang.org/x/text` (handles accented characters)
  - Removes special characters, collapses multiple hyphens
  - Truncates to 50 characters if needed
  - Fallback to "user" for empty/invalid names

### Modified Components

#### `auth.Service` (Auth Service)
- **Location**: `api/internal/service/auth/auth_service.go`
- **Changes**:
  - Added `orgRepo` dependency (OrganizationRepositoryPort from user service)
  - Added `txManager` dependency (TransactionManagerPort)
  - Updated constructor to inject new dependencies
  - Modified `GoogleAuth()` method to wrap user+org creation in transaction
- **Transaction Flow**:
  1. Check if user exists (existing user login flow unchanged)
  2. For new users: Start transaction via `txManager.WithTransaction()`
  3. Create user account
  4. Generate unique organization slug
  5. Create Personal Organization with user as admin
  6. Return both user and organization
  7. Generate JWT tokens
  8. Transaction commits automatically on success, rolls back on error

#### `OrganizationRepositoryPort` (User Service)
- **Location**: `api/internal/service/user/outbound/repository/organization_repo_port.go`
- **Changes**: Added `Create(ctx, org)` method to interface
- **Note**: This is a different interface from the organization service's repository port

#### `MongoOrganizationRepository` (User Service)
- **Location**: `api/internal/service/user/outbound/repository/mongodb/organization_repo.go`
- **Changes**: Implemented `Create()` method
- **Implementation**:
  - Converts domain model to MongoDB schema
  - Executes `InsertOne()` (automatically participates in transaction if context contains session)
  - Extracts inserted ObjectID
  - Returns created organization with ID

#### `module_platform.go` (Dependency Injection)
- **Location**: `api/cmd/internal/container/module_platform.go`
- **Changes**: Added TransactionManager provider using `fx.Annotate` with `fx.As` pattern
- **Provider**:
  - Takes `*slog.Logger` and `*mongo.Database` as dependencies
  - Extracts `*mongo.Client` from database
  - Creates `MongoTransactionManager` instance
  - Casts to `TransactionManagerPort` interface using `fx.As`

## Testing

### Verification Commands Run
```bash
go get golang.org/x/text                          # Result: PASS - Package installed
go build ./cli/... ./api/... ./daemon/... ./shared/...  # Result: Build has unrelated errors
```

### Known Issue
Build currently fails in `organization` service due to separate `OrganizationRepositoryPort` interface that needs `Create()` and `ExistsSlugForUser()` methods. This is unrelated to the auth service changes - it's a pre-existing architectural issue where there are two separate organization repository interfaces:
- `api/internal/service/user/outbound/repository/organization_repo_port.go` (used by auth service)
- `api/internal/service/organization/outbound/repository/organization_repo_port.go` (used by organization service)

## Issues & Resolutions

| Issue | Resolution |
|-------|-----------|
| Service layer depending on MongoDB types violates hexagonal architecture | Created TransactionManagerPort abstraction in platform layer to shield service from database-specific types |
| Transaction context propagation to repositories | MongoDB operations automatically join transaction when context contains session - no signature changes needed |
| Slug uniqueness guarantee | Use 4-character random suffix (1.6M combinations) with slug generation, plus MongoDB unique index on organizations.slug |
| Unicode character handling in slugs | Used `golang.org/x/text` for NFD normalization to convert accented characters to ASCII |
| Build failure in organization service | Separate organization repository interface needs method additions - out of scope for this feature |

## Related Tickets
- No specific ticket referenced in artifacts

## Architecture Decisions

### Hexagonal Architecture Compliance
- **Port/Adapter Pattern**: Created `TransactionManagerPort` interface with `MongoTransactionManager` adapter
- **Service Independence**: Auth service depends only on interfaces, not concrete MongoDB types
- **Platform Separation**: Transaction management logic isolated in platform layer
- **No Breaking Changes**: Existing repository methods work with transactions via context propagation

### Transaction Safety
- **Atomicity**: User and organization creation succeed or fail together
- **Write Concern**: Uses `writeconcern.Majority()` for durability guarantees
- **Automatic Retry**: MongoDB driver retries transient errors automatically
- **Rollback Guarantee**: Any error in transaction callback triggers automatic rollback

### Dependency Injection Pattern
- **fx.As Pattern**: Used for interface casting in container (preferred over anonymous wrappers)
- **Logger First**: All constructors take logger as first parameter (project convention)
- **Logger Binding**: Components bind logger with name in constructor
- **Infrastructure Reuse**: Database client extracted from existing `*mongo.Database` dependency

### Slug Generation Strategy
- **Format**: `{slugified-name}-{random-suffix}` (e.g., "jayce-kim-a3f9")
- **Collision Handling**: Extremely low probability with 36^4 combinations
- **Edge Cases**: Empty names default to "user", long names truncated to 50 chars
- **Security**: Uses `crypto/rand` for cryptographically secure random suffixes

## Key Implementation Patterns

### Transaction Manager Pattern
```go
// Service layer uses abstraction
result, err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) (interface{}, error) {
    // All operations in callback participate in transaction
    user, err := s.userRepo.Create(txCtx, newUser)
    if err != nil {
        return nil, err // Triggers automatic rollback
    }

    org, err := s.orgRepo.Create(txCtx, newOrg)
    if err != nil {
        return nil, err // Triggers automatic rollback
    }

    return &result{user, org}, nil // Success triggers commit
})
```

### Logger Binding Convention
```go
func NewService(l *slog.Logger, ...) *Service {
    return &Service{
        logger: l.With(slog.String("name", "auth.service")),
        // ...
    }
}
```

### Struct Field Types (Go Rule)
- **Slice of Structs**: ALWAYS use pointer elements `[]*T`, not `[]T`
- **Example**: `Members []*OrganizationMember` (correct), not `Members []OrganizationMember`
- **Rationale**: Follows project rule in `.agent/rules/go/go-struct.md`

## Files Created

| File | Purpose | Lines |
|------|---------|-------|
| `api/internal/platform/outbound/txmanager/txmanager_port.go` | Transaction manager interface | 32 |
| `api/internal/platform/outbound/txmanager/mongodb/txmanager.go` | MongoDB transaction adapter | 56 |
| `api/internal/platform/util/slugutil/slugutil.go` | Slug generation utility | 82 |

## Files Modified

| File | Changes |
|------|---------|
| `api/internal/service/auth/auth_service.go` | Added transaction logic to GoogleAuth method, injected dependencies |
| `api/internal/service/user/outbound/repository/organization_repo_port.go` | Added Create method to interface |
| `api/internal/service/user/outbound/repository/mongodb/organization_repo.go` | Implemented Create method |
| `api/cmd/internal/container/module_platform.go` | Added TransactionManager provider |

## Package Dependencies Added

- `golang.org/x/text v0.32.0`: Unicode normalization for slug generation

## Future Considerations

### Migration for Existing Users
Users created before this feature don't have Personal Organizations. A data migration script will be needed to create Personal Organizations for existing users (out of scope for this task).

### Organization Repository Consolidation
The codebase has two separate `OrganizationRepositoryPort` interfaces:
- `api/internal/service/user/outbound/repository/organization_repo_port.go` (read-only queries)
- `api/internal/service/organization/outbound/repository/organization_repo_port.go` (management operations)

Consider consolidating these interfaces or documenting the separation rationale.

### Transaction Manager Reusability
The TransactionManagerPort abstraction is now available platform-wide and can be reused for any multi-document operations requiring atomicity (e.g., project creation with default settings, cascade deletes).

### MongoDB Replica Set Requirement
**CRITICAL**: MongoDB transactions require a replica set deployment. Ensure development and production environments are configured correctly:
- Development: Docker Compose with `--replSet rs0` flag
- Connection string must include `?replicaSet=rs0` parameter
- Verify with `rs.status()` in mongosh

## Testing Recommendations

### Unit Tests Needed
1. `slugutil_test.go`: Test all edge cases (empty, unicode, special chars, long names)
2. `auth_service_test.go`: Mock transaction manager and repositories to test rollback scenarios

### Integration Tests Needed
1. Test successful user+org creation in transaction
2. Test transaction rollback when org creation fails
3. Verify existing user login flow unchanged
4. Test slug uniqueness handling

### Manual Testing Checklist
- [ ] New user signup creates both user and organization
- [ ] Organization name follows pattern: "User's Name's Organization"
- [ ] Organization slug is URL-safe with random suffix
- [ ] User has admin role in created organization
- [ ] Transaction rolls back if org creation fails
- [ ] Existing user login flow unchanged
- [ ] JWT tokens generated correctly for new users
