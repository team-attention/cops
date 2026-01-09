package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// EventRepositoryPort defines the interface for event persistence.
type EventRepositoryPort interface {
	// SaveEvent saves a single event to storage.
	SaveEvent(ctx context.Context, userID string, event *domain.Event) error

	// SaveEvents saves multiple events to storage in a batch.
	SaveEvents(ctx context.Context, userID string, events []*domain.Event) error
}
