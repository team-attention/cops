package repository

import (
	"context"

	"github.com/team-attention/cops/api/internal/platform/domain"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// RetryRepositoryPort defines the interface for retry event persistence.
type RetryRepositoryPort interface {
	// FindRetryableEvents finds events with status "failed" and retryCount < maxRetries.
	// Returns events sorted by createdAt ascending (oldest first).
	FindRetryableEvents(ctx context.Context, maxRetries int, limit int) ([]*domain.FailedEvent, error)

	// IncrementRetryCount atomically increments retry count and updates lastRetryAt.
	// Returns the updated event or nil if not found (already processed by another worker).
	IncrementRetryCount(ctx context.Context, eventID shareddomain.ID) (*domain.FailedEvent, error)

	// MarkPermanentlyFailed marks an event as permanently_failed with the given reason.
	MarkPermanentlyFailed(ctx context.Context, eventID shareddomain.ID, reason string) error

	// DeleteEvent removes a successfully processed event from the failed_events collection.
	DeleteEvent(ctx context.Context, eventID shareddomain.ID) error

	// SaveFailedEvent saves a new failed event to the collection.
	SaveFailedEvent(ctx context.Context, event *domain.FailedEvent) error
}
