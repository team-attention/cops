package repository

import (
	"context"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of session records from a daemon.
type LogBatch struct {
	Records   []shareddomain.SessionRecord
	DaemonID  string
	CreatedAt string
}

// SessionRecordRepositoryPort defines the interface for session record persistence.
type SessionRecordRepositoryPort interface {
	SaveBatch(ctx context.Context, batch *LogBatch) error
}
