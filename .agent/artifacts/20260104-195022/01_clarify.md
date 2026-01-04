# Requirements: Auto-create Personal Organization on User Signup

## Request Summary

When a new user signs up (logs in for the first time via Google OAuth), the system should automatically create a default Personal Organization for that user. Currently, only the user account is created, leaving new users without any organization membership. This prevents them from creating projects or using core functionality that requires organization context.

## Acceptance Criteria

- [ ] New user creation automatically creates a Personal Organization in the same transaction
- [ ] Personal Organization is named using pattern: `{User's Name}'s Organization` (e.g., "Jayce Kim's Organization")
- [ ] Personal Organization has a unique slug generated from user's name and random suffix
- [ ] User is added as the sole member with "admin" role in their Personal Organization
- [ ] Transaction ensures atomicity: if organization creation fails, user creation is also rolled back
- [ ] Existing user login flow (user already exists) is unchanged
- [ ] MongoDB transaction is used to ensure data consistency
- [ ] Logs clearly indicate Personal Organization creation success or failure

## Current Behavior Analysis

**File: `api/internal/service/auth/auth_service.go`**

The `GoogleAuth` method handles both existing user login and new user signup:

```go
// Lines 117-159: New User Creation Flow
newUser := &domain.User{
    Email:           userInfo.Email,
    Name:            userInfo.Name,
    ProfileImageURL: userInfo.Picture,
    Accounts: []*domain.Account{
        {
            Provider:   domain.AccountProviderGoogle,
            ProviderID: userInfo.ID,
        },
    },
}

createdUser, err := s.userRepo.Create(ctx, newUser)
// ... token generation ...
```

**Problem:**
- Only creates user account via `userRepo.Create()`
- Does not create any organization
- User ends up authenticated but without organization membership
- Cannot create projects (requires organizationID)

**Existing Resources:**
- `domain.Organization` model exists with Name, Slug, Members fields
- `domain.OrganizationMember` model exists with UserID and Role
- Organization repository likely exists (needs verification in planning phase)
- MongoDB collections: `users`, `organizations` already indexed

## Desired Behavior

**Updated New User Creation Flow:**

1. User logs in via Google OAuth for the first time (user does not exist)
2. System starts MongoDB transaction
3. System creates user account with OAuth details
4. System generates unique organization slug from user's name + random suffix
5. System creates Personal Organization with:
   - Name: `{user.Name}'s Organization` (e.g., "Jayce Kim's Organization")
   - Slug: `{slugified-name}-{random-suffix}` (e.g., "jayce-kim-a3f9")
   - Members: Single member (the new user) with role "admin"
6. System commits transaction
7. System generates JWT tokens
8. If any step fails: rollback entire transaction, return error

**User Experience:**
- User signs up once
- Immediately has their Personal Organization available
- Can start creating projects without additional setup

## Technical Requirements

### 1. Transaction Support

**Requirement:** Use MongoDB transaction to ensure atomicity of user + organization creation.

#### Transaction Manager Abstraction (Hexagonal Architecture)

**Architecture Decision**: To maintain hexagonal architecture, the service layer must NOT depend on MongoDB-specific types like `*mongo.Client` or `mongo.SessionContext`. Instead, create a **Transaction Manager** abstraction that hides database-specific implementation details.

**Package Structure:**

```
api/internal/platform/outbound/txmanager/
├── txmanager_port.go          # Interface (Port)
└── mongodb/
    └── txmanager.go            # MongoDB implementation (Adapter)
```

#### Transaction Manager Port (Interface)

Create database-agnostic transaction interface:

