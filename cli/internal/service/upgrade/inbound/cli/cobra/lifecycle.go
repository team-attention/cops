package cobra

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/service/upgrade"
)

// UpgradeLifecycle handles automatic update checks during CLI bootstrap.
type UpgradeLifecycle struct {
	logger *slog.Logger
	svc    *upgrade.Service
	cfg    *config.Config
}

// NewUpgradeLifecycle creates a new upgrade lifecycle handler.
func NewUpgradeLifecycle(l *slog.Logger, svc *upgrade.Service, cfg *config.Config) *UpgradeLifecycle {
	return &UpgradeLifecycle{
		logger: l.With(slog.String("name", "upgrade.lifecycle")),
		svc:    svc,
		cfg:    cfg,
	}
}

// OnPreRun checks for updates and prompts user to upgrade if available.
func (lc *UpgradeLifecycle) OnPreRun(ctx context.Context) error {
	// Check for updates using cache
	info, err := lc.svc.CheckUpdateWithCache(ctx)
	if err != nil {
		lc.logger.Debug("update check failed", slog.Any("error", err))
		return nil // Don't block on errors
	}

	if info == nil || !info.HasUpdate {
		return nil
	}

	// Prompt user for upgrade
	fmt.Printf("\n📦 New version available: %s → %s\n", info.CurrentVersion, info.LatestVersion)
	fmt.Print("Do you want to upgrade now? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	if answer == "" || strings.ToLower(answer) == "y" {
		fmt.Println()
		fmt.Printf("Downloading %s...\n", info.LatestVersion)

		result, err := lc.svc.Upgrade(ctx)
		if err != nil {
			fmt.Printf("Upgrade failed: %v\n", err)
			fmt.Println("You can try again later with 'cops upgrade'")
			fmt.Println()
			// Continue with original command
			return nil
		}

		fmt.Printf("✓ Successfully upgraded to %s\n", result.NewVersion)
		if result.DaemonUpgraded {
			fmt.Println("✓ Daemon binary updated (will restart automatically)")
		}
		fmt.Println()
	} else {
		fmt.Println()
	}

	return nil
}
