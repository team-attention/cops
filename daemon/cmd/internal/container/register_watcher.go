package container

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher"
	"github.com/team-attention/cops/daemon/internal/service/logprocessor"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	"github.com/team-attention/cops/daemon/internal/service/project"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

type watcherParams struct {
	fx.In

	Lifecycle     fx.Lifecycle
	Logger        *slog.Logger
	Config        *config.Config
	ConfigWatcher *configwatcher.Service
	Project       *project.Service
	LogWatcher    *logwatcher.Service
	LogProcessor  *logprocessor.Service
}

func registerWatcher(p watcherParams) {
	// Wire up event handlers
	p.ConfigWatcher.OnConfigChange(func(cfg domain.GlobalConfig) {
		p.Logger.Info("config changed, updating watch targets")
		if err := p.Project.UpdateProjects(cfg); err != nil {
			p.Logger.Error("failed to update projects", slog.Any("error", err))
			return
		}
		targets := p.Project.GetWatchTargets()
		if err := p.LogWatcher.UpdateTargets(targets); err != nil {
			p.Logger.Error("failed to update log watch targets", slog.Any("error", err))
		}
	})

	p.LogWatcher.OnLogEntry(func(record shareddomain.SessionRecord) {
		p.LogProcessor.AddEntry(record)
	})

	// Register lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Load initial config
			cfg, err := p.ConfigWatcher.LoadConfig()
			if err != nil {
				return fmt.Errorf("failed to load initial config: %w", err)
			}

			// Initialize projects
			if err := p.Project.UpdateProjects(*cfg); err != nil {
				return fmt.Errorf("failed to initialize projects: %w", err)
			}

			// Set initial watch targets
			targets := p.Project.GetWatchTargets()
			if err := p.LogWatcher.UpdateTargets(targets); err != nil {
				return fmt.Errorf("failed to set initial watch targets: %w", err)
			}

			// Start all services
			if err := p.ConfigWatcher.Start(ctx); err != nil {
				return fmt.Errorf("failed to start config watcher: %w", err)
			}
			if err := p.LogWatcher.Start(ctx); err != nil {
				return fmt.Errorf("failed to start log watcher: %w", err)
			}
			if err := p.LogProcessor.Start(ctx); err != nil {
				return fmt.Errorf("failed to start log processor: %w", err)
			}

			p.Logger.Info("daemon started",
				slog.Int("watchTargets", len(targets)),
				slog.String("configPath", p.Config.Cops.GlobalConfigPath),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("daemon stopping")

			// Stop in reverse order
			if err := p.LogProcessor.Stop(ctx); err != nil {
				p.Logger.Error("failed to stop log processor", slog.Any("error", err))
			}
			if err := p.LogWatcher.Stop(); err != nil {
				p.Logger.Error("failed to stop log watcher", slog.Any("error", err))
			}
			if err := p.ConfigWatcher.Stop(); err != nil {
				p.Logger.Error("failed to stop config watcher", slog.Any("error", err))
			}

			p.Logger.Info("daemon stopped")
			return nil
		},
	})
}
