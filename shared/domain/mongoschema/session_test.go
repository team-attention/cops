package mongoschema

import (
	"testing"
	"time"

	domain "github.com/team-attention/cops/shared/domain/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSession_MarshalUnmarshalBSON_Progress(t *testing.T) {
	hookEvent := "PostToolUse"
	hookName := "PostToolUse:Read"
	command := "callback"
	prompt := "Test prompt"
	agentID := "agent-123"

	tests := []struct {
		name     string
		progress *domain.Progress
		verify   func(t *testing.T, p *domain.Progress)
	}{
		{
			name: "hook_progress with all fields",
			progress: &domain.Progress{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "test-uuid",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				ToolExecutionID: "tool-exec-123",
				Data: domain.ProgressData{
					Type:      domain.ProgressTypeHook,
					HookEvent: &hookEvent,
					HookName:  &hookName,
					Command:   &command,
				},
			},
			verify: func(t *testing.T, p *domain.Progress) {
				if p.Data.Type != domain.ProgressTypeHook {
					t.Errorf("Data.Type = %v, expected %v", p.Data.Type, domain.ProgressTypeHook)
				}
				if p.Data.HookEvent == nil || *p.Data.HookEvent != hookEvent {
					t.Errorf("Data.HookEvent = %v, expected %v", p.Data.HookEvent, &hookEvent)
				}
				if p.Data.HookName == nil || *p.Data.HookName != hookName {
					t.Errorf("Data.HookName = %v, expected %v", p.Data.HookName, &hookName)
				}
				if p.Data.Command == nil || *p.Data.Command != command {
					t.Errorf("Data.Command = %v, expected %v", p.Data.Command, &command)
				}
			},
		},
		{
			name: "agent_progress with prompt and agentId",
			progress: &domain.Progress{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "test-uuid-2",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				Data: domain.ProgressData{
					Type:    domain.ProgressTypeAgent,
					Prompt:  &prompt,
					AgentID: &agentID,
				},
			},
			verify: func(t *testing.T, p *domain.Progress) {
				if p.Data.Type != domain.ProgressTypeAgent {
					t.Errorf("Data.Type = %v, expected %v", p.Data.Type, domain.ProgressTypeAgent)
				}
				if p.Data.Prompt == nil || *p.Data.Prompt != prompt {
					t.Errorf("Data.Prompt = %v, expected %v", p.Data.Prompt, &prompt)
				}
				if p.Data.AgentID == nil || *p.Data.AgentID != agentID {
					t.Errorf("Data.AgentID = %v, expected %v", p.Data.AgentID, &agentID)
				}
			},
		},
		{
			name: "skill_progress with prompt",
			progress: &domain.Progress{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "test-uuid-3",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				Data: domain.ProgressData{
					Type:   domain.ProgressTypeSkill,
					Prompt: &prompt,
				},
			},
			verify: func(t *testing.T, p *domain.Progress) {
				if p.Data.Type != domain.ProgressTypeSkill {
					t.Errorf("Data.Type = %v, expected %v", p.Data.Type, domain.ProgressTypeSkill)
				}
				if p.Data.Prompt == nil || *p.Data.Prompt != prompt {
					t.Errorf("Data.Prompt = %v, expected %v", p.Data.Prompt, &prompt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := &Session{
				Session:   &domain.Session{Type: domain.SessionTypeProgress, Data: tt.progress},
				ProjectID: bson.NewObjectID(),
				UserID:    bson.NewObjectID(),
			}

			// Marshal
			bytes, err := bson.Marshal(sess)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal
			var restored Session
			err = bson.Unmarshal(bytes, &restored)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Verify type
			if restored.Session.Type != domain.SessionTypeProgress {
				t.Errorf("Type = %v, expected %v", restored.Session.Type, domain.SessionTypeProgress)
			}

			// Verify Data is correct type
			p, ok := restored.Session.Data.(*domain.Progress)
			if !ok {
				t.Fatalf("Data is not *domain.Progress, got %T", restored.Session.Data)
			}

			// Verify TreeNodeMeta preserved
			if p.UUID != tt.progress.UUID {
				t.Errorf("UUID = %v, expected %v", p.UUID, tt.progress.UUID)
			}
			if p.SessionID != tt.progress.SessionID {
				t.Errorf("SessionID = %v, expected %v", p.SessionID, tt.progress.SessionID)
			}

			// Run specific verifications
			tt.verify(t, p)
		})
	}
}

func TestSession_MarshalUnmarshalBSON_HumanMessage(t *testing.T) {
	text := "Hello, Claude!"
	sess := &Session{
		Session: &domain.Session{
			Type: domain.SessionTypeHuman,
			Data: &domain.HumanMessage{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "human-uuid",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				Content: []*domain.HumanContentBlock{
					{Type: domain.HumanContentBlockTypeText, Text: &text},
				},
				IsMeta: false,
			},
		},
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
	}

	// Marshal
	bytes, err := bson.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify type
	if restored.Session.Type != domain.SessionTypeHuman {
		t.Errorf("Type = %v, expected %v", restored.Session.Type, domain.SessionTypeHuman)
	}

	// Verify Data
	h, ok := restored.Session.Data.(*domain.HumanMessage)
	if !ok {
		t.Fatalf("Data is not *domain.HumanMessage, got %T", restored.Session.Data)
	}

	if h.UUID != "human-uuid" {
		t.Errorf("UUID = %v, expected human-uuid", h.UUID)
	}

	if len(h.Content) != 1 {
		t.Fatalf("Content length = %d, expected 1", len(h.Content))
	}

	if h.Content[0].Type != domain.HumanContentBlockTypeText {
		t.Errorf("Content[0].Type = %v, expected text", h.Content[0].Type)
	}

	if h.Content[0].Text == nil || *h.Content[0].Text != text {
		t.Errorf("Content[0].Text = %v, expected %v", h.Content[0].Text, text)
	}
}

func TestSession_MarshalUnmarshalBSON_AgentMessage(t *testing.T) {
	textContent := "I'll help you with that."
	stopReason := "end_turn"
	sess := &Session{
		Session: &domain.Session{
			Type: domain.SessionTypeAgent,
			Data: &domain.AgentMessage{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "agent-uuid",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				Provider:   "anthropic",
				Model:      "claude-opus-4-5-20251101",
				RequestID:  "req_12345",
				StopReason: &stopReason,
				Content: []*domain.AgentContentBlock{
					{Type: domain.AgentContentBlockTypeText, Text: &textContent},
				},
				Usage: &domain.TokenUsage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
		},
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
	}

	// Marshal
	bytes, err := bson.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify type
	if restored.Session.Type != domain.SessionTypeAgent {
		t.Errorf("Type = %v, expected %v", restored.Session.Type, domain.SessionTypeAgent)
	}

	// Verify Data
	a, ok := restored.Session.Data.(*domain.AgentMessage)
	if !ok {
		t.Fatalf("Data is not *domain.AgentMessage, got %T", restored.Session.Data)
	}

	if a.UUID != "agent-uuid" {
		t.Errorf("UUID = %v, expected agent-uuid", a.UUID)
	}

	if a.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Model = %v, expected claude-opus-4-5-20251101", a.Model)
	}

	if a.StopReason == nil || *a.StopReason != stopReason {
		t.Errorf("StopReason = %v, expected %v", a.StopReason, stopReason)
	}

	if a.Usage == nil {
		t.Fatal("Usage is nil")
	}

	if a.Usage.InputTokens != 100 {
		t.Errorf("Usage.InputTokens = %d, expected 100", a.Usage.InputTokens)
	}
}

func TestSession_MarshalUnmarshalBSON_ToolExecution(t *testing.T) {
	content := "File contents here"
	result := &domain.ToolResult{
		Status:  domain.ToolResultStatusSuccess,
		Content: content,
	}

	sess := &Session{
		Session: &domain.Session{
			Type: domain.SessionTypeToolExecution,
			Data: &domain.ToolExecution{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "tool-uuid",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				ID:              "toolu_12345",
				ToolName:        "Read",
				Input:           map[string]any{"file_path": "/path/to/file.txt"},
				Result:          result,
				SourceAgentUUID: "agent-uuid",
			},
		},
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
	}

	// Marshal
	bytes, err := bson.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify type
	if restored.Session.Type != domain.SessionTypeToolExecution {
		t.Errorf("Type = %v, expected %v", restored.Session.Type, domain.SessionTypeToolExecution)
	}

	// Verify Data
	te, ok := restored.Session.Data.(*domain.ToolExecution)
	if !ok {
		t.Fatalf("Data is not *domain.ToolExecution, got %T", restored.Session.Data)
	}

	if te.ToolName != "Read" {
		t.Errorf("ToolName = %v, expected Read", te.ToolName)
	}

	if te.ID != "toolu_12345" {
		t.Errorf("ID = %v, expected toolu_12345", te.ID)
	}

	if te.Input["file_path"] != "/path/to/file.txt" {
		t.Errorf("Input[file_path] = %v, expected /path/to/file.txt", te.Input["file_path"])
	}

	if te.Result == nil {
		t.Fatal("Result is nil")
	}

	if te.Result.Status != domain.ToolResultStatusSuccess {
		t.Errorf("Result.Status = %v, expected success", te.Result.Status)
	}
}

func TestSession_MarshalUnmarshalBSON_SystemMessage(t *testing.T) {
	sess := &Session{
		Session: &domain.Session{
			Type: domain.SessionTypeSystem,
			Data: &domain.SystemMessage{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "system-uuid",
					SessionID: "test-session",
					Timestamp: time.Now().Truncate(time.Millisecond),
					Provider:  "claude_code",
				},
				Subtype: domain.SystemMessageSubtypeTurnDuration,
				TurnDuration: &domain.TurnDurationData{
					DurationMs: 5000,
				},
			},
		},
		ProjectID: bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
	}

	// Marshal
	bytes, err := bson.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify type
	if restored.Session.Type != domain.SessionTypeSystem {
		t.Errorf("Type = %v, expected %v", restored.Session.Type, domain.SessionTypeSystem)
	}

	// Verify Data
	s, ok := restored.Session.Data.(*domain.SystemMessage)
	if !ok {
		t.Fatalf("Data is not *domain.SystemMessage, got %T", restored.Session.Data)
	}

	if s.Subtype != domain.SystemMessageSubtypeTurnDuration {
		t.Errorf("Subtype = %v, expected turn_duration", s.Subtype)
	}

	if s.TurnDuration == nil {
		t.Fatal("TurnDuration is nil")
	}

	if s.TurnDuration.DurationMs != 5000 {
		t.Errorf("TurnDuration.DurationMs = %d, expected 5000", s.TurnDuration.DurationMs)
	}
}

func TestSession_MarshalUnmarshalBSON_PreservesProjectAndUserIDs(t *testing.T) {
	projectID := bson.NewObjectID()
	userID := bson.NewObjectID()

	sess := &Session{
		Session: &domain.Session{
			Type: domain.SessionTypeProgress,
			Data: &domain.Progress{
				TreeNodeMeta: domain.TreeNodeMeta{
					UUID:      "test-uuid",
					SessionID: "test-session",
				},
				Data: domain.ProgressData{
					Type: domain.ProgressTypeHook,
				},
			},
		},
		ProjectID: projectID,
		UserID:    userID,
	}

	// Marshal
	bytes, err := bson.Marshal(sess)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.ProjectID != projectID {
		t.Errorf("ProjectID = %v, expected %v", restored.ProjectID, projectID)
	}

	if restored.UserID != userID {
		t.Errorf("UserID = %v, expected %v", restored.UserID, userID)
	}
}

func TestSession_UnmarshalBSON_UnknownTypeFallsBackToMap(t *testing.T) {
	// Create a BSON document with unknown type
	doc := bson.M{
		"type":        "unknown_future_type",
		"projectId":   bson.NewObjectID(),
		"userId":      bson.NewObjectID(),
		"customField": "custom_value",
	}

	bytes, err := bson.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Session
	err = bson.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Session.Type != "unknown_future_type" {
		t.Errorf("Type = %v, expected unknown_future_type", restored.Session.Type)
	}

	// Data should be bson.M
	m, ok := restored.Session.Data.(bson.M)
	if !ok {
		t.Fatalf("Data is not bson.M, got %T", restored.Session.Data)
	}

	if m["customField"] != "custom_value" {
		t.Errorf("customField = %v, expected custom_value", m["customField"])
	}
}
