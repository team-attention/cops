package container

import (
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
)

// newPlatformModule registers all platform-level providers.
func newPlatformModule(c *dig.Container) error {
	providers := []interface{}{
		// Configuration (root - no dependencies)
		config.LoadConfig,

		// Logger (depends on config)
		logger.InitLogger,

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
		authstate_filesystem.NewFilesystemAuthState,
		dig.As(new(authstate.AuthStatePort)),
	); err != nil {
		return err
	}

	// Hook config adapter
	if err := c.Provide(
		hookconfig_filesystem.NewFilesystemHookConfig,
		dig.As(new(hookconfig.HookConfigPort)),
	); err != nil {
		return err
	}

	// API key adapter (for hook authentication)
	if err := c.Provide(
		apikey_filesystem.NewFilesystemAPIKey,
		dig.As(new(apikey.APIKeyPort)),
	); err != nil {
		return err
	}

	return nil
}
