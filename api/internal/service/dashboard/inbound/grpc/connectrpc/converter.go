package connectrpc

import (
	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	transcriptv1 "github.com/team-attention/cops/shared/gen/grpcstub/transcript/v1"
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
	transcripts := toProtoTranscripts(s.Transcripts)

	return &dashboardv1.SessionDetail{
		Id:          s.SessionBase.ID,
		ProjectId:   s.SessionBase.ProjectID,
		GitBranch:   s.SessionBase.GitBranch,
		Version:     s.Version,
		Usage:       toProtoTokenUsageSummary(s.Usage),
		StartedAt:   timestamppb.New(s.SessionBase.StartedAt),
		EndedAt:     timestamppb.New(s.SessionBase.EndedAt),
		Transcripts: transcripts,
	}
}

// toProtoTranscripts converts domain Transcripts to protobuf Transcripts.
func toProtoTranscripts(transcripts []shareddomain.Transcript) []*transcriptv1.Transcript {
	result := make([]*transcriptv1.Transcript, 0, len(transcripts))
	for _, transcript := range transcripts {
		protoTranscript := toProtoTranscript(transcript)
		if protoTranscript != nil {
			result = append(result, protoTranscript)
		}
	}
	return result
}

// toProtoTranscript converts a single domain Transcript to protobuf Transcript.
func toProtoTranscript(t shareddomain.Transcript) *transcriptv1.Transcript {
	proto := &transcriptv1.Transcript{}

	// Set Type based on t.Type using type switch on Data
	switch data := t.Data.(type) {
	case *shareddomain.UserTranscript:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_USER
		proto.Data = &transcriptv1.Transcript_UserData{
			UserData: toProtoUserTranscriptData(data),
		}
	case *shareddomain.AssistantTranscript:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_ASSISTANT
		proto.Data = &transcriptv1.Transcript_AssistantData{
			AssistantData: toProtoAssistantTranscriptData(data),
		}
	case *shareddomain.FileHistorySnapshotTranscript:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_FILE_HISTORY_SNAPSHOT
		proto.Data = &transcriptv1.Transcript_FileHistorySnapshotData{
			FileHistorySnapshotData: toProtoFileHistorySnapshotTranscriptData(data),
		}
	case *shareddomain.SystemTranscript:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_SYSTEM
		proto.Data = &transcriptv1.Transcript_SystemData{
			SystemData: toProtoSystemTranscriptData(data),
		}
	case *shareddomain.SummaryTranscript:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_SUMMARY
		proto.Data = &transcriptv1.Transcript_SummaryData{
			SummaryData: toProtoSummaryTranscriptData(data),
		}
	default:
		proto.Type = transcriptv1.TranscriptType_TRANSCRIPT_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoUserTranscriptData converts UserTranscript to proto UserTranscriptData.
func toProtoUserTranscriptData(u *shareddomain.UserTranscript) *transcriptv1.UserTranscriptData {
	// 1. Create base UserTranscriptData with metadata and other fields.
	data := &transcriptv1.UserTranscriptData{
		Metadata: toProtoTreeNodeMeta(u.TreeNodeMeta),
		Message: &transcriptv1.UserMessage{
			Role: u.Message.Role,
		},
		IsMeta: u.IsMeta,
	}

	// 2. Convert Content based on its type (string or []*UserMessageBlock).
	if u.Message.Content != nil {
		switch content := u.Message.Content.(type) {
		case string:
			// a. If string: Set Message.Content to UserMessage_Text.
			data.Message.Content = &transcriptv1.UserMessage_Text{
				Text: content,
			}
		case []*shareddomain.UserMessageBlock:
			// b. If []*UserMessageBlock: Convert to protobuf blocks.
			protoBlocks := make([]*transcriptv1.UserMessageBlock, len(content))
			for i, block := range content {
				protoBlocks[i] = toProtoUserMessageBlock(block)
			}
			// Set Message.Content to UserMessage_Blocks.
			data.Message.Content = &transcriptv1.UserMessage_Blocks{
				Blocks: &transcriptv1.UserMessageBlockList{
					Blocks: protoBlocks,
				},
			}
		}
	}

	// 3. Convert ThinkingMetadata if present.
	if u.ThinkingMetadata != nil {
		data.ThinkingMetadata = &transcriptv1.ThinkingMetadata{
			Level:    u.ThinkingMetadata.Level,
			Disabled: u.ThinkingMetadata.Disabled,
		}
		if len(u.ThinkingMetadata.Triggers) > 0 {
			data.ThinkingMetadata.Triggers = make([]*transcriptv1.ThinkingTrigger, len(u.ThinkingMetadata.Triggers))
			for i, trigger := range u.ThinkingMetadata.Triggers {
				data.ThinkingMetadata.Triggers[i] = &transcriptv1.ThinkingTrigger{
					Start: int32(trigger.Start),
					End:   int32(trigger.End),
					Text:  trigger.Text,
				}
			}
		}
	}

	// 4. Convert Todos if present.
	if len(u.Todos) > 0 {
		data.Todos = make([]*transcriptv1.Todo, len(u.Todos))
		for i, todo := range u.Todos {
			data.Todos[i] = &transcriptv1.Todo{
				Content:    todo.Content,
				Status:     todo.Status,
				ActiveForm: todo.ActiveForm,
			}
		}
	}

	// 5. Convert ToolUseResult if present.
	if u.ToolUseResult != nil {
		data.ToolUseResult = &transcriptv1.ToolUseResult{
			Success: u.ToolUseResult.Success,
		}
		if u.ToolUseResult.CommandName != nil {
			data.ToolUseResult.CommandName = *u.ToolUseResult.CommandName
		}
		if u.ToolUseResult.Model != nil {
			data.ToolUseResult.Model = *u.ToolUseResult.Model
		}
	}

	// 6. Set SourceToolAssistantUuid if present.
	if u.SourceToolAssistantUUID != nil {
		data.SourceToolAssistantUuid = *u.SourceToolAssistantUUID
	}

	// 7. Return the fully constructed data.
	return data
}

// toProtoUserMessageBlock converts a single domain UserMessageBlock to protobuf.
func toProtoUserMessageBlock(block *shareddomain.UserMessageBlock) *transcriptv1.UserMessageBlock {
	// 1. Create base protobuf block with type field.
	protoBlock := &transcriptv1.UserMessageBlock{
		Type: block.Type,
	}

	// 2. If Text is set, copy to proto.
	if block.Text != nil {
		protoBlock.Text = *block.Text
	}

	// 3. If Source is set (image block), convert to proto.
	if block.Source != nil {
		protoBlock.Source = &transcriptv1.ImageSource{
			Type:      block.Source.Type,
			MediaType: block.Source.MediaType,
			Data:      block.Source.Data,
		}
	}

	// 4. If ToolUseID is set (tool_result block), convert to proto.
	if block.ToolUseID != nil {
		// Convert Content to string (may be string or other type).
		contentStr := ""
		if block.Content != nil {
			switch c := block.Content.(type) {
			case string:
				contentStr = c
			default:
				// Marshal non-string content to JSON.
				if jsonBytes, err := sonic.Marshal(c); err == nil {
					contentStr = string(jsonBytes)
				}
			}
		}
		protoBlock.ToolResult = &transcriptv1.ToolResultContent{
			ToolUseId: *block.ToolUseID,
			Content:   contentStr,
		}
	}

	// 5. Return the converted block.
	return protoBlock
}

// toProtoAssistantTranscriptData converts AssistantTranscript to proto AssistantTranscriptData.
func toProtoAssistantTranscriptData(a *shareddomain.AssistantTranscript) *transcriptv1.AssistantTranscriptData {
	data := &transcriptv1.AssistantTranscriptData{
		Metadata:  toProtoTreeNodeMeta(a.TreeNodeMeta),
		RequestId: a.RequestID,
		Message: &transcriptv1.AssistantMessage{
			Model: a.Message.Model,
			Id:    a.Message.ID,
			Type:  a.Message.Type,
			Role:  a.Message.Role,
		},
	}

	// Set usage if present
	if a.Message.Usage != nil {
		data.Message.Usage = &transcriptv1.AssistantUsage{
			InputTokens:              int32(a.Message.Usage.InputTokens),
			OutputTokens:             int32(a.Message.Usage.OutputTokens),
			CacheCreationInputTokens: int32(a.Message.Usage.CacheCreationInputTokens),
			CacheReadInputTokens:     int32(a.Message.Usage.CacheReadInputTokens),
			ServiceTier:              a.Message.Usage.ServiceTier,
		}
		// Set cache creation if present
		if a.Message.Usage.CacheCreation != nil {
			data.Message.Usage.CacheCreation = &transcriptv1.CacheCreation{
				Ephemeral_5MInputTokens: int32(a.Message.Usage.CacheCreation.Ephemeral5mInputTokens),
				Ephemeral_1HInputTokens: int32(a.Message.Usage.CacheCreation.Ephemeral1hInputTokens),
			}
		}
	}

	if a.Message.StopReason != nil {
		data.Message.StopReason = *a.Message.StopReason
	}

	if a.Message.StopSequence != nil {
		data.Message.StopSequence = *a.Message.StopSequence
	}

	// Convert content blocks
	if len(a.Message.Content) > 0 {
		data.Message.Content = make([]*transcriptv1.AssistantContentBlock, len(a.Message.Content))
		for i, content := range a.Message.Content {
			protoContent := &transcriptv1.AssistantContentBlock{
				Type: content.Type,
			}

			switch content.Type {
			case "text":
				if content.Text != nil {
					protoContent.Text = *content.Text
				}
			case "thinking":
				if content.Thinking != nil {
					protoContent.Thinking = *content.Thinking
				}
				if content.Signature != nil {
					protoContent.Signature = *content.Signature
				}
			case "tool_use":
				if content.ID != nil {
					protoContent.Id = *content.ID
				}
				if content.Name != nil {
					protoContent.Name = *content.Name
				}
				if content.Input != nil {
					if inputBytes, err := sonic.Marshal(content.Input); err == nil {
						protoContent.InputJson = string(inputBytes)
					}
				}
			}

			data.Message.Content[i] = protoContent
		}
	}

	return data
}

// toProtoFileHistorySnapshotTranscriptData converts FileHistorySnapshotTranscript to proto.
func toProtoFileHistorySnapshotTranscriptData(f *shareddomain.FileHistorySnapshotTranscript) *transcriptv1.FileHistorySnapshotTranscriptData {
	data := &transcriptv1.FileHistorySnapshotTranscriptData{
		MessageId:        f.MessageID,
		IsSnapshotUpdate: f.IsSnapshotUpdate,
		Snapshot: &transcriptv1.FileSnapshot{
			MessageId:          f.Snapshot.MessageID,
			TrackedFileBackups: make(map[string]*transcriptv1.FileBackup),
			Timestamp:          timestamppb.New(f.Snapshot.Timestamp),
		},
	}

	for path, backup := range f.Snapshot.TrackedFileBackups {
		backupFileName := ""
		if backup.BackupFileName != nil {
			backupFileName = *backup.BackupFileName
		}
		data.Snapshot.TrackedFileBackups[path] = &transcriptv1.FileBackup{
			BackupFileName: backupFileName,
			Version:        int32(backup.Version),
			BackupTime:     timestamppb.New(backup.BackupTime),
		}
	}

	return data
}

// toProtoSystemTranscriptData converts SystemTranscript to proto SystemTranscriptData.
func toProtoSystemTranscriptData(s *shareddomain.SystemTranscript) *transcriptv1.SystemTranscriptData {
	data := &transcriptv1.SystemTranscriptData{
		Metadata:   toProtoTreeNodeMeta(s.TreeNodeMeta),
		DurationMs: int32(s.DurationMs),
		IsMeta:     s.IsMeta,
	}

	// Map subtype string to enum
	switch s.Subtype {
	case shareddomain.SystemTranscriptSubtypeTurnDuration:
		data.Subtype = transcriptv1.SystemTranscriptSubtype_SYSTEM_TRANSCRIPT_SUBTYPE_TURN_DURATION
	default:
		data.Subtype = transcriptv1.SystemTranscriptSubtype_SYSTEM_TRANSCRIPT_SUBTYPE_UNSPECIFIED
	}

	return data
}

// toProtoSummaryTranscriptData converts SummaryTranscript to proto SummaryTranscriptData.
func toProtoSummaryTranscriptData(s *shareddomain.SummaryTranscript) *transcriptv1.SummaryTranscriptData {
	return &transcriptv1.SummaryTranscriptData{
		Summary:  s.Summary,
		LeafUuid: s.LeafUUID,
	}
}

// toProtoTreeNodeMeta converts domain TreeNodeMeta to proto TreeNodeMeta.
func toProtoTreeNodeMeta(m shareddomain.TreeNodeMeta) *transcriptv1.TreeNodeMeta {
	proto := &transcriptv1.TreeNodeMeta{
		Uuid:        m.UUID,
		SessionId:   m.SessionID,
		Timestamp:   timestamppb.New(m.Timestamp),
		Version:     m.Version,
		Cwd:         m.CWD,
		GitBranch:   m.GitBranch,
		Slug:        m.Slug,
		UserType:    m.UserType,
		IsSidechain: m.IsSidechain,
	}

	// Set ParentUuid if present
	if m.ParentUUID != nil {
		proto.ParentUuid = *m.ParentUUID
	}

	return proto
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
