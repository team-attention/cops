package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	session "github.com/team-attention/cops/shared/domain/v2"
	"github.com/team-attention/cops/shared/domain/v2/claudecodeadapter"
	"github.com/team-attention/cops/shared/domain/v2/geminicliadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Service handles event processing and session conversion.
type Service struct {
	logger          *slog.Logger
	cfg             *config.Config
	eventQuery      repository.EventQueryPort
	sessionRepo     repository.SessionRepositoryPort
	resumeTokenRepo repository.ResumeTokenRepositoryPort
	claudeAdapter   *claudecodeadapter.Adapter
	geminiAdapter   *geminicliadapter.Adapter

	// Runtime state
	mu         sync.Mutex
	iterator   repository.ChangeEventIterator
	cancelFunc context.CancelFunc
}

// NewService creates a new Session service.
// Adapters are created inline since they have no dependencies (per go-container.md).
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	eventQuery repository.EventQueryPort,
	sessionRepo repository.SessionRepositoryPort,
	resumeTokenRepo repository.ResumeTokenRepositoryPort,
) *Service {
	return &Service{
		logger:          l.With(slog.String("name", "session.service")),
		cfg:             cfg,
		eventQuery:      eventQuery,
		sessionRepo:     sessionRepo,
		resumeTokenRepo: resumeTokenRepo,
		claudeAdapter:   claudecodeadapter.NewAdapter(),
		geminiAdapter:   geminicliadapter.NewAdapter(),
	}
}

// Start begins watching the events collection for new inserts.
// Called by fx lifecycle OnStart hook.
func (s *Service) Start(ctx context.Context) error {
	// Use context.Background() for the long-running goroutine.
	// The fx OnStart ctx may be cancelled after hook completion.
	cancelCtx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel

	token, err := s.resumeTokenRepo.GetResumeToken(ctx, repository.ResumeTokenKey)
	if err != nil {
		return fmt.Errorf("failed to load resume token: %w", err)
	}

	iterator, err := s.eventQuery.WatchInserts(cancelCtx, token)
	if err != nil {
		return fmt.Errorf("failed to open change stream: %w", err)
	}

	s.mu.Lock()
	s.iterator = iterator
	s.mu.Unlock()

	go s.processChangeStream(cancelCtx, iterator)

	s.logger.Info("session service started")

	return nil
}

// Stop closes the change stream and stops processing.
// Called by fx lifecycle OnStop hook.
func (s *Service) Stop(ctx context.Context) error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.iterator != nil {
		if err := s.iterator.Close(ctx); err != nil {
			s.logger.Warn("error closing change stream iterator", slog.Any("error", err))
		}
		s.iterator = nil
	}

	s.logger.Info("session service stopped")

	return nil
}

// processChangeStream continuously reads from the change stream iterator.
// Runs in a goroutine started by Start().
// Uses ChangeEventIterator interface to maintain port abstraction.
func (s *Service) processChangeStream(ctx context.Context, it repository.ChangeEventIterator) {
	defer s.logger.Info("change stream processing stopped")

	for {
		if !it.Next(ctx) {
			if err := it.Err(); err != nil {
				if ctx.Err() != nil {
					// Context cancelled, normal shutdown
					break
				}
				s.logger.Error("change stream error", slog.Any("error", err))
			}
			break
		}

		var changeEvent bson.M
		if err := it.Decode(&changeEvent); err != nil {
			s.logger.Error("failed to decode change event", slog.Any("error", err))
			continue
		}

		fullDocument, ok := changeEvent["fullDocument"].(bson.M)
		if !ok {
			s.logger.Warn("fullDocument not found in change event")
			continue
		}

		batchID, _ := fullDocument[mongoschema.EventBatchIDField].(string)
		if batchID == "" {
			s.logger.Warn("batchID is empty in change event")
			continue
		}

		if err := s.processBatch(ctx, batchID); err != nil {
			s.logger.Error("failed to process batch",
				slog.String("batchID", batchID),
				slog.Any("error", err),
			)
		}

		token := it.ResumeToken()
		if err := s.resumeTokenRepo.SaveResumeToken(ctx, repository.ResumeTokenKey, token); err != nil {
			s.logger.Warn("failed to save resume token", slog.Any("error", err))
		}
	}
}

