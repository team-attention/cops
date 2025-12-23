package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/dashboard"
	"github.com/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb"
)

func newDashboardModule() fx.Option {
	return fx.Module("dashboard",
		// MongoDB Repository Adapter
		fx.Provide(
			fx.Annotate(
				mongodb.NewAdapter,
				fx.As(new(repository.DashboardRepositoryPort)),
			),
		),

		// Service
		fx.Provide(dashboard.NewService),

		// gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewDashboardGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
