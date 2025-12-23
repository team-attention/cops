package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/tracking"
)

// TrackingCLIHandler handles CLI commands for tracking.
type TrackingCLIHandler struct {
	logger *slog.Logger
	svc    *tracking.Service
}

// NewTrackingCLIHandler creates a new CLI handler.
func NewTrackingCLIHandler(l *slog.Logger, svc *tracking.Service) *TrackingCLIHandler {
	return &TrackingCLIHandler{
		logger: l.With(slog.String("name", "tracking.cli.cobra")),
		svc:    svc,
	}
}

// Commands implements CLICommandProvider interface.
func (h *TrackingCLIHandler) Commands() []*cobra.Command {
	return []*cobra.Command{
		h.NewAddCommand(),
		h.NewListCommand(),
	}
}
