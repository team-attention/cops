package cobra

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// NewRootCommand creates the root cobra command with logger injection.
func NewRootCommand(l *slog.Logger) *cobra.Command {
	logger := l.With(slog.String("name", "root.command"))

	cmd := &cobra.Command{
		Use:   "cops",
		Short: "A CLI tool for managing code rules",
		Long:  "cops is a CLI tool that helps manage and enforce coding rules across your projects.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Debug("executing command", slog.String("command", cmd.Name()))
		},
	}

	return cmd
}
