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

	rootCmd.AddCommand(startCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
