package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
	logger *slog.Logger
	client aggregationv1connect.AggregationServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
	client := aggregationv1connect.NewAggregationServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &APIClient{
		logger: l.With(slog.String("name", "log.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of logs to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			Records:   convertRecords(batch.Records),
			ProjectId: batch.ProjectID.String(),
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}

func convertRecords(records []shareddomain.SessionRecord) []*aggregationv1.SessionRecord {
	result := make([]*aggregationv1.SessionRecord, len(records))
	for i, r := range records {
		result[i] = convertSessionRecord(r)
	}
	return result
}

func convertSessionRecord(r shareddomain.SessionRecord) *aggregationv1.SessionRecord {
	record := &aggregationv1.SessionRecord{
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
		Message:     convertMessage(r.Message),
	}

	return record
}

func convertSessionType(t shareddomain.SessionType) aggregationv1.SessionType {
	switch t {
	case shareddomain.SessionTypeUser:
		return aggregationv1.SessionType_SESSION_TYPE_USER
	case shareddomain.SessionTypeAssistant:
		return aggregationv1.SessionType_SESSION_TYPE_ASSISTANT
	case shareddomain.SessionTypeSystem:
		return aggregationv1.SessionType_SESSION_TYPE_SYSTEM
	case shareddomain.SessionTypeSummary:
		return aggregationv1.SessionType_SESSION_TYPE_SUMMARY
	case shareddomain.SessionTypeFileHistorySnapshot:
		return aggregationv1.SessionType_SESSION_TYPE_FILE_HISTORY_SNAPSHOT
	case shareddomain.SessionTypeQueueOperation:
		return aggregationv1.SessionType_SESSION_TYPE_QUEUE_OPERATION
	default:
		return aggregationv1.SessionType_SESSION_TYPE_USER
	}
}

func convertMessage(m *shareddomain.Message) *aggregationv1.Message {
	if m == nil {
		return nil
	}

	msg := &aggregationv1.Message{
		Id:         m.ID,
		Type:       m.Type,
		Role:       m.Role,
		Model:      m.Model,
		StopReason: m.StopReason,
	}

	if m.Content != nil && !m.Content.IsBlocks && m.Content.Text != nil {
		msg.Text = *m.Content.Text
	}

	if m.Usage != nil {
		msg.Usage = &aggregationv1.Usage{
			InputTokens:              int32(m.Usage.InputTokens),
			OutputTokens:             int32(m.Usage.OutputTokens),
			CacheCreationInputTokens: int32(m.Usage.CacheCreationInputTokens),
			CacheReadInputTokens:     int32(m.Usage.CacheReadInputTokens),
			ServiceTier:              m.Usage.ServiceTier,
		}
	}

	return msg
}
