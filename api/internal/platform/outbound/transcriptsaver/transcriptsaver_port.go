package transcriptsaver

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// TranscriptBatch represents a batch of transcripts to be saved.
// This is a method input DTO for the SaveTranscripts operation, bundling
// transcripts with their ownership context (project, organization, user).
type TranscriptBatch struct {
	// Transcripts is the list of transcripts to save.
	Transcripts []*domain.Transcript
	// ProjectID is the project these transcripts belong to.
	ProjectID string
	// OrganizationID is the organization these transcripts belong to.
	OrganizationID string
	// UserID is the user who submitted these transcripts.
	UserID string
}

// TranscriptSaverPort defines the interface for saving transcripts to storage.
type TranscriptSaverPort interface {
	// SaveTranscripts saves a batch of transcripts to the events collection.
	// Validates project belongs to organization before saving.
	// Returns errutil.NotFound if project not in organization.
	SaveTranscripts(ctx context.Context, batch *TranscriptBatch) error
}
