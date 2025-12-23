package container

import (
	"github.com/spf13/cobra"
	"go.uber.org/dig"
)

// CLICommandProvider provides CLI commands to be registered.
// Handlers implement this interface to participate in command registration.
type CLICommandProvider interface {
	// Commands returns the commands to register with the root command.
	Commands() []*cobra.Command
}

// cobraParams collects all CLI handlers via group.
type cobraParams struct {
	dig.In

	Root     *cobra.Command
	Handlers []CLICommandProvider `group:"cli_handlers"`
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

		// Execute (trigger)
		return params.Root.Execute()
	})
}