```go
// File: api/internal/platform/outbound/txmanager/txmanager_port.go
package txmanager

import "context"

// TransactionFunc is a function that executes within a transaction.
// The ctx parameter contains the transaction context and must be passed to all repository operations.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
type TransactionFunc func(ctx context.Context) (interface{}, error)

// TransactionManagerPort defines the interface for managing database transactions.
// This abstraction allows the service layer to use transactions without depending on
// database-specific types (e.g., mongo.Client, mongo.SessionContext).
type TransactionManagerPort interface {
    // WithTransaction executes a function within a transaction.
    // It automatically handles session creation, commit, rollback, and cleanup.
    //
    // Parameters:
    //   - ctx: Parent context for timeout/cancellation
    //   - fn: Function to execute within transaction
    //
    // Returns:
    //   - result: Value returned by fn if transaction commits successfully
    //   - error: Error from fn (triggers rollback) or transaction infrastructure error
    //
    // Behavior:
    //   - If fn returns (result, nil): Transaction commits, returns (result, nil)
    //   - If fn returns (nil, error): Transaction rolls back, returns (nil, error)
    //   - Automatically retries on transient errors (network issues, etc.)
    WithTransaction(ctx context.Context, fn TransactionFunc) (interface{}, error)
}
```

**Key Design Points:**

1. **No MongoDB Types**: Interface uses only `context.Context`, not `mongo.SessionContext`
2. **Callback Pattern**: Follows same pattern as MongoDB's `WithTransaction()` but abstracted
3. **Database Agnostic**: Could be implemented for PostgreSQL, MySQL, etc. in the future
4. **Automatic Management**: Handles session lifecycle, commit, rollback internally

#### MongoDB Transaction Manager Implementation

Implement the port using MongoDB Go Driver v2:

```go
// File: api/internal/platform/outbound/txmanager/mongodb/txmanager.go
package mongodb

import (
    "context"
    "fmt"
    "log/slog"

    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
    "go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

    "github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
)

// MongoTransactionManager implements TransactionManagerPort using MongoDB.
type MongoTransactionManager struct {
    client *mongo.Client
    logger *slog.Logger
}

// NewMongoTransactionManager creates a new MongoDB transaction manager.
func NewMongoTransactionManager(client *mongo.Client, logger *slog.Logger) *MongoTransactionManager {
    return &MongoTransactionManager{
        client: client,
        logger: logger.With(slog.String("name", "platform.txmanager.mongodb")),
    }
}

// WithTransaction executes a function within a MongoDB transaction.
func (m *MongoTransactionManager) WithTransaction(ctx context.Context, fn txmanager.TransactionFunc) (interface{}, error) {
    // 1. Configure transaction options with write concern majority
    txnOpts := options.Transaction().SetWriteConcern(writeconcern.Majority())
    sessOpts := options.Session().SetDefaultTransactionOptions(txnOpts)

    // 2. Start MongoDB session
    session, err := m.client.StartSession(sessOpts)
    if err != nil {
        m.logger.Error("failed to start MongoDB session",
            slog.Any("error", err),
        )
        return nil, fmt.Errorf("failed to start transaction session: %w", err)
    }
    defer session.EndSession(context.TODO())

    // 3. Execute transaction using MongoDB's WithTransaction helper
    result, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
        // Call the user-provided function with the session context
        // sessCtx implements context.Context interface, so it can be passed as context.Context
        return fn(sessCtx)
    })

    if err != nil {
        m.logger.Error("transaction failed and was rolled back",
            slog.Any("error", err),
        )
        return nil, err
    }

    return result, nil
}

// Interface verification
var _ txmanager.TransactionManagerPort = (*MongoTransactionManager)(nil)
```

**Implementation Details:**

1. **Wraps MongoDB Client**: Encapsulates `*mongo.Client` dependency
2. **Write Concern**: Configures `writeconcern.Majority()` for durability
3. **Session Management**: Handles session creation and cleanup
4. **Context Conversion**: Converts `mongo.SessionContext` to `context.Context` interface
5. **Error Propagation**: Forwards transaction errors to caller

#### Service Layer Usage

With the transaction manager abstraction, the service layer remains clean:

