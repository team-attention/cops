package connectrpc

import (
	"context"
	"log/slog"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
	logger *slog.Logger
	client eventv1connect.EventServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
	client := eventv1connect.NewEventServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
		apiClient.ConnectOptions()...,
	)

	return &APIClient{
		logger: l.With(slog.String("name", "log.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &eventv1.SendLogsReq{
		Batch: &eventv1.LogBatch{
			OrganizationId: batch.OrganizationID,
			ProjectId:      batch.ProjectID.String(),
			Jsonl:          batch.Lines,
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		code := connect.CodeOf(err)
		if code == connect.CodeUnknown {
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
		c.logger.Warn("API returned failure",
			slog.String("error", resp.Msg.ErrorMessage),
		)
	}

	c.logger.Debug("logs sent",
		slog.Int("count", len(batch.Lines)),
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}
