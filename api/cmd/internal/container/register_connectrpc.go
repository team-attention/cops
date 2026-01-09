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
	"github.com/team-attention/cops/api/internal/service/apikey"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

// PublicConnectHandler interface for ConnectRPC handlers that do not require authentication.
type PublicConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

// PrivateConnectHandler interface for ConnectRPC handlers that require authentication.
type PrivateConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

// APIKeyConnectHandler interface for ConnectRPC handlers that require API key authentication.
type APIKeyConnectHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

type connectRPCServerParams struct {
	fx.In

	Lifecycle        fx.Lifecycle
	Logger           *slog.Logger
	Config           *config.Config
	App              *fiber.App
	APIKeyService    *apikey.Service
	PublicHandlers   []PublicConnectHandler   `group:"public_connect_handlers"`
	PrivateHandlers  []PrivateConnectHandler  `group:"private_connect_handlers"`
	APIKeyHandlers   []APIKeyConnectHandler   `group:"apikey_connect_handlers"`
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

	// Create auth interceptor for JWT authentication
	authInterceptor := interceptor.NewAuthInterceptor(logger, jwtCfg)

	// Create API key interceptor for API key authentication
	// Apply to SendEvents endpoint
	apiKeyTargetProcedures := []string{
		eventv1connect.EventServiceSendEventsProcedure,
		// 1. Add SendEvents procedure to API key interceptor target list
		// 2. This ensures API key validation is applied before SendEvents handler
		// 3. Procedure value is "/event.v1.EventService/SendEvents"
	}
	apiKeyInterceptor := interceptor.NewAPIKeyInterceptor(logger, params.APIKeyService, apiKeyTargetProcedures)

	// Create handler options with interceptor for private handlers
	privateOpts := []connect.HandlerOption{connect.WithInterceptors(authInterceptor)}

	// Create handler options with API key interceptor
	apiKeyOpts := []connect.HandlerOption{connect.WithInterceptors(apiKeyInterceptor)}

	// Register public handlers WITHOUT any interceptor options
	for _, handler := range params.PublicHandlers {
		path, h := handler.GetHandler()
		params.App.All(path+"*", adaptor.HTTPHandler(h))
	}

	// Register private handlers WITH auth interceptor options
	for _, handler := range params.PrivateHandlers {
		path, h := handler.GetHandler(privateOpts...)
		params.App.All(path+"*", adaptor.HTTPHandler(h))
	}

	// Register API key handlers WITH API key interceptor options
	for _, handler := range params.APIKeyHandlers {
		path, h := handler.GetHandler(apiKeyOpts...)
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
