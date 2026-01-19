package cobra

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/tracking"
)

// NewRemoveCommand creates the 'remove' subcommand.
func (h *TrackingCLIHandler) NewRemoveCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove [directory]",
		Short: "Remove a project from tracking",
		Long: `Unregister a project directory from Claude Code session tracking.

This command removes the project from:
  - Global configuration (~/.cops/config.json)
  - Local configuration (.cops/ directory in project)

This does NOT delete:
  - Claude Code session logs
  - Project data from the server

Examples:
  cops remove .                # Remove current directory
  cops remove /path/to/project # Remove specific directory
  cops remove . --force        # Remove without confirmation`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// Confirmation prompt unless --force
			if !force {
				fmt.Printf("Remove project at '%s' from tracking?\n", path)
				fmt.Print("This will delete local .cops/ config. Continue? (y/N): ")

				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			params := tracking.RemoveProjectByPathParams{
				Path: path,
			}

			if err := h.svc.RemoveProjectByPath(context.Background(), params); err != nil {
				return err
			}

			fmt.Println("Project removed successfully!")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
