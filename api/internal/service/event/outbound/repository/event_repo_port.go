package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of records from a daemon for events collection.
type LogBatch struct {
	// Records contains the parsed Record instances (all types: user, assistant, file-history-snapshot).
	Records []*domain.Record
	// ProjectID is the project identifier for this batch.
	ProjectID string
	// OrganizationID is the organization identifier for RBAC validation.
	OrganizationID string
}

// EventRepositoryPort defines the interface for event persistence.
type EventRepositoryPort interface {
	// SaveEvent saves a single event to storage.
	SaveEvent(ctx context.Context, userID string, event *domain.Event) error

	// SaveEvents saves multiple events to storage in a batch.
	SaveEvents(ctx context.Context, userID string, events []*domain.Event) error

	// SaveLogBatch saves a batch of JSONL records to events collection.
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	SaveLogBatch(ctx context.Context, batch *LogBatch) error
}
