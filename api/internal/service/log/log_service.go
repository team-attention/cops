package log

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/log/outbound/repository"
)

// CollectLogsResult contains the result of log collection.
type CollectLogsResult struct {
	Success        bool
	ProcessedCount int32
	ErrorMessage   string
}

// Service handles log collection operations.
type Service struct {
	logger *slog.Logger
	repo   repository.SessionRecordRepositoryPort
}

// NewService creates a new log service.
func NewService(l *slog.Logger, repo repository.SessionRecordRepositoryPort) *Service {
	return &Service{
		logger: l.With(slog.String("name", "log.service")),
		repo:   repo,
	}
}

// CollectLogs processes a batch of session records and saves them to storage.
func (s *Service) CollectLogs(ctx context.Context, batch *repository.LogBatch) *CollectLogsResult {
	if len(batch.Records) == 0 {
		return &CollectLogsResult{
			Success:        true,
			ProcessedCount: 0,
		}
	}

	s.logger.Info("collecting log batch",
		slog.String("daemonId", batch.DaemonID),
		slog.Int("recordCount", len(batch.Records)),
	)

	if err := s.repo.SaveBatch(ctx, batch); err != nil {
		s.logger.Error("failed to save log batch",
			slog.String("daemonId", batch.DaemonID),
			slog.Any("error", err),
		)
		return &CollectLogsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}
	}

	return &CollectLogsResult{
		Success:        true,
		ProcessedCount: int32(len(batch.Records)),
	}
}
