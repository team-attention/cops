package mongoschema

import (
	"testing"

	"github.com/team-attention/cops/shared/domain"
)

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"session_id", "session_id", "sessionId"},
		{"tool_name", "tool_name", "toolName"},
		{"hook_event_name", "hook_event_name", "hookEventName"},
		{"stop_hook_active", "stop_hook_active", "stopHookActive"},
		{"agent_transcript_path", "agent_transcript_path", "agentTranscriptPath"},
		{"custom_instructions", "custom_instructions", "customInstructions"},
		{"notification_type", "notification_type", "notificationType"},
		{"permission_mode", "permission_mode", "permissionMode"},
		{"transcript_path", "transcript_path", "transcriptPath"},
		{"single word", "cwd", "cwd"},
		{"already camelCase", "sessionId", "sessionId"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snakeToCamel(tt.input)
			if result != tt.expected {
				t.Errorf("snakeToCamel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertKeysToCamel(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "simple flat map",
			input: map[string]any{
				"session_id": "abc123",
				"tool_name":  "Write",
				"cwd":        "/workspace",
			},
			expected: map[string]any{
				"sessionId": "abc123",
				"toolName":  "Write",
				"cwd":       "/workspace",
			},
		},
		{
			name: "nested map",
			input: map[string]any{
				"tool_input": map[string]any{
					"file_path": "/test.txt",
					"content":   "hello",
				},
			},
			expected: map[string]any{
				"toolInput": map[string]any{
					"filePath": "/test.txt",
					"content":  "hello",
				},
			},
		},
		{
			name: "array of maps",
			input: map[string]any{
				"items": []any{
					map[string]any{"item_name": "first"},
					map[string]any{"item_name": "second"},
				},
			},
			expected: map[string]any{
				"items": []any{
					map[string]any{"itemName": "first"},
					map[string]any{"itemName": "second"},
				},
			},
		},
		{
			name: "hook_event_name follows standard pattern",
			input: map[string]any{
				"hook_event_name": "PostToolUse",
				"session_id":      "abc123",
			},
			expected: map[string]any{
				"hookEventName": "PostToolUse",
				"sessionId":     "abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertKeysToCamel(tt.input).(map[string]any)
			compareMapKeys(t, result, tt.expected)
		})
	}
}

func TestEvent_ToBSONDocument_KeyConversion(t *testing.T) {
	tests := []struct {
		name           string
		event          *domain.Event
		expectedKeys   []string
		unexpectedKeys []string
	}{
		{
			name: "PostToolUse event",
			event: &domain.Event{
				Type: domain.EventTypePostToolUse,
				Data: &domain.PostToolUseEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "abc123",
						PermissionMode: "default",
					},
					ToolName:  "Write",
					ToolInput: map[string]any{"file_path": "/test.txt"},
				},
			},
			expectedKeys:   []string{"hookEventName", "sessionId", "permissionMode", "toolName", "toolInput"},
			unexpectedKeys: []string{"hook_event_name", "session_id", "permission_mode", "tool_name", "tool_input"},
		},
		{
			name: "SessionStart event",
			event: &domain.Event{
				Type: domain.EventTypeSessionStart,
				Data: &domain.SessionStartEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "abc123",
						PermissionMode: "default",
					},
					Source: "startup",
				},
			},
			expectedKeys:   []string{"hookEventName", "sessionId", "source"},
			unexpectedKeys: []string{"hook_event_name", "session_id"},
		},
		{
			name: "Stop event",
			event: &domain.Event{
				Type: domain.EventTypeStop,
				Data: &domain.StopEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "abc123",
						PermissionMode: "default",
					},
					StopHookActive: true,
				},
			},
			expectedKeys:   []string{"hookEventName", "sessionId", "stopHookActive"},
			unexpectedKeys: []string{"hook_event_name", "session_id", "stop_hook_active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{}
			e.FromDomain("", tt.event)

			doc, err := e.ToBSONDocument()
			if err != nil {
				t.Fatalf("ToBSONDocument() error = %v", err)
			}

			// Check expected keys exist
			for _, key := range tt.expectedKeys {
				if _, ok := doc[key]; !ok {
					t.Errorf("expected key %q not found in document", key)
				}
			}

			// Check unexpected keys do not exist
			for _, key := range tt.unexpectedKeys {
				if _, ok := doc[key]; ok {
					t.Errorf("unexpected key %q found in document", key)
				}
			}
		})
	}
}

func compareMapKeys(t *testing.T, result, expected map[string]any) {
	t.Helper()

	for key, expectedVal := range expected {
		resultVal, ok := result[key]
		if !ok {
			t.Errorf("expected key %q not found", key)
			continue
		}

		// Handle nested maps
		if expectedMap, ok := expectedVal.(map[string]any); ok {
			resultMap, ok := resultVal.(map[string]any)
			if !ok {
				t.Errorf("expected nested map for key %q, got %T", key, resultVal)
				continue
			}
			compareMapKeys(t, resultMap, expectedMap)
		}

		// Handle arrays
		if expectedArr, ok := expectedVal.([]any); ok {
			resultArr, ok := resultVal.([]any)
			if !ok {
				t.Errorf("expected array for key %q, got %T", key, resultVal)
				continue
			}
			if len(expectedArr) != len(resultArr) {
				t.Errorf("array length mismatch for key %q: got %d, expected %d", key, len(resultArr), len(expectedArr))
				continue
			}
			for i := range expectedArr {
				if expectedItemMap, ok := expectedArr[i].(map[string]any); ok {
					resultItemMap, ok := resultArr[i].(map[string]any)
					if !ok {
						t.Errorf("expected nested map in array for key %q[%d], got %T", key, i, resultArr[i])
						continue
					}
					compareMapKeys(t, resultItemMap, expectedItemMap)
				}
			}
		}
	}

	// Check for unexpected keys
	for key := range result {
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected key %q found", key)
		}
	}
}
