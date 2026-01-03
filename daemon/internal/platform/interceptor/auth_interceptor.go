package interceptor

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/service/auth"
)

// AuthInterceptor adds authentication to ConnectRPC requests and handles token refresh.
type AuthInterceptor struct {
	logger      *slog.Logger
	authService *auth.Service
}

// NewAuthInterceptor creates a new authentication interceptor.
func NewAuthInterceptor(l *slog.Logger, authService *auth.Service) *AuthInterceptor {
	return &AuthInterceptor{
		logger:      l.With(slog.String("name", "auth.interceptor")),
		authService: authService,
	}
}

// WrapUnary implements connect.Interceptor for unary RPCs.
func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Only intercept client requests
		if !req.Spec().IsClient {
			return next(ctx, req)
		}

		// Try to get access token
		token, err := i.authService.GetAccessToken()
		if err != nil {
			// User not logged in or token expired
			i.logger.Debug("no valid access token available, sending request without auth")

			// Send request without Authorization header
			resp, reqErr := next(ctx, req)
			if reqErr != nil && connect.CodeOf(reqErr) == connect.CodeUnauthenticated {
				i.logger.Warn("request failed with 401, user may need to re-authenticate via CLI")
				return nil, reqErr
			}
			return resp, reqErr
		}

		// Set Authorization header with valid token
		req.Header().Set("Authorization", "Bearer "+token)

		// Execute request
		resp, err := next(ctx, req)

		// Check for 401 Unauthenticated error
		if err != nil && connect.CodeOf(err) == connect.CodeUnauthenticated {
			i.logger.Info("received 401, attempting token refresh")

			// Attempt token refresh
			newToken, refreshErr := i.authService.RefreshAccessToken(ctx)
			if refreshErr != nil {
				i.logger.Warn("token refresh failed, user may need to re-authenticate via CLI",
					slog.Any("error", refreshErr),
				)
				return nil, err // Return original 401 error
			}

			// Update request header with new token
			req.Header().Set("Authorization", "Bearer "+newToken)

			// Retry request with new token
			i.logger.Debug("retrying request with new token")
			return next(ctx, req)
		}

		return resp, err
	}
}

// WrapStreamingClient implements connect.Interceptor for streaming client RPCs.
func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		// Get connection
		conn := next(ctx, spec)

		// Try to get access token
		token, err := i.authService.GetAccessToken()
		if err != nil {
			// User not logged in
			i.logger.Debug("no valid access token for streaming request")
			// Do NOT set Authorization header - let server decide
			return conn
		}

		// Set Authorization header with valid token
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
