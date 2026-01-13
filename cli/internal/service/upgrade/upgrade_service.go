package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/minio/selfupdate"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/cache"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/github"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/github/client"
)

const (
	cacheTTL         = 24 * time.Hour
	binaryName       = "cops"
	daemonBinaryName = "cops-daemon"
)

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	ReleaseNotes   string
}

// UpgradeResult contains the result of an upgrade operation.
type UpgradeResult struct {
	PreviousVersion string
	NewVersion      string
	DaemonUpgraded  bool
}

// extractedBinaries holds binaries extracted from a release archive.
type extractedBinaries struct {
	CLI    []byte
	Daemon []byte
}

// Service provides upgrade operations.
type Service struct {
	logger *slog.Logger
	cfg    *config.Config
	github github.GitHubPort
	cache  cache.CachePort
}

// NewService creates a new upgrade service.
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	github github.GitHubPort,
	cache cache.CachePort,
) *Service {
	return &Service{
		logger: l.With(slog.String("name", "upgrade.service")),
		cfg:    cfg,
		github: github,
		cache:  cache,
	}
}

// CheckUpdate checks if a new version is available.
func (s *Service) CheckUpdate(ctx context.Context) (*UpdateInfo, error) {
	release, err := s.github.GetLatestRelease(ctx, s.cfg.Upgrade.Owner, s.cfg.Upgrade.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	currentVersion := s.cfg.App.Version
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	hasUpdate, err := s.compareVersions(currentVersion, latestVersion)
	if err != nil {
		s.logger.Debug("version comparison failed, assuming no update",
			slog.String("current", currentVersion),
			slog.String("latest", latestVersion),
			slog.Any("error", err))
		hasUpdate = false
	}

	return &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		ReleaseNotes:   release.Body,
	}, nil
}

// CheckUpdateWithCache checks for updates using cached data when available.
func (s *Service) CheckUpdateWithCache(ctx context.Context) (*UpdateInfo, error) {
	// Check if auto-check is disabled
	if !s.cfg.Upgrade.AutoCheck {
		return nil, nil
	}

	// Skip for dev version
	if s.cfg.App.Version == "dev" {
		return nil, nil
	}

	// Try to get from cache
	cached, err := s.cache.Get(ctx)
	if err != nil {
		s.logger.Debug("failed to read cache", slog.Any("error", err))
	}

	// Return cached data if valid
	if s.cache.IsValid(cached, cacheTTL) {
		return &UpdateInfo{
			CurrentVersion: cached.CurrentVersion,
			LatestVersion:  cached.LatestVersion,
			HasUpdate:      cached.HasUpdate,
		}, nil
	}

	// Fetch fresh data
	info, err := s.CheckUpdate(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	cacheData := &cache.UpdateCheckCache{
		CheckedAt:      time.Now(),
		CurrentVersion: info.CurrentVersion,
		LatestVersion:  info.LatestVersion,
		HasUpdate:      info.HasUpdate,
	}
	if cacheErr := s.cache.Set(ctx, cacheData); cacheErr != nil {
		s.logger.Debug("failed to write cache", slog.Any("error", cacheErr))
	}

	return info, nil
}

// Upgrade downloads and applies the latest version.
func (s *Service) Upgrade(ctx context.Context) (*UpgradeResult, error) {
	// Get latest release
	release, err := s.github.GetLatestRelease(ctx, s.cfg.Upgrade.Owner, s.cfg.Upgrade.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Find asset for current platform
	asset := client.FindAssetForCurrentPlatform(release.Assets, binaryName)
	if asset == nil {
		return nil, fmt.Errorf("no release asset found for current platform")
	}

	s.logger.Info("downloading update",
		slog.String("version", release.TagName),
		slog.String("asset", asset.Name))

	// Download asset
	data, err := s.github.DownloadAsset(ctx, asset.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download asset: %w", err)
	}

	// Extract binaries from archive
	binaries, err := s.extractBinariesFromArchive(data, asset.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to extract binaries: %w", err)
	}

	// Apply CLI update
	if err := selfupdate.Apply(bytes.NewReader(binaries.CLI), selfupdate.Options{}); err != nil {
		return nil, fmt.Errorf("failed to apply CLI update: %w", err)
	}

	// Apply daemon update if binary was found
	daemonUpgraded := false
	if binaries.Daemon != nil {
		if err := s.writeDaemonBinary(binaries.Daemon); err != nil {
			s.logger.Warn("failed to update daemon binary", slog.Any("error", err))
		} else {
			daemonUpgraded = true
			s.logger.Info("daemon binary updated",
				slog.String("path", s.cfg.Daemon.BinaryPath))
		}
	}

	// Clear cache after successful upgrade
	if cacheErr := s.cache.Set(ctx, nil); cacheErr != nil {
		s.logger.Debug("failed to clear cache", slog.Any("error", cacheErr))
	}

	return &UpgradeResult{
		PreviousVersion: s.cfg.App.Version,
		NewVersion:      strings.TrimPrefix(release.TagName, "v"),
		DaemonUpgraded:  daemonUpgraded,
	}, nil
}

// compareVersions compares two semantic versions.
// Returns true if latest is greater than current.
func (s *Service) compareVersions(current, latest string) (bool, error) {
	// Handle dev version
	if current == "dev" {
		return false, nil
	}

	currentVer, err := semver.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("invalid current version %q: %w", current, err)
	}

	latestVer, err := semver.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("invalid latest version %q: %w", latest, err)
	}

	return latestVer.GreaterThan(currentVer), nil
}

// extractBinariesFromArchive extracts CLI and daemon binaries from a tar.gz archive.
func (s *Service) extractBinariesFromArchive(data []byte, assetName string) (*extractedBinaries, error) {
	// Check if it's a gzip archive
	if !strings.HasSuffix(assetName, ".tar.gz") && !strings.HasSuffix(assetName, ".tgz") {
		// Assume it's a raw CLI binary
		return &extractedBinaries{CLI: data}, nil
	}

	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	result := &extractedBinaries{}

	// Find binaries in the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		name := header.Name
		baseName := filepath.Base(name)

		switch baseName {
		case binaryName:
			binary, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read CLI binary from archive: %w", err)
			}
			result.CLI = binary
		case daemonBinaryName:
			binary, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read daemon binary from archive: %w", err)
			}
			result.Daemon = binary
		}
	}

	if result.CLI == nil {
		return nil, fmt.Errorf("CLI binary %q not found in archive", binaryName)
	}

	return result, nil
}

// writeDaemonBinary writes the daemon binary to the configured path.
func (s *Service) writeDaemonBinary(binary []byte) error {
	// Expand ~ in path
	path := s.cfg.Daemon.BinaryPath
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write binary with executable permissions
	if err := os.WriteFile(path, binary, 0755); err != nil {
		return fmt.Errorf("failed to write daemon binary: %w", err)
	}

	return nil
}
