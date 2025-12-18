---
trigger: always_on
globs: **/internal/**/*.go
paths: **/internal/**/*.go
---

# Logging Conventions

This document defines logging standards for all Go code under `internal/`.

## Logger Injection

**ALWAYS inject logger as the first parameter in constructors:**

```go
import "log/slog"

type UserService struct {
    logger *slog.Logger
    repo   UserRepositoryPort
}

func NewUserService(l *slog.Logger, repo UserRepositoryPort) *UserService {
    return &UserService{
        logger: l.With(slog.String("name", "user.service")),
        repo:   repo,
    }
}
```

**Key points:**
- Logger is ALWAYS the first parameter
- Type: `*slog.Logger`
- Bind context immediately in constructor with `l.With()`

## Logger Binding

### Bind in Constructor

ALWAYS bind logger context in constructor:

```go
func NewUserService(l *slog.Logger, ...) *UserService {
    return &UserService{
        logger: l.With(slog.String("name", "...")),  // Bind immediately
        // ...
    }
}
```

### Logger Naming by Layer

| Layer | Pattern | Example |
|-------|---------|---------|
| Service | `{domain}.service` | `user.service` |
| Inbound | `{domain}.{protocol}.{implementation}` | `user.http.fiber` |
| Outbound | `{domain}.{category}.{implementation}` | `user.repository.mongodb` |

## Logging Best Practices

### Include Context in Errors

```go
result, err := s.repo.Get(ctx, userID)
if err != nil {
    s.logger.Error("failed to get user",
        slog.String("userID", userID),
        slog.Any("error", err),
    )
    return nil, err
}
```

### Log Important State Changes

```go
func (s *OrderService) UpdateStatus(ctx context.Context, orderID string, status OrderStatus) error {
    s.logger.Info("updating order status",
        slog.String("orderID", orderID),
        slog.String("oldStatus", string(oldStatus)),
        slog.String("newStatus", string(status)),
    )
    // ...
}
```

## Summary

- **Inject logger**: First parameter in all constructors (`l *slog.Logger`)
- **Bind immediately**: `l.With(slog.String("name", "..."))` in constructor
- **Naming patterns**: See layer-specific documentation
- **Structured logging**: Use `slog.String`, `slog.Int`, `slog.Any` - not string formatting
- **Appropriate levels**: Debug, Info, Warn, Error
- **No sensitive data**: Never log passwords, tokens, or secrets
