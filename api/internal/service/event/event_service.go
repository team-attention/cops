package event

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/bytedance/sonic"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service handles event collection operations.
type Service struct {
	logger  *slog.Logger
	repo    repository.EventRepositoryPort
	rbacSvc *rbac.Service
}

// NewService creates a new event service.
func NewService(l *slog.Logger, repo repository.EventRepositoryPort, rbacSvc *rbac.Service) *Service {
	return &Service{
		logger:  l.With(slog.String("name", "event.service")),
		repo:    repo,
		rbacSvc: rbacSvc,
	}
}

// CollectLogsResult contains the result of log collection.
type CollectLogsResult struct {
	Success        bool
	ProcessedCount int32
	ErrorMessage   string
}

// ParseAndCollectLogs parses raw JSONL lines and saves them to events collection.
// Returns parse errors (for logging) and collection result.
func (s *Service) ParseAndCollectLogs(ctx context.Context, userID string, lines []string, projectID, organizationID string) (parseErrors []error, result *CollectLogsResult, err error) {
	if organizationID == "" {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	if userID == "" {
		return nil, nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	canAccess, err := s.rbacSvc.CanAccessOrganization(ctx, userID, organizationID)
	if err != nil {
		s.logger.Error("failed to check access",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check access"))
	}

	if !canAccess {
		s.logger.Info("access denied to organization",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
		)
		return nil, nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
	}

	// Parse JSONL lines into Transcript objects
	transcripts, parseErrors := s.parseJSONLLines(lines)

	// Log parse errors at ERROR level (Fire & Forget)
	if len(parseErrors) > 0 {
		s.logger.Error("failed to parse some JSONL lines",
			slog.String("projectId", projectID),
			slog.String("organizationId", organizationID),
			slog.Int("failedCount", len(parseErrors)),
			slog.Int("totalCount", len(lines)),
			slog.String("sampleError", parseErrors[0].Error()),
		)
	}

	if len(transcripts) == 0 {
		return parseErrors, &CollectLogsResult{
			Success:        true,
			ProcessedCount: 0,
		}, nil
	}

	batch := &repository.LogBatch{
		Transcripts:    transcripts,
		ProjectID:      projectID,
		OrganizationID: organizationID,
	}

	s.logger.Info("collecting log batch",
		slog.String("projectId", projectID),
		slog.String("organizationId", organizationID),
		slog.Int("transcriptCount", len(transcripts)),
	)

	if err := s.repo.SaveLogBatch(ctx, batch); err != nil {
		if errutil.IsNotFound(err) {
			s.logger.Warn("project not found in organization",
				slog.String("projectId", projectID),
				slog.String("organizationId", organizationID),
			)
			return parseErrors, &CollectLogsResult{
				Success:      false,
				ErrorMessage: "project not found in organization",
			}, nil
		}

		s.logger.Error("failed to save log batch",
			slog.String("projectId", projectID),
			slog.String("organizationId", organizationID),
			slog.Any("error", err),
		)
		return parseErrors, &CollectLogsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return parseErrors, &CollectLogsResult{
		Success:        true,
		ProcessedCount: int32(len(transcripts)),
	}, nil
}

// parseJSONLLines parses raw JSONL lines into Transcript domain objects.
// Returns the parsed transcripts and any parse errors encountered.
func (s *Service) parseJSONLLines(lines []string) ([]*shareddomain.Transcript, []error) {
	var transcripts []*shareddomain.Transcript
	var parseErrors []error

	for _, line := range lines {
		if line == "" {
			continue
		}

		var transcript shareddomain.Transcript
		if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("parse error: %s (line: %.100s...)", err.Error(), line))
			continue
		}

		transcripts = append(transcripts, &transcript)
	}

	return transcripts, parseErrors
}

// CollectLogs processes a batch of log transcripts and saves them to events collection.
func (s *Service) CollectLogs(ctx context.Context, userID string, batch *repository.LogBatch) (*CollectLogsResult, error) {
	if batch.OrganizationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	canAccess, err := s.rbacSvc.CanAccessOrganization(ctx, userID, batch.OrganizationID)
	if err != nil {
		s.logger.Error("failed to check access",
			slog.String("userID", userID),
			slog.String("organizationID", batch.OrganizationID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check access"))
	}

	if !canAccess {
		s.logger.Info("access denied to organization",
			slog.String("userID", userID),
			slog.String("organizationID", batch.OrganizationID),
		)
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
	}

	if len(batch.Transcripts) == 0 {
		return &CollectLogsResult{
			Success:        true,
			ProcessedCount: 0,
		}, nil
	}

	s.logger.Info("collecting log batch",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
		slog.Int("transcriptCount", len(batch.Transcripts)),
	)

	if err := s.repo.SaveLogBatch(ctx, batch); err != nil {
		if errutil.IsNotFound(err) {
			s.logger.Warn("project not found in organization",
				slog.String("projectId", batch.ProjectID),
				slog.String("organizationId", batch.OrganizationID),
			)
			return &CollectLogsResult{
				Success:      false,
				ErrorMessage: "project not found in organization",
			}, nil
		}

		s.logger.Error("failed to save log batch",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Any("error", err),
		)
		return &CollectLogsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &CollectLogsResult{
		Success:        true,
		ProcessedCount: int32(len(batch.Transcripts)),
	}, nil
}
