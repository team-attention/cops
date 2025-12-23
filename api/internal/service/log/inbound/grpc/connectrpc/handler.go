package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	logservice "github.com/team-attention/cops/api/internal/service/log"
	"github.com/team-attention/cops/api/internal/service/log/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	logv1 "github.com/team-attention/cops/shared/gen/grpcstub/log/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/log/v1/logv1connect"
)

// LogGRPCHandler handles log service gRPC endpoints.
type LogGRPCHandler struct {
	svc    *logservice.Service
	logger *slog.Logger
}

// NewLogGRPCHandler creates a new log gRPC handler.
func NewLogGRPCHandler(l *slog.Logger, svc *logservice.Service) *LogGRPCHandler {
	return &LogGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "log.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *LogGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return logv1connect.NewLogServiceHandler(h, opts...)
}

// SendLogs implements logv1connect.LogServiceHandler.
func (h *LogGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[logv1.SendLogsReq],
) (*connect.Response[logv1.SendLogsRes], error) {
	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&logv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch := convertToDomain(pbBatch)
	result := h.svc.CollectLogs(ctx, batch)

	res := &logv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}

func convertToDomain(pb *logv1.LogBatch) *repository.LogBatch {
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
		DaemonID:  pb.GetDaemonId(),
		CreatedAt: pb.GetCreatedAt().AsTime().String(),
	}
}

func convertSessionType(t logv1.SessionType) shareddomain.SessionType {
	switch t {
	case logv1.SessionType_SESSION_TYPE_USER:
		return shareddomain.SessionTypeUser
	case logv1.SessionType_SESSION_TYPE_ASSISTANT:
		return shareddomain.SessionTypeAssistant
	case logv1.SessionType_SESSION_TYPE_SYSTEM:
		return shareddomain.SessionTypeSystem
	case logv1.SessionType_SESSION_TYPE_SUMMARY:
		return shareddomain.SessionTypeSummary
	case logv1.SessionType_SESSION_TYPE_FILE_HISTORY_SNAPSHOT:
		return shareddomain.SessionTypeFileHistorySnapshot
	case logv1.SessionType_SESSION_TYPE_QUEUE_OPERATION:
		return shareddomain.SessionTypeQueueOperation
	default:
		return shareddomain.SessionTypeUser
	}
}

func convertMessage(m *logv1.Message) *shareddomain.Message {
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

	content := m.GetContent()
	return &shareddomain.Message{
		ID:         m.GetId(),
		Type:       m.GetType(),
		Role:       m.GetRole(),
		Model:      m.GetModel(),
		StopReason: m.GetStopReason(),
		Usage:      usage,
		Content: &shareddomain.MessageContent{
			IsBlocks: false,
			Text:     &content,
		},
	}
}

// Compile-time interface verification.
var _ logv1connect.LogServiceHandler = (*LogGRPCHandler)(nil)
