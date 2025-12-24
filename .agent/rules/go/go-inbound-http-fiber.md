---
trigger: glob
globs: **/internal/service/*/inbound/http/fiber/*.go
paths: **/internal/service/*/inbound/http/fiber/*.go
---

# Fiber HTTP Inbound Handler Guidelines

This document defines guidelines specific to Fiber implementation of HTTP handlers.

**For common patterns, see:**
- [go-inbound.md](./go-inbound.md) - Handler structure, module registration
- [go-inbound-http.md](./go-inbound-http.md) - HTTP common patterns
- [go-logging-conventions.md](./go-logging-conventions.md) - Logger binding

## Container Interfaces

```go
// HTTPRouter for public routes
type HTTPRouter interface {
    Register(router fiber.Router)
}

// HTTPAuthRouter for authenticated routes
type HTTPAuthRouter interface {
    RegisterWithAuth(router fiber.Router)
}
```

## Handler Implementation

```go
package fiber

import (
    "log/slog"

    "github.com/go-playground/validator/v10"
    "github.com/gofiber/fiber/v2"
    "{domain}"
)

type {Domain}HTTPHandler struct {
    svc      *{domain}.Service
    logger   *slog.Logger
    validate *validator.Validate
}

func New{Domain}HTTPHandler(l *slog.Logger, svc *{domain}.Service, v *validator.Validate) *{Domain}HTTPHandler {
    return &{Domain}HTTPHandler{
        svc:      svc,
        logger:   l.With(slog.String("name", "{domain}.http.fiber")),
        validate: v,
    }
}

// Register implements HTTPRouter (public routes)
func (h *{Domain}HTTPHandler) Register(router fiber.Router) {
    r := router.Group("/{domain}")
    r.Post("/", h.create{Domain})
    r.Get("/", h.list{Domain}s)
}

// RegisterWithAuth implements HTTPAuthRouter (authenticated routes)
func (h *{Domain}HTTPHandler) RegisterWithAuth(router fiber.Router) {
    r := router.Group("/{domain}")
    r.Get("/:id", h.get{Domain})
    r.Put("/:id", h.update{Domain})
    r.Delete("/:id", h.delete{Domain})
}
```

## Handler Method Pattern

```go
func (h *{Domain}HTTPHandler) create{Domain}(ctx *fiber.Ctx) error {
    // 1. Parse request
    var body CreateRequest
    if err := ctx.BodyParser(&body); err != nil {
        return httputil.ErrorResponse(ctx, err)
    }

    // 2. Validate
    if err := h.validate.Struct(body); err != nil {
        return httputil.ErrorResponse(ctx, err)
    }

    // 3. Extract user (if authenticated)
    userID := httputil.GetUserID(ctx)

    // 4. Call service
    result, err := h.svc.Create(ctx.UserContext(), userID, body)
    if err != nil {
        return httputil.ErrorResponse(ctx, err)
    }

    // 5. Return response
    return ctx.Status(fiber.StatusOK).JSON(result)
}
```

## Response Rules

| Scenario           | Response                                  |
| ------------------ | ----------------------------------------- |
| Success with data  | `ctx.Status(fiber.StatusOK).JSON(result)` |
| Success no content | `ctx.Status(fiber.StatusOK).Send(nil)`    |

**Never use**: `{"message": "ok"}` style responses

## Key Requirements

- **Package name**: `fiber`
- **Logger name**: `"{domain}.http.fiber"`
- **Validator**: Include `*validator.Validate` in constructor
