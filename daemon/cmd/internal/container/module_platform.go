package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/setup"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(setup.LoadConfig),

		// Logger (depends on config)
		fx.Provide(setup.InitLogger),

		// SQLite DB (depends on config and logger)
		fx.Provide(setup.InitSQLite),

		// API Client (depends on config)
		fx.Provide(setup.InitAPIClient),

		// Target PubSub (depends on logger)
		fx.Provide(setup.InitTargetPubSub),

		// Log Watcher (shared fsnotify.Watcher)
		fx.Provide(setup.InitLogWatcher),
	)
}
