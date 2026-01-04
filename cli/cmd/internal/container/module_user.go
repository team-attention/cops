package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/user"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api/connectrpc"
)

// newUserModule registers all user-related providers.
func newUserModule(c *dig.Container) error {
	// API client
	if err := c.Provide(
		connectrpc.NewUserAPIClient,
		dig.As(new(api.UserAPIPort)),
	); err != nil {
		return err
	}

	// Service
	return c.Provide(user.NewService)
}
