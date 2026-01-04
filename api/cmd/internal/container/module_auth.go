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

		// Service
		fx.Provide(auth.NewService),

		// Public ConnectRPC handler (no auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthPublicGRPCHandler,
				fx.As(new(PublicConnectHandler)),
				fx.ResultTags(`group:"public_connect_handlers"`),
			),
		),

		// Private ConnectRPC handler (auth required)
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthPrivateGRPCHandler,
				fx.As(new(PrivateConnectHandler)),
				fx.ResultTags(`group:"private_connect_handlers"`),
			),
		),
	)
}
