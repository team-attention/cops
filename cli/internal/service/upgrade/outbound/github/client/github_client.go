package client

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"

	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/github"
)

// githubReleaseResponse represents the GitHub API response for releases.
type githubReleaseResponse struct {
	TagName     string                `json:"tag_name"`
	Name        string                `json:"name"`
	Body        string                `json:"body"`
	PublishedAt string                `json:"published_at"`
	Assets      []*githubAssetResponse `json:"assets"`
}

// githubAssetResponse represents a release asset from GitHub API.
type githubAssetResponse struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// GitHubClient implements GitHubPort using imroc/req/v3.
type GitHubClient struct {
	logger *slog.Logger
	client *httpclient.GitHubHTTPClient
}

// NewGitHubClient creates a new GitHub API client.
func NewGitHubClient(l *slog.Logger, client *httpclient.GitHubHTTPClient) *GitHubClient {
	return &GitHubClient{
		logger: l.With(slog.String("name", "upgrade.github.client")),
		client: client,
	}
}

// GetLatestRelease fetches the latest release from the repository.
func (c *GitHubClient) GetLatestRelease(ctx context.Context, owner, repo string) (*github.Release, error) {
	url := fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)

	var resp githubReleaseResponse
	response, err := c.client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/vnd.github.v3+json").
		SetSuccessResult(&resp).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	if !response.IsSuccessState() {
		return nil, fmt.Errorf("GitHub API returned status %d", response.StatusCode)
	}

	assets := make([]*github.Asset, 0, len(resp.Assets))
	for _, a := range resp.Assets {
		assets = append(assets, &github.Asset{
			Name:        a.Name,
			DownloadURL: a.BrowserDownloadURL,
			Size:        a.Size,
			ContentType: a.ContentType,
		})
	}

	return &github.Release{
		TagName:     resp.TagName,
		Name:        resp.Name,
		Body:        resp.Body,
		PublishedAt: resp.PublishedAt,
		Assets:      assets,
	}, nil
}

// DownloadAsset downloads a release asset and returns the binary content.
func (c *GitHubClient) DownloadAsset(ctx context.Context, downloadURL string) ([]byte, error) {
	response, err := c.client.R().
		SetContext(ctx).
		Get(downloadURL)

	if err != nil {
		return nil, fmt.Errorf("failed to download asset: %w", err)
	}

	if !response.IsSuccessState() {
		return nil, fmt.Errorf("download returned status %d", response.StatusCode)
	}

	return response.Bytes(), nil
}

// FindAssetForCurrentPlatform finds the appropriate asset for the current OS/Arch.
// It looks for assets matching the pattern: {name}_{version}_{os}_{arch}.tar.gz
func FindAssetForCurrentPlatform(assets []*github.Asset, binaryName, version string) *github.Asset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Strip 'v' prefix from version if present (v0.1.0 -> 0.1.0)
	version = strings.TrimPrefix(version, "v")

	// Expected patterns:
	// - cops_0.1.0_darwin_amd64.tar.gz
	// - cops_0.1.0_darwin_arm64.tar.gz
	// - cops_0.1.0_linux_amd64.tar.gz
	expectedPattern := fmt.Sprintf("%s_%s_%s_%s", binaryName, version, goos, goarch)

	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasPrefix(name, expectedPattern) {
			return asset
		}
	}

	return nil
}

var _ github.GitHubPort = (*GitHubClient)(nil)
