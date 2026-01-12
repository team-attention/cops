package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/auth"
)

// AuthCLIHandler handles CLI commands for authentication.
type AuthCLIHandler struct {
	logger *slog.Logger
	svc    *auth.Service
}

// NewAuthCLIHandler creates a new CLI handler for auth operations.
func NewAuthCLIHandler(l *slog.Logger, svc *auth.Service) *AuthCLIHandler {
	return &AuthCLIHandler{
		logger: l.With(slog.String("name", "auth.cli.cobra")),
		svc:    svc,
	}
}

// Commands implements CLICommandProvider interface.
func (h *AuthCLIHandler) Commands() []*cobra.Command {
	// Create "auth" parent command (no action, just grouping)
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  `Manage authentication for the cops CLI.`,
	}

	// Add "login" subcommand
	authCmd.AddCommand(h.NewLoginCommand())

	return []*cobra.Command{authCmd}
}
