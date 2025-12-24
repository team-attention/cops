package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
	"github.com/team-attention/cops/daemon/internal/platform/setup/copsapi"
	"github.com/team-attention/cops/daemon/internal/platform/setup/logger"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(config.LoadConfig),

		// Logger (depends on config)
		fx.Provide(logger.InitLogger),

		// API Client (depends on config)
		fx.Provide(copsapi.InitAPIClient),
	)
}
