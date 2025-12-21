package container

import (
	"github.com/spf13/cobra"
	"go.uber.org/dig"

	setup_cobra "github.com/team-attention/cops/internal/platform/setup/cobra"
	"github.com/team-attention/cops/internal/platform/setup/config"
	"github.com/team-attention/cops/internal/platform/setup/logger"
)

// Run creates the container, registers all providers, and executes the root command.
func Run() error {
	c := dig.New()

	// Register all providers in dependency order.
	// Note: dig resolves the dependency graph automatically.
	providers := []interface{}{
		// Platform setup
		config.LoadConfig,  // *config.Config (root - no dependencies)
		logger.InitLogger,  // *slog.Logger (depends on *config.Config)

		// Command setup
		setup_cobra.NewRootCommand, // *cobra.Command (depends on *slog.Logger)

		// Add more providers here as the application grows
	}

	for _, p := range providers {
		if err := c.Provide(p); err != nil {
			return err
		}
	}

	// Execute the root command through the container
	return c.Invoke(func(cmd *cobra.Command) error {
		return cmd.Execute()
	})
}
