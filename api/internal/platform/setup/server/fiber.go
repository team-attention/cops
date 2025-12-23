package server

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// NewFiberApp creates a configured Fiber application.
func NewFiberApp(cfg *config.Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		DisableStartupMessage: cfg.App.Env == "production",
		ErrorHandler:          newErrorHandler(logger),
	})

	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())

	logger.Info("Fiber app initialized",
		slog.String("name", cfg.App.Name),
		slog.String("env", cfg.App.Env),
	)

	return app
}

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(ctx *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		logger.Error("Request error",
			slog.String("path", ctx.Path()),
			slog.String("method", ctx.Method()),
			slog.Int("status", code),
			slog.Any("error", err),
		)

		return ctx.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
