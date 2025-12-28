package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
	logger *slog.Logger
	client aggregationv1connect.AggregationServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
	client := aggregationv1connect.NewAggregationServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &APIClient{
		logger: l.With(slog.String("name", "log.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			Jsonl:     batch.Lines,
			ProjectId: batch.ProjectID.String(),
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}
