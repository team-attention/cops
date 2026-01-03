package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// ConnectHandler interface for ConnectRPC handlers.
type ConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

type connectRPCServerParams struct {
	fx.In

	Lifecycle       fx.Lifecycle
	Logger          *slog.Logger
	Config          *config.Config
	App             *fiber.App
	ConnectHandlers []ConnectHandler `group:"connect_handlers"`
}

func registerConnectRPCServer(params connectRPCServerParams) {
	logger := params.Logger.With(slog.String("name", "server.connectrpc"))

	// Create JWT config from params
	jwtCfg := &jwtutil.Config{
		SecretKey:            params.Config.JWT.SecretKey,
		AccessTokenDuration:  params.Config.JWT.AccessTokenDuration,
		RefreshTokenDuration: params.Config.JWT.RefreshTokenDuration,
		Issuer:               params.Config.JWT.Issuer,
	}

	// Create auth interceptor
	authInterceptor := interceptor.NewAuthInterceptor(logger, jwtCfg)

	// Create handler options with the interceptor
	opts := []connect.HandlerOption{connect.WithInterceptors(authInterceptor)}

	// Register ConnectRPC handlers with the interceptor options
	for _, handler := range params.ConnectHandlers {
		path, h := handler.GetHandler(opts...)
		params.App.All(path+"*", adaptor.HTTPHandler(h))
	}

	// Lifecycle hooks
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf(":%d", params.Config.HTTP.Port)
			logger.Info("Starting HTTP server", slog.String("addr", addr))

			go func() {
				if err := params.App.Listen(addr); err != nil {
					logger.Error("Server error", slog.Any("error", err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server")
			return params.App.ShutdownWithContext(ctx)
		},
	})
}
