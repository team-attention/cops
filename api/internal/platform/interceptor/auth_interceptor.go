package interceptor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/apikeyutil"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/service/apikey"
)

// userIDContextKey is the context key for storing userID.
type userIDContextKey struct{}

// UserIDFromContext extracts userID from context.
// Returns empty string if not found.
func UserIDFromContext(ctx context.Context) string {
	value := ctx.Value(userIDContextKey{})
	if value == nil {
		return ""
	}

	userID, ok := value.(string)
	if !ok {
		return ""
	}

	return userID
}

// NewAuthInterceptor creates a ConnectRPC unary interceptor for authentication.
// It supports both JWT tokens and API keys (cops_ prefix).
// It validates the Authorization header and adds userID to context.
func NewAuthInterceptor(l *slog.Logger, jwtCfg *jwtutil.Config, apiKeySvc *apikey.Service) connect.UnaryInterceptorFunc {
	logger := l.With(slog.String("name", "interceptor.auth"))

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				return next(ctx, req)
			}

			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing authorization header"))
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid authorization header format"))
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			var userID string

			// Check if token is API key (cops_ prefix)
			if strings.HasPrefix(tokenString, apikeyutil.KeyPrefix) {
				// API Key authentication
				result, err := apiKeySvc.ValidateAPIKey(ctx, tokenString)
				if err != nil {
					logger.Error("failed to validate API key", slog.Any("error", err))
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to validate API key"))
				}
				if !result.Valid {
					logger.Warn("invalid API key", slog.String("errorMessage", result.ErrorMessage))
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New(result.ErrorMessage))
				}
				userID = result.UserID
			} else {
				// JWT authentication
				var err error
				userID, err = jwtutil.ValidateAccessToken(jwtCfg, tokenString)
				if err != nil {
					logger.Warn("invalid access token", slog.String("error", err.Error()))
					return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired token"))
				}
			}

			ctx = context.WithValue(ctx, userIDContextKey{}, userID)

			return next(ctx, req)
		}
	}
}
