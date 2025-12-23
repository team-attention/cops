package fiber

import (
	"github.com/gofiber/fiber/v2"
)

// health handles GET /health - liveness probe.
func (h *HealthHTTPHandler) health(ctx *fiber.Ctx) error {
	status := h.svc.Health(ctx.UserContext())
	return ctx.Status(fiber.StatusOK).JSON(status)
}

// ready handles GET /ready - readiness probe.
func (h *HealthHTTPHandler) ready(ctx *fiber.Ctx) error {
	status := h.svc.Ready(ctx.UserContext())

	httpStatus := fiber.StatusOK
	if status.Status != "ok" {
		httpStatus = fiber.StatusServiceUnavailable
	}

	return ctx.Status(httpStatus).JSON(status)
}
