---
trigger: always_on
globs: **/cmd/internal/container/*.go, cmd/internal/container/*.go
paths: **/cmd/internal/container/*.go, cmd/internal/container/*.go
---

# Container Package Guidelines

This document describes the best practices for dependency injection in the `cmd/internal/container` package using `go.uber.org/fx`.

## Dependency Injection Patterns

### Use `fx.As` for Interface Type Conversion

When you need to provide a concrete type as an interface, use `fx.As` instead of anonymous function wrappers.

**Correct:**
```go
fx.Provide(
    fx.Annotate(
        implementation.NewHandler,
        fx.As(new(InterfaceType)),
        fx.ResultTags(`group:"interface_group"`),
    ),
)
```

**Incorrect:**
```go
fx.Provide(
    fx.Annotate(
        func(handler *implementation.Handler) InterfaceType {
            return handler
        },
        fx.ResultTags(`group:"interface_group"`),
    ),
)
```

For concrete examples, see:
- [ConnectRPC handlers](./go-inbound-grpc-connectrpc.md#module-registration-pattern)
- [Fiber HTTP handlers](./go-inbound-http-fiber.md#module-registration-pattern)
- [Queue workers](./go-inbound-worker-queue.md#module-registration-pattern)

### Group Pattern

When collecting multiple implementations of the same interface, combine `fx.As` with `fx.ResultTags`:

**Provider side:**
```go
fx.Provide(
    fx.Annotate(
        service.NewImplementation,
        fx.As(new(InterfaceType)),
        fx.ResultTags(`group:"interface_group"`),
    ),
)
```

**Consumer side:**
```go
type Params struct {
    fx.In
    Handlers []InterfaceType `group:"interface_group"`
}
```

For registration file implementations, see:
- [ConnectRPC registration](./go-inbound-grpc-connectrpc.md#container-registration-file) - `group:"connect_handlers"`
- [Fiber HTTP registration](./go-inbound-http-fiber.md#container-registration-file) - `group:"public_routers"`, `group:"auth_routers"`
- [Queue worker registration](./go-inbound-worker-queue.md#container-registration-file) - `group:"workers"`

## Module Organization

Each feature module should:
1. Group related providers together
2. Use descriptive module names
3. Separate concerns (service, repository, inbound handlers)

**General pattern:**
```go
func new{Domain}Module() fx.Option {
    return fx.Module("{domain}",
        // Core services
        fx.Provide({domain}.NewService),

        // Repositories
        fx.Provide(
            fx.Annotate(
                {domain}repo.NewRepository,
                fx.As(new({domain}repo.RepositoryPort)),
            ),
        ),

        // Inbound handlers (gRPC, HTTP, Workers, etc.)
        // See go-inbound.md for patterns
    )
}
```

## File Organization

```
cmd/internal/container/
├── module_platform.go     # Platform module (logger, db, config)
├── module_{domain}.go     # Domain modules
├── application.go         # Application composition
└── register_{type}.go     # Runtime registration (fiber, connectrpc, queue)
```

## Registration Files

Registration files (`register_{type}.go`) handle runtime initialization of handlers.

### Pattern

```go
// register_{type}.go
package container

type {type}HandlerParams struct {
    fx.In

    Lifecycle fx.Lifecycle
    Logger    *slog.Logger
    Config    *config.Config

    Handlers []{Type}Handler `group:"{type}_handlers"`
}

func register{Type}Handlers(params {type}HandlerParams) {
    params.Logger.Info("Registering handlers", slog.Int("count", len(params.Handlers)))

    // Register all handlers
    for _, handler := range params.Handlers {
        // Protocol-specific registration
    }

    // Lifecycle management
    params.Lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // Start server
            return nil
        },
        OnStop: func(ctx context.Context) error {
            // Graceful shutdown
            return nil
        },
    })
}
```

### Registration Types

| Type      | Interface        | Group Tag          | Registration File        |
| --------- | ---------------- | ------------------ | ------------------------ |
| HTTP      | `HTTPRouter`     | `public_routers`   | `register_fiber.go`      |
| HTTP Auth | `HTTPAuthRouter` | `auth_routers`     | `register_fiber.go`      |
| gRPC      | `ConnectHandler` | `connect_handlers` | `register_connectrpc.go` |
| Worker    | `Worker`         | `workers`          | `register_queue.go`      |

**See also:** [go-inbound.md](./go-inbound.md) - Handler interface patterns