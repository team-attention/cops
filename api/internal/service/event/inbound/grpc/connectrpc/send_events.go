package connectrpc

import (
	"context"
	"encoding/json"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/shared/domain"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
)

// SendEvents receives a batch of hook events from Claude Code.
func (h *EventGRPCHandler) SendEvents(
	ctx context.Context,
	req *connect.Request[eventv1.SendEventsReq],
) (*connect.Response[eventv1.SendEventsRes], error) {
	// 1. Extract userID from context (set by API key interceptor)
	//    - Call interceptor.UserIDFromContext(ctx)
	//    - If empty, log warning and return empty response (Fire & Forget)
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("SendEvents: userID not found in context")
		return connect.NewResponse(&eventv1.SendEventsRes{}), nil
	}

	// 2. Parse raw JSON strings to domain.Event slice
	//    - Iterate over req.Msg.Events
	//    - For each JSON string:
	//      a. Create new domain.Event
	//      b. Call json.Unmarshal([]byte(jsonStr), &event)
	//      c. If unmarshal fails, log error with event index and continue (skip)
	//      d. If success, append to events slice
	var events []*domain.Event
	for i, jsonStr := range req.Msg.Events {
		var event domain.Event
		if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
			h.logger.Error("SendEvents: failed to unmarshal event",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			continue
		}

		// Skip events with empty type (log full event for debugging)
		if event.Type == "" {
			h.logger.Warn("SendEvents: skipping event with empty type",
				slog.Int("index", i),
				slog.String("rawJSON", jsonStr),
			)
			continue
		}

		events = append(events, &event)
	}

	// 3. Save events to repository (Fire & Forget)
	//    - If events slice is not empty:
	//      a. Call h.repo.SaveEvents(ctx, userID, events)
	//      b. If error, log error (do NOT return error - Fire & Forget)
	if len(events) > 0 {
		if err := h.repo.SaveEvents(ctx, userID, events); err != nil {
			h.logger.Error("SendEvents: failed to save events",
				slog.String("userID", userID),
				slog.Int("eventCount", len(events)),
				slog.Any("error", err),
			)
		}
	}

	// 4. Return empty response
	//    - Always return success regardless of parsing/saving errors
	return connect.NewResponse(&eventv1.SendEventsRes{}), nil
}
