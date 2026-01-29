package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// EventBatch represents a batch of event data for storage.
type EventBatch struct {
	// DataLines contains the raw JSONL strings exactly as received.
	// JSON parsing to Data field is performed at save time.
	DataLines []string
	// Version is the event structure version.
	Version string
	// Provider is the source provider (claude, gemini, codex).
	Provider domain.EventProvider
	// ProjectID is the project identifier for this batch.
	ProjectID string
	// OrganizationID is the organization identifier for RBAC validation.
	OrganizationID string
	// UserID is the user who sent this batch.
	UserID string
	// BatchID groups events from the same request for batch-level operations.
	BatchID string
}

// EventRepositoryPort defines the interface for event persistence.
type EventRepositoryPort interface {
	// SaveEventBatch saves a batch of event data to the events collection.
	// Each non-empty line is parsed and stored as an Event with status "pending".
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	// Returns the count of events stored (excluding empty lines).
	SaveEventBatch(ctx context.Context, batch *EventBatch) (int, error)
}
