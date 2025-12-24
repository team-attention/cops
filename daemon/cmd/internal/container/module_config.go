package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub/inmemory"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
	fsnotifyhandler "github.com/team-attention/cops/daemon/internal/service/configwatcher/inbound/worker/fsnotify"
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

		// Service (pure business logic)
		fx.Provide(configwatcher.NewService),

		// Inbound: FsnotifyHandler with fx.Group
		fx.Provide(fx.Annotate(
			fsnotifyhandler.NewConfigWatcherFsnotifyHandler,
			fx.As(new(FsnotifyHandler)),
			fx.ResultTags(`group:"fsnotify_handlers"`),
		)),
	)
}
