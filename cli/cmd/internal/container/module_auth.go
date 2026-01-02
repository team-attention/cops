package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/auth"
	"github.com/team-attention/cops/cli/internal/service/auth/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc"
)

// newAuthModule registers all auth-related providers.
func newAuthModule(c *dig.Container) error {
	// API client
	if err := c.Provide(
		connectrpc.NewAuthAPIClient,
		dig.As(new(api.AuthAPIPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(auth.NewService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cobra.NewAuthCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
