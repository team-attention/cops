package cobra

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// NewLoginCommand creates the login command.
func (h *AuthCLIHandler) NewLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to C-Ops",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			result, err := h.svc.InitiateLogin(ctx)
			if err != nil {
				return fmt.Errorf("failed to initiate login: %w", err)
			}

			fmt.Println("To sign in, open this URL in your browser:")
			fmt.Printf("\n  %s\n\n", result.VerificationURL)
			fmt.Printf("Device code: %s\n\n", result.UserCode)
			fmt.Println("Waiting for authentication...")

			ticker := time.NewTicker(time.Duration(result.Interval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("authentication timeout")
				case <-ticker.C:
					complete, err := h.svc.PollLogin(ctx, result.DeviceCode)
					if err != nil {
						return fmt.Errorf("failed to poll for authentication: %w", err)
					}

					if complete {
						fmt.Println("\nAuthentication successful!")
						return nil
					}
				}
			}
		},
	}
}
