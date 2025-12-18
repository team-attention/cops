---
trigger: always_on
globs: **/internal/**/*.go
paths: **/internal/**/*.go
---

# Internal Package Guidelines

This document defines the coding standards and architectural patterns for the go codebase.

## Architecture Overview

This package follows **Hexagonal Architecture** (Ports and Adapters) with clear separation of concerns:

```
internal/
├── gen/          # Generated code (DO NOT EDIT MANUALLY)
│   └── grpcstub/ # Generated gRPC/Connect stubs from protobuf
├── platform/     # Shared infrastructure, utilities, and domain models
└── service/      # Business logic organized by domain
    ├── core/     # Cross-cutting services (RBAC, Audit, etc.)
    └── {domain}/ # Domain-specific services
```

## Generated Code Rules

**IMPORTANT: Files under `internal/gen/` are programmatically generated and must NOT be edited manually.**

- **`gen/grpcstub/`**: Generated from Protocol Buffers using `buf generate`
  - Contains Go types and ConnectRPC service handlers
  - Regenerated whenever proto files change
  - Import from `github.com/rcvbridge/recoverybridge/server/api/internal/gen/grpcstub/{service}/v1`

**To modify generated code:**
- Update `.proto` files in `idl/protobuf/` and run `buf generate`
- Service implementations should import and use these generated types


## General Rules

### Package Structure
- Each service follows the hexagonal pattern with `inbound`, `outbound`, and core business logic
- Dependencies flow inward: `inbound → service → outbound`
- Never import from `inbound` packages into service or outbound layers

### Service Isolation

**CRITICAL RULE: Service Independence**

Each service (except Core) MUST be completely independent. Services cannot directly import or depend on other services.

**FORBIDDEN - Cross-Service Imports:**
- ❌ WRONG: `import "github.com/.../internal/service/user"` in `address` service
- ❌ WRONG: Importing another service's inbound/outbound code

**ALLOWED - Cross-Service Dependencies:**

1. **Core services**: Services under `internal/service/core/` can be injected into other services
2. **Platform components**: All platform modules can be used across services:
   - `internal/platform/domain/` - Shared domain models
   - `internal/platform/util/` - Shared utilities
   - `internal/platform/outbound/` - Shared outbound adapters

**Examples:**

```go
// ❌ WRONG - address service importing user service directly
import "github.com/.../internal/service/user" // FORBIDDEN

// ✅ CORRECT - using Core service (if exists)
import "github.com/.../internal/service/core/auth" // OK - Core services can be injected

// ✅ CORRECT - using platform components
import "github.com/.../internal/platform/domain" // OK - shared domain models
import "github.com/.../internal/platform/util/errutil" // OK - shared utilities
import "github.com/.../internal/platform/outbound/email" // OK - shared outbound adapters
```

### Dependency Injection
- All dependencies MUST be injected through constructors
- Use `New{ComponentName}` pattern for constructors
- Return concrete types from constructors, not interfaces

### Error Handling
- Use `platform/util/errutil` package for all error creation
- Return appropriate HTTP status codes using error types
- Never panic in application code

### Logging
- Pass logger as first parameter in constructors
- Log errors at the service layer, not in repositories

### Context Usage
- Always pass `context.Context` as the first parameter when it is required
- Never store business data in context except user identity

### Type Usage
- Use pointer types for structs in response models and service return values unless there is a specific reason to use value types.