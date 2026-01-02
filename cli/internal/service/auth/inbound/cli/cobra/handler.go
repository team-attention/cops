package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/auth"
)

// AuthCLIHandler handles auth CLI commands.
type AuthCLIHandler struct {
	logger *slog.Logger
	svc    *auth.Service
}

// NewAuthCLIHandler creates a new auth CLI handler.
func NewAuthCLIHandler(l *slog.Logger, svc *auth.Service) *AuthCLIHandler {
	return &AuthCLIHandler{
		logger: l.With(slog.String("name", "auth.cli.cobra")),
		svc:    svc,
	}
}

// Commands implements CLICommandProvider interface.
func (h *AuthCLIHandler) Commands() []*cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	authCmd.AddCommand(
		h.NewLoginCommand(),
		h.NewLogoutCommand(),
		h.NewStatusCommand(),
	)

	return []*cobra.Command{authCmd}
}
