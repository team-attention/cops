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
	records := toProtoRecords(s.Records)

	return &dashboardv1.SessionDetail{
		Id:        s.SessionBase.ID,
		ProjectId: s.SessionBase.ProjectID,
		GitBranch: s.SessionBase.GitBranch,
		Version:   s.Version,
		Usage:     toProtoTokenUsageSummary(s.Usage),
		StartedAt: timestamppb.New(s.SessionBase.StartedAt),
		EndedAt:   timestamppb.New(s.SessionBase.EndedAt),
		Records:   records,
	}
}

// toProtoRecords converts domain Records to protobuf Records.
func toProtoRecords(records []shareddomain.Record) []*aggregationv1.Record {
	result := make([]*aggregationv1.Record, 0, len(records))
	for _, record := range records {
		protoRec := toProtoRecord(record)
		if protoRec != nil {
			result = append(result, protoRec)
		}
	}
	return result
}

// toProtoRecord converts a single domain Record to protobuf.
func toProtoRecord(r shareddomain.Record) *aggregationv1.Record {
	proto := &aggregationv1.Record{}

	// Set Type based on r.Type
	switch r.Type {
	case shareddomain.RecordTypeUser:
		proto.Type = aggregationv1.RecordType_RECORD_TYPE_USER
		if userRec, ok := r.Data.(*shareddomain.UserRecord); ok {
			proto.Data = &aggregationv1.Record_UserData{
				UserData: toProtoUserRecordData(userRec),
			}
		}
	case shareddomain.RecordTypeMessage:
		proto.Type = aggregationv1.RecordType_RECORD_TYPE_ASSISTANT
		if assistantRec, ok := r.Data.(*shareddomain.AssistantRecord); ok {
			proto.Data = &aggregationv1.Record_AssistantData{
				AssistantData: toProtoAssistantRecordData(assistantRec),
			}
		}
	case shareddomain.RecordTypeFileHistorySnapshot:
		proto.Type = aggregationv1.RecordType_RECORD_TYPE_FILE_HISTORY_SNAPSHOT
		if fileHistoryRec, ok := r.Data.(*shareddomain.FileHistorySnapshotRecord); ok {
			proto.Data = &aggregationv1.Record_FileHistorySnapshotData{
				FileHistorySnapshotData: toProtoFileHistorySnapshotRecordData(fileHistoryRec),
			}
		}
	default:
		proto.Type = aggregationv1.RecordType_RECORD_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoUserRecordData converts UserRecord to proto UserRecordData.
func toProtoUserRecordData(u *shareddomain.UserRecord) *aggregationv1.UserRecordData {
	data := &aggregationv1.UserRecordData{
		Metadata: &aggregationv1.MessageMetadata{
			ParentUuid:  "",
			IsSidechain: u.IsSidechain,
			UserType:    string(u.UserType),
			SessionId:   u.SessionID,
			Version:     u.Version,
			GitBranch:   u.GitBranch,
			Uuid:        u.UUID,
			Timestamp:   timestamppb.New(u.Timestamp),
		},
		Message: &aggregationv1.UserMessage{
			Role:    string(u.Message.Role),
			Content: u.Message.Content,
		},
		IsMeta: u.IsMeta,
	}

	if u.ParentUUID != nil {
		data.Metadata.ParentUuid = *u.ParentUUID
	}

	if u.ThinkingMetadata != nil {
		data.ThinkingMetadata = &aggregationv1.UserRecordThinkingMetadata{
			Level:    u.ThinkingMetadata.Level,
			Disabled: u.ThinkingMetadata.Disabled,
		}
		if len(u.ThinkingMetadata.Triggers) > 0 {
			data.ThinkingMetadata.Triggers = make([]*aggregationv1.UserRecordThinkingMetadataTrigger, len(u.ThinkingMetadata.Triggers))
			for i, trigger := range u.ThinkingMetadata.Triggers {
				data.ThinkingMetadata.Triggers[i] = &aggregationv1.UserRecordThinkingMetadataTrigger{
					Start: int32(trigger.Start),
					End:   int32(trigger.End),
					Text:  trigger.Text,
				}
			}
		}
	}

	if len(u.Todos) > 0 {
		data.Todos = make([]*aggregationv1.UserRecordTodo, len(u.Todos))
		for i, todo := range u.Todos {
			data.Todos[i] = &aggregationv1.UserRecordTodo{
				Content:    todo.Content,
				Status:     todo.Status,
				ActiveForm: todo.ActiveForm,
			}
		}
	}

	return data
}

// toProtoAssistantRecordData converts AssistantRecord to proto AssistantRecordData.
func toProtoAssistantRecordData(a *shareddomain.AssistantRecord) *aggregationv1.AssistantRecordData {
	data := &aggregationv1.AssistantRecordData{
		Metadata: &aggregationv1.MessageMetadata{
			ParentUuid:  "",
			IsSidechain: a.IsSidechain,
			UserType:    string(a.UserType),
			SessionId:   a.SessionID,
			Version:     a.Version,
			GitBranch:   a.GitBranch,
			Uuid:        a.UUID,
			Timestamp:   timestamppb.New(a.Timestamp),
		},
		RequestId: a.RequestID,
		Message: &aggregationv1.AssistantMessage{
			Model: a.Message.Model,
			Id:    a.Message.ID,
			Type:  string(a.Message.Type),
			Role:  string(a.Message.Role),
			Usage: &aggregationv1.AssistantMessageUsage{
				InputTokens:              int32(a.Message.Usage.InputTokens),
				OutputTokens:             int32(a.Message.Usage.OutputTokens),
				CacheCreationInputTokens: int32(a.Message.Usage.CacheCreationInputTokens),
				CacheReadInputTokens:     int32(a.Message.Usage.CacheReadInputTokens),
				ServiceTier:              a.Message.Usage.ServiceTier,
			},
		},
	}

	if a.ParentUUID != nil {
		data.Metadata.ParentUuid = *a.ParentUUID
	}

	if a.Message.StopReason != nil {
		data.Message.StopReason = *a.Message.StopReason
	}

	if a.Message.StopSequence != nil {
		data.Message.StopSequence = int32(*a.Message.StopSequence)
	}

	// Convert content blocks
	if len(a.Message.Content) > 0 {
		data.Message.Content = make([]*aggregationv1.AssistantMessageContent, len(a.Message.Content))
		for i, content := range a.Message.Content {
			protoContent := &aggregationv1.AssistantMessageContent{
				Type: string(content.Type),
			}

			switch content.Type {
			case shareddomain.AssistantMessageContentTypeText:
				if content.Text != nil {
					protoContent.Text = *content.Text
				}
			case shareddomain.AssistantMessageContentTypeTool:
				if content.Thinking != nil {
					protoContent.Thinking = *content.Thinking
				}
			case shareddomain.AssistantMessageContentTypeToolUse:
				if content.AssistantMessageToolUseContent != nil {
					protoContent.ToolUseId = content.ID
					protoContent.ToolUseName = content.Name
					if inputBytes, err := sonic.Marshal(content.Input); err == nil {
						protoContent.ToolUseInputJson = string(inputBytes)
					}
				}
			}

			data.Message.Content[i] = protoContent
		}
	}

	return data
}

// toProtoFileHistorySnapshotRecordData converts FileHistorySnapshotRecord to proto.
func toProtoFileHistorySnapshotRecordData(f *shareddomain.FileHistorySnapshotRecord) *aggregationv1.FileHistorySnapshotRecordData {
	data := &aggregationv1.FileHistorySnapshotRecordData{
		MessageId:         f.MessageID,
		IsSnapshotUpdate: f.IsSnapshotUpdate,
		Snapshot: &aggregationv1.FileHistorySnapshot{
			MessageId:          f.Snapshot.MessageID,
			TrackedFileBackups: make(map[string]*aggregationv1.FileHistorySnapshotTrackedBackup),
		},
	}

	for path, backup := range f.Snapshot.TrackedFileBackups {
		backupFileName := ""
		if backup.BackupFileName != nil {
			backupFileName = *backup.BackupFileName
		}
		data.Snapshot.TrackedFileBackups[path] = &aggregationv1.FileHistorySnapshotTrackedBackup{
			BackupFileName: backupFileName,
			Version:        int32(backup.Version),
			BackupTime:     timestamppb.New(backup.BackupTime),
		}
	}

	return data
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
