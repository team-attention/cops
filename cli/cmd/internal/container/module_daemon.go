package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/daemon"
	cliadapter "github.com/team-attention/cops/cli/internal/service/daemon/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer"
	kardianosadapter "github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer/kardianos"
)

// newDaemonModule registers all daemon-related providers.
func newDaemonModule(c *dig.Container) error {
	// Outbound adapter - kardianos service installer
	if err := c.Provide(
		kardianosadapter.NewKardianosInstaller,
		dig.As(new(installer.InstallerPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(daemon.NewService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cliadapter.NewDaemonCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
