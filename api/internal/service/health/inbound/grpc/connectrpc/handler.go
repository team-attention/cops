package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/health"
	healthv1 "github.com/team-attention/cops/shared/gen/grpcstub/health/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/health/v1/healthv1connect"
)

// HealthGRPCHandler handles health check gRPC endpoints.
type HealthGRPCHandler struct {
	svc    *health.Service
	logger *slog.Logger
}

// NewHealthGRPCHandler creates a new health gRPC handler.
func NewHealthGRPCHandler(l *slog.Logger, svc *health.Service) *HealthGRPCHandler {
	return &HealthGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "health.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *HealthGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return healthv1connect.NewHealthServiceHandler(h, opts...)
}

// Check implements healthv1connect.HealthServiceHandler.
func (h *HealthGRPCHandler) Check(
	ctx context.Context,
	req *connect.Request[healthv1.CheckReq],
) (*connect.Response[healthv1.CheckRes], error) {
	serviceName := req.Msg.GetService()

	var status healthv1.ServingStatus
	if h.svc.IsServing(ctx, serviceName) {
		status = healthv1.ServingStatus_SERVING_STATUS_SERVING
	} else {
		status = healthv1.ServingStatus_SERVING_STATUS_NOT_SERVING
	}

	res := &healthv1.CheckRes{
		Status: status,
	}

	return connect.NewResponse(res), nil
}

// Watch implements healthv1connect.HealthServiceHandler.
func (h *HealthGRPCHandler) Watch(
	ctx context.Context,
	req *connect.Request[healthv1.WatchReq],
	stream *connect.ServerStream[healthv1.WatchRes],
) error {
	serviceName := req.Msg.GetService()

	var status healthv1.ServingStatus
	if h.svc.IsServing(ctx, serviceName) {
		status = healthv1.ServingStatus_SERVING_STATUS_SERVING
	} else {
		status = healthv1.ServingStatus_SERVING_STATUS_NOT_SERVING
	}

	// Send initial status
	if err := stream.Send(&healthv1.WatchRes{Status: status}); err != nil {
		return err
	}

	// Keep connection open until context is cancelled
	<-ctx.Done()
	return nil
}

// Compile-time interface verification
var _ healthv1connect.HealthServiceHandler = (*HealthGRPCHandler)(nil)
