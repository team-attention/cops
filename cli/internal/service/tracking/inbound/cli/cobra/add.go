package cobra

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/tracking"
)

// NewAddCommand creates the 'add' subcommand.
func (h *TrackingCLIHandler) NewAddCommand() *cobra.Command {
	var noGit bool

	cmd := &cobra.Command{
		Use:   "add [directory]",
		Short: "Add a project to tracking",
		Long: `Register a project directory for Claude Code session tracking.

This command launches an interactive TUI to configure the project:
1. Git repository detection - finds git repos in parent directories
2. Project name - customize the display name for the project
3. Data sync - choose whether to upload past Claude Code logs

If the directory is a git repository, the main repo path will be registered
(not the worktree path). This ensures the same project ID is used across
all worktrees.

Examples:
  cops add .                # Add current directory (interactive)
  cops add /path/to/project # Add specific directory (interactive)
  cops add . --no-git       # Treat as non-git project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			// Run the TUI
			result, err := runAddTUI(path, noGit)
			if err != nil {
				return err
			}

			// Check if user cancelled
			if result.Cancelled {
				fmt.Println("Operation cancelled.")
				return nil
			}

			// Create params from TUI result
			params := tracking.AddProjectParams{
				Path:  result.ProjectPath,
				Name:  result.ProjectName,
				NoGit: !result.IsGitProject, // NoGit=true means non-git project
				Sync:  result.SyncPastLogs,
			}

			// Call service to register project
			project, err := h.svc.AddProject(context.Background(), params)
			if err != nil {
				return err
			}

			// Display success message
			fmt.Println("\nProject added successfully!")
			fmt.Printf("  ID:   %s\n", project.ID)
			fmt.Printf("  Name: %s\n", project.Name)
			fmt.Printf("  Path: %s\n", project.Path)
			fmt.Printf("  Git:  %t\n", project.IsGitProject)

			if result.SyncPastLogs {
				fmt.Println("  Sync: requested (will sync past logs)")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noGit, "no-git", false, "Treat directory as non-git project")

	return cmd
}
