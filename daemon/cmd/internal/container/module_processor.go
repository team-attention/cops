package container

import (
	"go.uber.org/fx"

	"github.com/team-attention/cops/daemon/internal/service/logprocessor"
	"github.com/team-attention/cops/daemon/internal/service/logprocessor/outbound/api/connectrpc"
)

func newProcessorModule() fx.Option {
	return fx.Module("processor",
		// ConnectRPC API client adapter
		fx.Provide(
			fx.Annotate(
				connectrpc.NewAdapter,
				fx.As(new(logprocessor.APIClientPort)),
			),
		),
		// LogProcessor service
		fx.Provide(logprocessor.NewService),
	)
}
