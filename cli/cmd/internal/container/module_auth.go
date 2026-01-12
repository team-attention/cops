package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/auth"
	cliadapter "github.com/team-attention/cops/cli/internal/service/auth/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	connectrpcadapter "github.com/team-attention/cops/cli/internal/service/auth/outbound/api/connectrpc"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/storage"
	filesystemadapter "github.com/team-attention/cops/cli/internal/service/auth/outbound/storage/filesystem"
)

// newAuthModule registers auth service and CLI handler providers.
func newAuthModule(c *dig.Container) error {
	// Provide auth API client (outbound adapter)
	if err := c.Provide(
		connectrpcadapter.NewAuthClient,
		dig.As(new(api.AuthAPIPort)),
	); err != nil {
		return err
	}

	// Provide API key storage (outbound adapter)
	if err := c.Provide(
		filesystemadapter.NewFilesystemAPIKeyStorage,
		dig.As(new(storage.APIKeyStoragePort)),
	); err != nil {
		return err
	}

	// Provide auth service
	if err := c.Provide(auth.NewService); err != nil {
		return err
	}

	// Provide CLI handler with group registration
	return c.Provide(
		cliadapter.NewAuthCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
