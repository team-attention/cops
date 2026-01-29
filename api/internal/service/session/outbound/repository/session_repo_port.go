package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
	session "github.com/team-attention/cops/shared/domain/v2"
)

// SessionRepositoryPort defines the interface for Session persistence.
type SessionRepositoryPort interface {
	// SaveBatch saves multiple Session records to the sessions collection.
	// All sessions in the batch should belong to the same project.
	SaveBatch(ctx context.Context, projectID, userID domain.ID, sessions []*session.Session) error
}
