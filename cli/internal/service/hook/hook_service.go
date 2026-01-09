package hook

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/cli/internal/platform/outbound/apikey"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
)

// Service provides hook-related operations.
type Service struct {
	logger   *slog.Logger
	apiKey   apikey.APIKeyPort
	eventAPI api.EventAPIPort
}

// NewService creates a new hook service.
func NewService(l *slog.Logger, apiKey apikey.APIKeyPort, eventAPI api.EventAPIPort) *Service {
	return &Service{
		logger:   l.With(slog.String("name", "hook.service")),
		apiKey:   apiKey,
		eventAPI: eventAPI,
	}
}

// PostEventParams contains parameters for posting an event.
type PostEventParams struct {
	RawJSON string
}

// PostEvent sends a single hook event to the API server.
func (s *Service) PostEvent(ctx context.Context, params PostEventParams) error {
	// 1. Get API key from APIKeyPort
	//    - Call s.apiKey.GetAPIKey(ctx)
	//    - If error, return error (caller handles logging)
	apiKey, err := s.apiKey.GetAPIKey(ctx)
	if err != nil {
		return err
	}

	// 2. Send event to API server
	//    - Call s.eventAPI.SendEvents(ctx, apiKey, []string{params.RawJSON})
	//    - If error, return error
	if err := s.eventAPI.SendEvents(ctx, apiKey, []string{params.RawJSON}); err != nil {
		return err
	}

	// 3. Return nil on success
	return nil
}
