package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup/config"
	"github.com/team-attention/cops/daemon/internal/platform/setup/copsapi"
	shareddomain "github.com/team-attention/cops/shared/domain"
	collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect"
)

// Adapter implements APIClientPort using ConnectRPC.
type Adapter struct {
	logger *slog.Logger
	client collectorv1connect.CollectorServiceClient
}

// NewAdapter creates a new ConnectRPC API client adapter.
func NewAdapter(l *slog.Logger, apiClient *copsapi.APIClient, cfg *config.Config) *Adapter {
	client := collectorv1connect.NewCollectorServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &Adapter{
		logger: l.With(slog.String("name", "logprocessor.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of logs to the API server.
func (a *Adapter) SendLogs(ctx context.Context, batch domain.LogBatch) error {
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

	resp, err := a.client.SendRecords(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		a.logger.Warn("API returned failure")
	}

	a.logger.Debug("logs sent",
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
