package container

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/setup"
)

// ipcHandlerParams collects IPC handlers for registration.
type ipcHandlerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Logger    *slog.Logger
	Server    *setup.IPCServer
	Handlers  []IPCHandler `group:"ipc_handlers"`
}

// registerIPCHandlers registers all IPC handlers with the Unix socket server.
func registerIPCHandlers(params ipcHandlerParams) {
	params.Logger.Info("registering IPC handlers",
		slog.Int("count", len(params.Handlers)),
	)

	// Register each handler with the server
	for _, handler := range params.Handlers {
		path, httpHandler := handler.GetHandler()
		params.Server.RegisterHandler(path, httpHandler)
	}

	// Register server lifecycle
	setup.RegisterIPCServerLifecycle(params.Lifecycle, params.Server)
}
