package installer

import "context"

// InstallerPort defines the interface for system service installation operations.
type InstallerPort interface {
	// Install registers the daemon as a system service.
	Install(ctx context.Context) error

	// Uninstall removes the daemon from system services.
	Uninstall(ctx context.Context) error

	// Status returns the current status of the daemon service.
	Status() (ServiceStatus, error)
}

// ServiceStatus represents the current status of the daemon service.
type ServiceStatus int

const (
	// StatusUnknown indicates the service status cannot be determined.
	StatusUnknown ServiceStatus = iota

	// StatusRunning indicates the service is currently running.
	StatusRunning

	// StatusStopped indicates the service is installed but not running.
	StatusStopped

	// StatusNotInstalled indicates the service is not installed.
	StatusNotInstalled
)

// String returns a human-readable representation of the service status.
func (s ServiceStatus) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	case StatusNotInstalled:
		return "not installed"
	default:
		return "unknown"
	}
}
