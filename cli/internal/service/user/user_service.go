package user

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	"github.com/team-attention/cops/cli/internal/service/user/outbound/api"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/connectrpcschema"
)

// Service provides user operations.
type Service struct {
	logger    *slog.Logger
	apiClient api.UserAPIPort
	authState authstate.AuthStatePort
}

// NewService creates a new user service.
func NewService(l *slog.Logger, apiClient api.UserAPIPort, authState authstate.AuthStatePort) *Service {
	return &Service{
		logger:    l.With(slog.String("name", "user.service")),
		apiClient: apiClient,
		authState: authState,
	}
}

// GetMyOrganizations fetches the authenticated user's organizations.
// Returns domain.Organization (not protobuf types) for use in TUI and business logic.
func (s *Service) GetMyOrganizations(ctx context.Context) ([]*domain.Organization, error) {
	accessToken, err := s.authState.GetAccessToken(ctx)
	if err != nil {
		s.logger.Error("failed to get access token",
			slog.Any("error", err),
		)
		return nil, err
	}

	result, err := s.apiClient.GetMe(ctx, accessToken)
	if err != nil {
		s.logger.Error("failed to get user organizations",
			slog.Any("error", err),
		)
		return nil, err
	}

	organizations := connectrpcschema.OrganizationsFromProto(result.Organizations)

	return organizations, nil
}
