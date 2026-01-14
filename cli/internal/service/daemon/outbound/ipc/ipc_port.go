package ipc

import "context"

// ScanLogsParams contains parameters for requesting log scan.
type ScanLogsParams struct {
	ProjectID      string
	ProjectPath    string
	OrganizationID string
}

// ScanLogsResult contains the result of a scan request.
type ScanLogsResult struct {
	Success bool
	Message string
}

// IPCPort defines the interface for CLI-Daemon IPC communication.
type IPCPort interface {
	// ScanLogs requests the daemon to scan logs for a specific project.
	ScanLogs(ctx context.Context, params ScanLogsParams) (*ScanLogsResult, error)

	// IsAvailable checks if the daemon is running and accessible.
	IsAvailable(ctx context.Context) bool
}
