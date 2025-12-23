package connectrpc

import (
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoTokenUsageSummary converts repository token usage to protobuf.
func toProtoTokenUsageSummary(usage repository.TokenUsageSummary) *dashboardv1.TokenUsageSummary {
	return &dashboardv1.TokenUsageSummary{
		TotalInputTokens:         usage.TotalInputTokens,
		TotalOutputTokens:        usage.TotalOutputTokens,
		TotalCacheCreationTokens: usage.TotalCacheCreationTokens,
		TotalCacheReadTokens:     usage.TotalCacheReadTokens,
	}
}

// toProtoProjectSummary converts repository project summary to protobuf.
func toProtoProjectSummary(p repository.ProjectSummary) *dashboardv1.ProjectSummary {
	return &dashboardv1.ProjectSummary{
		Id:           p.ID,
		Name:         p.Name,
		Path:         p.Path,
		GitBranch:    p.GitBranch,
		SessionCount: p.SessionCount,
		Usage:        toProtoTokenUsageSummary(p.Usage),
		LastActivity: timestamppb.New(p.LastActivity),
	}
}

// toProtoProjectDetail converts repository project detail to protobuf.
func toProtoProjectDetail(p *repository.ProjectDetail) *dashboardv1.ProjectDetail {
	return &dashboardv1.ProjectDetail{
		Id:           p.ID,
		Name:         p.Name,
		Path:         p.Path,
		GitBranch:    p.GitBranch,
		Worktrees:    p.Worktrees,
		SessionCount: p.SessionCount,
		Usage:        toProtoTokenUsageSummary(p.Usage),
		CreatedAt:    timestamppb.New(p.CreatedAt),
		LastActivity: timestamppb.New(p.LastActivity),
	}
}

// toProtoSessionSummary converts repository session summary to protobuf.
func toProtoSessionSummary(s repository.SessionSummary) *dashboardv1.SessionSummary {
	return &dashboardv1.SessionSummary{
		Id:           s.ID,
		ProjectId:    s.ProjectID,
		GitBranch:    s.GitBranch,
		MessageCount: s.MessageCount,
		Usage:        toProtoTokenUsageSummary(s.Usage),
		StartedAt:    timestamppb.New(s.StartedAt),
		EndedAt:      timestamppb.New(s.EndedAt),
	}
}

// toProtoSessionDetail converts repository session detail to protobuf.
func toProtoSessionDetail(s *repository.SessionDetail) *dashboardv1.SessionDetail {
	records := make([]*collectorv1.SessionRecord, len(s.Records))
	for i, r := range s.Records {
		records[i] = toProtoSessionRecord(r)
	}

	return &dashboardv1.SessionDetail{
		Id:        s.ID,
		ProjectId: s.ProjectID,
		GitBranch: s.GitBranch,
		Cwd:       s.CWD,
		Version:   s.Version,
		Usage:     toProtoTokenUsageSummary(s.Usage),
		StartedAt: timestamppb.New(s.StartedAt),
		EndedAt:   timestamppb.New(s.EndedAt),
		Records:   records,
	}
}

// toProtoSessionRecord converts domain session record to protobuf.
func toProtoSessionRecord(r shareddomain.SessionRecord) *collectorv1.SessionRecord {
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

	// Add message data if available
	if r.Message != nil {
		record.Role = r.Message.Role
		if r.Message.Content != nil && !r.Message.Content.IsBlocks && r.Message.Content.Text != nil {
			record.Content = *r.Message.Content.Text
		}

		// Add usage metadata
		if r.Message.Usage != nil {
			record.Usage = &collectorv1.UsageMetadata{
				InputTokens:              int32(r.Message.Usage.InputTokens),
				OutputTokens:             int32(r.Message.Usage.OutputTokens),
				CacheCreationInputTokens: int32(r.Message.Usage.CacheCreationInputTokens),
				CacheReadInputTokens:     int32(r.Message.Usage.CacheReadInputTokens),
				ServiceTier:              r.Message.Usage.ServiceTier,
			}

			// Add cache creation metadata if available
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

// toProtoPagination converts pagination metadata to protobuf.
func toProtoPagination(currentPage, pageSize, totalPages int32, totalCount int64) *dashboardv1.PaginationResponse {
	return &dashboardv1.PaginationResponse{
		CurrentPage: currentPage,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
}
