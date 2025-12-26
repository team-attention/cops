package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/team-attention/cops/cli/internal/platform/setup/config"
	"github.com/team-attention/cops/cli/internal/platform/setup/httpclient"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/shared/domain"
	collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect"
)

// CollectorClient implements CollectorPort using ConnectRPC.
type CollectorClient struct {
	logger *slog.Logger
	client collectorv1connect.CollectorServiceClient
}

// NewCollectorClient creates a new ConnectRPC collector client.
func NewCollectorClient(l *slog.Logger, cfg *config.Config, httpClient *httpclient.CollectorHTTPClient) *CollectorClient {
	logger := l.With(slog.String("name", "tracking.api.connectrpc"))

	client := collectorv1connect.NewCollectorServiceClient(
		httpClient.StandardHTTPClient(),
		cfg.Collector.URL,
	)

	return &CollectorClient{
		logger: logger,
		client: client,
	}
}

// SendRecords sends session records to the collector server.
func (c *CollectorClient) SendRecords(ctx context.Context, project *domain.Project, records []*domain.SessionRecord) error {
	protoRecords := make([]*collectorv1.SessionRecord, 0, len(records))

	for _, r := range records {
		protoRecord := &collectorv1.SessionRecord{
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
			protoRecord.Role = r.Message.Role
			// Handle content - use direct field access
			if r.Message.Content != nil && r.Message.Content.Text != nil {
				protoRecord.Content = *r.Message.Content.Text
			}

			// Usage is nested inside Message
			if r.Message.Usage != nil {
				usage := r.Message.Usage
				protoRecord.Usage = &collectorv1.UsageMetadata{
					InputTokens:              int32(usage.InputTokens),
					OutputTokens:             int32(usage.OutputTokens),
					CacheCreationInputTokens: int32(usage.CacheCreationInputTokens),
					CacheReadInputTokens:     int32(usage.CacheReadInputTokens),
					ServiceTier:              usage.ServiceTier,
				}
				if usage.CacheCreation != nil {
					protoRecord.Usage.CacheCreation = &collectorv1.CacheCreation{
						Ephemeral_5MInputTokens: int32(usage.CacheCreation.Ephemeral5mInputTokens),
						Ephemeral_1HInputTokens: int32(usage.CacheCreation.Ephemeral1hInputTokens),
					}
				}
			}
		}

		protoRecords = append(protoRecords, protoRecord)
	}

	req := connect.NewRequest(&collectorv1.SendRecordsReq{
		Project: &collectorv1.ProjectMetadata{
			Id:         project.ID.String(),
			Name:       project.Name,
			Path:       project.Path,
			GitProject: project.IsGitProject,
		},
		Records: protoRecords,
	})

	resp, err := c.client.SendRecords(ctx, req)
	if err != nil {
		c.logger.Error("failed to send records", slog.Any("error", err))
		return err
	}

	c.logger.Info("records sent successfully",
		slog.Int("count", int(resp.Msg.RecordsReceived)))

	return nil
}

// Compile-time interface verification
var _ api.CollectorPort = (*CollectorClient)(nil)
