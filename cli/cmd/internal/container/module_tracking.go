package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	cliadapter "github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	connectrpcadapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
	configport "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	filesystemadapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	jsonladapter "github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"
)

// newTrackingModule registers all tracking-related providers.
func newTrackingModule(c *dig.Container) error {
	// Outbound adapters with dig.As for interface casting
	if err := c.Provide(
		filesystemadapter.NewFilesystemConfigAdapter,
		dig.As(new(configport.ConfigPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		jsonladapter.NewJSONLParser,
		dig.As(new(parser.ParserPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		connectrpcadapter.NewCollectorClient,
		dig.As(new(api.CollectorPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(tracking.NewService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cliadapter.NewTrackingCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