```go
// In auth.Service.GoogleAuth() method

// Execute user signup within transaction
result, err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) (interface{}, error) {
    // All operations inside this callback participate in the transaction

    // Create user (pass txCtx to repository)
    createdUser, err := s.userRepo.Create(txCtx, newUser)
    if err != nil {
        s.logger.Error("failed to create user in transaction",
            slog.String("email", newUser.Email),
            slog.Any("error", err),
        )
        return nil, err // Returning error triggers automatic rollback
    }

    // Generate organization slug
    orgSlug, err := generateOrganizationSlug(createdUser.Name)
    if err != nil {
        s.logger.Error("failed to generate organization slug",
            slog.String("userID", string(createdUser.ID)),
            slog.Any("error", err),
        )
        return nil, err // Triggers rollback
    }

    // Create organization
    newOrg := &domain.Organization{
        Name: fmt.Sprintf("%s's Organization", createdUser.Name),
        Slug: orgSlug,
        Members: []*domain.OrganizationMember{
            {
                UserID: createdUser.ID,
                Role:   domain.MemberRoleAdmin,
            },
        },
    }

    createdOrg, err := s.orgRepo.Create(txCtx, newOrg)
    if err != nil {
        s.logger.Error("failed to create personal organization in transaction",
            slog.String("userID", string(createdUser.ID)),
            slog.String("orgSlug", orgSlug),
            slog.Any("error", err),
        )
        return nil, err // Triggers rollback
    }

    // Return both created entities
    return map[string]interface{}{
        "user": createdUser,
        "org":  createdOrg,
    }, nil
    // If function returns nil error, transaction commits automatically
    // If function returns error, transaction rolls back automatically
})

// Check transaction result
if err != nil {
    // Transaction was rolled back
    s.logger.Error("user signup transaction failed and was rolled back",
        slog.String("email", userInfo.Email),
        slog.Any("error", err),
    )
    return nil, fmt.Errorf("failed to create user account: %w", err)
}

// Extract results from transaction
resultMap := result.(map[string]interface{})
createdUser := resultMap["user"].(*domain.User)
createdOrg := resultMap["org"].(*domain.Organization)

s.logger.Info("new user created with personal organization",
    slog.String("userID", string(createdUser.ID)),
    slog.String("email", createdUser.Email),
    slog.String("organizationID", string(createdOrg.ID)),
    slog.String("organizationSlug", createdOrg.Slug),
)

// Continue with JWT token generation...
```

**Service Layer Benefits:**

- ✅ No MongoDB imports in service layer
- ✅ Uses `context.Context` instead of `mongo.SessionContext`
- ✅ Clean, testable code (can mock `TransactionManagerPort`)
- ✅ Follows hexagonal architecture principles
- ✅ Database implementation can be swapped without changing service code

**How WithTransaction Works:**

- `session.WithTransaction()` is a helper method that handles the full transaction lifecycle
- **Automatic Commit**: If callback returns `(result, nil)`, transaction commits automatically
- **Automatic Rollback**: If callback returns `(nil, error)`, transaction rolls back automatically
- **Retry Logic**: Automatically retries on transient transaction errors
- **No Manual Commit/Abort**: Don't call `session.CommitTransaction()` or `session.AbortTransaction()` when using `WithTransaction()`

#### Context Propagation to Repositories

**Critical Requirement**: All repository operations must receive `mongo.SessionContext` to participate in the transaction.

**Current Repository Signatures** (need verification):
```go
// auth/outbound/repository/user_repo_port.go
type UserRepositoryPort interface {
    Create(ctx context.Context, user *domain.User) (*domain.User, error)
}

// Will use the same method but pass mongo.SessionContext
// mongo.SessionContext satisfies context.Context interface
```

**Repository Implementation Pattern**:

Repository methods already accept `context.Context`. When called with `mongo.SessionContext`, MongoDB operations automatically join the transaction:

```go
// In mongodb/user_repo.go
func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    var schema mongoschema.User
    schema.FromDomain(user)

    // InsertOne automatically uses transaction if ctx contains session
    result, err := r.usersColl.InsertOne(ctx, schema)
    if err != nil {
        return nil, err
    }

    // ... rest of implementation
}
```

**Key Points:**

