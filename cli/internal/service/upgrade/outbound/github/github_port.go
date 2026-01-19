package github

import (
	"context"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string
	Name        string
	Body        string
	PublishedAt string
	Assets      []*Asset
}

// Asset represents a release asset (binary file).
type Asset struct {
	Name        string
	DownloadURL string
	Size        int64
	ContentType string
}

// GitHubPort defines operations for GitHub API interactions.
type GitHubPort interface {
	// GetLatestRelease fetches the latest release from the repository.
	GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error)

	// DownloadAsset downloads a release asset and returns the binary content.
	DownloadAsset(ctx context.Context, downloadURL string) ([]byte, error)
}
