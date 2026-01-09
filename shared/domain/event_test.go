package domain_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("Event", func() {
	Describe("UnmarshalJSON", func() {
		Context("when parsing session_start events", func() {
			It("parses valid session_start JSON", func() {
				jsonData := []byte(`{
					"type": "session_start",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"session_type": "interactive",
					"tools": ["bash", "read", "write"],
					"mcp_servers": ["server1", "server2"],
					"model": "claude-3-opus",
					"perm_mode": "auto",
					"max_turns": 100
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionStart))

				sessionStart, ok := event.Data.(*domain.SessionStartEvent)
				Expect(ok).To(BeTrue())
				Expect(string(sessionStart.SessionID)).To(Equal("test-session-123"))
				Expect(*sessionStart.SessionType).To(Equal("interactive"))
				Expect(sessionStart.Tools).To(Equal([]string{"bash", "read", "write"}))
				Expect(sessionStart.McpServers).To(Equal([]string{"server1", "server2"}))
				Expect(*sessionStart.Model).To(Equal("claude-3-opus"))
				Expect(*sessionStart.PermMode).To(Equal("auto"))
				Expect(*sessionStart.MaxTurns).To(Equal(100))
			})
		})

		Context("when parsing post_tool_use events", func() {
			It("parses valid post_tool_use JSON", func() {
				jsonData := []byte(`{
					"type": "post_tool_use",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"tool_name": "bash",
					"tool_id": "toolu_123",
					"success": true,
					"duration_ms": 1500
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypePostToolUse))

				postToolUse, ok := event.Data.(*domain.PostToolUseEvent)
				Expect(ok).To(BeTrue())
				Expect(postToolUse.ToolName).To(Equal("bash"))
				Expect(postToolUse.ToolID).To(Equal("toolu_123"))
				Expect(postToolUse.Success).To(BeTrue())
				Expect(*postToolUse.DurationMs).To(Equal(int64(1500)))
				Expect(postToolUse.Error).To(BeNil())
			})

			It("parses post_tool_use with error", func() {
				jsonData := []byte(`{
					"type": "post_tool_use",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"tool_name": "bash",
					"tool_id": "toolu_456",
					"success": false,
					"error": "command failed"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				postToolUse, ok := event.Data.(*domain.PostToolUseEvent)
				Expect(ok).To(BeTrue())
				Expect(postToolUse.Success).To(BeFalse())
				Expect(*postToolUse.Error).To(Equal("command failed"))
			})
		})

		Context("when parsing notification events", func() {
			It("parses valid notification JSON", func() {
				jsonData := []byte(`{
					"type": "notification",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"level": "info",
					"message": "Task completed successfully",
					"category": "task"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeNotification))

				notification, ok := event.Data.(*domain.NotificationEvent)
				Expect(ok).To(BeTrue())
				Expect(notification.Level).To(Equal("info"))
				Expect(notification.Message).To(Equal("Task completed successfully"))
				Expect(*notification.Category).To(Equal("task"))
			})
		})

		Context("when parsing user_prompt_submit events", func() {
			It("parses valid user_prompt_submit JSON", func() {
				jsonData := []byte(`{
					"type": "user_prompt_submit",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"prompt_length": 256,
					"has_images": true
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeUserPromptSubmit))

				userPrompt, ok := event.Data.(*domain.UserPromptSubmitEvent)
				Expect(ok).To(BeTrue())
				Expect(*userPrompt.PromptLength).To(Equal(256))
				Expect(*userPrompt.HasImages).To(BeTrue())
			})
		})

		Context("when parsing stop events", func() {
			It("parses valid stop JSON", func() {
				jsonData := []byte(`{
					"type": "stop",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"stop_reason": "end_turn",
					"total_turns": 5,
					"input_tokens": 10000,
					"output_tokens": 5000
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeStop))

				stopEvent, ok := event.Data.(*domain.StopEvent)
				Expect(ok).To(BeTrue())
				Expect(*stopEvent.StopReason).To(Equal("end_turn"))
				Expect(*stopEvent.TotalTurns).To(Equal(5))
				Expect(*stopEvent.InputTokens).To(Equal(int64(10000)))
				Expect(*stopEvent.OutputTokens).To(Equal(int64(5000)))
			})
		})

		Context("when parsing subagent_stop events", func() {
			It("parses valid subagent_stop JSON", func() {
				jsonData := []byte(`{
					"type": "subagent_stop",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"subagent_id": "subagent-abc",
					"stop_reason": "task_complete",
					"input_tokens": 3000,
					"output_tokens": 1500
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSubagentStop))

				subagentStop, ok := event.Data.(*domain.SubagentStopEvent)
				Expect(ok).To(BeTrue())
				Expect(*subagentStop.SubagentID).To(Equal("subagent-abc"))
				Expect(*subagentStop.StopReason).To(Equal("task_complete"))
				Expect(*subagentStop.InputTokens).To(Equal(int64(3000)))
				Expect(*subagentStop.OutputTokens).To(Equal(int64(1500)))
			})
		})

		Context("when parsing session_end events", func() {
			It("parses valid session_end JSON", func() {
				jsonData := []byte(`{
					"type": "session_end",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"exit_code": 0,
					"total_duration_ms": 120000,
					"total_input_tokens": 50000,
					"total_output_tokens": 25000
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionEnd))

				sessionEnd, ok := event.Data.(*domain.SessionEndEvent)
				Expect(ok).To(BeTrue())
				Expect(*sessionEnd.ExitCode).To(Equal(0))
				Expect(*sessionEnd.TotalDurationMs).To(Equal(int64(120000)))
				Expect(*sessionEnd.TotalInputTokens).To(Equal(int64(50000)))
				Expect(*sessionEnd.TotalOutputTokens).To(Equal(int64(25000)))
			})
		})

		Context("when parsing unknown event types", func() {
			It("stores unknown type as map[string]any", func() {
				jsonData := []byte(`{
					"type": "future_event_type",
					"sessionId": "test-session-123",
					"timestamp": "2025-01-01T00:00:00Z",
					"custom_field": "custom_value"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventType("future_event_type")))

				mapData, ok := event.Data.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(mapData).To(HaveKey("type"))
				Expect(mapData).To(HaveKeyWithValue("custom_field", "custom_value"))
			})
		})

		Context("when JSON is invalid", func() {
			It("returns error for malformed JSON", func() {
				jsonData := []byte(`{not valid`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when marshaling typed events", func() {
			It("produces flat JSON with type field", func() {
				sessionType := "interactive"
				model := "claude-3-opus"
				sessionStart := &domain.SessionStartEvent{
					SessionType: &sessionType,
					Model:       &model,
					Tools:       []string{"bash", "read"},
				}
				sessionStart.SessionID = "test-session-123"

				event := domain.Event{
					Type: domain.EventTypeSessionStart,
					Data: sessionStart,
				}

				jsonData, err := json.Marshal(event)
				Expect(err).NotTo(HaveOccurred())

				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(HaveKeyWithValue("type", "session_start"))
				Expect(result).To(HaveKeyWithValue("session_type", "interactive"))
				Expect(result).To(HaveKeyWithValue("model", "claude-3-opus"))
				Expect(result).To(HaveKeyWithValue("sessionId", "test-session-123"))
				Expect(result).To(HaveKey("tools"))
			})
		})

		Context("when marshaling nil Data", func() {
			It("produces JSON with only type field", func() {
				event := domain.Event{
					Type: domain.EventTypeSessionStart,
					Data: nil,
				}

				jsonData, err := json.Marshal(event)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(jsonData)).To(Equal(`{"type":"session_start"}`))
			})
		})
	})

	Describe("Round-trip serialization", func() {
		It("preserves event through marshal/unmarshal cycle", func() {
			// Create a session_start event with all fields
			sessionType := "interactive"
			model := "claude-3-opus"
			permMode := "auto"
			maxTurns := 100

			originalEvent := domain.Event{
				Type: domain.EventTypeSessionStart,
				Data: &domain.SessionStartEvent{
					SessionType: &sessionType,
					Tools:       []string{"bash", "read", "write"},
					McpServers:  []string{"server1"},
					Model:       &model,
					PermMode:    &permMode,
					MaxTurns:    &maxTurns,
				},
			}
			originalSessionStart := originalEvent.Data.(*domain.SessionStartEvent)
			originalSessionStart.SessionID = "test-session-123"

			// Marshal to JSON
			jsonData, err := json.Marshal(originalEvent)
			Expect(err).NotTo(HaveOccurred())

			// Unmarshal back
			var roundTrippedEvent domain.Event
			err = json.Unmarshal(jsonData, &roundTrippedEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify type
			Expect(roundTrippedEvent.Type).To(Equal(originalEvent.Type))

			// Verify data
			roundTrippedSessionStart, ok := roundTrippedEvent.Data.(*domain.SessionStartEvent)
			Expect(ok).To(BeTrue())
			Expect(*roundTrippedSessionStart.SessionType).To(Equal(*originalSessionStart.SessionType))
			Expect(roundTrippedSessionStart.Tools).To(Equal(originalSessionStart.Tools))
			Expect(*roundTrippedSessionStart.Model).To(Equal(*originalSessionStart.Model))
			Expect(string(roundTrippedSessionStart.SessionID)).To(Equal(string(originalSessionStart.SessionID)))
		})

		It("preserves post_tool_use event through cycle", func() {
			durationMs := int64(1500)
			errorMsg := "test error"

			originalEvent := domain.Event{
				Type: domain.EventTypePostToolUse,
				Data: &domain.PostToolUseEvent{
					ToolName:   "bash",
					ToolID:     "toolu_123",
					Success:    false,
					Error:      &errorMsg,
					DurationMs: &durationMs,
				},
			}

			jsonData, err := json.Marshal(originalEvent)
			Expect(err).NotTo(HaveOccurred())

			var roundTrippedEvent domain.Event
			err = json.Unmarshal(jsonData, &roundTrippedEvent)
			Expect(err).NotTo(HaveOccurred())

			postToolUse, ok := roundTrippedEvent.Data.(*domain.PostToolUseEvent)
			Expect(ok).To(BeTrue())
			Expect(postToolUse.ToolName).To(Equal("bash"))
			Expect(postToolUse.ToolID).To(Equal("toolu_123"))
			Expect(postToolUse.Success).To(BeFalse())
			Expect(*postToolUse.Error).To(Equal("test error"))
			Expect(*postToolUse.DurationMs).To(Equal(int64(1500)))
		})
	})
})
