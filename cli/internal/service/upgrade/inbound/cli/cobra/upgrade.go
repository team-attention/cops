package cobra

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewUpgradeCommand creates the upgrade command.
func (h *UpgradeCLIHandler) NewUpgradeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade cops to the latest version",
		Long:  "Check for updates and upgrade cops to the latest version from GitHub Releases.",
		RunE:  h.runUpgrade,
	}

	cmd.Flags().Bool("check", false, "Check for updates without installing")

	return cmd
}

func (h *UpgradeCLIHandler) runUpgrade(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	checkOnly, _ := cmd.Flags().GetBool("check")

	if checkOnly {
		return h.checkUpdate(cmd)
	}

	// Check for updates
	info, err := h.svc.CheckUpdate(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !info.HasUpdate {
		fmt.Printf("You are already on the latest version (%s)\n", info.CurrentVersion)
		return nil
	}

	fmt.Printf("Upgrading cops: %s → %s\n", info.CurrentVersion, info.LatestVersion)

	// Perform upgrade
	result, err := h.svc.Upgrade(ctx)
	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	fmt.Printf("✓ Successfully upgraded from %s to %s\n", result.PreviousVersion, result.NewVersion)
	if result.DaemonUpgraded {
		fmt.Println("✓ Daemon binary updated (will restart automatically)")
	}

	if info.ReleaseNotes != "" {
		fmt.Println("\nRelease notes:")
		fmt.Println(info.ReleaseNotes)
	}

	return nil
}

func (h *UpgradeCLIHandler) checkUpdate(cmd *cobra.Command) error {
	ctx := cmd.Context()

	info, err := h.svc.CheckUpdate(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if info.HasUpdate {
		fmt.Printf("New version available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
		fmt.Println("Run 'cops upgrade' to update")
	} else {
		fmt.Printf("You are on the latest version (%s)\n", info.CurrentVersion)
	}

	return nil
}
