package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/featured"
	"github.com/team-attention/cops/api/internal/service/featured/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/featured/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/featured/outbound/repository/mongodb"
)

func newFeaturedModule() fx.Option {
	return fx.Module("featured",
		// MongoDB Repository Adapter
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoFeaturedBoardRepository,
				fx.As(new(repository.FeaturedBoardRepositoryPort)),
			),
		),

		// Service
		fx.Provide(featured.NewFeaturedBoardService),

		// gRPC Handler (public - no auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewFeaturedBoardGRPCHandler,
				fx.As(new(PublicConnectHandler)),
				fx.ResultTags(`group:"public_connect_handlers"`),
			),
		),
	)
}
