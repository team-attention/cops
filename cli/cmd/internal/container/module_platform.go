package container

import (
	"log/slog"

	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/platform/outbound/apikey"
	apikey_filesystem "github.com/team-attention/cops/cli/internal/platform/outbound/apikey/filesystem"
	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	authstate_filesystem "github.com/team-attention/cops/cli/internal/platform/outbound/authstate/filesystem"
	"github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig"
	hookconfig_filesystem "github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig/filesystem"
	setup_cobra "github.com/team-attention/cops/cli/internal/platform/setup/cobra"
	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/platform/setup/logger"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
)

// newPlatformModule registers all platform-level providers.
func newPlatformModule(c *dig.Container) error {
	providers := []any{
		// Configuration (root - no dependencies)
		config.LoadConfig,

		// Logger (depends on config)
		logger.InitLogger,

		// ExpandedPaths (depends on config)
		provideExpandedPaths,

		// HTTP clients (depends on config)
		httpclient.InitAPIHTTPClient,
		httpclient.InitGitHubHTTPClient,

		// Cobra root command (depends on logger)
		setup_cobra.NewRootCommand,
	}

	for _, p := range providers {
		if err := c.Provide(p); err != nil {
			return err
		}
	}

	// Platform outbound adapters
	if err := c.Provide(
		provideFilesystemAuthState,
		dig.As(new(authstate.AuthStatePort)),
	); err != nil {
		return err
	}

	// Hook config adapter
	if err := c.Provide(
		provideFilesystemHookConfig,
		dig.As(new(hookconfig.HookConfigPort)),
	); err != nil {
		return err
	}

	// API key adapter (for hook authentication)
	if err := c.Provide(
		provideFilesystemAPIKey,
		dig.As(new(apikey.APIKeyPort)),
	); err != nil {
		return err
	}

	return nil
}

// provideExpandedPaths creates ExpandedPaths from config.
func provideExpandedPaths(cfg *config.Config) (*pathutil.ExpandedPaths, error) {
	return pathutil.NewExpandedPaths(cfg.Paths.BaseDir, cfg.Paths.LocalConfigDir)
}

// provideFilesystemAuthState wraps the authstate adapter constructor.
func provideFilesystemAuthState(l *slog.Logger, paths *pathutil.ExpandedPaths) authstate.AuthStatePort {
	return authstate_filesystem.NewFilesystemAuthState(l, paths.AuthPath)
}

// provideFilesystemHookConfig wraps the hookconfig adapter constructor.
func provideFilesystemHookConfig(l *slog.Logger, paths *pathutil.ExpandedPaths) hookconfig.HookConfigPort {
	return hookconfig_filesystem.NewFilesystemHookConfig(l, paths.AuthPath)
}

// provideFilesystemAPIKey wraps the apikey adapter constructor.
func provideFilesystemAPIKey(l *slog.Logger, paths *pathutil.ExpandedPaths) apikey.APIKeyPort {
	return apikey_filesystem.NewFilesystemAPIKey(l, paths.AuthPath)
}
