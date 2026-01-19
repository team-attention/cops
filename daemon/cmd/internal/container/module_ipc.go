package container

import (
	"net/http"

	"connectrpc.com/connect"
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/service/ipc"
	connectrpchandler "github.com/team-attention/cops/daemon/internal/service/ipc/inbound/grpc/connectrpc"
)

// IPCHandler interface for handler registration.
type IPCHandler interface {
	GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}

func newIPCModule() fx.Option {
	return fx.Module("ipc",
		// IPC Server for Unix socket
		fx.Provide(setup.NewIPCServer),

		// IPC Service
		fx.Provide(ipc.NewService),

		// IPC gRPC Handler
		fx.Provide(
			fx.Annotate(
				connectrpchandler.NewIPCGRPCHandler,
				fx.As(new(IPCHandler)),
				fx.ResultTags(`group:"ipc_handlers"`),
			),
		),
	)
}
