package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/hook"
	"github.com/team-attention/cops/cli/internal/service/hook/inbound/cli/cobra"
)

// newHookModule registers all hook-related providers.
func newHookModule(c *dig.Container) error {
	// 1. Provide hook.Service
	//    - Dependencies (apikey.APIKeyPort, api.EventAPIPort) are already registered
	//      in module_platform.go and module_tracking.go
	if err := c.Provide(hook.NewService); err != nil {
		return err
	}

	// 2. Provide HookCLIHandler with dig.As + dig.Group
	//    - Cast to CLICommandProvider interface
	//    - Register to "cli_handlers" group
	return c.Provide(
		cobra.NewHookCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
