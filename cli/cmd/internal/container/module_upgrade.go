package container

import (
	"log/slog"

	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/upgrade"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/cache"
	filesystemcache "github.com/team-attention/cops/cli/internal/service/upgrade/outbound/cache/filesystem"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/github"
	githubclient "github.com/team-attention/cops/cli/internal/service/upgrade/outbound/github/client"

	cliadapter "github.com/team-attention/cops/cli/internal/service/upgrade/inbound/cli/cobra"
)

// newUpgradeModule registers all upgrade-related providers.
func newUpgradeModule(c *dig.Container) error {
	// Outbound adapter - GitHub client
	if err := c.Provide(
		githubclient.NewGitHubClient,
		dig.As(new(github.GitHubPort)),
	); err != nil {
		return err
	}

	// Outbound adapter - Filesystem cache
	if err := c.Provide(
		provideFilesystemCache,
		dig.As(new(cache.CachePort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(provideUpgradeService); err != nil {
		return err
	}

	// CLI handler (CLICommandProvider)
	if err := c.Provide(
		cliadapter.NewUpgradeCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	); err != nil {
		return err
	}

	// Lifecycle handler (CLILifecycle)
	return c.Provide(
		cliadapter.NewUpgradeLifecycle,
		dig.As(new(CLILifecycle)),
		dig.Group("cli_lifecycles"),
	)
}

// provideFilesystemCache wraps the cache adapter constructor.
func provideFilesystemCache(l *slog.Logger, paths *pathutil.ExpandedPaths) (*filesystemcache.FilesystemCache, error) {
	return filesystemcache.NewFilesystemCache(l, paths.CacheDir)
}

// provideUpgradeService wraps the upgrade service constructor.
func provideUpgradeService(
	l *slog.Logger,
	cfg *config.Config,
	paths *pathutil.ExpandedPaths,
	github github.GitHubPort,
	cache cache.CachePort,
) *upgrade.Service {
	return upgrade.NewService(l, cfg, paths.DaemonBinaryPath, github, cache)
}
