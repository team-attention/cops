package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub/inmemory"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	fsnotifyhandler "github.com/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/fsnotify"
	pollinghandler "github.com/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/polling"
	pubsubhandler "github.com/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/pubsub"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
	connectrpc "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
	fsnotifyadapter "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem/fsnotify"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/repository"
	sqlite "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/repository/sqlite"
)

func newLogModule() fx.Option {
	return fx.Module("log",
		// Outbound: FileWatchPort (wraps shared watcher from platform)
		fx.Provide(fx.Annotate(
			fsnotifyadapter.NewFileWatchAdapter,
			fx.As(new(filesystem.FileWatchPort)),
		)),

		// Outbound: PubSub ReaderPort
		fx.Provide(fx.Annotate(
			func(ps *inmemory.PubSub[[]domain.WatchTarget]) pubsub.ReaderPort[[]domain.WatchTarget] {
				return ps
			},
			fx.As(new(pubsub.ReaderPort[[]domain.WatchTarget])),
		)),

		// Outbound: StateRepositoryPort
		fx.Provide(fx.Annotate(
			sqlite.NewSQLiteStateRepository,
			fx.As(new(repository.StateRepositoryPort)),
		)),

		// Outbound: APIClientPort
		fx.Provide(fx.Annotate(
			connectrpc.NewAPIClient,
			fx.As(new(api.APIClientPort)),
		)),

		// Service
		fx.Provide(logwatcher.NewService),

		// Inbound 1: Fsnotify Handler (reads watcher.Events)
		fx.Provide(fx.Annotate(
			fsnotifyhandler.NewLogFsnotifyHandler,
			fx.As(new(FsnotifyHandler)),
			fx.ResultTags(`group:"fsnotify_handlers"`),
		)),

		// Inbound 2: SubscriberHandler - subscribes to pubsub for target updates
		fx.Provide(fx.Annotate(
			pubsubhandler.NewLogPubsubHandler,
			fx.As(new(SubscriberHandler)),
			fx.ResultTags(`group:"subscriber_handlers"`),
		)),

		// Inbound 3: OpenCode Polling Handler (LifecycleHandler, not FsnotifyHandler)
		fx.Provide(fx.Annotate(
			pollinghandler.NewLogPollingHandler,
			fx.ParamTags(``, ``, `name:"opencode_db"`, `name:"opencode_data_dir"`),
			fx.As(new(LifecycleHandler)),
			fx.ResultTags(`group:"lifecycle_handlers"`),
		)),
	)
}
