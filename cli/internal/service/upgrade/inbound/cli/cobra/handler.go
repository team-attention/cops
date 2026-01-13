package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/service/upgrade"
)

// UpgradeCLIHandler handles CLI commands for upgrade operations.
type UpgradeCLIHandler struct {
	logger *slog.Logger
	svc    *upgrade.Service
	cfg    *config.Config
}

// NewUpgradeCLIHandler creates a new CLI handler for upgrade operations.
func NewUpgradeCLIHandler(l *slog.Logger, svc *upgrade.Service, cfg *config.Config) *UpgradeCLIHandler {
	return &UpgradeCLIHandler{
		logger: l.With(slog.String("name", "upgrade.cli.cobra")),
		svc:    svc,
		cfg:    cfg,
	}
}

// Commands implements CLICommandProvider interface.
func (h *UpgradeCLIHandler) Commands() []*cobra.Command {
	return []*cobra.Command{
		h.NewUpgradeCommand(),
	}
}
