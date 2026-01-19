package cobra

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
)

// NewRootCommand creates the root cobra command with logger injection.
func NewRootCommand(l *slog.Logger, cfg *config.Config) *cobra.Command {
	logger := l.With(slog.String("name", "root.command"))

	cmd := &cobra.Command{
		Use:   "cops",
		Short: "A CLI tool for managing code rules",
		Long:  "cops is a CLI tool that helps manage and enforce coding rules across your projects.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logger.Debug("executing command", slog.String("command", cmd.Name()))
		},
	}

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cfg.App.Version)
		},
	}
	cmd.AddCommand(versionCmd)

	return cmd
}
