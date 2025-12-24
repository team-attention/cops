package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/health"
	"github.com/team-attention/cops/api/internal/service/health/inbound/grpc/connectrpc"
)

func newHealthModule() fx.Option {
	return fx.Module("health",
		// Service
		fx.Provide(health.NewService),

		// gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewHealthGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
