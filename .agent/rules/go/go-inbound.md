---
trigger: glob
globs: **/internal/service/*/inbound/**/*.go
paths: **/internal/service/*/inbound/**/*.go
---

# Inbound Adapter Guidelines

Inbound adapters handle incoming requests from external sources (HTTP, gRPC, workers, etc.) and translate them into service calls.

## Directory Structure Rule

**CRITICAL: This structure is MANDATORY. NO exceptions. NO shortcuts.**

```
{domain}/inbound/
└── {category}/             # Protocol category
    └── {implementation}/   # Specific implementation
        └── *.go            # Handler files
```

### Structure Formula

```
ALWAYS: inbound/{category}/{implementation}/

Category:       Protocol type (grpc, http, worker, websocket)
Implementation: Specific technology (connectrpc, fiber, queue, gorilla)
```

## Naming Conventions

### Handler Struct Names

**Pattern:** `{Domain}{Protocol}Handler`

```go
// gRPC
type ProjectGRPCHandler struct { ... }

// HTTP
type UserHTTPHandler struct { ... }

// Worker
type SourceWorkerHandler struct { ... }
```

### Constructor Names

**Pattern:** `New{StructName}`

```go
func NewProjectGRPCHandler(...) *ProjectGRPCHandler { ... }
func NewUserHTTPHandler(...) *UserHTTPHandler { ... }
func NewSourceWorkerHandler(...) *SourceWorkerHandler { ... }
```

### File Names

**Multi-file Pattern (ALL protocols):**

```
handler.go # Constructor + Container interface implementations
{feature}.go # Handler methods
{optional...} # Optional functions (ex. common helper functions)
```

## Handler File Structure

**All protocols follow Multi-file Pattern**

### File: `handler.go`

**Purpose:** Struct definition, constructor, DI Container interface implementations

```go
package {implementation}  // e.g., connectrpc, fiber, queue

type {Domain}{Protocol}Handler struct {
    logger *slog.Logger
    svc    *{domain}.Service
    // ... other dependencies
}

func New{Domain}{Protocol}Handler(l *slog.Logger, svc *{domain}.Service /* dependencies */) *{Domain}{Protocol}Handler {
    return &{Domain}{Protocol}Handler{
        logger: l.With(slog.String("name", "{domain}.{protocol}.{implementation}")),
        svc:    svc,
        // Initialize
    }
}

// Or, If initialization can fail, return error:
func New{Domain}{Protocol}Handler(l *slog.Logger, svc *{domain}.Service /* dependencies */) (*{Domain}{Protocol}Handler, error) {
    // Perform initialization that might fail
    if err := doSomething(svc); err != nil {
        return nil, fmt.Errorf("failed to initialize handler: %w", err)
    }

    return &{Domain}{Protocol}Handler{
        logger: l.With(slog.String("name", "{domain}.{protocol}.{implementation}")),
        svc:    svc,
        // Initialize
    }, nil
}
```

### File: `{feature}.go`

**Purpose:** Handler method implementations

```go
package {implementation}

// Handler methods attached to {Domain}{Protocol}Handler

func (h *{Domain}{Protocol}Handler) {MethodName}(...) ... {
    // 1. Parse/validate input
    // 2. Call service layer
    // 3. Convert response
    // 4. Return result
}

// Helper functions (if needed, or use other files for common helper functions)
func helper(...) ... {
    // ...
}
```

### Key Principles

1. **`handler.go`**: Setup code (struct, constructor, Container interfaces)
2. **`{feature}.go`**: Handling logic (methods, request/response processing)
3. **Container Integration**: See [go-container.md](./go-container.md)

**See also:**
- [go-logging-conventions.md](./go-logging-conventions.md) - Logger binding patterns
- [go-port-adapter-pattern.md](./go-port-adapter-pattern.md) - Port/Adapter fundamentals

## Common Handler Interface Pattern

All inbound handlers implement a container interface for DI collection. Each protocol defines its own interface in `cmd/internal/container/register_{type}.go`.

### Interface Examples

```go
// gRPC (ConnectRPC)
type ConnectHandler interface {
    GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

// HTTP (Fiber)
type HTTPRouter interface {
    Register(router fiber.Router)
}

// Worker (Queue)
type Worker interface {
    Start(ctx context.Context)
}
```

## Module Registration Pattern

Handlers are registered to DI container using `fx.Annotate` pattern:

```go
// cmd/internal/container/module_{domain}.go
func new{Domain}Module() fx.Option {
    return fx.Module("{domain}",
        fx.Provide({domain}.NewService),

        // Register handler with group tag
        fx.Provide(
            fx.Annotate(
                {impl}.New{Domain}{Protocol}Handler,
                fx.As(new({Protocol}Handler)),
                fx.ResultTags(`group:"{protocol}_handlers"`),
            ),
        ),
    )
}
```

### Container Collection

Handlers are collected using `fx.In` with group tags:

```go
type {protocol}HandlerParams struct {
    fx.In
    Handlers []{Protocol}Handler `group:"{protocol}_handlers"`
}

func register{Protocol}Handlers(params {protocol}HandlerParams) {
    for _, handler := range params.Handlers {
        // Register each handler
    }
}
```

## Common Mistakes to Avoid

1. **Skipping category directory**
   - ❌ `inbound/connectrpc/`
   - ✅ `inbound/grpc/connectrpc/`

2. **Skipping implementation directory**
   - ❌ `inbound/grpc/handler.go`
   - ✅ `inbound/grpc/connectrpc/handler.go`

3. **Adding extra nesting**
   - ❌ `inbound/grpc/connectrpc/handlers/handler.go`
   - ✅ `inbound/grpc/connectrpc/handler.go`

4. **Using wrong package name**
   - ❌ `package grpc` (in grpc/connectrpc/)
   - ✅ `package connectrpc`

## Protocol/Implementation-Specific Guidelines

For detailed implementation guidelines, see:
- **gRPC (ConnectRPC)**: [go-inbound-grpc-connectrpc.md](./go-inbound-grpc-connectrpc.md)
- **HTTP (Fiber)**: [go-inbound-http-fiber.md](./go-inbound-http-fiber.md)
- **Worker (Queue)**: [go-inbound-worker-queue.md](./go-inbound-worker-queue.md)
