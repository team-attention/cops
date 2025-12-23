package fiber

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/team-attention/cops/api/internal/service/health"
)

// HealthHTTPHandler handles health check HTTP endpoints.
type HealthHTTPHandler struct {
	svc    *health.Service
	logger *slog.Logger
}

// NewHealthHTTPHandler creates a new health HTTP handler.
func NewHealthHTTPHandler(l *slog.Logger, svc *health.Service) *HealthHTTPHandler {
	return &HealthHTTPHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "health.http.fiber")),
	}
}

// Register implements HTTPRouter interface.
func (h *HealthHTTPHandler) Register(app *fiber.App) {
	app.Get("/health", h.health)
	app.Get("/ready", h.ready)
}
