package container

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/interceptor"
	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate/filesystem"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(setup.LoadConfig),

		// Logger (depends on config)
		fx.Provide(setup.InitLogger),

		// ExpandedPaths (depends on config)
		fx.Provide(provideExpandedPaths),

		// SQLite DB (depends on config and logger)
		fx.Provide(setup.InitSQLite),

		// API Client (depends on config)
		fx.Provide(setup.InitAPIClient),

		// FilesystemAuthState adapter
		fx.Provide(fx.Annotate(
			provideFilesystemAuthState,
			fx.As(new(authstate.AuthStatePort)),
		)),

		// Auth Interceptor (depends on logger and authstate)
		fx.Provide(interceptor.NewAuthInterceptor),

		// Target PubSub (depends on logger)
		fx.Provide(setup.InitTargetPubSub),

		// Log Watcher (shared fsnotify.Watcher)
		fx.Provide(setup.InitLogWatcher),

		// OpenCode DB (read-only, nil if not installed)
		fx.Provide(fx.Annotate(
			setup.InitOpenCodeDB,
			fx.ResultTags(`name:"opencode_db"`),
		)),

		// OpenCode data dir (from pathutil, for DI into polling handler)
		fx.Provide(fx.Annotate(
			pathutil.GetOpenCodeDataDir,
			fx.ResultTags(`name:"opencode_data_dir"`),
		)),

		// Invoke to wire interceptor to API client (side effect)
		fx.Invoke(func(apiClient *setup.APIClient, authInterceptor *interceptor.AuthInterceptor) {
			apiClient.SetInterceptor(authInterceptor)
		}),
	)
}

// provideExpandedPaths creates ExpandedPaths from config.
func provideExpandedPaths(cfg *setup.Config) *setup.ExpandedPaths {
	return setup.NewExpandedPaths(&cfg.Cops)
}

// provideFilesystemAuthState creates the auth state adapter with the correct auth path.
func provideFilesystemAuthState(l *slog.Logger, paths *setup.ExpandedPaths) *filesystem.FilesystemAuthState {
	return filesystem.NewFilesystemAuthState(l, paths.BaseDir)
}
