package domain

import shareddomain "github.com/team-attention/cops/shared/domain"

// FailedEventStatus represents the status of a failed event.
type FailedEventStatus string

const (
	// FailedEventStatusFailed indicates the event failed and is eligible for retry.
	FailedEventStatusFailed FailedEventStatus = "failed"
	// FailedEventStatusPermanentlyFailed indicates the event exceeded max retries.
	FailedEventStatusPermanentlyFailed FailedEventStatus = "permanently_failed"
)

// FailedEvent represents an event that failed during processing and is eligible for retry.
// ID fields use bson:"-" as they are handled separately in mongoschema.
// Non-ID fields include bson tags for inline embedding.
type FailedEvent struct {
	// ID is the unique identifier for this failed event record.
	ID shareddomain.ID `json:"id,omitempty" bson:"-"`
	// Status indicates whether the event is retryable or permanently failed.
	Status FailedEventStatus `json:"status" bson:"status"`
	// RetryCount tracks how many times this event has been retried.
	RetryCount int `json:"retryCount" bson:"retryCount"`
	// FailureReason describes why the event failed.
	FailureReason string `json:"failureReason" bson:"failureReason"`
	// RawLine contains the original JSONL line that failed to process.
	RawLine string `json:"rawLine" bson:"rawLine"`
	// ProjectID identifies the project this event belongs to.
	ProjectID shareddomain.ID `json:"projectId" bson:"-"`
	// OrganizationID identifies the organization this event belongs to.
	OrganizationID shareddomain.ID `json:"organizationId" bson:"-"`
	// UserID identifies the user who submitted this event.
	UserID shareddomain.ID `json:"userId" bson:"-"`
}
