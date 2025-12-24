---
trigger: glob
globs: **/internal/service/*/inbound/http/**/*.go
paths: **/internal/service/*/inbound/http/**/*.go
---

# HTTP Inbound Handler Guidelines

This document defines HTTP-specific patterns for inbound handlers.

**For common patterns, see [go-inbound.md](./go-inbound.md).**

## Container Interfaces

HTTP handlers implement one or both interfaces for DI collection:

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

## HTTP Handler Pattern

```go
type {Domain}HTTPHandler struct {
    svc      *{domain}.Service
    logger   *slog.Logger
    validate *validator.Validate
}

func (h *{Domain}HTTPHandler) Register(router fiber.Router) {
    r := router.Group("/{domain}")
    r.Post("/", h.create{Domain})
    r.Get("/", h.list{Domain}s)
}

func (h *{Domain}HTTPHandler) RegisterWithAuth(router fiber.Router) {
    r := router.Group("/{domain}")
    r.Get("/:id", h.get{Domain})
    r.Put("/:id", h.update{Domain})
    r.Delete("/:id", h.delete{Domain})
}
```

## Request Handling Pattern

```go
func (h *{Domain}HTTPHandler) create{Domain}(ctx *fiber.Ctx) error {
    // 1. Parse request body
    var body CreateRequest
    if err := ctx.BodyParser(&body); err != nil {
        return httputil.ErrorResponse(ctx, err)
    }

    // 2. Validate
    if err := h.validate.Struct(body); err != nil {
        return httputil.ErrorResponse(ctx, err)
    }

    // 3. Extract user context (if authenticated route)
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

| Scenario           | Response                                       |
| ------------------ | ---------------------------------------------- |
| Success with data  | `ctx.Status(fiber.StatusOK).JSON(result)`      |
| Success no content | `ctx.Status(fiber.StatusOK).Send(nil)`         |
| Created            | `ctx.Status(fiber.StatusCreated).JSON(result)` |
| Error              | `httputil.ErrorResponse(ctx, err)`             |

**Never use**: `{"message": "ok"}` style responses

## Validation

Use `go-playground/validator` for request validation:

```go
type CreateUserRequest struct {
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"gte=0,lte=150"`
}
```

## See Also

- [go-inbound.md](./go-inbound.md) - General inbound structure, module registration
- [go-inbound-http-fiber.md](./go-inbound-http-fiber.md) - Fiber-specific implementation
- [go-logging-conventions.md](./go-logging-conventions.md) - Logger binding patterns