1. **No signature changes needed**: `mongo.SessionContext` implements `context.Context` interface
2. **Automatic participation**: MongoDB operations detect session in context and join transaction
3. **Existing methods work**: Current repository methods already support transactions via context
4. **Type assertion**: `mongo.SessionContext` is a `context.Context` with additional session methods

#### Transaction Manager Dependency Injection

**Updated auth.Service Structure:**

```go
// File: api/internal/service/auth/auth_service.go
type Service struct {
    logger         *slog.Logger
    cfg            *config.Config
    oauthPort      oauth.GoogleOAuthPort
    userRepo       repository.UserRepositoryPort
    deviceCodeRepo repository.DeviceCodeRepositoryPort
    orgRepo        userrepo.OrganizationRepositoryPort  // NEW: For creating organizations
    txManager      txmanager.TransactionManagerPort      // NEW: For transaction management
}

func NewService(
    l *slog.Logger,
    cfg *config.Config,
    oauthPort oauth.GoogleOAuthPort,
    userRepo repository.UserRepositoryPort,
    deviceCodeRepo repository.DeviceCodeRepositoryPort,
    orgRepo userrepo.OrganizationRepositoryPort,  // NEW
    txManager txmanager.TransactionManagerPort,    // NEW
) *Service {
    return &Service{
        logger:         l.With(slog.String("name", "auth.service")),
        cfg:            cfg,
        oauthPort:      oauthPort,
        userRepo:       userRepo,
        deviceCodeRepo: deviceCodeRepo,
        orgRepo:        orgRepo,    // NEW
        txManager:      txManager,  // NEW
    }
}
```

**Container Module Updates:**

**1. Register Transaction Manager** (in `api/cmd/internal/container/module_platform.go` or new file):

```go
// File: api/cmd/internal/container/module_platform.go
package container

import (
    "go.uber.org/fx"
    "log/slog"

    "github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
    mongotx "github.com/team-attention/cops/api/internal/platform/outbound/txmanager/mongodb"
    "go.mongodb.org/mongo-driver/v2/mongo"
)

var PlatformModule = fx.Module("platform",
    // Provide transaction manager
    fx.Provide(
        func(db *mongo.Database, l *slog.Logger) txmanager.TransactionManagerPort {
            // Extract client from database
            client := db.Client()
            return mongotx.NewMongoTransactionManager(client, l)
        },
    ),
)
```

**2. Update Auth Service Module** (in `api/cmd/internal/container/module_auth.go`):

```go
// File: api/cmd/internal/container/module_auth.go
package container

import (
    "go.uber.org/fx"
    "log/slog"

    "github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
    "github.com/team-attention/cops/api/internal/platform/setup/config"
    "github.com/team-attention/cops/api/internal/service/auth"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
    authrepo "github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
    userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
)

var AuthModule = fx.Module("auth",
    // ... existing oauth and repository providers ...

    // Provide auth service with transaction manager
    fx.Provide(
        func(
            l *slog.Logger,
            cfg *config.Config,
            oauth oauth.GoogleOAuthPort,
            userRepo authrepo.UserRepositoryPort,
            deviceRepo authrepo.DeviceCodeRepositoryPort,
            orgRepo userrepo.OrganizationRepositoryPort,  // NEW: Inject org repo
            txManager txmanager.TransactionManagerPort,    // NEW: Inject tx manager
        ) *auth.Service {
            return auth.NewService(
                l,
                cfg,
                oauth,
                userRepo,
                deviceRepo,
                orgRepo,    // NEW
                txManager,  // NEW
            )
        },
    ),

    // ... existing handler providers ...
)
```

**Why Transaction Manager Abstraction:**

- ✅ **Hexagonal Architecture**: Service layer has no database-specific dependencies
- ✅ **Testability**: Can mock `TransactionManagerPort` in unit tests
- ✅ **Flexibility**: Can swap database implementation (PostgreSQL, etc.) without changing service
- ✅ **Clean Code**: Transaction logic encapsulated in platform layer, not service layer
- ✅ **Separation of Concerns**: Database details hidden from business logic

