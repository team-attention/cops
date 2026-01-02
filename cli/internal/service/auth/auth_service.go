package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
)

// AuthState represents the local authentication state.
// Stores only token information.
type AuthState struct {
	Tokens *TokenInfo `json:"tokens"`
}

// TokenInfo contains token data.
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"` // Unix timestamp
}

// Service implements CLI authentication logic.
type Service struct {
	logger    *slog.Logger
	apiClient api.AuthAPIPort
	authPath  string
}

// NewService creates a new CLI auth service.
func NewService(l *slog.Logger, apiClient api.AuthAPIPort, homeDir string) *Service {
	return &Service{
		logger:    l.With(slog.String("name", "auth.service")),
		apiClient: apiClient,
		authPath:  filepath.Join(homeDir, ".cops", "auth.json"),
	}
}

// LoginResult contains the result of login flow for display.
type LoginResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	Interval        int
}

// InitiateLogin starts the device flow and returns display info.
func (s *Service) InitiateLogin(ctx context.Context) (*LoginResult, error) {
	result, err := s.apiClient.DeviceCode(ctx)
	if err != nil {
		s.logger.Error("failed to initiate device flow",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}

	return &LoginResult{
		DeviceCode:      result.DeviceCode,
		UserCode:        result.UserCode,
		VerificationURL: result.VerificationURL,
		Interval:        result.Interval,
	}, nil
}

// PollLogin polls for authentication completion.
func (s *Service) PollLogin(ctx context.Context, deviceCode string) (bool, error) {
	result, err := s.apiClient.DevicePoll(ctx, deviceCode)
	if err != nil {
		s.logger.Error("failed to poll device code",
			slog.Any("error", err),
		)
		return false, fmt.Errorf("failed to poll device code: %w", err)
	}

	if result.Pending {
		return false, nil
	}

	state := &AuthState{
		Tokens: &TokenInfo{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		},
	}

	if err := s.saveAuthState(state); err != nil {
		s.logger.Error("failed to save auth state",
			slog.Any("error", err),
		)
		return false, fmt.Errorf("failed to save auth state: %w", err)
	}

	s.logger.Info("authentication successful")

	return true, nil
}

// Logout removes the local authentication state.
func (s *Service) Logout(ctx context.Context) error {
	if _, err := os.Stat(s.authPath); os.IsNotExist(err) {
		s.logger.Info("no auth state to remove")
		return nil
	}

	if err := os.Remove(s.authPath); err != nil {
		s.logger.Error("failed to remove auth file",
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to remove auth file: %w", err)
	}

	s.logger.Info("logged out successfully")
	return nil
}

// GetAuthState returns the current authentication state.
func (s *Service) GetAuthState() (*AuthState, error) {
	if _, err := os.Stat(s.authPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(s.authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	return &state, nil
}

// IsLoggedIn checks if user is currently logged in.
func (s *Service) IsLoggedIn() bool {
	state, err := s.GetAuthState()
	if err != nil || state == nil || state.Tokens == nil {
		return false
	}

	return true
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (s *Service) GetAccessToken(ctx context.Context) (string, error) {
	state, err := s.GetAuthState()
	if err != nil {
		return "", fmt.Errorf("failed to get auth state: %w", err)
	}

	if state == nil || state.Tokens == nil {
		return "", fmt.Errorf("not logged in")
	}

	now := time.Now().Unix()
	tokenExpiry := state.Tokens.ExpiresAt
	refreshBuffer := int64(300)

	if tokenExpiry-now > refreshBuffer {
		return state.Tokens.AccessToken, nil
	}

	s.logger.Info("access token near expiry, refreshing")

	result, err := s.apiClient.RefreshToken(ctx, state.Tokens.RefreshToken)
	if err != nil {
		s.logger.Error("failed to refresh token",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	state.Tokens.AccessToken = result.AccessToken
	state.Tokens.RefreshToken = result.RefreshToken
	state.Tokens.ExpiresAt = result.ExpiresAt

	if err := s.saveAuthState(state); err != nil {
		s.logger.Error("failed to save refreshed tokens",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to save refreshed tokens: %w", err)
	}

	s.logger.Info("token refreshed successfully")

	return state.Tokens.AccessToken, nil
}

// saveAuthState writes auth state to file with secure permissions.
func (s *Service) saveAuthState(state *AuthState) error {
	copsDir := filepath.Dir(s.authPath)
	if err := os.MkdirAll(copsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .cops directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth state: %w", err)
	}

	if err := os.WriteFile(s.authPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}
