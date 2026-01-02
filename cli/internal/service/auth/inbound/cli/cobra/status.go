package cobra

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewStatusCommand creates the status command.
func (h *AuthCLIHandler) NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !h.svc.IsLoggedIn() {
				fmt.Println("Not logged in")
				fmt.Println("Run 'cops auth login' to authenticate")
				return nil
			}

			fmt.Println("Logged in")
			return nil
		},
	}
}
