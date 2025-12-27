package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/shared/domain"
	projectv1 "github.com/team-attention/cops/shared/gen/grpcstub/project/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/project/v1/projectv1connect"
)

// ProjectClient implements ProjectPort using ConnectRPC.
type ProjectClient struct {
	logger *slog.Logger
	client projectv1connect.ProjectServiceClient
}

// NewProjectClient creates a new ConnectRPC project client.
func NewProjectClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *ProjectClient {
	logger := l.With(slog.String("name", "tracking.api.connectrpc"))

	client := projectv1connect.NewProjectServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &ProjectClient{
		logger: logger,
		client: client,
	}
}

// RegisterProject registers a project or returns existing project ID if already registered.
func (c *ProjectClient) RegisterProject(ctx context.Context, params api.RegisterProjectParams) (*api.RegisterProjectResult, error) {
	req := connect.NewRequest(&projectv1.RegisterProjectReq{
		ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
		ActualRemoteUrl:     params.ActualRemoteURL,
		ExistingProjectId:   params.ExistingProjectID,
		Name:                params.Name,
		IsGitProject:        params.IsGitProject,
	})

	resp, err := c.client.RegisterProject(ctx, req)
	if err != nil {
		c.logger.Error("failed to register project", slog.Any("error", err))
		return nil, err
	}

	result := &api.RegisterProjectResult{
		ProjectID:    domain.ID(resp.Msg.ProjectId),
		IsNew:        resp.Msg.IsNew,
		Name:         resp.Msg.Name,
		IsGitProject: resp.Msg.IsGitProject,
	}

	c.logger.Info("project registered successfully",
		slog.String("projectID", string(result.ProjectID)),
		slog.Bool("isNew", result.IsNew))

	return result, nil
}

// Compile-time interface verification
var _ api.ProjectPort = (*ProjectClient)(nil)
