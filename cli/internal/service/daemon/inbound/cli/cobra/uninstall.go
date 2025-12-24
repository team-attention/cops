package cobra

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewUninstallCommand creates the 'uninstall' subcommand.
func (h *DaemonCLIHandler) NewUninstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the daemon service",
		Long: `Remove the cops daemon from system services.

This command will:
- Stop the running daemon service (if running)
- Remove the service registration from the system service manager
- Remove auto-start configuration

The daemon binary will not be deleted; only the service registration is removed.

Examples:
  cops uninstall    # Uninstall the daemon service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := h.svc.Uninstall(context.Background()); err != nil {
				return err
			}

			fmt.Println("Daemon service uninstalled successfully!")
			fmt.Println("The service has been removed from the system.")

			return nil
		},
	}

	return cmd
}
