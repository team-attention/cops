package cobra

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewLogoutCommand creates the logout command.
func (h *AuthCLIHandler) NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out and remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := h.svc.Logout(cmd.Context()); err != nil {
				return fmt.Errorf("failed to logout: %w", err)
			}

			fmt.Println("Logged out successfully")
			return nil
		},
	}
}
