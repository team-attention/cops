package connectrpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/event"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
)

// SendLogs receives a batch of JSONL log lines and saves them to events collection.
func (h *EventGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[eventv1.SendLogsReq],
) (*connect.Response[eventv1.SendLogsRes], error) {
	// 1. Extract userID from context using interceptor.UserIDFromContext(ctx).
	userID := interceptor.UserIDFromContext(ctx)

	// 2. Get batch from request: pbBatch := req.Msg.GetBatch().
	pbBatch := req.Msg.GetBatch()

	// 3. Validate batch is not nil.
	if pbBatch == nil {
		return connect.NewResponse(&eventv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	// 4. Call service with all batch fields including version and provider.
	result, err := h.svc.ParseAndCollectLogs(ctx, event.ParseAndCollectLogsParams{
		UserID:         userID,
		Lines:          pbBatch.GetJsonl(),
		ProjectID:      pbBatch.GetProjectId(),
		OrganizationID: pbBatch.GetOrganizationId(),
		Version:        pbBatch.GetVersion(),
		Provider:       pbBatch.GetProvider(),
	})

	// 5. If error, return nil, err.
	if err != nil {
		return nil, err
	}

	// 6. Build response mapping result.StoredCount to ProcessedCount field.
	res := &eventv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.StoredCount,
	}

	// 7. Return connect.NewResponse(res), nil.
	return connect.NewResponse(res), nil
}
