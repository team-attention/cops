package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/service/ipc"
	daemonv1 "github.com/team-attention/cops/shared/gen/grpcstub/daemon/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/daemon/v1/daemonv1connect"
)

// IPCGRPCHandler implements the DaemonService gRPC interface.
type IPCGRPCHandler struct {
	logger *slog.Logger
	svc    *ipc.Service
}

// Interface verification
var _ daemonv1connect.DaemonServiceHandler = (*IPCGRPCHandler)(nil)

// NewIPCGRPCHandler creates a new IPC gRPC handler.
func NewIPCGRPCHandler(l *slog.Logger, svc *ipc.Service) *IPCGRPCHandler {
	return &IPCGRPCHandler{
		logger: l.With(slog.String("name", "ipc.grpc.connectrpc")),
		svc:    svc,
	}
}

// GetHandler returns the HTTP path and handler for this service.
func (h *IPCGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return daemonv1connect.NewDaemonServiceHandler(h, opts...)
}

// ScanLogs implements daemonv1connect.DaemonServiceHandler.
func (h *IPCGRPCHandler) ScanLogs(
	ctx context.Context,
	req *connect.Request[daemonv1.ScanLogsReq],
) (*connect.Response[daemonv1.ScanLogsRes], error) {
	h.logger.Debug("ScanLogs request received",
		slog.String("projectID", req.Msg.ProjectId),
		slog.String("projectPath", req.Msg.ProjectPath),
	)

	result, err := h.svc.ScanLogs(ctx, ipc.ScanLogsParams{
		ProjectID:      req.Msg.ProjectId,
		ProjectPath:    req.Msg.ProjectPath,
		OrganizationID: req.Msg.OrganizationId,
	})
	if err != nil {
		h.logger.Error("ScanLogs failed", slog.Any("error", err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&daemonv1.ScanLogsRes{
		Success: result.Success,
		Message: result.Message,
	}), nil
}
