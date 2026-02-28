package retry

import (
	"context"
	"log/slog"

	"github.com/bytedance/sonic"

	"github.com/team-attention/cops/api/internal/platform/domain"
	"github.com/team-attention/cops/api/internal/platform/outbound/transcriptsaver"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/retry/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service handles retry processing for failed events.
type Service struct {
	logger          *slog.Logger
	retryRepo       repository.RetryRepositoryPort
	transcriptSaver transcriptsaver.TranscriptSaverPort
	maxRetries      int
	batchSize       int
}

// NewService creates a new retry service.
func NewService(
	l *slog.Logger,
	retryRepo repository.RetryRepositoryPort,
	transcriptSaver transcriptsaver.TranscriptSaverPort,
	cfg *config.Config,
) *Service {
	return &Service{
		logger:          l.With(slog.String("name", "retry.service")),
		retryRepo:       retryRepo,
		transcriptSaver: transcriptSaver,
		maxRetries:      cfg.Retry.MaxRetries,
		batchSize:       cfg.Retry.BatchSize,
	}
}

// ProcessRetryBatch processes a batch of retryable events.
// Returns the number of successfully processed events.
func (s *Service) ProcessRetryBatch(ctx context.Context) (int, error) {
	events, err := s.retryRepo.FindRetryableEvents(ctx, s.maxRetries, s.batchSize)
	if err != nil {
		s.logger.Error("failed to find retryable events",
			slog.Any("error", err),
		)
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	s.logger.Info("processing retry batch",
		slog.Int("eventCount", len(events)),
	)

	successCount := 0
	for _, event := range events {
		if err := s.processEvent(ctx, event); err != nil {
			s.logger.Warn("failed to process event",
				slog.String("eventID", string(event.ID)),
				slog.Any("error", err),
			)
			continue
		}
		successCount++
	}

	s.logger.Info("retry batch completed",
		slog.Int("total", len(events)),
		slog.Int("success", successCount),
	)

	return successCount, nil
}

// processEvent handles a single retry attempt for one event.
func (s *Service) processEvent(ctx context.Context, event *domain.FailedEvent) error {
	updatedEvent, err := s.retryRepo.IncrementRetryCount(ctx, event.ID)
	if err != nil {
		return err
	}

	if updatedEvent == nil {
		return nil
	}

	var transcript shareddomain.Transcript
	if err := sonic.Unmarshal([]byte(event.RawLine), &transcript); err != nil {
		s.handleParseFailure(ctx, updatedEvent, err)
		return nil
	}

	batch := &transcriptsaver.TranscriptBatch{
		Transcripts:    []*shareddomain.Transcript{&transcript},
		ProjectID:      string(event.ProjectID),
		OrganizationID: string(event.OrganizationID),
		UserID:         string(event.UserID),
	}

	if err := s.transcriptSaver.SaveTranscripts(ctx, batch); err != nil {
		s.handleSaveFailure(ctx, updatedEvent, err)
		return nil
	}

	// TODO: Event deletion is temporarily disabled to retain raw event data for analysis.
	//       Re-enable when raw data retention is no longer needed.

	s.logger.Info("successfully retried event",
		slog.String("eventID", string(event.ID)),
		slog.Int("retryCount", updatedEvent.RetryCount),
	)

	return nil
}

// handleParseFailure handles events that fail JSON parsing.
func (s *Service) handleParseFailure(ctx context.Context, event *domain.FailedEvent, parseErr error) {
	reason := "parse error: " + parseErr.Error()

	if event.RetryCount >= s.maxRetries {
		if err := s.retryRepo.MarkPermanentlyFailed(ctx, event.ID, reason); err != nil {
			s.logger.Error("failed to mark event as permanently failed",
				slog.String("eventID", string(event.ID)),
				slog.Any("error", err),
			)
			return
		}
		s.logger.Warn("event permanently failed due to parse error",
			slog.String("eventID", string(event.ID)),
			slog.String("reason", reason),
		)
	}
}

// handleSaveFailure handles events that fail during transcript saving.
func (s *Service) handleSaveFailure(ctx context.Context, event *domain.FailedEvent, saveErr error) {
	reason := "save error: " + saveErr.Error()

	if event.RetryCount >= s.maxRetries {
		if err := s.retryRepo.MarkPermanentlyFailed(ctx, event.ID, reason); err != nil {
			s.logger.Error("failed to mark event as permanently failed",
				slog.String("eventID", string(event.ID)),
				slog.Any("error", err),
			)
			return
		}
		s.logger.Warn("event permanently failed due to save error",
			slog.String("eventID", string(event.ID)),
			slog.String("reason", reason),
		)
	} else {
		s.logger.Warn("retry failed, will retry again",
			slog.String("eventID", string(event.ID)),
			slog.Int("retryCount", event.RetryCount),
			slog.String("reason", reason),
		)
	}
}
