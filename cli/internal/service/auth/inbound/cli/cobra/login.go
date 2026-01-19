package cobra

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/team-attention/cops/cli/internal/service/auth"
)

// NewLoginCommand creates the 'login' subcommand under 'auth'.
func (h *AuthCLIHandler) NewLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to cops via browser",
		Long: `Authenticate with cops using your browser.

This command initiates a device authentication flow:
1. A device code is displayed for you to enter in your browser
2. Your browser opens to the verification URL
3. After approving the device, an API key is stored locally

The API key is stored in ~/.cops/auth.json and is used for subsequent CLI operations.

Examples:
  cops auth login    # Start the login flow`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Check if already logged in
			loggedIn, err := h.svc.IsLoggedIn(ctx)
			if err != nil {
				h.logger.Error("failed to check login status",
					slog.Any("error", err),
				)
				// Continue with login if we can't check status
			}

			if loggedIn {
				// Prompt: "Already logged in. Do you want to login again? [y/N]"
				if !auth.PromptConfirmation("Already logged in. Do you want to login again?") {
					fmt.Println("Login cancelled.")
					return nil
				}
			}

			// Perform login
			params := auth.LoginParams{}
			result, err := h.svc.Login(ctx, params)
			if err != nil {
				// Handle specific error cases
				if err.Error() == "device code expired" {
					fmt.Println("Login timed out. Please try again.")
					return nil
				}
				return err
			}

			// Display success
			fmt.Println(result.Message)
			fmt.Println("API key stored in ~/.cops/auth.json")

			return nil
		},
	}

	return cmd
}
