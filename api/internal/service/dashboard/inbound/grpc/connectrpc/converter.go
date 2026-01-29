package connectrpc

import (
	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	session "github.com/team-attention/cops/shared/domain/v2"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	sessionv1 "github.com/team-attention/cops/shared/gen/grpcstub/session/v1"
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
	sessions := toProtoSessions(s.Sessions)

	return &dashboardv1.SessionDetail{
		Id:        s.SessionBase.ID,
		ProjectId: s.SessionBase.ProjectID,
		GitBranch: s.SessionBase.GitBranch,
		Version:   s.Version,
		Usage:     toProtoTokenUsageSummary(s.Usage),
		StartedAt: timestamppb.New(s.SessionBase.StartedAt),
		EndedAt:   timestamppb.New(s.SessionBase.EndedAt),
		Sessions:  sessions,
	}
}

// toProtoSessions converts domain Sessions to protobuf Sessions.
func toProtoSessions(sessions []*session.Session) []*sessionv1.Session {
	result := make([]*sessionv1.Session, 0, len(sessions))
	for _, s := range sessions {
		protoSession := toProtoSession(s)
		if protoSession != nil {
			result = append(result, protoSession)
		}
	}
	return result
}

// toProtoSession converts a single domain Session to protobuf Session.
func toProtoSession(s *session.Session) *sessionv1.Session {
	if s == nil {
		return nil
	}

	proto := &sessionv1.Session{}

	switch s.Type {
	case session.SessionTypeHuman:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_HUMAN
		if data, ok := s.Data.(*session.HumanMessage); ok {
			proto.Data = &sessionv1.Session_HumanData{
				HumanData: toProtoHumanMessage(data),
			}
		}
	case session.SessionTypeAgent:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_AGENT
		if data, ok := s.Data.(*session.AgentMessage); ok {
			proto.Data = &sessionv1.Session_AgentData{
				AgentData: toProtoAgentMessage(data),
			}
		}
	case session.SessionTypeToolExecution:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_TOOL_EXECUTION
		if data, ok := s.Data.(*session.ToolExecution); ok {
			proto.Data = &sessionv1.Session_ToolExecutionData{
				ToolExecutionData: toProtoToolExecution(data),
			}
		}
	case session.SessionTypeSystem:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_SYSTEM
		if data, ok := s.Data.(*session.SystemMessage); ok {
			proto.Data = &sessionv1.Session_SystemData{
				SystemData: toProtoSystemMessage(data),
			}
		}
	case session.SessionTypeProgress:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_PROGRESS
		if data, ok := s.Data.(*session.Progress); ok {
			proto.Data = &sessionv1.Session_ProgressData{
				ProgressData: toProtoProgress(data),
			}
		}
	default:
		proto.Type = sessionv1.SessionType_SESSION_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoHumanMessage converts domain HumanMessage to protobuf.
func toProtoHumanMessage(h *session.HumanMessage) *sessionv1.HumanMessage {
	if h == nil {
		return nil
	}

	proto := &sessionv1.HumanMessage{
		Metadata: toProtoTreeNodeMeta(h.TreeNodeMeta),
		IsMeta:   h.IsMeta,
	}

	// Convert content blocks
	if len(h.Content) > 0 {
		proto.Content = make([]*sessionv1.HumanContentBlock, len(h.Content))
		for i, block := range h.Content {
			proto.Content[i] = toProtoHumanContentBlock(block)
		}
	}

	// Convert todos
	if len(h.Todos) > 0 {
		proto.Todos = make([]*sessionv1.Todo, len(h.Todos))
		for i, todo := range h.Todos {
			proto.Todos[i] = &sessionv1.Todo{
				Content:    todo.Content,
				Status:     todo.Status,
				ActiveForm: todo.ActiveForm,
			}
		}
	}

	return proto
}

// toProtoHumanContentBlock converts domain HumanContentBlock to protobuf.
func toProtoHumanContentBlock(block *session.HumanContentBlock) *sessionv1.HumanContentBlock {
	if block == nil {
		return nil
	}

	proto := &sessionv1.HumanContentBlock{}

	switch block.Type {
	case session.HumanContentBlockTypeText:
		proto.Type = sessionv1.HumanContentBlockType_HUMAN_CONTENT_BLOCK_TYPE_TEXT
		if block.Text != nil {
			proto.Text = *block.Text
		}
	case session.HumanContentBlockTypeImage:
		proto.Type = sessionv1.HumanContentBlockType_HUMAN_CONTENT_BLOCK_TYPE_IMAGE
		if block.Image != nil {
			proto.Image = &sessionv1.ImageData{
				MediaType: block.Image.MediaType,
				Data:      block.Image.Data,
			}
		}
	default:
		proto.Type = sessionv1.HumanContentBlockType_HUMAN_CONTENT_BLOCK_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoAgentMessage converts domain AgentMessage to protobuf.
func toProtoAgentMessage(a *session.AgentMessage) *sessionv1.AgentMessage {
	if a == nil {
		return nil
	}

	proto := &sessionv1.AgentMessage{
		Metadata:  toProtoTreeNodeMeta(a.TreeNodeMeta),
		Provider:  a.Provider,
		Model:     a.Model,
		RequestId: a.RequestID,
	}

	if a.StopReason != nil {
		proto.StopReason = *a.StopReason
	}

	// Convert usage
	if a.Usage != nil {
		proto.Usage = &sessionv1.TokenUsage{
			InputTokens:             int32(a.Usage.InputTokens),
			OutputTokens:            int32(a.Usage.OutputTokens),
			CacheCreationInputTokens: int32(a.Usage.CacheCreationInputTokens),
			CacheReadInputTokens:    int32(a.Usage.CacheReadInputTokens),
			ServiceTier:             a.Usage.ServiceTier,
		}
	}

	// Convert content blocks
	if len(a.Content) > 0 {
		proto.Content = make([]*sessionv1.AgentContentBlock, len(a.Content))
		for i, block := range a.Content {
			proto.Content[i] = toProtoAgentContentBlock(block)
		}
	}

	return proto
}

// toProtoAgentContentBlock converts domain AgentContentBlock to protobuf.
func toProtoAgentContentBlock(block *session.AgentContentBlock) *sessionv1.AgentContentBlock {
	if block == nil {
		return nil
	}

	proto := &sessionv1.AgentContentBlock{}

	switch block.Type {
	case session.AgentContentBlockTypeText:
		proto.Type = sessionv1.AgentContentBlockType_AGENT_CONTENT_BLOCK_TYPE_TEXT
		if block.Text != nil {
			proto.Text = *block.Text
		}
	case session.AgentContentBlockTypeThinking:
		proto.Type = sessionv1.AgentContentBlockType_AGENT_CONTENT_BLOCK_TYPE_THINKING
		if block.Thinking != nil {
			proto.Thinking = &sessionv1.ThinkingBlock{
				Content: block.Thinking.Content,
			}
			if block.Thinking.Signature != nil {
				proto.Thinking.Signature = *block.Thinking.Signature
			}
		}
	case session.AgentContentBlockTypeToolCallRef:
		proto.Type = sessionv1.AgentContentBlockType_AGENT_CONTENT_BLOCK_TYPE_TOOL_CALL_REF
		if block.ToolCallRef != nil {
			proto.ToolCallRef = &sessionv1.ToolCallReference{
				ToolExecutionId: block.ToolCallRef.ToolExecutionID,
				ToolName:        block.ToolCallRef.ToolName,
			}
		}
	default:
		proto.Type = sessionv1.AgentContentBlockType_AGENT_CONTENT_BLOCK_TYPE_UNSPECIFIED
	}

	return proto
}

// toProtoToolExecution converts domain ToolExecution to protobuf.
func toProtoToolExecution(t *session.ToolExecution) *sessionv1.ToolExecution {
	if t == nil {
		return nil
	}

	proto := &sessionv1.ToolExecution{
		Metadata:       toProtoTreeNodeMeta(t.TreeNodeMeta),
		Id:             t.ID,
		ToolName:       t.ToolName,
		SourceAgentUuid: t.SourceAgentUUID,
	}

	// Convert input to JSON
	if t.Input != nil {
		if inputBytes, err := sonic.Marshal(t.Input); err == nil {
			proto.InputJson = string(inputBytes)
		}
	}

	// Convert result
	if t.Result != nil {
		proto.Result = &sessionv1.ToolResult{
			Status: toProtoToolResultStatus(t.Result.Status),
		}

		// Convert content to string
		if t.Result.Content != nil {
			switch c := t.Result.Content.(type) {
			case string:
				proto.Result.Content = c
			default:
				if contentBytes, err := sonic.Marshal(c); err == nil {
					proto.Result.Content = string(contentBytes)
				}
			}
		}

		if t.Result.Error != nil {
			proto.Result.Error = *t.Result.Error
		}
	}

	// Convert metadata
	if t.Metadata != nil {
		proto.ExecMetadata = &sessionv1.ToolExecutionMetadata{
			ApprovalMode: t.Metadata.ApprovalMode,
			IsSubagent:   t.Metadata.IsSubagent,
			SubagentId:   t.Metadata.SubagentID,
		}
		if t.Metadata.DurationMs != nil {
			proto.ExecMetadata.DurationMs = int32(*t.Metadata.DurationMs)
		}
		if t.Metadata.ExecutedAt != nil {
			proto.ExecMetadata.ExecutedAt = timestamppb.New(*t.Metadata.ExecutedAt)
		}
	}

	return proto
}

// toProtoToolResultStatus converts domain ToolResultStatus to protobuf.
func toProtoToolResultStatus(status session.ToolResultStatus) sessionv1.ToolResultStatus {
	switch status {
	case session.ToolResultStatusSuccess:
		return sessionv1.ToolResultStatus_TOOL_RESULT_STATUS_SUCCESS
	case session.ToolResultStatusError:
		return sessionv1.ToolResultStatus_TOOL_RESULT_STATUS_ERROR
	case session.ToolResultStatusTimeout:
		return sessionv1.ToolResultStatus_TOOL_RESULT_STATUS_TIMEOUT
	case session.ToolResultStatusSkipped:
		return sessionv1.ToolResultStatus_TOOL_RESULT_STATUS_SKIPPED
	default:
		return sessionv1.ToolResultStatus_TOOL_RESULT_STATUS_UNSPECIFIED
	}
}

// toProtoSystemMessage converts domain SystemMessage to protobuf.
func toProtoSystemMessage(s *session.SystemMessage) *sessionv1.SystemMessage {
	if s == nil {
		return nil
	}

	proto := &sessionv1.SystemMessage{
		Metadata: toProtoTreeNodeMeta(s.TreeNodeMeta),
		IsMeta:   s.IsMeta,
	}

	switch s.Subtype {
	case session.SystemMessageSubtypeTurnDuration:
		proto.Subtype = sessionv1.SystemMessageSubtype_SYSTEM_MESSAGE_SUBTYPE_TURN_DURATION
		if s.TurnDuration != nil {
			proto.TurnDuration = &sessionv1.TurnDurationData{
				DurationMs: int32(s.TurnDuration.DurationMs),
			}
		}
	case session.SystemMessageSubtypeSummary:
		proto.Subtype = sessionv1.SystemMessageSubtype_SYSTEM_MESSAGE_SUBTYPE_SUMMARY
		if s.Summary != nil {
			proto.Summary = &sessionv1.SummaryData{
				Summary:  s.Summary.Summary,
				LeafUuid: s.Summary.LeafUUID,
			}
		}
	case session.SystemMessageSubtypeFileSnapshot:
		proto.Subtype = sessionv1.SystemMessageSubtype_SYSTEM_MESSAGE_SUBTYPE_FILE_SNAPSHOT
		if s.FileSnapshot != nil {
			proto.FileSnapshot = &sessionv1.FileSnapshotData{
				MessageId:          s.FileSnapshot.MessageID,
				IsSnapshotUpdate:   s.FileSnapshot.IsSnapshotUpdate,
				SnapshotTimestamp:  timestamppb.New(s.FileSnapshot.SnapshotTimestamp),
				TrackedFileBackups: make(map[string]*sessionv1.FileBackup),
			}
			for path, backup := range s.FileSnapshot.TrackedFileBackups {
				protoBackup := &sessionv1.FileBackup{
					Version:    int32(backup.Version),
					BackupTime: timestamppb.New(backup.BackupTime),
				}
				if backup.BackupFileName != nil {
					protoBackup.BackupFileName = *backup.BackupFileName
				}
				proto.FileSnapshot.TrackedFileBackups[path] = protoBackup
			}
		}
	case session.SystemMessageSubtypeInfo:
		proto.Subtype = sessionv1.SystemMessageSubtype_SYSTEM_MESSAGE_SUBTYPE_INFO
	default:
		proto.Subtype = sessionv1.SystemMessageSubtype_SYSTEM_MESSAGE_SUBTYPE_UNSPECIFIED
	}

	return proto
}

// toProtoProgress converts domain Progress to protobuf.
func toProtoProgress(p *session.Progress) *sessionv1.Progress {
	if p == nil {
		return nil
	}

	proto := &sessionv1.Progress{
		Metadata:              toProtoTreeNodeMeta(p.TreeNodeMeta),
		ToolExecutionId:       p.ToolExecutionID,
		ParentToolExecutionId: p.ParentToolExecutionID,
		Data:                  toProtoProgressData(p.Data),
	}

	return proto
}

// toProtoProgressData converts domain ProgressData to protobuf.
func toProtoProgressData(d session.ProgressData) *sessionv1.ProgressData {
	proto := &sessionv1.ProgressData{}

	switch d.Type {
	case session.ProgressTypeAgent:
		proto.Type = sessionv1.ProgressType_PROGRESS_TYPE_AGENT
	case session.ProgressTypeHook:
		proto.Type = sessionv1.ProgressType_PROGRESS_TYPE_HOOK
	case session.ProgressTypeBash:
		proto.Type = sessionv1.ProgressType_PROGRESS_TYPE_BASH
	case session.ProgressTypeMcp:
		proto.Type = sessionv1.ProgressType_PROGRESS_TYPE_MCP
	default:
		proto.Type = sessionv1.ProgressType_PROGRESS_TYPE_UNSPECIFIED
	}

	// Serialize Message to JSON
	if d.Message != nil {
		if jsonBytes, err := sonic.Marshal(d.Message); err == nil {
			proto.MessageJson = string(jsonBytes)
		}
	}

	// Serialize NormalizedMessages to JSON
	if len(d.NormalizedMessages) > 0 {
		if jsonBytes, err := sonic.Marshal(d.NormalizedMessages); err == nil {
			proto.NormalizedMessagesJson = string(jsonBytes)
		}
	}

	if d.Prompt != nil {
		proto.Prompt = *d.Prompt
	}

	if d.AgentID != nil {
		proto.AgentId = *d.AgentID
	}

	// Hook progress fields
	if d.HookEvent != nil {
		proto.HookEvent = *d.HookEvent
	}
	if d.HookName != nil {
		proto.HookName = *d.HookName
	}
	if d.Command != nil {
		proto.Command = *d.Command
	}

	return proto
}

// toProtoTreeNodeMeta converts domain TreeNodeMeta to protobuf.
func toProtoTreeNodeMeta(m session.TreeNodeMeta) *sessionv1.TreeNodeMeta {
	proto := &sessionv1.TreeNodeMeta{
		Uuid:        m.UUID,
		SessionId:   m.SessionID,
		Timestamp:   timestamppb.New(m.Timestamp),
		Provider:    m.Provider,
		Version:     m.Version,
		GitBranch:   m.GitBranch,
		IsSidechain: m.IsSidechain,
	}

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
