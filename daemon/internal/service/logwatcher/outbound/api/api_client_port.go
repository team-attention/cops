package api

import (
	"context"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
)

// APIClientPort is the port interface for sending logs to the API server.
type APIClientPort interface {
	// SendLogs sends a batch of logs to the API server.
	// Returns errutil.ErrorTypePayloadTooLarge if the server rejects with 413.
	SendLogs(ctx context.Context, batch domain.LogBatch) error
}
