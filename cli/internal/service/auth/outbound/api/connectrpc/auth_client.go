package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

type AuthAPIClient struct {
	logger *slog.Logger
	client authv1connect.AuthServiceClient
}

func NewAuthAPIClient(l *slog.Logger, httpClient *http.Client, baseURL string) *AuthAPIClient {
	return &AuthAPIClient{
		logger: l.With(slog.String("name", "auth.api.connectrpc")),
		client: authv1connect.NewAuthServiceClient(httpClient, baseURL),
	}
}

func (c *AuthAPIClient) DeviceCode(ctx context.Context) (*api.DeviceCodeResult, error) {
	req := connect.NewRequest(&authv1.DeviceCodeReq{})

	resp, err := c.client.DeviceCode(ctx, req)
	if err != nil {
		c.logger.Error("failed to get device code",
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

func (c *AuthAPIClient) DevicePoll(ctx context.Context, deviceCode string) (*api.PollResult, error) {
	req := connect.NewRequest(&authv1.DevicePollReq{
		DeviceCode: deviceCode,
	})

	resp, err := c.client.DevicePoll(ctx, req)
	if err != nil {
		c.logger.Error("failed to poll device code",
			slog.Any("error", err),
		)
		return nil, err
	}

	result := &api.PollResult{
		Pending: resp.Msg.Pending,
	}

	if !resp.Msg.Pending && resp.Msg.Tokens != nil {
		result.AccessToken = resp.Msg.Tokens.AccessToken
		result.RefreshToken = resp.Msg.Tokens.RefreshToken
		result.ExpiresAt = resp.Msg.Tokens.ExpiresAt
	}

	return result, nil
}

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

var _ api.AuthAPIPort = (*AuthAPIClient)(nil)
