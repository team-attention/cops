package connectrpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
)

// SendLogs receives a batch of JSONL log lines and saves them to events collection.
func (h *EventGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[eventv1.SendLogsReq],
) (*connect.Response[eventv1.SendLogsRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&eventv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	_, result, err := h.svc.ParseAndCollectLogs(
		ctx,
		userID,
		pbBatch.GetJsonl(),
		pbBatch.GetProjectId(),
		pbBatch.GetOrganizationId(),
	)
	if err != nil {
		return nil, err
	}

	res := &eventv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}
