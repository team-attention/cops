package connectrpc

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

// EventGRPCHandler handles gRPC requests for event service.
type EventGRPCHandler struct {
	logger *slog.Logger
	repo   repository.EventRepositoryPort
}

// NewEventGRPCHandler creates a new event gRPC handler.
func NewEventGRPCHandler(l *slog.Logger, repo repository.EventRepositoryPort) *EventGRPCHandler {
	return &EventGRPCHandler{
		logger: l.With(slog.String("name", "event.grpc.connectrpc")),
		repo:   repo,
	}
}

// GetHandler implements ConnectHandler interface.
func (h *EventGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return eventv1connect.NewEventServiceHandler(h, opts...)
}

// Interface verification
var _ eventv1connect.EventServiceHandler = (*EventGRPCHandler)(nil)
