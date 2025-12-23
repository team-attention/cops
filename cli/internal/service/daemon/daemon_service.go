package daemon

import (
	"context"
	"log/slog"
	"os"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer"
)

// Service provides daemon service installation operations.
type Service struct {
	logger    *slog.Logger
	cfg       *config.Config
	installer installer.InstallerPort
}

// NewService creates a new daemon service.
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	installer installer.InstallerPort,
) *Service {
	return &Service{
		logger:    l.With(slog.String("name", "daemon.service")),
		cfg:       cfg,
		installer: installer,
	}
}

// Install registers the daemon as a system service.
// It verifies that the daemon binary exists before attempting installation.
func (s *Service) Install(ctx context.Context) error {
	// Expand tilde in path
	binaryPath, err := pathutil.ExpandPath(s.cfg.Daemon.BinaryPath)
	if err != nil {
		return errutil.BadRequestf("invalid binary path: %v", err)
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return errutil.NotFoundf("daemon binary not found at %s", binaryPath)
	}

	// Check if already installed
	status, err := s.installer.Status()
	if err != nil {
		return errutil.Internalf("failed to check service status: %v", err)
	}

	if status != installer.StatusNotInstalled {
		s.logger.Info("daemon service is already installed",
			slog.String("status", status.String()))
		// TODO: Ask user for reinstall confirmation
		return errutil.BadRequestf("service is already installed (status: %s)", status)
	}

	// Install the service
	if err := s.installer.Install(ctx); err != nil {
		return errutil.Internalf("failed to install daemon service: %v", err)
	}

	s.logger.Info("daemon service installed successfully",
		slog.String("binary", binaryPath))
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
