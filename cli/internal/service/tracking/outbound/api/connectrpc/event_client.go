package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

// EventAPIClient implements EventAPIPort using ConnectRPC.
type EventAPIClient struct {
	logger *slog.Logger
	client eventv1connect.EventServiceClient
}

// NewEventAPIClient creates a new ConnectRPC event client.
func NewEventAPIClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.APIHTTPClient) *EventAPIClient {
	logger := l.With(slog.String("name", "tracking.api.event.connectrpc"))

	client := eventv1connect.NewEventServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &EventAPIClient{
		logger: logger,
		client: client,
	}
}

// SendEvents sends a batch of raw JSON event strings to the API server.
func (c *EventAPIClient) SendEvents(ctx context.Context, apiKey string, events []string) error {
	req := connect.NewRequest(&eventv1.SendEventsReq{
		Events: events,
	})
	req.Header().Set("Authorization", "Bearer "+apiKey)

	_, err := c.client.SendEvents(ctx, req)
	if err != nil {
		c.logger.Error("failed to send events",
			slog.Int("eventCount", len(events)),
			slog.Any("error", err),
		)
		return err
	}

	c.logger.Debug("events sent successfully",
		slog.Int("eventCount", len(events)),
	)

	return nil
}

var _ api.EventAPIPort = (*EventAPIClient)(nil)
