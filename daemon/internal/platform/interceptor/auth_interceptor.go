package interceptor

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
)

// AuthInterceptor adds authentication to ConnectRPC requests.
type AuthInterceptor struct {
	logger    *slog.Logger
	authState authstate.AuthStatePort
}

// NewAuthInterceptor creates a new authentication interceptor.
func NewAuthInterceptor(l *slog.Logger, authState authstate.AuthStatePort) *AuthInterceptor {
	return &AuthInterceptor{
		logger:    l.With(slog.String("name", "auth.interceptor")),
		authState: authState,
	}
}

// WrapUnary implements connect.Interceptor for unary RPCs.
func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if !req.Spec().IsClient {
			return next(ctx, req)
		}

		token, err := i.authState.GetAccessToken(ctx)
		if err != nil {
			i.logger.Error("failed to get access token",
				slog.Any("error", err),
			)
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		req.Header().Set("Authorization", "Bearer "+token)

		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor for streaming client RPCs.
func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)

		token, err := i.authState.GetAccessToken(ctx)
		if err != nil {
			i.logger.Warn("no valid access token for streaming request",
				slog.Any("error", err),
			)
			return conn
		}

		conn.RequestHeader().Set("Authorization", "Bearer "+token)

		return conn
	}
}

// WrapStreamingHandler implements connect.Interceptor for streaming handler RPCs.
// This is a no-op for client-side interceptor.
func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// Compile-time interface verification.
var _ connect.Interceptor = (*AuthInterceptor)(nil)
