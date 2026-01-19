package connectrpc

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	eventservice "github.com/team-attention/cops/api/internal/service/event"
	"github.com/team-attention/cops/shared/gen/grpcstub/event/v1/eventv1connect"
)

// EventGRPCHandler handles gRPC requests for event service.
type EventGRPCHandler struct {
	logger *slog.Logger
	svc    *eventservice.Service
}

// NewEventGRPCHandler creates a new event gRPC handler.
func NewEventGRPCHandler(l *slog.Logger, svc *eventservice.Service) *EventGRPCHandler {
	return &EventGRPCHandler{
		logger: l.With(slog.String("name", "event.grpc.connectrpc")),
		svc:    svc,
	}
}

// GetHandler implements ConnectHandler interface.
func (h *EventGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return eventv1connect.NewEventServiceHandler(h, opts...)
}

// Interface verification
var _ eventv1connect.EventServiceHandler = (*EventGRPCHandler)(nil)
