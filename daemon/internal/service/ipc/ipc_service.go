package ipc

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// ScanLogsParams contains parameters for ScanLogs operation.
type ScanLogsParams struct {
	ProjectID      string
	ProjectPath    string
	OrganizationID string
}

// ScanLogsResult contains the result of ScanLogs operation.
type ScanLogsResult struct {
	Success bool
	Message string
}

// Service provides IPC operations for CLI-Daemon communication.
type Service struct {
	logger     *slog.Logger
	logWatcher *logwatcher.Service
}

// NewService creates a new IPC service.
func NewService(l *slog.Logger, logWatcher *logwatcher.Service) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "ipc.service")),
		logWatcher: logWatcher,
	}
}

// ScanLogs triggers log file scanning for a specific project.
func (s *Service) ScanLogs(ctx context.Context, params ScanLogsParams) (*ScanLogsResult, error) {
	// Validate params
	if params.ProjectID == "" {
		return nil, errutil.BadRequestf("project_id is required")
	}
	if params.ProjectPath == "" {
		return nil, errutil.BadRequestf("project_path is required")
	}
	if params.OrganizationID == "" {
		return nil, errutil.BadRequestf("organization_id is required")
	}

	s.logger.Info("scan request received",
		slog.String("projectID", params.ProjectID),
		slog.String("projectPath", params.ProjectPath),
		slog.String("organizationID", params.OrganizationID),
	)

	// Call logWatcher.ScanProjectLogs
	err := s.logWatcher.ScanProjectLogs(ctx, logwatcher.ScanProjectLogsParams{
		ProjectID:      shareddomain.ID(params.ProjectID),
		ProjectPath:    params.ProjectPath,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		s.logger.Error("scan failed",
			slog.String("projectID", params.ProjectID),
			slog.Any("error", err),
		)
		return &ScanLogsResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	s.logger.Info("scan completed successfully",
		slog.String("projectID", params.ProjectID),
	)

	return &ScanLogsResult{
		Success: true,
		Message: "scan completed successfully",
	}, nil
}
