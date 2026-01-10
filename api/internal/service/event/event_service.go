package event

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
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

// CollectLogs processes a batch of log records and saves them to events collection.
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
		ProcessedCount: int32(len(batch.Records)),
	}, nil
}