// processBatch processes all events with the given batchID.
func (s *Service) processBatch(ctx context.Context, batchID string) error {
	events, err := s.eventQuery.FindByBatchID(ctx, batchID, s.cfg.Retry.MaxRetries)
	if err != nil {
		return fmt.Errorf("failed to find events: %w", err)
	}

	if len(events) == 0 {
		s.logger.Warn("no events found for batch", slog.String("batchID", batchID))
		return nil
	}

	// Group events by (projectID, userID) key
	type groupKey struct {
		ProjectID domain.ID
		UserID    domain.ID
	}
	groups := make(map[groupKey][]*domain.Event)
	for _, event := range events {
		key := groupKey{ProjectID: event.ProjectID, UserID: event.UserID}
		groups[key] = append(groups[key], event)
	}

	var successfulIDs []domain.ID

	for key, groupEvents := range groups {
		sessions, failedIDs, err := s.convertEventsToSessions(groupEvents)
		if err != nil {
			s.logger.Error("failed to convert events",
				slog.String("batchID", batchID),
				slog.String("projectID", string(key.ProjectID)),
				slog.Any("error", err),
			)
			for _, failedID := range failedIDs {
				if incErr := s.eventQuery.IncrementRetryCount(ctx, failedID); incErr != nil {
					s.logger.Warn("failed to increment retry count",
						slog.String("eventID", string(failedID)),
						slog.Any("error", incErr),
					)
				}
			}
			continue
		}

		// Increment retry count for any failed events
		for _, failedID := range failedIDs {
			if incErr := s.eventQuery.IncrementRetryCount(ctx, failedID); incErr != nil {
				s.logger.Warn("failed to increment retry count",
					slog.String("eventID", string(failedID)),
					slog.Any("error", incErr),
				)
			}
		}

		if len(sessions) > 0 {
			if saveErr := s.sessionRepo.SaveBatch(ctx, key.ProjectID, key.UserID, sessions); saveErr != nil {
				s.logger.Error("failed to save sessions",
					slog.String("batchID", batchID),
					slog.String("projectID", string(key.ProjectID)),
					slog.Any("error", saveErr),
				)
				continue
			}
		}

		// Collect successfully processed event IDs (excluding failed ones)
		failedSet := make(map[domain.ID]bool)
		for _, id := range failedIDs {
			failedSet[id] = true
		}
		for _, event := range groupEvents {
			if !failedSet[event.ID] {
				successfulIDs = append(successfulIDs, event.ID)
			}
		}
	}

	// TODO: Event deletion is temporarily disabled to retain raw event data for analysis.
	//       Re-enable when raw data retention is no longer needed.
	if len(successfulIDs) > 0 {
		s.logger.Debug("event deletion disabled, retaining processed events",
			slog.Int("count", len(successfulIDs)),
		)
	}

	s.logger.Info("batch processed",
		slog.String("batchID", batchID),
		slog.Int("processedCount", len(successfulIDs)),
		slog.Int("totalEvents", len(events)),
	)

	return nil
}

// convertEventsToSessions converts event data to Session records.
func (s *Service) convertEventsToSessions(events []*domain.Event) ([]*session.Session, []domain.ID, error) {
	var sessions []*session.Session
	var failedIDs []domain.ID

	for _, event := range events {
		provider := DetectProvider(event.Data)

		switch provider {
		case ProviderClaudeCode:
			converted, err := s.convertClaudeCodeEvent(event.Data)
			if err != nil {
				s.logger.Warn("failed to convert Claude Code event",
					slog.String("eventID", string(event.ID)),
					slog.Any("error", err),
				)
				failedIDs = append(failedIDs, event.ID)
				continue
			}
			sessions = append(sessions, converted...)

		case ProviderGeminiCLI:
			converted, err := s.convertGeminiCLIEvent(event.Data)
			if err != nil {
				s.logger.Warn("failed to convert Gemini CLI event",
					slog.String("eventID", string(event.ID)),
					slog.Any("error", err),
				)
				failedIDs = append(failedIDs, event.ID)
				continue
			}
			sessions = append(sessions, converted...)

		case ProviderUnknown:
			s.logger.Warn("unknown provider for event", slog.String("eventID", string(event.ID)))
			failedIDs = append(failedIDs, event.ID)
		}
	}

	return sessions, failedIDs, nil
}

// convertClaudeCodeEvent converts a single Claude Code event data to sessions.
func (s *Service) convertClaudeCodeEvent(data any) ([]*session.Session, error) {
	// Re-marshal and unmarshal to convert map[string]any to domain.Transcript
	jsonBytes, err := sonic.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var transcript domain.Transcript
	if err := sonic.Unmarshal(jsonBytes, &transcript); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Claude Code transcript: %w", err)
	}

	return s.claudeAdapter.Adapt(&transcript)
}

// convertGeminiCLIEvent converts a Gemini CLI event data to sessions.
func (s *Service) convertGeminiCLIEvent(data any) ([]*session.Session, error) {
	// Re-marshal and unmarshal to convert map[string]any to GeminiSession
	jsonBytes, err := sonic.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var geminiSession geminicliadapter.GeminiSession
	if err := sonic.Unmarshal(jsonBytes, &geminiSession); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Gemini CLI session: %w", err)
	}

	return s.geminiAdapter.AdaptSession(&geminiSession)
}
