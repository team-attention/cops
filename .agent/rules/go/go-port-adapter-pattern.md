---
trigger: glob
globs: **/internal/service/**/inbound/**/*.go, **/internal/service/**/outbound/**/*.go, **/internal/platform/outbound/**/*.go
paths: **/internal/service/**/inbound/**/*.go, **/internal/service/**/outbound/**/*.go, **/internal/platform/outbound/**/*.go
---

# Port/Adapter Pattern (Hexagonal Architecture)

This document explains the Port/Adapter pattern used throughout the codebase for both inbound and outbound dependencies.

**For detailed layer-specific guidelines, see:**
- [go-inbound.md](./go-inbound.md) - Inbound adapter implementation details
- [go-outbound.md](./go-outbound.md) - Outbound adapter implementation details

## Core Concept

The Port/Adapter pattern (also known as Hexagonal Architecture) separates:
- **Port (Interface)**: Interface defining what operations are available
- **Adapter (Implementation)**: Concrete implementation of those operations

### Pattern Formula

```
Port (Interface) → Adapter (Implementation)
```

**Examples:**
```
UserRepositoryPort (Interface) → MongoUserRepository (Adapter)
ConnectHandler (Interface) → AddressGRPCHandler (Adapter)
HTTPRouter (Interface) → UserHTTPHandler (Adapter)
```

## Interface Definition Pattern in Go

Go's interfaces are implicitly implemented (no explicit inheritance needed):

```go
// Port definition
type UserRepositoryPort interface {
    Create(ctx context.Context, user *domain.User) (*domain.User, error)
    Get(ctx context.Context, userID string) (*domain.User, error)
}

// Adapter implementation - no explicit "implements" keyword
type MongoUserRepository struct {
    db     *mongo.Database
    logger *slog.Logger
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    // Implementation
}

func (r *MongoUserRepository) Get(ctx context.Context, userID string) (*domain.User, error) {
    // Implementation
}
```

### Interface Verification (Optional but Recommended)

Use compile-time verification to ensure interface compliance:

```go
var _ UserRepositoryPort = (*MongoUserRepository)(nil)
```

## DTO Model Patterns

**IMPORTANT - Be Conservative with DTOs:**
- **Default to domain models**: Return domain models directly when safe
- **Check for existing models**: Before creating DTOs, verify if shared/common models exist
- **Minimize code**: Only create DTOs when absolutely necessary

### When to Use DTOs

Create dedicated DTOs **only when**:
1. Domain model contains security-sensitive fields (passwords, internal IDs)
2. Response/output requires computed/derived fields not in domain
3. Data format significantly differs from domain model
4. Need to combine multiple domain models
5. Input validation requirements differ from domain model constraints

### When to Return Domain Models Directly (PREFERRED)

Return domain models directly when:
1. Domain model is safe to expose (no sensitive fields)
2. No transformation or computation needed
3. Data structure matches domain model

## File Organization Principles

### General Structure

```
{domain}/{layer}/{category}/
├── {name}_port.go      # Port (Interface) - optional, can be in service
└── {implementation}/   # Implementation directory
    └── {name}.go       # Adapter (Implementation)
```

### Inbound Example

```
user/inbound/http/
├── fiber/
│   ├── handler.go      # Handler struct + constructor
│   └── user.go         # Handler methods
```

### Outbound Example

```
user/outbound/repository/
├── user_repo_port.go   # UserRepositoryPort interface (optional)
└── mongodb/
    └── user_repo.go    # MongoUserRepository implementation
```

## Container Integration

**Key principle**: Container injects concrete adapters, but services receive Port types.

```go
type UserService struct {
    logger *slog.Logger
    repo   UserRepositoryPort  // Port type, not concrete
}

func NewUserService(l *slog.Logger, repo UserRepositoryPort) *UserService {
    return &UserService{
        logger: l.With(slog.String("name", "user.service")),
        repo:   repo,  // Works with any adapter implementing the Port
    }
}
```

This allows:
- **Flexibility**: Container can inject `MongoUserRepository`, `PostgresUserRepository`, or any adapter
- **Type Safety**: Service depends on Port interface, not concrete implementation
- **Testability**: Easy to inject mock adapters for testing

For detailed container configuration patterns, see [go-container.md](./go-container.md).

## Best Practices

1. **Interfaces in Go are implicit**: No explicit inheritance needed
2. **Interface verification**: Use `var _ Port = (*Adapter)(nil)` for compile-time checks
3. **Consistent naming**: Port (`{Domain}{Category}Port`) and implementation (`{Tech}{Domain}{Category}`)
4. **One adapter per file**: Each implementation in its own file
5. **Full type annotations**: All parameters and return types must be typed
6. **Context first**: Use `ctx context.Context` as first parameter for I/O operations
7. **Descriptive names**: `UserRepositoryPort`, `MongoUserRepository` (not `UserRepo`, `Mongo`)

## Summary

- **Port (Interface)**: Interface defining operations
- **Adapter (Implementation)**: Concrete technology-specific code
- **Implicit implementation**: Go interfaces don't need explicit inheritance
- **Interface verification**: Use compile-time checks for safety
- **Full type annotations**: All parameters and return types typed
- **Flexibility**: Easy to swap implementations (MongoDB → PostgreSQL)
- **Testability**: Easy to mock Port interface for testing

This pattern is used consistently across:
- **Inbound adapters**: HTTP, gRPC, Workers (see [go-inbound.md](./go-inbound.md))
- **Outbound adapters**: Repositories, external services (see [go-outbound.md](./go-outbound.md))
