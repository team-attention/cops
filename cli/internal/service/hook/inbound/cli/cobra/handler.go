package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/hook"
)

// HookCLIHandler handles CLI commands for hook operations.
type HookCLIHandler struct {
	logger *slog.Logger
	svc    *hook.Service
}

// NewHookCLIHandler creates a new CLI handler for hook commands.
func NewHookCLIHandler(l *slog.Logger, svc *hook.Service) *HookCLIHandler {
	return &HookCLIHandler{
		logger: l.With(slog.String("name", "hook.cli.cobra")),
		svc:    svc,
	}
}

// Commands implements CLICommandProvider interface.
// Returns the "hook" parent command with subcommands.
func (h *HookCLIHandler) Commands() []*cobra.Command {
	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Hook event management commands",
		// 1. Create parent "hook" command
		// 2. No Run function (parent command only)
	}

	// Add subcommands
	hookCmd.AddCommand(h.NewPostCommand())

	return []*cobra.Command{hookCmd}
}
