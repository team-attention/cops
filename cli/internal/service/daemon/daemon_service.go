package daemon

import (
	"context"
	"log/slog"
	"os"

	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/ipc"
)

// Service provides daemon service installation operations.
type Service struct {
	logger     *slog.Logger
	binaryPath string
	installer  installer.InstallerPort
	ipcClient  ipc.IPCPort
}

// NewService creates a new daemon service.
func NewService(
	l *slog.Logger,
	binaryPath string,
	installer installer.InstallerPort,
	ipcClient ipc.IPCPort,
) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "daemon.service")),
		binaryPath: binaryPath,
		installer:  installer,
		ipcClient:  ipcClient,
	}
}

// Install registers the daemon as a system service.
// It verifies that the daemon binary exists before attempting installation.
func (s *Service) Install(ctx context.Context) error {
	// Check if binary exists (binaryPath is already expanded)
	if _, err := os.Stat(s.binaryPath); os.IsNotExist(err) {
		return errutil.NotFoundf("daemon binary not found at %s", s.binaryPath)
	}

	// Check if already installed
	status, err := s.installer.Status()
	if err != nil {
		return errutil.Internalf("failed to check service status: %v", err)
	}

	if status == installer.StatusNotInstalled {
		// Install the service
		if err := s.installer.Install(ctx); err != nil {
			return errutil.Internalf("failed to install daemon service: %v", err)
		}

		// Start the service after installation
		if err := s.installer.Start(ctx); err != nil {
			return errutil.Internalf("failed to start daemon service: %v", err)
		}

		s.logger.Info("daemon service installed and started",
			slog.String("binary", s.binaryPath))
	} else {
		// Already installed - restart
		if err := s.installer.Restart(ctx); err != nil {
			return errutil.Internalf("failed to restart daemon service: %v", err)
		}

		s.logger.Info("daemon service restarted",
			slog.String("binary", s.binaryPath))
	}

	return nil
}

// Uninstall removes the daemon from system services.
func (s *Service) Uninstall(ctx context.Context) error {
	// Check if service is installed
	status, err := s.installer.Status()
	if err != nil {
		return errutil.Internalf("failed to check service status: %v", err)
	}

	if status == installer.StatusNotInstalled {
		return errutil.NotFoundf("daemon service is not installed")
	}

	// Uninstall the service
	if err := s.installer.Uninstall(ctx); err != nil {
		return errutil.Internalf("failed to uninstall daemon service: %v", err)
	}

	s.logger.Info("daemon service uninstalled successfully")
	return nil
}

// Status returns the current status of the daemon service.
func (s *Service) Status(ctx context.Context) (installer.ServiceStatus, error) {
	status, err := s.installer.Status()
	if err != nil {
		return installer.StatusUnknown, errutil.Internalf("failed to get service status: %v", err)
	}

	return status, nil
}

// RequestLogScanParams contains parameters for RequestLogScan.
type RequestLogScanParams struct {
	ProjectID      string
	ProjectPath    string
	OrganizationID string
}

// RequestLogScan requests the daemon to scan logs for a project.
// Returns error if daemon is not running.
func (s *Service) RequestLogScan(ctx context.Context, params RequestLogScanParams) error {
	// Check if daemon is available
	if !s.ipcClient.IsAvailable(ctx) {
		return errutil.NotFoundf("daemon is not running")
	}

	// Request log scan
	result, err := s.ipcClient.ScanLogs(ctx, ipc.ScanLogsParams{
		ProjectID:      params.ProjectID,
		ProjectPath:    params.ProjectPath,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		return errutil.Internalf("failed to request log scan: %v", err)
	}

	if !result.Success {
		return errutil.Internalf("log scan failed: %s", result.Message)
	}

	s.logger.Info("log scan request completed",
		slog.String("projectID", params.ProjectID),
		slog.String("message", result.Message),
	)

	return nil
}
