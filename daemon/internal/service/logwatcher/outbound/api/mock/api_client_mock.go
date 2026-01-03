package mock

import (
	"context"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
)

// APIClient implements api.APIClientPort for testing.
type APIClient struct {
	// SendLogsFunc is the behavior to execute when SendLogs is called.
	SendLogsFunc func(ctx context.Context, batch domain.LogBatch) error
	// CallCount tracks the number of SendLogs calls.
	CallCount int
	// Batches records all batches sent.
	Batches []domain.LogBatch
}

// SendLogs implements api.APIClientPort.
func (m *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	// 1. Increment CallCount
	m.CallCount++

	// 2. If SendLogsFunc is set, call it
	var err error
	if m.SendLogsFunc != nil {
		err = m.SendLogsFunc(ctx, batch)
	}

	// 3. Only record successful batches (when err is nil)
	if err == nil {
		batchCopy := domain.LogBatch{
			Lines:     append([]string(nil), batch.Lines...),
			ProjectID: batch.ProjectID,
		}
		m.Batches = append(m.Batches, batchCopy)
	}

	// 4. Return the error (or nil)
	return err
}

// Compile-time interface verification.
var _ api.APIClientPort = (*APIClient)(nil)
