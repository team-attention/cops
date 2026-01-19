package container

import (
	"go.uber.org/fx"

	eventservice "github.com/team-attention/cops/api/internal/service/event"
	"github.com/team-attention/cops/api/internal/service/event/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository/mongodb"
)

func newEventModule() fx.Option {
	return fx.Module("event",
		// Repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoEventRepository,
				fx.As(new(repository.EventRepositoryPort)),
			),
		),

		// Service
		fx.Provide(eventservice.NewService),

		// ConnectRPC handler (API key auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewEventGRPCHandler,
				fx.As(new(APIKeyConnectHandler)),
				fx.ResultTags(`group:"apikey_connect_handlers"`),
				// 1. Cast to APIKeyConnectHandler interface for API key authentication
				// 2. Register to "apikey_connect_handlers" group
				// 3. This group is processed by API key interceptor in register_connectrpc.go
			),
		),
	)
}
