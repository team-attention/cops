package connectrpc

import (
	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	recordv1 "github.com/team-attention/cops/shared/gen/grpcstub/record/v1"
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
func toProtoRecords(records []shareddomain.Record) []*recordv1.Record {
	result := make([]*recordv1.Record, 0, len(records))
	for _, record := range records {
		protoRec := toProtoRecord(record)
		if protoRec != nil {
			result = append(result, protoRec)
		}
	}
	return result
}

// toProtoRecord converts a single domain Record to protobuf.
func toProtoRecord(r shareddomain.Record) *recordv1.Record {
	proto := &recordv1.Record{}

	// Set Type based on r.Type
	switch r.Type {
	case shareddomain.RecordTypeUser:
		proto.Type = recordv1.RecordType_RECORD_TYPE_USER
		if userRec, ok := r.Data.(*shareddomain.UserRecord); ok {
			proto.Data = &recordv1.Record_UserData{
				UserData: toProtoUserRecordData(userRec),
			}
		}
	case shareddomain.RecordTypeMessage:
		proto.Type = recordv1.RecordType_RECORD_TYPE_ASSISTANT
		if assistantRec, ok := r.Data.(*shareddomain.AssistantRecord); ok {
			proto.Data = &recordv1.Record_AssistantData{
				AssistantData: toProtoAssistantRecordData(assistantRec),
			}
		}
	case shareddomain.RecordTypeFileHistorySnapshot:
		proto.Type = recordv1.RecordType_RECORD_TYPE_FILE_HISTORY_SNAPSHOT
		if fileHistoryRec, ok := r.Data.(*shareddomain.FileHistorySnapshotRecord); ok {
			proto.Data = &recordv1.Record_FileHistorySnapshotData{
				FileHistorySnapshotData: toProtoFileHistorySnapshotRecordData(fileHistoryRec),
			}
		}
	default:
		proto.Type = recordv1.RecordType_RECORD_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoUserRecordData converts UserRecord to proto UserRecordData.
func toProtoUserRecordData(u *shareddomain.UserRecord) *recordv1.UserRecordData {
	// 1. Create base UserRecordData with metadata and other fields.
	data := &recordv1.UserRecordData{
		Metadata: &recordv1.MessageMetadata{
			ParentUuid:  "",
			IsSidechain: u.IsSidechain,
			UserType:    string(u.UserType),
			SessionId:   u.SessionID,
			Version:     u.Version,
			GitBranch:   u.GitBranch,
			Uuid:        u.UUID,
			Timestamp:   timestamppb.New(u.Timestamp),
		},
		Message: &recordv1.UserMessage{
			Role: string(u.Message.Role),
		},
		IsMeta: u.IsMeta,
	}

	// 2. Set ParentUuid if present.
	if u.ParentUUID != nil {
		data.Metadata.ParentUuid = *u.ParentUUID
	}

	// 3. Convert Content based on its type (string or []*UserMessageBlockContent).
	if u.Message.Content != nil {
		switch content := u.Message.Content.(type) {
		case string:
			// a. If string: Set Message.Content to UserMessage_Text.
			data.Message.Content = &recordv1.UserMessage_Text{
				Text: content,
			}
		case []*shareddomain.UserMessageBlockContent:
			// b. If []*UserMessageBlockContent: Convert to protobuf blocks.
			protoBlocks := make([]*recordv1.UserMessageBlockContent, len(content))
			for i, block := range content {
				protoBlocks[i] = toProtoUserMessageBlockContent(block)
			}
			// Set Message.Content to UserMessage_Blocks.
			data.Message.Content = &recordv1.UserMessage_Blocks{
				Blocks: &recordv1.UserMessageBlockContentList{
					Blocks: protoBlocks,
				},
			}
		}
	}

	// 4. Convert ThinkingMetadata if present.
	if u.ThinkingMetadata != nil {
		data.ThinkingMetadata = &recordv1.UserRecordThinkingMetadata{
			Level:    u.ThinkingMetadata.Level,
			Disabled: u.ThinkingMetadata.Disabled,
		}
		if len(u.ThinkingMetadata.Triggers) > 0 {
			data.ThinkingMetadata.Triggers = make([]*recordv1.UserRecordThinkingMetadataTrigger, len(u.ThinkingMetadata.Triggers))
			for i, trigger := range u.ThinkingMetadata.Triggers {
				data.ThinkingMetadata.Triggers[i] = &recordv1.UserRecordThinkingMetadataTrigger{
					Start: int32(trigger.Start),
					End:   int32(trigger.End),
					Text:  trigger.Text,
				}
			}
		}
	}

	// 5. Convert Todos if present.
	if len(u.Todos) > 0 {
		data.Todos = make([]*recordv1.UserRecordTodo, len(u.Todos))
		for i, todo := range u.Todos {
			data.Todos[i] = &recordv1.UserRecordTodo{
				Content:    todo.Content,
				Status:     todo.Status,
				ActiveForm: todo.ActiveForm,
			}
		}
	}

	// 6. Return the fully constructed data.
	return data
}

// toProtoUserMessageBlockContent converts a single domain UserMessageBlockContent to protobuf.
func toProtoUserMessageBlockContent(block *shareddomain.UserMessageBlockContent) *recordv1.UserMessageBlockContent {
	// 1. Create base protobuf block with type field.
	protoBlock := &recordv1.UserMessageBlockContent{
		Type: block.Type,
	}

	// 2. If Text is set, copy to proto.
	if block.Text != nil {
		protoBlock.Text = *block.Text
	}

	// 3. If Source is set (image block), convert to proto.
	if block.Source != nil {
		protoBlock.Source = &recordv1.UserMessageBlockContentSource{
			Type:      block.Source.Type,
			MediaType: block.Source.Media_type,
			Data:      block.Source.Data,
		}
	}

	// 4. If UserMessageBlockContentToolResult is set (tool_result block), convert to proto.
	if block.UserMessageBlockContentToolResult != nil {
		// Convert Content to string (may be string or other type).
		contentStr := ""
		if block.UserMessageBlockContentToolResult.Content != nil {
			switch c := block.UserMessageBlockContentToolResult.Content.(type) {
			case string:
				contentStr = c
			default:
				// Marshal non-string content to JSON.
				if jsonBytes, err := sonic.Marshal(c); err == nil {
					contentStr = string(jsonBytes)
				}
			}
		}
		protoBlock.ToolResult = &recordv1.UserMessageBlockContentToolResult{
			ToolUseId: block.UserMessageBlockContentToolResult.ToolUseID,
			Content:   contentStr,
		}
	}

	// 5. Return the converted block.
	return protoBlock
}

// toProtoAssistantRecordData converts AssistantRecord to proto AssistantRecordData.
func toProtoAssistantRecordData(a *shareddomain.AssistantRecord) *recordv1.AssistantRecordData {
	data := &recordv1.AssistantRecordData{
		Metadata: &recordv1.MessageMetadata{
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
		Message: &recordv1.AssistantMessage{
			Model: a.Message.Model,
			Id:    a.Message.ID,
			Type:  string(a.Message.Type),
			Role:  string(a.Message.Role),
			Usage: &recordv1.AssistantMessageUsage{
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
		data.Message.Content = make([]*recordv1.AssistantMessageContent, len(a.Message.Content))
		for i, content := range a.Message.Content {
			protoContent := &recordv1.AssistantMessageContent{
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
func toProtoFileHistorySnapshotRecordData(f *shareddomain.FileHistorySnapshotRecord) *recordv1.FileHistorySnapshotRecordData {
	data := &recordv1.FileHistorySnapshotRecordData{
		MessageId:         f.MessageID,
		IsSnapshotUpdate: f.IsSnapshotUpdate,
		Snapshot: &recordv1.FileHistorySnapshot{
			MessageId:          f.Snapshot.MessageID,
			TrackedFileBackups: make(map[string]*recordv1.FileHistorySnapshotTrackedBackup),
		},
	}

	for path, backup := range f.Snapshot.TrackedFileBackups {
		backupFileName := ""
		if backup.BackupFileName != nil {
			backupFileName = *backup.BackupFileName
		}
		data.Snapshot.TrackedFileBackups[path] = &recordv1.FileHistorySnapshotTrackedBackup{
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
