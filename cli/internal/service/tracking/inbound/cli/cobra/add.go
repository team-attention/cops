package cobra

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/tracking"
)

// NewAddCommand creates the 'add' subcommand.
func (h *TrackingCLIHandler) NewAddCommand() *cobra.Command {
	var sync bool
	var noGit bool

	cmd := &cobra.Command{
		Use:   "add [directory]",
		Short: "Add a project to tracking",
		Long: `Register a project directory for Claude Code session tracking.

If the directory is a git repository, the main repo path will be registered
(not the worktree path). This ensures the same project ID is used across
all worktrees.

Examples:
  cops add .                # Add current directory
  cops add /path/to/project # Add specific directory
  cops add . --sync         # Add and sync past records
  cops add . --no-git       # Treat as non-git project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			params := tracking.AddProjectParams{
				Path:  path,
				NoGit: noGit,
				Sync:  sync,
			}

			project, err := h.svc.AddProject(context.Background(), params)
			if err != nil {
				return err
			}

			fmt.Println("Project added successfully!")
			fmt.Printf("  ID:   %s\n", project.ID)
			fmt.Printf("  Path: %s\n", project.Path)
			fmt.Printf("  Git:  %t\n", project.GitProject)

			if sync {
				fmt.Println("  Sync: completed")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&sync, "sync", "s", false, "Sync past records immediately")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "Treat directory as non-git project")

	return cmd
}
