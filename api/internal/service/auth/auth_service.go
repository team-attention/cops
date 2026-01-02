package auth

import (
	"context"
	"fmt"
	"log/slog"

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
	logger    *slog.Logger
	jwtCfg    *jwtutil.Config
	oauthPort oauth.GoogleOAuthPort
	userRepo  repository.UserRepositoryPort
}

// NewService creates a new auth service.
func NewService(
	l *slog.Logger,
	jwtCfg *jwtutil.Config,
	oauthPort oauth.GoogleOAuthPort,
	userRepo repository.UserRepositoryPort,
) *Service {
	return &Service{
		logger:    l.With(slog.String("name", "auth.service")),
		jwtCfg:    jwtCfg,
		oauthPort: oauthPort,
		userRepo:  userRepo,
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
		tokens, err := jwtutil.GenerateTokenPair(s.jwtCfg, string(user.ID))
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

	tokens, err := jwtutil.GenerateTokenPair(s.jwtCfg, string(createdUser.ID))
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
	resp, err := s.oauthPort.InitiateDeviceFlow(ctx)
	if err != nil {
		s.logger.Error("failed to initiate device flow",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}

	return &DeviceCodeResult{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURL: resp.VerificationURL,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        resp.Interval,
	}, nil
}

// DevicePoll checks if device authentication is complete.
func (s *Service) DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error) {
	tokenResp, err := s.oauthPort.PollDeviceCode(ctx, deviceCode)
	if err != nil {
		s.logger.Error("failed to poll device code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to poll device code: %w", err)
	}

	if tokenResp == nil {
		return &DevicePollResult{Pending: true}, nil
	}

	userInfo, err := s.oauthPort.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		s.logger.Error("failed to get user info from device flow",
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

	if user == nil {
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

		user, err = s.userRepo.Create(ctx, newUser)
		if err != nil {
			s.logger.Error("failed to create user from device flow",
				slog.String("email", newUser.Email),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		s.logger.Info("new user created via device flow",
			slog.String("userID", string(user.ID)),
			slog.String("email", user.Email),
		)
	}

	tokens, err := jwtutil.GenerateTokenPair(s.jwtCfg, string(user.ID))
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
	userID, err := jwtutil.ValidateRefreshToken(s.jwtCfg, refreshToken)
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

	tokens, err := jwtutil.GenerateTokenPair(s.jwtCfg, string(user.ID))
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