#### Transaction Options

Transaction options are configured inside the `MongoTransactionManager` implementation, not in the service layer:

```go
// In api/internal/platform/outbound/txmanager/mongodb/txmanager.go
func (m *MongoTransactionManager) WithTransaction(ctx context.Context, fn txmanager.TransactionFunc) (interface{}, error) {
    // Configure transaction options with write concern majority
    txnOpts := options.Transaction().SetWriteConcern(writeconcern.Majority())
    sessOpts := options.Session().SetDefaultTransactionOptions(txnOpts)

    session, err := m.client.StartSession(sessOpts)
    // ...
}
```

**Write Concern Configuration:**
- Use `writeconcern.Majority()` for production (already configured in implementation above)
- Ensures transaction is committed to majority of replica set members
- Provides durability guarantee even if primary fails
- Service layer doesn't need to worry about these details - they're encapsulated in the transaction manager

#### Error Handling in Transactions

**Errors that Trigger Rollback:**

Any error returned from `WithTransaction` callback triggers automatic rollback:

```go
session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
    if err := operation1(sessCtx); err != nil {
        return nil, err  // Triggers rollback, no manual cleanup needed
    }
    if err := operation2(sessCtx); err != nil {
        return nil, err  // Triggers rollback
    }
    return result, nil  // Triggers commit
})
```

**Common Transaction Errors:**

| Error Type | Cause | Handling |
|------------|-------|----------|
| Duplicate key error | Organization slug collision | Retry slug generation (up to 3 attempts) |
| Write conflict | Concurrent transaction on same documents | MongoDB driver retries automatically |
| Transient transaction error | Network issue, replica set election | MongoDB driver retries automatically |
| Non-transient error | Validation failure, schema error | Return error, transaction rolls back |

**MongoDB Driver v2 Automatic Retry:**

The `WithTransaction()` helper automatically retries on:
- `UnknownTransactionCommitResult` errors
- Transient transaction errors (network issues)
- Does NOT retry on validation or duplicate key errors

#### Transaction Implementation Summary

**What You Need to Do:**

1. **Create Transaction Manager Abstraction (Platform Layer)**
   - Create interface: `api/internal/platform/outbound/txmanager/txmanager_port.go`
   - Create MongoDB implementation: `api/internal/platform/outbound/txmanager/mongodb/txmanager.go`
   - Register in fx container: `api/cmd/internal/container/module_platform.go`

2. **Add Dependencies to auth.Service**
   - Inject `TransactionManagerPort` (for transactions)
   - Inject `OrganizationRepositoryPort` (for creating organizations)
   - Update service constructor

3. **Add Organization Repository Create Method**
   - Add `Create()` method to `OrganizationRepositoryPort` interface (if missing)
   - Implement in `mongodb/organization_repo.go`

4. **Wrap User Creation in Transaction**
   - Call `s.txManager.WithTransaction(ctx, func(txCtx context.Context) {...})`
   - Pass `txCtx` to all repository methods inside callback
   - Return both user and org from callback
   - Transaction manager handles commit/rollback automatically

5. **No Repository Signature Changes Needed**
   - Existing methods work with transactions via `context.Context`
   - MongoDB operations automatically join transaction when session is in context

**Key Architectural Points:**

- ✅ **Hexagonal Architecture**: Service layer has no `mongo.*` imports
- ✅ **Port & Adapter**: `TransactionManagerPort` is port, `MongoTransactionManager` is adapter
- ✅ **Clean Separation**: Database logic in platform layer, business logic in service layer
- ✅ **Testability**: Mock `TransactionManagerPort` in unit tests
- ✅ **Use `txManager.WithTransaction()`** - it handles everything automatically
- ✅ **Pass transaction context to repositories** - propagates through `context.Context`
- ✅ **Return error to rollback** - return `(nil, err)` from callback
- ✅ **Return result to commit** - return `(result, nil)` from callback
- ❌ **Don't inject `*mongo.Client`** into service layer - violates hexagonal architecture
- ❌ **Don't call MongoDB transaction APIs** in service - use transaction manager abstraction
- ❌ **Don't create special transaction methods** in repositories - use existing methods with transaction context

