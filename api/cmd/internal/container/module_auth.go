package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/service/auth"
	"github.com/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth/google"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb"
)

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// JWT config from main config
		fx.Provide(func(cfg *config.Config) *jwtutil.Config {
			return &jwtutil.Config{
				SecretKey:            cfg.JWT.SecretKey,
				AccessTokenDuration:  cfg.JWT.AccessTokenDuration,
				RefreshTokenDuration: cfg.JWT.RefreshTokenDuration,
				Issuer:               cfg.JWT.Issuer,
			}
		}),

		// OAuth adapter (config injected via constructor)
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

		// Service
		fx.Provide(auth.NewService),

		// ConnectRPC handler
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAuthGRPCHandler,
				fx.As(new(ConnectHandler)),
				fx.ResultTags(`group:"connect_handlers"`),
			),
		),
	)
}
