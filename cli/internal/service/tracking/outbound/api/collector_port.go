package api

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// CollectorPort defines the interface for sending records to the collector.
type CollectorPort interface {
	// SendRecords sends session records to the collector server.
	SendRecords(ctx context.Context, projectID domain.ID, records []*domain.SessionRecord) error
}
