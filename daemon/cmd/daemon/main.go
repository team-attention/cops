package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/daemon/cmd/internal/container"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cops-daemon",
		Short: "COps background daemon for watching Claude Code logs",
		Long: `COps daemon watches Claude Code log directories and sends log entries
to the API server. It can run as a foreground process or as a launchd service.`,
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		Long:  "Start the daemon to watch Claude Code logs and send them to the API server.",
		Run: func(cmd *cobra.Command, args []string) {
			container.Run()
		},
	}

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install as launchd service",
		Long:  "Generate and install a launchd plist to run the daemon as a background service.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("launchd installation not yet implemented")
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall launchd service",
		Long:  "Remove the launchd plist and stop the daemon service.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("launchd uninstallation not yet implemented")
		},
	}

	rootCmd.AddCommand(startCmd, installCmd, uninstallCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
