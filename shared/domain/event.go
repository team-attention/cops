package domain

// EventStatus represents the processing state of an event.
type EventStatus string

const (
	EventStatusPending    EventStatus = "pending"
	EventStatusProcessing EventStatus = "processing"
	EventStatusCompleted  EventStatus = "completed"
	EventStatusFailed     EventStatus = "failed"
)

// EventProvider represents the source provider of the event.
type EventProvider string

const (
	EventProviderClaude   EventProvider = "claude"
	EventProviderGemini   EventProvider = "gemini"
	EventProviderCodex    EventProvider = "codex"
	EventProviderOpenCode EventProvider = "opencode"
)

// Event represents a log event received from the Daemon.
type Event struct {
	ID             ID            `json:"-" bson:"-"`
	Data           any           `json:"data" bson:"data"`
	Version        string        `json:"version" bson:"version"`
	Provider       EventProvider `json:"provider" bson:"provider"`
	ProjectID      ID            `json:"-" bson:"-"`
	OrganizationID ID            `json:"-" bson:"-"`
	UserID         ID            `json:"-" bson:"-"`
	BatchID        string        `json:"batchId" bson:"batchId"`
	RetryCount     int           `json:"retryCount" bson:"retryCount"`
	Status         EventStatus   `json:"status" bson:"status"`
}
