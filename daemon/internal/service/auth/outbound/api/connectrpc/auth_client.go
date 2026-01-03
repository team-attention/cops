package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

// AuthAPIClient implements AuthAPIPort using ConnectRPC.
type AuthAPIClient struct {
	logger *slog.Logger
	client authv1connect.AuthServiceClient
}

// NewAuthAPIClient creates a new ConnectRPC auth API client adapter.
func NewAuthAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *AuthAPIClient {
	logger := l.With(slog.String("name", "auth.api.connectrpc"))

	client := authv1connect.NewAuthServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &AuthAPIClient{
		logger: logger,
		client: client,
	}
}

// RefreshToken exchanges a refresh token for a new token pair.
func (c *AuthAPIClient) RefreshToken(ctx context.Context, refreshToken string) (*api.TokenResult, error) {
	req := connect.NewRequest(&authv1.RefreshTokenReq{
		RefreshToken: refreshToken,
	})

	resp, err := c.client.RefreshToken(ctx, req)
	if err != nil {
		c.logger.Error("failed to refresh token",
			slog.Any("error", err),
		)
		return nil, err
	}

	return &api.TokenResult{
		AccessToken:  resp.Msg.Tokens.AccessToken,
		RefreshToken: resp.Msg.Tokens.RefreshToken,
		ExpiresAt:    resp.Msg.Tokens.ExpiresAt,
	}, nil
}

// Compile-time interface verification.
var _ api.AuthAPIPort = (*AuthAPIClient)(nil)
