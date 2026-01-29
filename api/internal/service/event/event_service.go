package event

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
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
	Success      bool
	StoredCount  int32
	ErrorMessage string
}

// ParseAndCollectLogsParams contains parameters for ParseAndCollectLogs.
type ParseAndCollectLogsParams struct {
	UserID         string
	Lines          []string
	ProjectID      string
	OrganizationID string
	Version        string
	Provider       string
}

// ParseAndCollectLogs stores raw JSONL lines as Event records for async processing.
// Empty lines are filtered out. Each non-empty line is stored with status "pending".
// Returns the count of events stored (not processed).
func (s *Service) ParseAndCollectLogs(ctx context.Context, params ParseAndCollectLogsParams) (*CollectLogsResult, error) {
	// 1. Validate organizationID is not empty.
	if params.OrganizationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// 2. Validate userID is not empty.
	if params.UserID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// 3. Check RBAC: call s.rbacSvc.CanAccessOrganization(ctx, userID, organizationID).
	canAccess, err := s.rbacSvc.CanAccessOrganization(ctx, params.UserID, params.OrganizationID)
	if err != nil {
		s.logger.Error("failed to check access",
			slog.String("userID", params.UserID),
			slog.String("organizationID", params.OrganizationID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check access"))
	}

	if !canAccess {
		s.logger.Info("access denied to organization",
			slog.String("userID", params.UserID),
			slog.String("organizationID", params.OrganizationID),
		)
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
	}

	// 4. Generate batchID using uuid.NewString().
	batchID := uuid.NewString()

	// 5. Create EventBatch with all fields including version and provider.
	batch := &repository.EventBatch{
		DataLines:      params.Lines,
		Version:        params.Version,
		Provider:       domain.EventProvider(params.Provider),
		ProjectID:      params.ProjectID,
		OrganizationID: params.OrganizationID,
		UserID:         params.UserID,
		BatchID:        batchID,
	}

	// 6. Log info: "storing events" with projectId, organizationId, batchId, lineCount.
	s.logger.Info("storing events",
		slog.String("projectId", params.ProjectID),
		slog.String("organizationId", params.OrganizationID),
		slog.String("batchId", batchID),
		slog.Int("lineCount", len(params.Lines)),
	)

	// 7. Call storedCount, err := s.repo.SaveEventBatch(ctx, batch).
	storedCount, err := s.repo.SaveEventBatch(ctx, batch)

	// 8. Handle repository errors.
	if err != nil {
		if errutil.IsNotFound(err) {
			s.logger.Warn("project not found in organization",
				slog.String("projectId", params.ProjectID),
				slog.String("organizationId", params.OrganizationID),
			)
			return &CollectLogsResult{
				Success:      false,
				ErrorMessage: "project not found in organization",
			}, nil
		}

		s.logger.Error("failed to save events",
			slog.String("projectId", params.ProjectID),
			slog.String("organizationId", params.OrganizationID),
			slog.String("batchId", batchID),
			slog.Any("error", err),
		)
		return &CollectLogsResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// 9. Return CollectLogsResult with success=true and storedCount.
	return &CollectLogsResult{
		Success:     true,
		StoredCount: int32(storedCount),
	}, nil
}
