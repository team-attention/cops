---
trigger: always_on
globs: **/cmd/internal/container/*.go, cmd/internal/container/*.go
paths: **/cmd/internal/container/*.go, cmd/internal/container/*.go
---

# Dig Container Package Guidelines

This document describes the best practices for dependency injection in the `cmd/internal/container` package using `go.uber.org/dig`.

Unlike `fx`, `dig` is a pure dependency injection container without lifecycle management. This makes it ideal for CLI applications where commands execute and exit immediately.

## Dependency Injection Patterns

### Basic Provider

Providers are functions that return values to be injected into the container.

**Correct:**
```go
func NewConfig() *Config {
    return &Config{
        AppName: "cops",
    }
}
```

**Incorrect:**
```go
// Don't create values inline in Provide
c.Provide(func() *Config {
    return &Config{} // Hard to test and reuse
})
```

### Provider with Dependencies

Providers can depend on other providers. dig resolves the dependency graph automatically.

```go
func NewService(cfg *Config, logger *slog.Logger) *Service {
    return &Service{
        config: cfg,
        logger: logger,
    }
}
```

### Provider with Error

Providers can return errors. dig will propagate errors during Invoke.

```go
func NewDatabase(cfg *Config) (*sql.DB, error) {
    return sql.Open("postgres", cfg.DatabaseURL)
}
```

## Container Entry Point

The container should expose a single `Run()` function that:
1. Creates the dig container
2. Registers all providers
3. Invokes the root command through the container

**Pattern:**
```go
func Run() error {
    c := dig.New()

    providers := []interface{}{
        setup.NewRootCommand,
        NewConfig,
        NewLogger,
    }

    for _, p := range providers {
        if err := c.Provide(p); err != nil {
            return err
        }
    }

    return c.Invoke(func(cmd *cobra.Command) error {
        return cmd.Execute()
    })
}
```

## Subcommand Registration

Subcommands are defined in inbound adapters and registered through the container.

### Inbound Adapter Pattern

**Subcommand definition (`internal/service/core/inbound/cli/init.go`):**
```go
package cli

func NewInitCommand(svc *service.CoreService) *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize code rules",
        RunE: func(cmd *cobra.Command, args []string) error {
            return svc.Initialize()
        },
    }
}
```

### Container Registration

```go
// Register subcommand providers
providers := []interface{}{
    setup.NewRootCommand,
    cli.NewInitCommand,
}

// Use Invoke to attach subcommands to root
c.Invoke(func(root *cobra.Command, initCmd *cobra.Command) {
    root.AddCommand(initCmd)
})
```

## File Organization

```
cmd/internal/
└── container/
    └── container.go      # Run() function and provider registration

internal/platform/setup/
└── cobra/
    └── root.go           # Root command definition

internal/service/{domain}/inbound/cli/
└── *.go                  # Subcommand definitions (inbound adapters)
```

**See also:** [go-inbound.md](./go-inbound.md) - Inbound adapter patterns