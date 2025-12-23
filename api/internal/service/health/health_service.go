package health

import (
	"context"
	"log/slog"
	"time"
)

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// ReadyStatus represents the readiness check response.
type ReadyStatus struct {
	Status     string            `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Components map[string]string `json:"components,omitempty"`
}

// Service handles health check operations.
type Service struct {
	logger  *slog.Logger
	version string
}

// NewService creates a new health service.
func NewService(l *slog.Logger) *Service {
	return &Service{
		logger:  l.With(slog.String("name", "health.service")),
		version: "0.0.1",
	}
}

// Health returns the liveness status.
func (s *Service) Health(ctx context.Context) *HealthStatus {
	return &HealthStatus{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   s.version,
	}
}

// Ready returns the readiness status.
func (s *Service) Ready(ctx context.Context) *ReadyStatus {
	components := map[string]string{
		"api": "ok",
	}

	return &ReadyStatus{
		Status:     "ok",
		Timestamp:  time.Now().UTC(),
		Components: components,
	}
}

// IsServing returns whether the service is serving requests.
func (s *Service) IsServing(ctx context.Context, serviceName string) bool {
	return true
}
