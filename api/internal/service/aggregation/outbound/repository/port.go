package repository

import (
	"context"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of records from a daemon.
type LogBatch struct {
	// Records contains the parsed Record instances (all types).
	Records []shareddomain.Record
	// ProjectID is the project identifier for this batch.
	ProjectID string
}

// SessionRecordRepositoryPort defines the interface for record persistence.
// NOTE: Name kept for minimal interface change; consider renaming to RecordRepositoryPort in future.
type SessionRecordRepositoryPort interface {
	// SaveBatch saves a batch of records to storage.
	SaveBatch(ctx context.Context, batch *LogBatch) error
}