### 2. Organization Slug Generation

**Requirement:** Generate unique, URL-safe slug from user's name with random suffix.

**Algorithm:**
```
1. Take user's name (e.g., "Jayce Kim")
2. Convert to lowercase
3. Replace spaces with hyphens
4. Remove non-alphanumeric characters (except hyphens)
5. Append 4-character random alphanumeric suffix
6. Result: "jayce-kim-a3f9"
```

**Uniqueness Strategy:**
- 4-character alphanumeric suffix = 36^4 = 1,679,616 combinations
- Slug collision probability is extremely low
- If collision occurs (MongoDB unique index violation), retry with new suffix
- Maximum 3 retry attempts before returning error

**Edge Cases:**
- Empty name: Use "user" as base (slug: "user-a3f9")
- Very long name: Truncate to 50 characters before adding suffix
- Special characters: Remove all except alphanumeric and hyphens
- Unicode characters: Transliterate to ASCII (e.g., "José" → "jose")

### 3. Repository Changes

**Auth Service Dependencies:**

The `auth.Service` currently only has:
- `userRepo` (UserRepositoryPort)
- `deviceCodeRepo` (DeviceCodeRepositoryPort)
- `mongoClient` **(NEW)** - MongoDB client for starting sessions

**New Dependency Required:**
- Add `orgRepo` (OrganizationRepositoryPort) to `auth.Service`
  - Can reuse existing organization repository from `user` service
  - Port: `api/internal/service/user/outbound/repository/organization_repo_port.go`
  - Implementation: `api/internal/service/user/outbound/repository/mongodb/organization_repo.go`

**Organization Repository Port:**

Check if `OrganizationRepositoryPort` has a `Create` method. If not, add:

```go
// In api/internal/service/user/outbound/repository/organization_repo_port.go
type OrganizationRepositoryPort interface {
    // Create creates a new organization.
    // Participates in transaction if ctx contains mongo.SessionContext.
    // Returns created organization with generated ID.
    Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

    // ... other existing methods (GetUserOrganizations, etc.) ...
}
```

**Organization Repository Implementation:**

Add corresponding MongoDB implementation:

```go
// In api/internal/service/user/outbound/repository/mongodb/organization_repo.go
func (r *MongoOrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
    var schema mongoschema.Organization
    schema.FromDomain(org)

    // InsertOne automatically participates in transaction if ctx is mongo.SessionContext
    result, err := r.orgColl.InsertOne(ctx, schema)
    if err != nil {
        r.logger.Error("failed to create organization",
            slog.String("slug", org.Slug),
            slog.Any("error", err),
        )
        return nil, err
    }

    insertedID, ok := result.InsertedID.(bson.ObjectID)
    if !ok {
        return nil, fmt.Errorf("failed to get inserted organization ID")
    }

    schema.ID = insertedID
    return schema.ToDomain(), nil
}
```

**No Special Transaction Methods Needed:**

- **Existing repository methods work with transactions** via `context.Context`
- `mongo.SessionContext` implements `context.Context` interface
- MongoDB operations automatically detect session in context and participate in transaction
- **No need for separate `CreateWithTransaction()` methods**
- Just pass `mongo.SessionContext` to existing `Create()` methods

**Dependency Injection:**

The auth service needs to be able to use the organization repository:

```go
// In api/cmd/internal/container/module_auth.go
import (
    userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
)

fx.Provide(
    func(
        l *slog.Logger,
        cfg *config.Config,
        oauth oauth.GoogleOAuthPort,
        userRepo authrepo.UserRepositoryPort,
        deviceRepo authrepo.DeviceCodeRepositoryPort,
        orgRepo userrepo.OrganizationRepositoryPort,  // NEW: Inject org repo
        db *mongo.Database,
    ) *auth.Service {
        return auth.NewService(
            l,
            cfg,
            oauth,
            userRepo,
            deviceRepo,
            orgRepo,        // NEW
            db.Client(),    // NEW: For transactions
        )
    },
),
```

