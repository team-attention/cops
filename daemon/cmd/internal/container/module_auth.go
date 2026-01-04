package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
	connectrpc "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api/connectrpc"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// Outbound: AuthAPIPort
		fx.Provide(fx.Annotate(
			connectrpc.NewAuthAPIClient,
			fx.As(new(api.AuthAPIPort)),
		)),
	)
}
