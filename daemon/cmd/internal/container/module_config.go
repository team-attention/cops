package container

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub/inmemory"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
	fsnotifyhandler "github.com/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig/filesystem"
)

func newConfigModule() fx.Option {
	return fx.Module("config",
		// Outbound: PubSub WriterPort
		fx.Provide(fx.Annotate(
			func(ps *inmemory.PubSub[[]domain.WatchTarget]) pubsub.WriterPort[[]domain.WatchTarget] {
				return ps
			},
			fx.As(new(pubsub.WriterPort[[]domain.WatchTarget])),
		)),

		// Outbound: LocalConfigPort
		fx.Provide(fx.Annotate(
			provideFilesystemLocalConfigAdapter,
			fx.As(new(localconfig.LocalConfigPort)),
		)),

		// Service (pure business logic)
		fx.Provide(configwatcher.NewService),

		// Inbound: PublisherHandler - publishes to pubsub on config change
		fx.Provide(fx.Annotate(
			fsnotifyhandler.NewConfigWatcherFsnotifyHandler,
			fx.As(new(PublisherHandler)),
			fx.ResultTags(`group:"publisher_handlers"`),
		)),
	)
}

// provideFilesystemLocalConfigAdapter wraps the local config adapter constructor.
func provideFilesystemLocalConfigAdapter(l *slog.Logger, paths *setup.ExpandedPaths) *filesystem.FilesystemLocalConfigAdapter {
	return filesystem.NewFilesystemLocalConfigAdapter(l, paths.LocalConfigDir)
}
