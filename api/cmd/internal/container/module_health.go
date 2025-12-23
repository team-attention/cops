package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/health"
	"github.com/team-attention/cops/api/internal/service/health/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/health/inbound/http/fiber"
)

func newHealthModule() fx.Option {
	return fx.Module("health",
		// Service
		fx.Provide(health.NewService),

		// HTTP Handler
		fx.Provide(
			fx.Annotate(
				fiber.NewHealthHTTPHandler,
				fx.As(new(HTTPRouter)),
				fx.ResultTags(`group:"http_routers"`),
			),
		),

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
