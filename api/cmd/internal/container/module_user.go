package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/user"
	"github.com/team-attention/cops/api/internal/service/user/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/user/outbound/repository/mongodb"
)

func newUserModule() fx.Option {
	return fx.Module("user",
		// User repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoUserRepository,
				fx.As(new(repository.UserRepositoryPort)),
			),
		),

		// Organization repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoOrganizationRepository,
				fx.As(new(repository.OrganizationRepositoryPort)),
			),
		),

		// Cascade delete repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoCascadeDeleteRepository,
				fx.As(new(repository.CascadeDeleteRepositoryPort)),
			),
		),

		// Service
		fx.Provide(user.NewService),

		// ConnectRPC handler (private - requires auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewUserGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
