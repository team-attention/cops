package aggregation

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	"github.com/team-attention/cops/api/internal/service/core/rbac"
)

// Service handles log collection operations.
// Injects RBAC service for authorization checks.
type Service struct {
	logger  *slog.Logger
	repo    repository.SessionRecordRepositoryPort
	rbacSvc *rbac.Service
}

// NewService creates a new aggregation service.
func NewService(l *slog.Logger, repo repository.SessionRecordRepositoryPort, rbacSvc *rbac.Service) *Service {
	return &Service{
		logger:  l.With(slog.String("name", "aggregation.service")),
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

// CollectLogs processes a batch of session records and saves them to storage.
// Validates RBAC at the start, then validates project belongs to organization.
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

	if len(batch.Records) == 0 {
		return &CollectLogsResult{
			Success:        true,
			ProcessedCount: 0,
		}, nil
	}

	s.logger.Info("collecting log batch",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
		slog.Int("recordCount", len(batch.Records)),
	)

	if err := s.repo.SaveBatch(ctx, batch); err != nil {
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
		ProcessedCount: int32(len(batch.Records)),
	}, nil
}
