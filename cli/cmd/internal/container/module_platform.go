package container

import (
	"go.uber.org/dig"

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
		httpclient.InitCollectorHTTPClient,
		httpclient.InitAPIHTTPClient,

		// Cobra root command (depends on logger)
		setup_cobra.NewRootCommand,
	}

	for _, p := range providers {
		if err := c.Provide(p); err != nil {
			return err
		}
	}

	return nil
}
