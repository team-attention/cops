package container

import (
	"os"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/service/auth"
	"github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
	connectrpc "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api/connectrpc"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// Provide home directory for auth file path
		fx.Provide(func() string {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "."
			}
			return homeDir
		}),

		// Outbound: AuthAPIPort
		fx.Provide(fx.Annotate(
			connectrpc.NewAuthAPIClient,
			fx.As(new(api.AuthAPIPort)),
		)),

		// Service
		fx.Provide(auth.NewService),
	)
}
