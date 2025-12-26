package container

import (
	"go.uber.org/fx"

	aggregationservice "github.com/team-attention/cops/api/internal/service/aggregation"
	"github.com/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb"
)

func newAggregationModule() fx.Option {
	return fx.Module("aggregation",
		// Repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoSessionRecordRepository,
				fx.As(new(repository.SessionRecordRepositoryPort)),
			),
		),

		// Service
		fx.Provide(aggregationservice.NewService),

		// gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAggregationGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
