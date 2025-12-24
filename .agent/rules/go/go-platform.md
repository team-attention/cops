---
trigger: always_on
globs: **/internal/platform/**/*.go
paths: **/internal/platform/**/*.go
---

# Platform Package Guidelines

The platform package contains shared infrastructure, utilities, domain models, and setup code used across all services.

## Directory Structure

```
platform/
├── domain/       # Core domain models and business entities
├── outbound/     # External service implementations (storage, etc.)
├── pkg/          # Reusable packages safe for external import (e.g., queue)
├── setup/        # Application initialization and configuration
├── structure/    # Shared data structures (cursor, resource, etc.)
└── util/         # Utility packages and helpers
```

## Domain Package (`domain/`)

### Domain Models
- Define core business entities (User, Organization, Agent, etc.)
- Use value objects for complex types
- Keep domain models framework-agnostic
- Include validation logic within domain models when appropriate

### Naming Conventions
- Domain models: Singular nouns (e.g., `Agent`, `Organization`)
- Enums: Type + constants pattern

Example:
```go
type MemberRole string

const (
    MemberRoleOwner  MemberRole = "owner"
    MemberRoleAdmin  MemberRole = "admin"
    MemberRoleMember MemberRole = "member"
)
```

### ID Type
- Use `domain.ID` type for all entity identifiers
- Never use raw strings for IDs in domain models

### MongoDB Schemas
- Place MongoDB-specific schemas in `domain/mongoschema/`
- Use `{Model}Schema` naming pattern
- Include BSON tags for field mapping
- **Field Constants**: Define constants for all field names to avoid magic strings in queries (e.g., `UserEmailField = "email"`)

Example:
```go
const (
    UserCollection = "users"
)

const (
    UserIDField    = "_id"
    UserEmailField = "email"
    UserNameField  = "name"
)

type User struct {
    ID    bson.ObjectID `bson:"_id,omitempty"`
    Email string        `bson:"email"`
    Name  string        `bson:"name"`
}
```

## Outbound Package (`outbound/`)
- Contains common outbound adapters that can be shared across multiple services
- **IMPORTANT**: Add new implementations very conservatively - only when truly reusable
- Most outbound implementations should remain in individual service packages

## Pkg Package (`pkg/`)
- Contains standalone, reusable packages that are safe to be imported by other projects or services.
- Code here should be designed as a library with clear interfaces and minimal dependencies on the rest of the monolith if possible.
- Example: `queue` generic implementation.

## Setup Package (`setup/`)

**For detailed setup patterns, see [go-platform-setup.md](./go-platform-setup.md).**

### Configuration (`config/`)
- Define all application configuration structures
- Use environment-based configuration
- Provide sensible defaults
- Never hardcode secrets

### Initialization
- Each setup package should provide an `Init{Service}` function
- Return initialized clients/connections
- Handle connection failures gracefully
- Log initialization steps

## Structure Package (`structure/`)

### Shared Structures
- `CursorParams`: Pagination parameters
- `Resource`: Resource identification (OrgID + UserID)
- Common request/response structures

### Conventions
- Use generics where appropriate for type safety
- Include `SetDefault()` methods for parameter structs
- Validate structures at service layer, not here

## Util Package (`util/`)

### Error Utilities (`errutil/`)
- **Centralized error management**: All errors must be converted to or created as errutil errors
- **Error normalization**: Use httputil, k8sutil, mongoutil to convert specific errors to general errutil errors
- **Standard error types**: `BadRequestError`, `UnauthorizedError`, `ForbiddenError`, `NotFoundError`, `InternalError`

```go
// Always use errutil errors
return errutil.NotFoundError("agent not found")

// Convert specific errors using utility helpers
result, err := collection.FindOne(ctx, filter).Decode(&doc)
if err != nil {
    return nil, mongoutil.ErrWrapper(err) // Converts mongo.ErrNoDocuments to errutil.NotFound
}

// mongoutil.ErrWrapper handles:
// - mongo.ErrNoDocuments → errutil.NotFound
// - mongo.IsDuplicateKeyError → errutil.BadRequest  
// - bson.ErrInvalidHex → errutil.BadRequest
```

### Other Utilities
- **httputil**: HTTP middleware, response helpers, context extractors
- **mongoutil**: MongoDB helpers with error conversion to errutil  
- **jwtutil**: JWT token operations
- **factory**: External service client initialization
