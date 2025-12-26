package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"
)

// newTrackingModule registers all tracking-related providers.
func newTrackingModule(c *dig.Container) error {
	// Outbound adapters with dig.As for interface casting
	if err := c.Provide(
		filesystem.NewFilesystemConfigAdapter,
		dig.As(new(config.ConfigPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		jsonl.NewJSONLParser,
		dig.As(new(parser.ParserPort)),
	); err != nil {
		return err
	}
	if err := c.Provide(
		connectrpc.NewProjectClient,
		dig.As(new(api.ProjectPort)),
	); err != nil {
		return err
	}

	// Service
	if err := c.Provide(tracking.NewService); err != nil {
		return err
	}

	// CLI handler with dig.As + dig.Group
	return c.Provide(
		cobra.NewTrackingCLIHandler,
		dig.As(new(CLICommandProvider)),
		dig.Group("cli_handlers"),
	)
}
