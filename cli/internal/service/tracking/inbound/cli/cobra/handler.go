package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/cli/internal/service/user"
)

// TrackingCLIHandler handles CLI commands for tracking.
type TrackingCLIHandler struct {
	logger  *slog.Logger
	svc     *tracking.Service
	userSvc *user.Service
}

// NewTrackingCLIHandler creates a new CLI handler.
func NewTrackingCLIHandler(
	l *slog.Logger,
	svc *tracking.Service,
	userSvc *user.Service,
) *TrackingCLIHandler {
	return &TrackingCLIHandler{
		logger:  l.With(slog.String("name", "tracking.cli.cobra")),
		svc:     svc,
		userSvc: userSvc,
	}
}

// Commands implements CLICommandProvider interface.
func (h *TrackingCLIHandler) Commands() []*cobra.Command {
	return []*cobra.Command{
		h.NewAddCommand(),
		h.NewListCommand(),
		h.NewRemoveCommand(),
	}
}
