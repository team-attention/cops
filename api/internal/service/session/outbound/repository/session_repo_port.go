package repository

import (
	"context"

	session "github.com/team-attention/cops/shared/domain/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SessionRepositoryPort defines the interface for Session persistence.
type SessionRepositoryPort interface {
	// SaveBatch saves multiple Session records to the sessions collection.
	// All sessions in the batch should belong to the same project.
	SaveBatch(ctx context.Context, projectID, userID bson.ObjectID, sessions []*session.Session) error
}
