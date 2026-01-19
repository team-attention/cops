package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	userv1 "github.com/team-attention/cops/shared/gen/grpcstub/user/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect"
)

// UserAPIClient implements UserAPIPort using ConnectRPC.
type UserAPIClient struct {
	logger *slog.Logger
	client userv1connect.UserServiceClient
}

// NewUserAPIClient creates a new ConnectRPC user client.
//
// Dependencies:
// - l: Logger for structured logging
// - cfg: Configuration containing API server URL
// - httpClient: Typed HTTP client wrapper providing standard http.Client
//
// Returns:
// - *UserAPIClient: Initialized user API client ready for use
func NewUserAPIClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *UserAPIClient {
	logger := l.With(slog.String("name", "user.api.connectrpc"))

	client := userv1connect.NewUserServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &UserAPIClient{
		logger: logger,
		client: client,
	}
}

// GetMe fetches the authenticated user's information and organizations.
func (c *UserAPIClient) GetMe(ctx context.Context, accessToken string) (*api.GetMeResult, error) {
	req := connect.NewRequest(&userv1.GetMeReq{})
	req.Header().Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.GetMe(ctx, req)
	if err != nil {
		c.logger.Error("failed to get user info",
			slog.Any("error", err),
		)
		return nil, err
	}

	return &api.GetMeResult{
		UserID:        resp.Msg.User.Id,
		Organizations: resp.Msg.Organizations,
	}, nil
}

var _ api.UserAPIPort = (*UserAPIClient)(nil)