### 4. Error Handling

**Transaction Failures:**

| Scenario | Action | Log Level | Error Message |
|----------|--------|-----------|---------------|
| User creation fails | Rollback transaction | ERROR | "failed to create user in transaction: {error}" |
| Organization creation fails | Rollback transaction | ERROR | "failed to create personal organization: {error}" |
| Slug generation fails (3 retries) | Rollback transaction | ERROR | "failed to generate unique organization slug after retries" |
| Transaction commit fails | Transaction auto-rollback | ERROR | "failed to commit user signup transaction: {error}" |
| MongoDB session creation fails | Return error immediately | ERROR | "failed to start transaction session: {error}" |

**Rollback Behavior:**
- MongoDB driver handles automatic rollback on error
- No manual cleanup needed (transaction guarantees)
- Return error to caller (ConnectRPC handler returns connect.CodeInternal)

### 5. Logging Requirements

**Success Case:**
```go
s.logger.Info("new user created with personal organization",
    slog.String("userID", string(createdUser.ID)),
    slog.String("email", createdUser.Email),
    slog.String("organizationID", string(createdOrg.ID)),
    slog.String("organizationSlug", createdOrg.Slug),
)
```

**Failure Cases:**
- Log transaction start
- Log each creation step (user, org)
- Log transaction rollback with reason
- Include userEmail in all error logs (userID not available yet)

## Scope

### In Scope

- Create transaction manager abstraction (platform layer)
  - `api/internal/platform/outbound/txmanager/txmanager_port.go` - Interface
  - `api/internal/platform/outbound/txmanager/mongodb/txmanager.go` - MongoDB implementation
  - `api/cmd/internal/container/module_platform.go` - Fx module registration
- Modify `auth.Service.GoogleAuth()` to create Personal Organization for new users
- Add organization repository dependency to auth service
- Add transaction manager dependency to auth service
- Implement slug generation utility function
- Wrap user + organization creation in transaction using transaction manager
- Add `Create()` method to organization repository if missing
- Update auth service constructor to inject new dependencies
- Update fx modules to provide transaction manager and organization repository
- Error handling and automatic rollback logic
- Logging for transaction flow

### Out of Scope

- Creating Personal Organization for existing users (migration script - separate task)
- UI changes to display Personal Organization
- Ability to rename Personal Organization (use existing organization update functionality)
- Device code flow changes (only affects Google OAuth flow)
- CLI authentication flow (uses same backend API)
- Organization deletion on user deletion (already handled by existing cascade delete logic)

## Constraints

### Technical Constraints

1. **MongoDB Transaction Requirements:**

   **CRITICAL**: MongoDB transactions require a replica set deployment. Standalone MongoDB instances do not support transactions.

   **Replica Set Requirement:**
   - Transactions only work on MongoDB running as a replica set
   - Minimum: 1-node replica set (for development)
   - Production: 3-node replica set (for high availability)

   **Development Environment Setup:**

   If using Docker Compose for local development, MongoDB must be initialized as a replica set:

   ```yaml
   # docker-compose.yml
   mongodb:
     image: mongo:8.0
     command: ["--replSet", "rs0", "--bind_ip_all"]
     ports:
       - "27017:27017"
     environment:
       MONGO_INITDB_ROOT_USERNAME: admin
       MONGO_INITDB_ROOT_PASSWORD: password
     healthcheck:
       test: echo "try { rs.status() } catch (err) { rs.initiate({_id:'rs0',members:[{_id:0,host:'localhost:27017'}]}) }" | mongosh --username admin --password password --authenticationDatabase admin
       interval: 5s
       timeout: 5s
       retries: 3
   ```

   **Connection String:**
   ```
   mongodb://admin:password@localhost:27017/?replicaSet=rs0
   ```

   **Important**: The connection string must include `?replicaSet=rs0` parameter.

   **Verification:**

   To verify replica set is configured:
   ```bash
   mongosh "mongodb://admin:password@localhost:27017/?replicaSet=rs0"
   > rs.status()  # Should show replica set status, not error
   ```

   **Error if Replica Set Not Configured:**
   ```
   (CommandNotSupported) This MongoDB deployment does not support retryable writes.
   Please add retryWrites=false to your connection string.
   ```

   If you see this error, MongoDB is running in standalone mode, not replica set mode.

