package kardianos

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kardianos/service"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer"
)

// KardianosInstaller implements InstallerPort using github.com/kardianos/service.
// This provides cross-platform service management for Windows, macOS (launchd), and Linux (systemd).
type KardianosInstaller struct {
	logger *slog.Logger
	cfg    *config.Config
}

// NewKardianosInstaller creates a new kardianos-based service installer.
func NewKardianosInstaller(l *slog.Logger, cfg *config.Config) *KardianosInstaller {
	return &KardianosInstaller{
		logger: l.With(slog.String("name", "daemon.installer.kardianos")),
		cfg:    cfg,
	}
}

// Install registers the daemon as a system service.
func (i *KardianosInstaller) Install(ctx context.Context) error {
	svc, err := i.createService()
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Install(); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	i.logger.Info("daemon service installed successfully",
		slog.String("binary", i.cfg.Daemon.BinaryPath))
	return nil
}

// Uninstall removes the daemon from system services.
func (i *KardianosInstaller) Uninstall(ctx context.Context) error {
	svc, err := i.createService()
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	if err := svc.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	i.logger.Info("daemon service uninstalled successfully")
	return nil
}

// Status returns the current status of the daemon service.
func (i *KardianosInstaller) Status() (installer.ServiceStatus, error) {
	svc, err := i.createService()
	if err != nil {
		return installer.StatusUnknown, fmt.Errorf("failed to create service: %w", err)
	}

	status, err := svc.Status()
	if err != nil {
		// If the service is not installed, kardianos returns an error
		return installer.StatusNotInstalled, nil
	}

	switch status {
	case service.StatusRunning:
		return installer.StatusRunning, nil
	case service.StatusStopped:
		return installer.StatusStopped, nil
	default:
		return installer.StatusUnknown, nil
	}
}

// createService creates a kardianos service instance with the appropriate configuration.
func (i *KardianosInstaller) createService() (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "com.cops.daemon",
		DisplayName: "C-Ops Daemon",
		Description: "C-Ops background service for Claude Code session tracking",
		Executable:  i.cfg.Daemon.BinaryPath,
		Arguments:   []string{"start"},
		Option: service.KeyValue{
			"KeepAlive": true,
			"RunAtLoad": true,
		},
	}

	// Pass nil for the Interface since we're only using this for install/uninstall/status operations
	// The actual daemon process handles the service lifecycle
	return service.New(nil, svcConfig)
}
