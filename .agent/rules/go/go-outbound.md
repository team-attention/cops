---
trigger: always_on
globs: **/internal/**/outbound/**/*.go
paths: **/internal/**/outbound/**/*.go
---

# Outbound Adapters

Follow **Port (Interface) → Adapter (Implementation)** pattern.

**For core Port/Adapter pattern fundamentals, see [go-port-adapter-pattern.md](./go-port-adapter-pattern.md).**

## Directory Structure Rule

```
outbound/                   # External dependencies
├── externalsvc/            # External service interfaces (impl in internal/service/core/{extsvc}/)
│   └── auth/               # Authentication service interfaces
├── repository/             # Data persistence interfaces
│   ├── user_repo_port.go   # User repository interface
│   ├── post_repo_port.go   # Post repository interface
│   └── mongodb/
│       ├── user_repo.go    # User repository implementation
│       └── post_repo.go    # Post repository implementation
└── {category}/             # Other outbound dependencies
    ├── {interface}.go      # Interface
    └── {implmentaion}/     # Implementation package
        └── {impl_file}.go  # Implementation
```

> Omit directories if not needed (e.g., no repository if no data persistence)

**Categories:**
- `repository/` - Data persistence (MongoDB, etc.)
- `externalsvc/` - External service dependencies (interfaces only)
- ... Other categories as needed

## Naming Conventions

### Interface Names
- `{Domain}{Category}Port` (ex. `OrganizationRepositoryPort`, `SecretResourcePort`)

### Struct Names
- `{Technology or Service}{Domain}{Category}` (ex. `MongoAgentRepository`, `RBACAgentService`)

### Constructor Names
- `New{StructName}(...) {Category Interface} { ... }` (ex. `func NewMongoAgentRepository(...) repository.OrganizationRepositoryPort { ... }`, `func NewK8SSecretResource(...) resource.SecretResourcePort { ... }`)

## Constructor Dependency Injection

### Outbound adapters should depend on platform-initialized infrastructure

When an adapter requires external resources (database, HTTP client, etc.):
- **DO**: Accept pre-initialized infrastructure from `platform/setup` (e.g., `*sql.DB`, `*mongo.Database`)
- **DON'T**: Initialize infrastructure directly in the adapter
- **DON'T**: Accept configuration and initialize infrastructure yourself

Example:
```go
// CORRECT: Accept pre-initialized DB from platform/setup
func NewSQLiteStateRepository(l *slog.Logger, db *sql.DB) (repository.StateRepositoryPort, error) {
    // Create table if not exists
    if err := createTable(db); err != nil {
        return nil, err
    }
    // ...
}

// WRONG: Don't initialize infrastructure in adapter
func NewSQLiteStateRepository(l *slog.Logger, cfg *config.Config) (repository.StateRepositoryPort, error) {
    db, err := sql.Open("sqlite3", cfg.DaemonDataDir+"/state.db")
    // ...
}

// WRONG: Don't inject individual primitive values
func NewSQLiteStateRepository(l *slog.Logger, dataDir string) (repository.StateRepositoryPort, error) {
    db, err := sql.Open("sqlite3", dataDir+"/state.db")
    // ...
}
```

This ensures:
1. Infrastructure initialization is centralized in `platform/setup`
2. Adapters focus on business logic, not infrastructure setup
3. Database connections are shared and managed via DI container lifecycle
4. Consistent pattern across all outbound adapters

# Common Patterns

## List Operations
- Always use cursor-based pagination (`structure.CursorParams`)
- Sort by creation/update time by default
- Apply filters at repository level

## RBAC Integration
- Check permissions before any operation
- Use appropriate permission level (Read, Write, Execute)
- Pass `structure.Resource` containing OrgID and UserID
- Return `errutil.ForbiddenError` for permission denials

## Error Handling
- Use `errutil` package for consistent error types
- Return errors from repository/resource layers as-is
- Log errors at service layer with appropriate level
- Never expose internal details in error messages

## Core Service Usage
- Use `externalsvc` outbound adapter when using other Core Services