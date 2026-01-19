package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of transcripts from a daemon for events collection.
type LogBatch struct {
	// Transcripts contains the parsed Transcript instances (all types: user, assistant, system, summary, file-history-snapshot).
	Transcripts []*domain.Transcript
	// ProjectID is the project identifier for this batch.
	ProjectID string
	// OrganizationID is the organization identifier for RBAC validation.
	OrganizationID string
	// UserID is the user who sent this batch.
	UserID string
}

// EventRepositoryPort defines the interface for event persistence.
type EventRepositoryPort interface {
	// SaveLogBatch saves a batch of JSONL records to events collection.
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	SaveLogBatch(ctx context.Context, batch *LogBatch) error
}
