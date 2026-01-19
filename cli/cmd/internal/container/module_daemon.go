package container

import (
	"log/slog"

	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/daemon"
	cliadapter "github.com/team-attention/cops/cli/internal/service/daemon/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer"
	"github.com/team-attention/cops/cli/internal/service/daemon/outbound/ipc"
	connectrpcadapter "github.com/team-attention/cops/cli/internal/service/daemon/outbound/ipc/connectrpc"
	kardianosadapter "github.com/team-attention/cops/cli/internal/service/daemon/outbound/installer/kardianos"
)

// newDaemonModule registers all daemon-related providers.
func newDaemonModule(c *dig.Container) error {
	// Outbound adapter - kardianos service installer
	if err := c.Provide(
		provideKardianosInstaller,
		dig.As(new(installer.InstallerPort)),
	); err != nil {
		return err
	}

	// Outbound adapter - IPC client for CLI-Daemon communication
	if err := c.Provide(
		provideIPCClient,
		dig.As(new(ipc.IPCPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(provideDaemonService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cliadapter.NewDaemonCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}

// provideKardianosInstaller wraps the installer constructor.
func provideKardianosInstaller(l *slog.Logger, paths *pathutil.ExpandedPaths) *kardianosadapter.KardianosInstaller {
	return kardianosadapter.NewKardianosInstaller(l, paths.DaemonBinaryPath)
}

// provideIPCClient wraps the IPC client constructor.
func provideIPCClient(l *slog.Logger, paths *pathutil.ExpandedPaths) *connectrpcadapter.IPCClient {
	return connectrpcadapter.NewIPCClient(l, paths.SocketPath)
}

// provideDaemonService wraps the daemon service constructor.
func provideDaemonService(
	l *slog.Logger,
	paths *pathutil.ExpandedPaths,
	installer installer.InstallerPort,
	ipcClient ipc.IPCPort,
) *daemon.Service {
	return daemon.NewService(l, paths.DaemonBinaryPath, installer, ipcClient)
}
