package container

import (
	"github.com/spf13/cobra"
	"go.uber.org/dig"

	setup "github.com/team-attention/code-rules/internal/platform/setup/cobra"
)

// Run creates the container, registers all providers, and executes the root command
func Run() error {
	c := dig.New()

	// Register all providers
	providers := []interface{}{
		setup.NewRootCommand,
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
