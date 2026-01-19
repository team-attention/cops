package cobra

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewInstallCommand creates the 'install' subcommand.
func (h *DaemonCLIHandler) NewInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the daemon as a system service",
		Long: `Register the cops daemon as a system service.

This command will:
- Verify that the daemon binary exists
- Register the service with the system service manager (launchd on macOS, systemd on Linux)
- Configure the service to start automatically

The daemon binary must be located at ~/.cops/bin/cops-daemon before installation.

Examples:
  cops install    # Install the daemon service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := h.svc.Install(context.Background()); err != nil {
				return err
			}

			fmt.Println("Daemon service installed successfully!")
			fmt.Println("The service will start automatically on system boot.")
			fmt.Println()
			fmt.Println("To start the service now:")
			fmt.Println("  macOS:   launchctl load ~/Library/LaunchAgents/com.cops.daemon.plist")
			fmt.Println("  Linux:   systemctl --user start cops-daemon")

			return nil
		},
	}

	return cmd
}
