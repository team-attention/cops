package container

import (
	"go.uber.org/fx"

	logservice "github.com/team-attention/cops/api/internal/service/log"
	"github.com/team-attention/cops/api/internal/service/log/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/log/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/log/outbound/repository/mongodb"
)

func newLogModule() fx.Option {
	return fx.Module("log",
		// Repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewAdapter,
				fx.As(new(repository.SessionRecordRepositoryPort)),
			),
		),

		// Service
		fx.Provide(logservice.NewService),

		// gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewLogGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
