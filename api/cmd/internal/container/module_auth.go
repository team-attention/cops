package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/service/auth"
	"github.com/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth/google"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// OAuth adapter
		fx.Provide(
			fx.Annotate(
				google.NewGoogleOAuthAdapter,
				fx.As(new(oauth.GoogleOAuthPort)),
			),
		),

		// User repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoUserRepository,
				fx.As(new(repository.UserRepositoryPort)),
			),
		),

		// Device code repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoDeviceCodeRepository,
				fx.As(new(repository.DeviceCodeRepositoryPort)),
			),
		),

		// Service - receives *config.Config directly
		fx.Provide(auth.NewService),

		// ConnectRPC handler - receives *config.Config directly
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
