package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
	shareddomain "github.com/team-attention/cops/shared/domain"
	logv1 "github.com/team-attention/cops/shared/gen/grpcstub/log/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/log/v1/logv1connect"
)

// Adapter implements APIClientPort using ConnectRPC.
type Adapter struct {
	logger *slog.Logger
	client logv1connect.LogServiceClient
}

// NewAdapter creates a new ConnectRPC API client adapter.
func NewAdapter(l *slog.Logger, cfg *config.Config) *Adapter {
	client := logv1connect.NewLogServiceClient(
		&http.Client{Timeout: cfg.API.Timeout},
		cfg.API.URL,
	)

	return &Adapter{
		logger: l.With(slog.String("name", "logprocessor.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of logs to the API server.
func (a *Adapter) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &logv1.SendLogsReq{
		Batch: convertBatch(batch),
	}

	resp, err := a.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		a.logger.Warn("API returned failure",
			slog.String("error", resp.Msg.ErrorMessage),
		)
	}

	a.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}

func convertBatch(batch domain.LogBatch) *logv1.LogBatch {
	records := make([]*logv1.SessionRecord, len(batch.Records))
	for i, r := range batch.Records {
		records[i] = convertSessionRecord(r)
	}

	return &logv1.LogBatch{
		Records:   records,
		DaemonId:  batch.DaemonID,
		CreatedAt: timestamppb.New(batch.CreatedAt),
	}
}

func convertSessionRecord(r shareddomain.SessionRecord) *logv1.SessionRecord {
	record := &logv1.SessionRecord{
		Uuid:        r.UUID,
		ParentUuid:  r.ParentUUID,
		SessionId:   r.SessionID,
		Type:        convertSessionType(r.Type),
		Timestamp:   timestamppb.New(r.Timestamp),
		Cwd:         r.CWD,
		GitBranch:   r.GitBranch,
		Version:     r.Version,
		UserType:    r.UserType,
		IsSidechain: r.IsSidechain,
		IsMeta:      r.IsMeta,
		Slug:        r.Slug,
		RequestId:   r.RequestID,
	}

	if r.Message != nil {
		record.Message = convertMessage(r.Message)
	}

	return record
}

func convertSessionType(t shareddomain.SessionType) logv1.SessionType {
	switch t {
	case shareddomain.SessionTypeUser:
		return logv1.SessionType_SESSION_TYPE_USER
	case shareddomain.SessionTypeAssistant:
		return logv1.SessionType_SESSION_TYPE_ASSISTANT
	case shareddomain.SessionTypeSystem:
		return logv1.SessionType_SESSION_TYPE_SYSTEM
	case shareddomain.SessionTypeSummary:
		return logv1.SessionType_SESSION_TYPE_SUMMARY
	case shareddomain.SessionTypeFileHistorySnapshot:
		return logv1.SessionType_SESSION_TYPE_FILE_HISTORY_SNAPSHOT
	case shareddomain.SessionTypeQueueOperation:
		return logv1.SessionType_SESSION_TYPE_QUEUE_OPERATION
	default:
		return logv1.SessionType_SESSION_TYPE_UNSPECIFIED
	}
}

func convertMessage(m *shareddomain.Message) *logv1.Message {
	msg := &logv1.Message{
		Id:         m.ID,
		Type:       m.Type,
		Role:       m.Role,
		Model:      m.Model,
		StopReason: m.StopReason,
	}

	if m.Content != nil && !m.Content.IsBlocks && m.Content.Text != nil {
		msg.Content = *m.Content.Text
	}

	if m.Usage != nil {
		msg.Usage = &logv1.Usage{
			InputTokens:              int32(m.Usage.InputTokens),
			OutputTokens:             int32(m.Usage.OutputTokens),
			CacheCreationInputTokens: int32(m.Usage.CacheCreationInputTokens),
			CacheReadInputTokens:     int32(m.Usage.CacheReadInputTokens),
			ServiceTier:              m.Usage.ServiceTier,
		}
	}

	return msg
}
