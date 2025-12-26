package cobra

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewListCommand creates the 'list' subcommand.
func (h *TrackingCLIHandler) NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered projects",
		Long: `Display all projects registered for Claude Code session tracking.

For git projects, discovered worktrees will also be shown.

Examples:
  cops list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := h.svc.ListProjects(context.Background())
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				fmt.Println("No projects registered. Use 'cops add' to add a project.")
				return nil
			}

			table := tablewriter.NewTable(os.Stdout)
			table.Header("ID", "Name", "Path", "Git", "Worktrees")

			for _, p := range projects {
				gitStatus := "No"
				if p.IsGitProject {
					gitStatus = "Yes"
				}

				worktreeCount := ""
				if len(p.Worktrees) > 0 {
					worktreeCount = formatWorktrees(p.Worktrees)
				}

				// Truncate ID for display
				idDisplay := p.ID.String()
				if len(idDisplay) > 8 {
					idDisplay = idDisplay[:8] + "..."
				}

				table.Append([]string{
					idDisplay,
					p.Name,
					p.Path,
					gitStatus,
					worktreeCount,
				})
			}

			table.Render()
			return nil
		},
	}

	return cmd
}

// formatWorktrees formats worktree paths for display.
func formatWorktrees(worktrees []string) string {
	if len(worktrees) == 0 {
		return ""
	}

	if len(worktrees) == 1 {
		return "1 worktree"
	}

	// Show count and first worktree name
	names := make([]string, 0, len(worktrees))
	for _, wt := range worktrees {
		// Get the last component of the path
		parts := strings.Split(wt, string(os.PathSeparator))
		if len(parts) > 0 {
			names = append(names, parts[len(parts)-1])
		}
	}

	if len(names) <= 2 {
		return strings.Join(names, ", ")
	}

	return strings.Join(names[:2], ", ") + "..."
}
