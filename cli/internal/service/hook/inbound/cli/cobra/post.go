package cobra

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/hook"
)

// NewPostCommand creates the "hook post" subcommand.
func (h *HookCLIHandler) NewPostCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "post",
		Short: "Post a hook event to the API server",
		Long:  "Reads a JSON event from stdin and sends it to the C-Ops API server.",
		// SilenceUsage: true to prevent usage output on error
		SilenceUsage: true,
		// SilenceErrors: true to handle error output manually
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.runPost(cmd.Context())
		},
	}
}

// runPost executes the post command logic.
func (h *HookCLIHandler) runPost(ctx context.Context) error {
	// 1. Read all data from stdin
	//    - Call io.ReadAll(os.Stdin)
	//    - If error, log to stderr and return error (exit 1)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		h.logger.Error("failed to read from stdin", slog.Any("error", err))
		return err
	}

	// 2. Check if stdin is empty
	//    - If len(data) == 0, log warning and return nil (exit 0, nothing to send)
	if len(data) == 0 {
		h.logger.Warn("no data received from stdin, nothing to send")
		return nil
	}

	// 3. Convert to string (Raw JSON)
	//    - rawJSON := string(data)
	rawJSON := string(data)

	// 4. Call service.PostEvent
	//    - params := hook.PostEventParams{RawJSON: rawJSON}
	//    - err := h.svc.PostEvent(ctx, params)
	//    - If error, log error to stderr with slog and return error (exit 1)
	params := hook.PostEventParams{RawJSON: rawJSON}
	if err := h.svc.PostEvent(ctx, params); err != nil {
		h.logger.Error("failed to post event", slog.Any("error", err))
		return err
	}

	// 5. Return nil on success (exit 0, no stdout output)
	return nil
}
