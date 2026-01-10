package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/apikey"
	"github.com/team-attention/cops/api/internal/service/apikey/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository/mongodb"
)

func newAPIKeyModule() fx.Option {
	return fx.Module("apikey",
		// Repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoAPIKeyRepository,
				fx.As(new(repository.APIKeyRepositoryPort)),
			),
		),

		// Service
		fx.Provide(apikey.NewService),

		// ConnectRPC handler (private - requires JWT auth)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAPIKeyGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
