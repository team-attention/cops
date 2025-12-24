package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	shareddomain "github.com/team-attention/cops/shared/domain"
	collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
	logger *slog.Logger
	client collectorv1connect.CollectorServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
	client := collectorv1connect.NewCollectorServiceClient(
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
	// Note: Collector API uses SendRecords, adapt batch to match
	req := &collectorv1.SendRecordsReq{
		Project: &collectorv1.ProjectMetadata{
			Id:         batch.ProjectID,
			Name:       batch.ProjectName,
			Path:       batch.ProjectPath,
			GitProject: batch.IsGitProject,
		},
		Records: convertRecords(batch.Records),
	}

	resp, err := c.client.SendRecords(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.RecordsReceived)),
	)

	return nil
}

func convertRecords(records []shareddomain.SessionRecord) []*collectorv1.SessionRecord {
	result := make([]*collectorv1.SessionRecord, len(records))
	for i, r := range records {
		result[i] = convertSessionRecord(r)
	}
	return result
}

func convertSessionRecord(r shareddomain.SessionRecord) *collectorv1.SessionRecord {
	record := &collectorv1.SessionRecord{
		Uuid:        r.UUID,
		ParentUuid:  r.ParentUUID,
		SessionId:   r.SessionID,
		Type:        string(r.Type),
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
		record.Role = r.Message.Role
		if r.Message.Content != nil && !r.Message.Content.IsBlocks && r.Message.Content.Text != nil {
			record.Content = *r.Message.Content.Text
		}

		if r.Message.Usage != nil {
			record.Usage = &collectorv1.UsageMetadata{
				InputTokens:              int32(r.Message.Usage.InputTokens),
				OutputTokens:             int32(r.Message.Usage.OutputTokens),
				CacheCreationInputTokens: int32(r.Message.Usage.CacheCreationInputTokens),
				CacheReadInputTokens:     int32(r.Message.Usage.CacheReadInputTokens),
				ServiceTier:              r.Message.Usage.ServiceTier,
			}

			if r.Message.Usage.CacheCreation != nil {
				record.Usage.CacheCreation = &collectorv1.CacheCreation{
					Ephemeral_5MInputTokens: int32(r.Message.Usage.CacheCreation.Ephemeral5mInputTokens),
					Ephemeral_1HInputTokens: int32(r.Message.Usage.CacheCreation.Ephemeral1hInputTokens),
				}
			}
		}
	}

	return record
}
