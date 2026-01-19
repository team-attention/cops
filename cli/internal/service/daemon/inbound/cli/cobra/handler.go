package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/daemon"
)

// DaemonCLIHandler handles CLI commands for daemon service management.
type DaemonCLIHandler struct {
	logger *slog.Logger
	svc    *daemon.Service
}

// NewDaemonCLIHandler creates a new CLI handler for daemon operations.
func NewDaemonCLIHandler(l *slog.Logger, svc *daemon.Service) *DaemonCLIHandler {
	return &DaemonCLIHandler{
		logger: l.With(slog.String("name", "daemon.cli.cobra")),
		svc:    svc,
	}
}

// Commands implements CLICommandProvider interface.
func (h *DaemonCLIHandler) Commands() []*cobra.Command {
	return []*cobra.Command{
		h.NewInstallCommand(),
		h.NewUninstallCommand(),
	}
}
