package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
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

// NewAuthInterceptor creates a ConnectRPC unary interceptor for JWT authentication.
// It validates the Authorization header and adds userID to context.
func NewAuthInterceptor(l *slog.Logger, jwtCfg *jwtutil.Config) connect.UnaryInterceptorFunc {
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

			userID, err := jwtutil.ValidateAccessToken(jwtCfg, tokenString)
			if err != nil {
				logger.Warn("invalid access token",
					slog.String("error", err.Error()),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired token"))
			}

			ctx = context.WithValue(ctx, userIDContextKey{}, userID)

			return next(ctx, req)
		}
	}
}
