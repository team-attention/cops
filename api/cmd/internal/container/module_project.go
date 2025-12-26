package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/project"
	"github.com/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/project/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/project/outbound/repository/mongodb"
)

func newProjectModule() fx.Option {
	return fx.Module("project",
		// MongoDB Repository Adapter
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoProjectRepository,
				fx.As(new(repository.ProjectRepositoryPort)),
			),
		),

		// Service
		fx.Provide(project.NewService),

		// gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewProjectGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
