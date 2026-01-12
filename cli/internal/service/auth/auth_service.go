package auth

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/auth/outbound/storage"
)

// LoginParams contains parameters for the login flow.
type LoginParams struct {
	// SkipConfirmation skips the "already logged in" prompt when true
	SkipConfirmation bool
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	// APIKeyPrefix is the obtained API key (masked for display)
	APIKeyPrefix string
	// Message is a user-friendly success message
	Message string
}

// Service provides authentication operations for CLI.
type Service struct {
	logger     *slog.Logger
	authClient api.AuthAPIPort
	apikeyPort storage.APIKeyStoragePort
}

// NewService creates a new auth service.
func NewService(l *slog.Logger, authClient api.AuthAPIPort, apikeyPort storage.APIKeyStoragePort) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "auth.service")),
		authClient: authClient,
		apikeyPort: apikeyPort,
	}
}

// IsLoggedIn checks if an API key exists in local storage.
func (s *Service) IsLoggedIn(ctx context.Context) (bool, error) {
	exists, err := s.apikeyPort.HasAPIKey(ctx)
	if err != nil {
		s.logger.Error("failed to check API key existence",
			slog.Any("error", err),
		)
		return false, err
	}
	return exists, nil
}

// Login performs the device flow authentication.
// Returns LoginResult on success, error on failure.
func (s *Service) Login(ctx context.Context, params LoginParams) (*LoginResult, error) {
	// 1. Initiate device code flow
	deviceResult, err := s.authClient.DeviceCode(ctx)
	if err != nil {
		s.logger.Error("failed to initiate device code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to initiate device code: %w", err)
	}

	// 2. Display device code info to user
	fmt.Println()
	fmt.Println("To authenticate, please visit the following URL in your browser:")
	fmt.Println()
	fmt.Printf("  %s\n", deviceResult.VerificationURL)
	fmt.Println()
	fmt.Println("And enter this code:")
	fmt.Println()
	fmt.Printf("  %s\n", deviceResult.UserCode)
	fmt.Println()

	// Attempt to open browser automatically
	if err := openBrowser(deviceResult.VerificationURL); err != nil {
		s.logger.Debug("failed to open browser automatically",
			slog.Any("error", err),
		)
	}

	fmt.Println("Waiting for approval...")
	fmt.Println()

	// 3. Poll for approval with spinner
	deadline := time.Now().Add(time.Duration(deviceResult.ExpiresIn) * time.Second)
	interval := time.Duration(deviceResult.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	spinnerChars := []rune{'|', '/', '-', '\\'}
	spinnerIdx := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			// Check if deadline exceeded
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("device code expired")
			}

			// Update spinner
			fmt.Printf("\r  %c Polling for approval...", spinnerChars[spinnerIdx])
			spinnerIdx = (spinnerIdx + 1) % len(spinnerChars)

			// Poll for status
			pollResult, err := s.authClient.DevicePoll(ctx, deviceResult.DeviceCode)
			if err != nil {
				s.logger.Error("failed to poll device status",
					slog.Any("error", err),
				)
				fmt.Println()
				return nil, fmt.Errorf("failed to poll for approval: %w", err)
			}

			if pollResult.Pending {
				// Continue polling
				continue
			}

			// Approval received - clear spinner line
			fmt.Print("\r                              \r")

			// Issue API key using the access token
			apiKey, err := s.authClient.IssueAPIKey(ctx, pollResult.AccessToken, "CLI Device")
			if err != nil {
				s.logger.Error("failed to issue API key",
					slog.Any("error", err),
				)
				return nil, fmt.Errorf("failed to issue API key: %w", err)
			}

			// Store API key
			if err := s.apikeyPort.SaveAPIKey(ctx, apiKey); err != nil {
				s.logger.Error("failed to save API key",
					slog.Any("error", err),
				)
				return nil, fmt.Errorf("failed to save API key: %w", err)
			}

			// Return success with masked API key prefix
			prefix := apiKey
			if len(prefix) > 12 {
				prefix = prefix[:12] + "..."
			}

			return &LoginResult{
				APIKeyPrefix: prefix,
				Message:      "Login successful!",
			}, nil
		}
	}
}

// PromptConfirmation prompts the user for confirmation and returns the result.
func PromptConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// openBrowser attempts to open a URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
