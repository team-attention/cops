package interceptor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/apikey"
)

// NewAPIKeyInterceptor creates a ConnectRPC unary interceptor for API key authentication.
// It validates the API key from Authorization header and adds userID to context.
// Only applies to procedures listed in targetProcedures.
func NewAPIKeyInterceptor(l *slog.Logger, apiKeySvc *apikey.Service, targetProcedures []string) connect.UnaryInterceptorFunc {
	logger := l.With(slog.String("name", "interceptor.apikey"))

	// Build a map for quick lookup
	procedureMap := make(map[string]bool)
	for _, proc := range targetProcedures {
		procedureMap[proc] = true
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				return next(ctx, req)
			}

			// Check if procedure is in targetProcedures list
			procedure := req.Spec().Procedure
			if !procedureMap[procedure] {
				return next(ctx, req)
			}

			// Extract Authorization header
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing authorization header"))
			}

			// Parse "Bearer {key}" format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid authorization header format"))
			}

			apiKey := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate API key
			result, err := apiKeySvc.ValidateAPIKey(ctx, apiKey)
			if err != nil {
				logger.Error("failed to validate API key",
					slog.String("procedure", procedure),
					slog.Any("error", err),
				)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to validate API key"))
			}

			if !result.Valid {
				logger.Warn("invalid API key",
					slog.String("procedure", procedure),
					slog.String("errorMessage", result.ErrorMessage),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New(result.ErrorMessage))
			}

			// Inject userID into context
			ctx = context.WithValue(ctx, userIDContextKey{}, result.UserID)

			return next(ctx, req)
		}
	}
}
