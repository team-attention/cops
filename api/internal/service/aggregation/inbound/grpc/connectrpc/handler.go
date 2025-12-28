package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	aggregationservice "github.com/team-attention/cops/api/internal/service/aggregation"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// AggregationGRPCHandler handles aggregation service gRPC endpoints.
type AggregationGRPCHandler struct {
	svc    *aggregationservice.Service
	logger *slog.Logger
}

// NewAggregationGRPCHandler creates a new aggregation gRPC handler.
func NewAggregationGRPCHandler(l *slog.Logger, svc *aggregationservice.Service) *AggregationGRPCHandler {
	return &AggregationGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "aggregation.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *AggregationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return aggregationv1connect.NewAggregationServiceHandler(h, opts...)
}

// SendLogs implements aggregationv1connect.AggregationServiceHandler.
func (h *AggregationGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&aggregationv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch := convertToDomain(pbBatch)
	result := h.svc.CollectLogs(ctx, batch)

	res := &aggregationv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}

func convertToDomain(pb *aggregationv1.LogBatch) *repository.LogBatch {
	records := make([]shareddomain.SessionRecord, len(pb.GetRecords()))
	for i, r := range pb.GetRecords() {
		records[i] = shareddomain.SessionRecord{
			UUID:        r.GetUuid(),
			ParentUUID:  r.GetParentUuid(),
			SessionID:   r.GetSessionId(),
			Type:        convertSessionType(r.GetType()),
			Timestamp:   r.GetTimestamp().AsTime(),
			CWD:         r.GetCwd(),
			GitBranch:   r.GetGitBranch(),
			Version:     r.GetVersion(),
			UserType:    r.GetUserType(),
			IsSidechain: r.GetIsSidechain(),
			IsMeta:      r.GetIsMeta(),
			Slug:        r.GetSlug(),
			RequestID:   r.GetRequestId(),
			Message:     convertMessage(r.GetMessage()),
		}
	}

	return &repository.LogBatch{
		Records:   records,
		ProjectID: pb.GetProjectId(),
	}
}

func convertSessionType(t aggregationv1.SessionType) shareddomain.SessionType {
	switch t {
	case aggregationv1.SessionType_SESSION_TYPE_USER:
		return shareddomain.SessionTypeUser
	case aggregationv1.SessionType_SESSION_TYPE_ASSISTANT:
		return shareddomain.SessionTypeAssistant
	case aggregationv1.SessionType_SESSION_TYPE_SYSTEM:
		return shareddomain.SessionTypeSystem
	case aggregationv1.SessionType_SESSION_TYPE_SUMMARY:
		return shareddomain.SessionTypeSummary
	case aggregationv1.SessionType_SESSION_TYPE_FILE_HISTORY_SNAPSHOT:
		return shareddomain.SessionTypeFileHistorySnapshot
	case aggregationv1.SessionType_SESSION_TYPE_QUEUE_OPERATION:
		return shareddomain.SessionTypeQueueOperation
	default:
		return shareddomain.SessionTypeUser
	}
}

func convertMessage(m *aggregationv1.Message) *shareddomain.Message {
	if m == nil {
		return nil
	}

	var usage *shareddomain.Usage
	if m.GetUsage() != nil {
		usage = &shareddomain.Usage{
			InputTokens:              int(m.GetUsage().GetInputTokens()),
			OutputTokens:             int(m.GetUsage().GetOutputTokens()),
			CacheCreationInputTokens: int(m.GetUsage().GetCacheCreationInputTokens()),
			CacheReadInputTokens:     int(m.GetUsage().GetCacheReadInputTokens()),
			ServiceTier:              m.GetUsage().GetServiceTier(),
		}
	}

	text := m.GetText()
	return &shareddomain.Message{
		ID:         m.GetId(),
		Type:       m.GetType(),
		Role:       m.GetRole(),
		Model:      m.GetModel(),
		StopReason: m.GetStopReason(),
		Usage:      usage,
		Content: &shareddomain.MessageContent{
			IsBlocks: false,
			Text:     &text,
		},
	}
}

// Compile-time interface verification.
var _ aggregationv1connect.AggregationServiceHandler = (*AggregationGRPCHandler)(nil)