2. **Service and Repository Layers:**
   - Must maintain hexagonal architecture
   - **Service Layer**: No database-specific imports (no `mongo.*` types)
   - **Service Layer**: Uses `TransactionManagerPort` interface for transactions
   - **Platform Layer**: Contains transaction manager implementation with database-specific code
   - **Repository Layer**: Receives transaction context via `context.Context` interface
   - Transaction session propagated through context, not explicit parameters

3. **Backward Compatibility:**
   - Existing users (already created without organization) will not be affected
   - No breaking changes to API contracts
   - GoogleAuth response format unchanged

### Business Constraints

1. **One Personal Organization per User:**
   - Each new user gets exactly one Personal Organization
   - No automatic creation on subsequent logins

2. **No Personal Organization Marker:**
   - Personal Organizations are indistinguishable from team organizations
   - No special "isPersonal" field or flag
   - User can delete their Personal Organization if desired (existing functionality)

## Additional Context

### Related Code Files

**Platform Layer (NEW - Transaction Manager):**
- `api/internal/platform/outbound/txmanager/txmanager_port.go` - Transaction manager interface (to be created)
- `api/internal/platform/outbound/txmanager/mongodb/txmanager.go` - MongoDB implementation (to be created)
- `api/cmd/internal/container/module_platform.go` - Platform module for fx (to be created or updated)

**Service Layer (Modifications):**
- `api/internal/service/auth/auth_service.go` - Main modification target (add transaction logic)

**Repository Layer (Check/Add):**
- `api/internal/service/user/outbound/repository/organization_repo_port.go` - Check for Create method
- `api/internal/service/user/outbound/repository/mongodb/organization_repo.go` - Add Create implementation if missing

**Domain Layer (Reference):**
- `shared/domain/organization.go` - Domain model
- `shared/domain/mongoschema/organization.go` - MongoDB schema

**Container/DI (Updates):**
- `api/cmd/internal/container/module_auth.go` - Update auth service provider with new dependencies

### MongoDB Collections

- `users` - User accounts with OAuth credentials
- `organizations` - Organization documents with embedded members array
- Indexes: `organizations.slug` (unique), verified in migration `20260104000000-create-initial-indexes.js`

### Architecture Pattern

This codebase follows **Hexagonal Architecture**:
- **Domain Layer:** `shared/domain/` - Pure business entities
- **Service Layer:** `api/internal/service/auth/` - Business logic (our modification target)
- **Outbound Ports:** `api/internal/service/auth/outbound/repository/` - Repository interfaces
- **Outbound Adapters:** `.../mongodb/` - MongoDB implementations

Service layer must depend on ports (interfaces), not concrete implementations.

## Questions Resolved

| Question | Answer |
|----------|--------|
| Should personal organization be named after the user? | Yes, use pattern: "{User's Name}'s Organization" (e.g., "Jayce Kim's Organization") |
| How should slug be generated? | Slugify user's name + 4-character random alphanumeric suffix (e.g., "jayce-kim-a3f9") |
| What role should user have in personal org? | Admin role (can manage organization and projects) |
| Should personal orgs be marked differently? | No, treat as regular organizations (no special field/flag) |
| What if organization creation fails? | Rollback user creation using MongoDB transaction (all-or-nothing) |
| Should we handle existing users without orgs? | No, out of scope (requires separate data migration task) |
| Does MongoDB need special setup for transactions? | Yes, must run as replica set (verify dev/prod configs) |
| Should slug generation retry on collision? | Yes, up to 3 attempts with different random suffixes |
