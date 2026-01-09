package container

import (
	"go.uber.org/dig"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/claudesettings"
	claudesettingsfs "github.com/team-attention/cops/cli/internal/service/tracking/outbound/claudesettings/filesystem"
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

	// Event API client
	if err := c.Provide(
		connectrpc.NewEventAPIClient,
		dig.As(new(api.EventAPIPort)),
	); err != nil {
		return err
	}

	// Claude settings adapter
	// Algorithm:
	//   1. Provide NewFilesystemClaudeSettings constructor
	//   2. Cast to ClaudeSettingsPort interface
	if err := c.Provide(
		claudesettingsfs.NewFilesystemClaudeSettings,
		dig.As(new(claudesettings.ClaudeSettingsPort)),
	); err != nil {
		return err
	}

	// Service (automatically picks up new dependency)
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
