---
trigger: always_on
globs: **/cmd/internal/container/*.go, cmd/internal/container/*.go
paths: **/cmd/internal/container/*.go, cmd/internal/container/*.go
---

# Dig Container Package Guidelines

This document describes the best practices for dependency injection in the `cmd/internal/container` package using `go.uber.org/dig`.

Unlike lifecycle-managed DI containers, `dig` is a pure dependency injection container. This makes it ideal for CLI applications where commands execute and exit immediately.

## File Organization

```
cmd/internal/container/
├── container.go           # Run() entry point
├── module_platform.go     # Platform providers (config, logger, HTTP clients)
├── module_{domain}.go     # Domain providers + group registration
└── register_{type}.go     # Interface definition + group collection + trigger
```

## Dependency Injection Patterns

### Use `dig.As` for Interface Type Conversion

When you need to provide a concrete type as an interface, use `dig.As`:

```go
c.Provide(
    adapter.NewFilesystemConfigAdapter,
    dig.As(new(config.ConfigPort)),
)
```

### Group Pattern

When collecting multiple implementations of the same interface, combine `dig.As` with `dig.Group`:

**Provider side:**
```go
c.Provide(
    handler.NewTrackingCLIHandler,
    dig.As(new(CLICommandProvider)),
    dig.Group("cli_handlers"),
)
```

**Consumer side:**
```go
type cobraParams struct {
    dig.In
    Handlers []CLICommandProvider `group:"cli_handlers"`
}
```

## Module Organization

Each feature module should:
1. Group related providers together
2. Use descriptive module names
3. Separate concerns (service, repository, inbound handlers)

**General pattern:**
```go
func new{Domain}Module(c *dig.Container) error {
    // Outbound adapters with dig.As for interface casting
    if err := c.Provide(
        adapter.NewRepository,
        dig.As(new(repository.RepositoryPort)),
    ); err != nil {
        return err
    }

    // Services
    if err := c.Provide({domain}.NewService); err != nil {
        return err
    }

    // CLI handlers with dig.As + dig.Group
    return c.Provide(
        cli.New{Domain}CLIHandler,
        dig.As(new(CLICommandProvider)),
        dig.Group("cli_handlers"),
    )
}
```

## Registration Files

Registration files (`register_{type}.go`) handle runtime initialization of handlers.

### Pattern

```go
// register_cobra.go
package container

// CLICommandProvider provides CLI commands to be registered.
type CLICommandProvider interface {
    Commands() []*cobra.Command
}

// cobraParams collects all CLI handlers via group.
type cobraParams struct {
    dig.In

    Root     *cobra.Command
    Handlers []CLICommandProvider `group:"cli_handlers"`
}

// RegisterCobraCommands collects handlers via group, registers commands, and executes.
func RegisterCobraCommands(c *dig.Container) error {
    return c.Invoke(func(params cobraParams) error {
        // Collect and register all commands from handlers
        for _, handler := range params.Handlers {
            for _, cmd := range handler.Commands() {
                params.Root.AddCommand(cmd)
            }
        }

        // Execute (trigger)
        return params.Root.Execute()
    })
}
```

### Registration Types

| Type | Interface | Group Tag | Registration File |
|------|-----------|-----------|-------------------|
| CLI  | `CLICommandProvider` | `cli_handlers` | `register_cobra.go` |

## Container Entry Point

```go
func Run() error {
    c := dig.New()

    // Register modules (providers + group registrations)
    if err := newPlatformModule(c); err != nil { return err }
    if err := newTrackingModule(c); err != nil { return err }

    // Register and execute commands (trigger)
    return RegisterCobraCommands(c)
}
```

## CLI Handler Implementation

CLI handlers implement `CLICommandProvider` interface in inbound adapters:

```go
// internal/service/{domain}/inbound/cli/cobra/handler.go
package cobra

type {Domain}CLIHandler struct {
    logger  *slog.Logger
    svc     *{domain}.Service
}

func New{Domain}CLIHandler(l *slog.Logger, svc *{domain}.Service) *{Domain}CLIHandler {
    return &{Domain}CLIHandler{
        logger: l.With(slog.String("name", "{domain}.cli.cobra")),
        svc:    svc,
    }
}

// Commands implements CLICommandProvider interface.
func (h *{Domain}CLIHandler) Commands() []*cobra.Command {
    return []*cobra.Command{
        h.NewFooCommand(),
        h.NewBarCommand(),
    }
}
```

**See also:** [go-inbound.md](./go-inbound.md) - Inbound adapter patterns
