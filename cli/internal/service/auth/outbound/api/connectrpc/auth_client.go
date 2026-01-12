package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	apikeyv1 "github.com/team-attention/cops/shared/gen/grpcstub/apikey/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/apikey/v1/apikeyv1connect"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

// AuthClient implements AuthAPIPort using ConnectRPC.
type AuthClient struct {
	logger       *slog.Logger
	authClient   authv1connect.AuthServiceClient
	apikeyClient apikeyv1connect.APIKeyServiceClient
}

// NewAuthClient creates a new auth API client.
func NewAuthClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *AuthClient {
	logger := l.With(slog.String("name", "auth.api.connectrpc"))

	authClient := authv1connect.NewAuthServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	apikeyClient := apikeyv1connect.NewAPIKeyServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &AuthClient{
		logger:       logger,
		authClient:   authClient,
		apikeyClient: apikeyClient,
	}
}

// DeviceCode initiates device flow authentication.
func (c *AuthClient) DeviceCode(ctx context.Context) (*api.DeviceCodeResult, error) {
	req := connect.NewRequest(&authv1.DeviceCodeReq{})

	resp, err := c.authClient.DeviceCode(ctx, req)
	if err != nil {
		c.logger.Error("failed to initiate device code",
			slog.Any("error", err),
		)
		return nil, err
	}

	return &api.DeviceCodeResult{
		DeviceCode:      resp.Msg.DeviceCode,
		UserCode:        resp.Msg.UserCode,
		VerificationURL: resp.Msg.VerificationUrl,
		ExpiresIn:       int(resp.Msg.ExpiresIn),
		Interval:        int(resp.Msg.Interval),
	}, nil
}

// DevicePoll polls for device code approval status.
func (c *AuthClient) DevicePoll(ctx context.Context, deviceCode string) (*api.DevicePollResult, error) {
	req := connect.NewRequest(&authv1.DevicePollReq{
		DeviceCode: deviceCode,
	})

	resp, err := c.authClient.DevicePoll(ctx, req)
	if err != nil {
		c.logger.Error("failed to poll device status",
			slog.Any("error", err),
		)
		return nil, err
	}

	result := &api.DevicePollResult{
		Pending: resp.Msg.Pending,
	}

	if resp.Msg.Tokens != nil {
		result.AccessToken = resp.Msg.Tokens.AccessToken
		result.RefreshToken = resp.Msg.Tokens.RefreshToken
		result.ExpiresAt = resp.Msg.Tokens.ExpiresAt
	}

	return result, nil
}

// IssueAPIKey creates a new API key using the access token.
func (c *AuthClient) IssueAPIKey(ctx context.Context, accessToken string, name string) (string, error) {
	req := connect.NewRequest(&apikeyv1.IssueAPIKeyReq{
		Name:          name,
		ExpiresInDays: 0, // Permanent - no expiration
	})
	req.Header().Set("Authorization", "Bearer "+accessToken)

	resp, err := c.apikeyClient.IssueAPIKey(ctx, req)
	if err != nil {
		c.logger.Error("failed to issue API key",
			slog.Any("error", err),
		)
		return "", err
	}

	return resp.Msg.ApiKey, nil
}

// Compile-time interface verification
var _ api.AuthAPIPort = (*AuthClient)(nil)
