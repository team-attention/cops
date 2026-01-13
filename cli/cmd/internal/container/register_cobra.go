package container

import (
	"context"

	"github.com/spf13/cobra"
	"go.uber.org/dig"
)

// CLICommandProvider provides CLI commands to be registered.
// Handlers implement this interface to participate in command registration.
type CLICommandProvider interface {
	// Commands returns the commands to register with the root command.
	Commands() []*cobra.Command
}

// CLILifecycle provides lifecycle hooks for CLI execution.
// Implementations are called during PersistentPreRun.
type CLILifecycle interface {
	// OnPreRun is called before any command runs (blocking).
	// Returns error to abort command execution.
	OnPreRun(ctx context.Context) error
}

// cobraParams collects all CLI handlers via group.
type cobraParams struct {
	dig.In

	Root       *cobra.Command
	Handlers   []CLICommandProvider `group:"cli_handlers"`
	Lifecycles []CLILifecycle       `group:"cli_lifecycles"`
}

// RegisterCobraCommands collects handlers via group, registers commands, and executes.
func RegisterCobraCommands(c *dig.Container) error {
	return c.Invoke(func(params cobraParams) error {
		// Collect and register all commands from handlers
		for _, handler := range params.Handlers {
			for _, cmd := range handler.Commands() {
				params.Root.AddCommand(cmd)
			}
		}

		// Register lifecycle hooks
		if len(params.Lifecycles) > 0 {
			params.Root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				if ctx == nil {
					ctx = context.Background()
				}
				for _, lc := range params.Lifecycles {
					if err := lc.OnPreRun(ctx); err != nil {
						return err
					}
				}
				return nil
			}
		}

		// Execute (trigger)
		return params.Root.Execute()
	})
}
