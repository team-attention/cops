package connectrpc

import (
	"context"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
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
		// Check if error indicates payload too large
		code := connect.CodeOf(err)
		if code == connect.CodeUnknown {
			// Check error message for "413" or "Request Entity Too Large"
			errMsg := err.Error()
			if strings.Contains(errMsg, "413") || strings.Contains(strings.ToLower(errMsg), "request entity too large") {
				return errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err)
			}
		}
		if code == connect.CodeResourceExhausted {
			return errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err)
		}
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("count", len(batch.Lines)),
	)

	return nil
}
