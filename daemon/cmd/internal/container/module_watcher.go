package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
	configfilesystem "github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/filesystem"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	logfilesystem "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
	"github.com/team-attention/cops/daemon/internal/service/project"
)

func newWatcherModule() fx.Option {
	return fx.Module("watcher",
		// ConfigWatcher filesystem adapter
		fx.Provide(
			fx.Annotate(
				configfilesystem.NewAdapter,
				fx.As(new(configwatcher.FileWatchPort)),
			),
		),
		// ConfigWatcher service
		fx.Provide(configwatcher.NewService),

		// Project service
		fx.Provide(project.NewService),

		// LogWatcher filesystem adapter
		fx.Provide(
			fx.Annotate(
				logfilesystem.NewAdapter,
				fx.As(new(logwatcher.FileWatchPort)),
			),
		),
		// LogWatcher service
		fx.Provide(logwatcher.NewService),
	)
}
