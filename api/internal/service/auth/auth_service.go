package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// GoogleAuthParams contains parameters for Google OAuth authentication.
type GoogleAuthParams struct {
	AuthorizationCode string
	RedirectURI       string
}

// DeviceCodeResult contains device code for CLI flow.
type DeviceCodeResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

// DevicePollResult contains result of device code polling.
type DevicePollResult struct {
	Pending bool
	Tokens  *jwtutil.TokenPair
}

// Service implements authentication business logic.
type Service struct {
	logger         *slog.Logger
	cfg            *config.Config
	oauthPort      oauth.GoogleOAuthPort
	userRepo       repository.UserRepositoryPort
	deviceCodeRepo repository.DeviceCodeRepositoryPort
}

// NewService creates a new auth service.
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	oauthPort oauth.GoogleOAuthPort,
	userRepo repository.UserRepositoryPort,
	deviceCodeRepo repository.DeviceCodeRepositoryPort,
) *Service {
	return &Service{
		logger:         l.With(slog.String("name", "auth.service")),
		cfg:            cfg,
		oauthPort:      oauthPort,
		userRepo:       userRepo,
		deviceCodeRepo: deviceCodeRepo,
	}
}

// GoogleAuth handles Google OAuth code exchange and user creation/lookup.
func (s *Service) GoogleAuth(ctx context.Context, params GoogleAuthParams) (*jwtutil.TokenPair, error) {
	tokenResp, err := s.oauthPort.ExchangeCode(ctx, params.AuthorizationCode, params.RedirectURI)
	if err != nil {
		s.logger.Error("failed to exchange authorization code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	userInfo, err := s.oauthPort.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		s.logger.Error("failed to get user info",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := s.userRepo.FindByAccountProvider(ctx, domain.AccountProviderGoogle, userInfo.ID)
	if err != nil {
		s.logger.Error("failed to find user by account provider",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if user != nil {
		jwtCfg := &jwtutil.Config{
			SecretKey:            s.cfg.JWT.SecretKey,
			AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
			RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
			Issuer:               s.cfg.JWT.Issuer,
		}

		tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(user.ID))
		if err != nil {
			s.logger.Error("failed to generate tokens for existing user",
				slog.String("userID", string(user.ID)),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to generate tokens: %w", err)
		}

		s.logger.Info("user logged in",
			slog.String("userID", string(user.ID)),
			slog.String("email", user.Email),
		)

		return tokens, nil
	}

	newUser := &domain.User{
		Email:           userInfo.Email,
		Name:            userInfo.Name,
		ProfileImageURL: userInfo.Picture,
		Accounts: []*domain.Account{
			{
				Provider:   domain.AccountProviderGoogle,
				ProviderID: userInfo.ID,
			},
		},
	}

	createdUser, err := s.userRepo.Create(ctx, newUser)
	if err != nil {
		s.logger.Error("failed to create user",
			slog.String("email", newUser.Email),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	jwtCfg := &jwtutil.Config{
		SecretKey:            s.cfg.JWT.SecretKey,
		AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
		RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
		Issuer:               s.cfg.JWT.Issuer,
	}

	tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(createdUser.ID))
	if err != nil {
		s.logger.Error("failed to generate tokens for new user",
			slog.String("userID", string(createdUser.ID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Info("new user created and logged in",
		slog.String("userID", string(createdUser.ID)),
		slog.String("email", createdUser.Email),
	)

	return tokens, nil
}

// DeviceCode initiates device flow authentication.
func (s *Service) DeviceCode(ctx context.Context) (*DeviceCodeResult, error) {
	userCode, err := generateUserCode()
	if err != nil {
		s.logger.Error("failed to generate user code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	expiresAt := time.Now().Add(s.cfg.DeviceCode.Expiration)

	deviceCodeData := &domain.DeviceCode{
		UserCode:  normalizeUserCode(userCode),
		Approved:  false,
		ExpiresAt: expiresAt,
	}

	createdCode, err := s.deviceCodeRepo.Create(ctx, deviceCodeData)
	if err != nil {
		s.logger.Error("failed to create device code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to create device code: %w", err)
	}

	verificationURL := fmt.Sprintf("%s/auth/device?code=%s", s.cfg.DeviceCode.WebBaseURL, userCode)

	return &DeviceCodeResult{
		DeviceCode:      string(createdCode.ID),
		UserCode:        userCode,
		VerificationURL: verificationURL,
		ExpiresIn:       int(s.cfg.DeviceCode.Expiration.Seconds()),
		Interval:        s.cfg.DeviceCode.Interval,
	}, nil
}

// DevicePoll checks if device authentication is complete.
func (s *Service) DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error) {
	code, err := s.deviceCodeRepo.GetByID(ctx, deviceCode)
	if err != nil {
		s.logger.Error("failed to get device code",
			slog.String("deviceCode", deviceCode),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to get device code: %w", err)
	}

	if code == nil {
		return nil, fmt.Errorf("device code not found")
	}

	if code.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("device code expired")
	}

	if !code.Approved {
		return &DevicePollResult{Pending: true}, nil
	}

	if code.UserID == nil {
		s.logger.Error("device code approved but no user ID",
			slog.String("deviceCode", deviceCode),
		)
		return nil, fmt.Errorf("device code approved but no user ID")
	}

	user, err := s.userRepo.GetByID(ctx, string(*code.UserID))
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("userID", string(*code.UserID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user == nil {
		s.logger.Error("user not found for device code",
			slog.String("userID", string(*code.UserID)),
		)
		return nil, fmt.Errorf("user not found")
	}

	jwtCfg := &jwtutil.Config{
		SecretKey:            s.cfg.JWT.SecretKey,
		AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
		RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
		Issuer:               s.cfg.JWT.Issuer,
	}

	tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(user.ID))
	if err != nil {
		s.logger.Error("failed to generate tokens from device flow",
			slog.String("userID", string(user.ID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &DevicePollResult{
		Pending: false,
		Tokens:  tokens,
	}, nil
}

// RefreshToken exchanges refresh token for new token pair.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*jwtutil.TokenPair, error) {
	jwtCfg := &jwtutil.Config{
		SecretKey:            s.cfg.JWT.SecretKey,
		AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
		RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
		Issuer:               s.cfg.JWT.Issuer,
	}

	userID, err := jwtutil.ValidateRefreshToken(jwtCfg, refreshToken)
	if err != nil {
		s.logger.Warn("invalid refresh token",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user for refresh token",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user == nil {
		s.logger.Warn("user not found for refresh token",
			slog.String("userID", userID),
		)
		return nil, fmt.Errorf("user not found")
	}

	tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(user.ID))
	if err != nil {
		s.logger.Error("failed to generate new tokens",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Info("token refreshed",
		slog.String("userID", userID),
	)

	return tokens, nil
}

// DeviceCodeApproveParams contains parameters for device code approval.
type DeviceCodeApproveParams struct {
	UserCode string
	UserID   domain.ID
}

// DeviceCodeApprove approves a device code and links it to the authenticated user.
func (s *Service) DeviceCodeApprove(ctx context.Context, params DeviceCodeApproveParams) error {
	normalizedCode := normalizeUserCode(params.UserCode)

	user, err := s.userRepo.GetByID(ctx, string(params.UserID))
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("userID", string(params.UserID)),
			slog.Any("error", err),
		)
		return fmt.Errorf("user not found: %w", err)
	}

	if user == nil {
		s.logger.Warn("user not found for device code approval",
			slog.String("userID", string(params.UserID)),
		)
		return fmt.Errorf("user not found")
	}

	err = s.deviceCodeRepo.Approve(ctx, normalizedCode, params.UserID)
	if err != nil {
		s.logger.Error("failed to approve device code",
			slog.String("userCode", params.UserCode),
			slog.String("userID", string(params.UserID)),
			slog.Any("error", err),
		)
		return err
	}

	s.logger.Info("device code approved",
		slog.String("userCode", params.UserCode),
		slog.String("userID", string(params.UserID)),
	)

	return nil
}

// userCodeChars contains characters for human-friendly codes.
// Excludes ambiguous characters: 0, O, I, 1, L
const userCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateDeviceCodeID generates a cryptographically secure device code ID.
// Returns a 32-character hex string (16 bytes of randomness).
func generateDeviceCodeID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateUserCode generates a human-friendly 8-character code with hyphen.
// Format: XXXX-XXXX (e.g., "ABCD-EFGH")
func generateUserCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	var code strings.Builder
	code.Grow(9) // 8 chars + 1 hyphen

	for i, b := range bytes {
		if i == 4 {
			code.WriteByte('-')
		}
		code.WriteByte(userCodeChars[int(b)%len(userCodeChars)])
	}

	return code.String(), nil
}

// normalizeUserCode normalizes user code input by removing hyphens and converting to uppercase.
func normalizeUserCode(code string) string {
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, "-", "")
	return code
}
