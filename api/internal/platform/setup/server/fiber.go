package server

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// InitFiber creates a configured Fiber application.
func InitFiber(cfg *config.Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           cfg.HTTP.ReadTimeout,
		WriteTimeout:          cfg.HTTP.WriteTimeout,
		DisableStartupMessage: cfg.App.Env == "production",
		ErrorHandler:          newErrorHandler(logger),
	})

	// Middleware
	app.Use(recover.New())

	// CORS middleware - must be before routes and other middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.HTTP.CORSAllowOrigins,
		AllowMethods: cfg.HTTP.CORSAllowMethods,
		AllowHeaders: cfg.HTTP.CORSAllowHeaders,
	}))

	app.Use(requestid.New())

	logger.Info("Fiber app initialized",
		slog.String("name", cfg.App.Name),
		slog.String("env", cfg.App.Env),
		slog.String("cors_origins", cfg.HTTP.CORSAllowOrigins),
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

		// For 413 errors on ConnectRPC endpoints, return proper Connect error format
		if code == fiber.StatusRequestEntityTooLarge {
			contentType := ctx.Get("Content-Type")
			if strings.Contains(contentType, "application/connect") ||
				strings.Contains(contentType, "application/grpc") ||
				strings.Contains(contentType, "application/proto") {
				ctx.Set("Content-Type", "application/json")
				return ctx.Status(code).JSON(fiber.Map{
					"code":    "resource_exhausted",
					"message": "request body too large",
				})
			}
		}

		return ctx.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
