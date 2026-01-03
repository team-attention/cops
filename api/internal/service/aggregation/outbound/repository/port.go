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
	// OrganizationID is the organization identifier for RBAC validation.
	// Each batch is for one Project that belongs to this Organization.
	OrganizationID string
}

// SessionRecordRepositoryPort defines the interface for record persistence.
// NOTE: Name kept for minimal interface change; consider renaming to RecordRepositoryPort in future.
type SessionRecordRepositoryPort interface {
	// SaveBatch saves a batch of records to storage.
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	SaveBatch(ctx context.Context, batch *LogBatch) error
}
