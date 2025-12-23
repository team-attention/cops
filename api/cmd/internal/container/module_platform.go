package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/setup/logger"
	"github.com/team-attention/cops/api/internal/platform/setup/mongodb"
	"github.com/team-attention/cops/api/internal/platform/setup/server"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(config.LoadConfig),

		// Logger (depends on config)
		fx.Provide(logger.InitLogger),

		// MongoDB (depends on config, logger)
		fx.Provide(mongodb.InitMongoDB),

		// Fiber app (depends on config, logger)
		fx.Provide(server.NewFiberApp),
	)
}
