package connectrpc

import (
	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
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
		Id:           string(p.ProjectAbstract.ID),
		Name:         p.ProjectAbstract.Name,
		Path:         p.ProjectAbstract.Path,
		SessionCount: p.ProjectAggregation.SessionCount,
		Usage:        toProtoTokenUsageSummary(p.ProjectAggregation.Usage),
		LastActivity: timestamppb.New(p.ProjectAggregation.LastActivity),
	}
}

// toProtoProjectDetail converts repository project detail to protobuf.
func toProtoProjectDetail(p *repository.ProjectDetail) *dashboardv1.ProjectDetail {
	return &dashboardv1.ProjectDetail{
		Id:           string(p.Project.ID),
		Name:         p.Project.Name,
		Path:         p.Project.Path,
		SessionCount: p.ProjectAggregation.SessionCount,
		Usage:        toProtoTokenUsageSummary(p.ProjectAggregation.Usage),
		CreatedAt:    timestamppb.New(p.Project.RegisteredAt),
		LastActivity: timestamppb.New(p.ProjectAggregation.LastActivity),
	}
}

// toProtoSessionSummary converts repository session summary to protobuf.
func toProtoSessionSummary(s repository.SessionSummary) *dashboardv1.SessionSummary {
	return &dashboardv1.SessionSummary{
		Id:           s.SessionBase.ID,
		ProjectId:    s.SessionBase.ProjectID,
		GitBranch:    s.SessionBase.GitBranch,
		MessageCount: s.MessageCount,
		Usage:        toProtoTokenUsageSummary(s.Usage),
		StartedAt:    timestamppb.New(s.SessionBase.StartedAt),
		EndedAt:      timestamppb.New(s.SessionBase.EndedAt),
	}
}

// toProtoSessionDetail converts repository session detail to protobuf.
func toProtoSessionDetail(s *repository.SessionDetail) *dashboardv1.SessionDetail {
	records := make([]*aggregationv1.SessionRecord, len(s.Records))
	for i, r := range s.Records {
		records[i] = toProtoSessionRecord(r)
	}

	return &dashboardv1.SessionDetail{
		Id:        s.SessionBase.ID,
		ProjectId: s.SessionBase.ProjectID,
		GitBranch: s.SessionBase.GitBranch,
		Cwd:       s.CWD,
		Version:   s.Version,
		Usage:     toProtoTokenUsageSummary(s.Usage),
		StartedAt: timestamppb.New(s.SessionBase.StartedAt),
		EndedAt:   timestamppb.New(s.SessionBase.EndedAt),
		Records:   records,
	}
}

// toProtoSessionRecord converts domain session record to protobuf.
func toProtoSessionRecord(r shareddomain.SessionRecord) *aggregationv1.SessionRecord {
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

	if m.Content != nil {
		if m.Content.IsBlocks {
			msg.ContentBlocks = convertContentBlocks(m.Content.Blocks)
		} else if m.Content.Text != nil {
			msg.Text = *m.Content.Text
		}
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

// toProtoPagination converts pagination metadata to protobuf.
func toProtoPagination(currentPage, pageSize, totalPages int32, totalCount int64) *dashboardv1.PaginationRes {
	return &dashboardv1.PaginationRes{
		CurrentPage: currentPage,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
}

// convertContentBlocks converts domain ContentBlocks to protobuf ContentBlocks.
func convertContentBlocks(blocks []shareddomain.ContentBlock) []*aggregationv1.ContentBlock {
	if len(blocks) == 0 {
		return nil
	}

	result := make([]*aggregationv1.ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		pb := convertContentBlock(block)
		if pb != nil {
			result = append(result, pb)
		}
	}
	return result
}

// convertContentBlock converts a single domain ContentBlock to protobuf.
func convertContentBlock(block shareddomain.ContentBlock) *aggregationv1.ContentBlock {
	switch b := block.(type) {
	case *shareddomain.TextContentBlock:
		return &aggregationv1.ContentBlock{
			Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TEXT,
			Block: &aggregationv1.ContentBlock_Text{
				Text: &aggregationv1.TextContentBlock{
					Text: b.Text,
				},
			},
		}
	case *shareddomain.ToolUseContentBlock:
		inputJSON := ""
		if b.Input != nil {
			if bytes, err := sonic.Marshal(b.Input); err == nil {
				inputJSON = string(bytes)
			}
		}
		return &aggregationv1.ContentBlock{
			Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_USE,
			Block: &aggregationv1.ContentBlock_ToolUse{
				ToolUse: &aggregationv1.ToolUseContentBlock{
					Id:        b.ID,
					Name:      b.Name,
					InputJson: inputJSON,
				},
			},
		}
	case *shareddomain.ToolResultContentBlock:
		return &aggregationv1.ContentBlock{
			Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_RESULT,
			Block: &aggregationv1.ContentBlock_ToolResult{
				ToolResult: &aggregationv1.ToolResultContentBlock{
					ToolUseId: b.ToolUseID,
					Content:   b.Content,
					IsError:   b.IsError,
				},
			},
		}
	case *shareddomain.ThinkingContentBlock:
		return &aggregationv1.ContentBlock{
			Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING,
			Block: &aggregationv1.ContentBlock_Thinking{
				Thinking: &aggregationv1.ThinkingContentBlock{
					Thinking:  b.Thinking,
					Signature: b.Signature,
				},
			},
		}
	default:
		return nil
	}
}
