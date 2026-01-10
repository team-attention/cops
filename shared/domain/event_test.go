package domain_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("Event", func() {
	Describe("UnmarshalJSON", func() {
		Context("when parsing PreToolUse events", func() {
			It("parses valid PreToolUse JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"cwd": "/Users/test/project",
					"permission_mode": "default",
					"hook_event_name": "PreToolUse",
					"tool_name": "Write",
					"tool_input": {
						"file_path": "/path/to/file.txt",
						"content": "file content"
					},
					"tool_use_id": "toolu_01ABC123"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypePreToolUse))

				preToolUse, ok := event.Data.(*domain.PreToolUseEvent)
				Expect(ok).To(BeTrue())
				Expect(preToolUse.SessionID).To(Equal("abc123"))
				Expect(preToolUse.PermissionMode).To(Equal("default"))
				Expect(preToolUse.ToolName).To(Equal("Write"))
				Expect(preToolUse.ToolInput).To(HaveKeyWithValue("file_path", "/path/to/file.txt"))
				Expect(preToolUse.ToolUseID).To(Equal("toolu_01ABC123"))
			})
		})

		Context("when parsing PostToolUse events", func() {
			It("parses valid PostToolUse JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"cwd": "/Users/test/project",
					"permission_mode": "default",
					"hook_event_name": "PostToolUse",
					"tool_name": "Write",
					"tool_input": {
						"file_path": "/path/to/file.txt",
						"content": "file content"
					},
					"tool_response": {
						"filePath": "/path/to/file.txt",
						"success": true
					},
					"tool_use_id": "toolu_01ABC123"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypePostToolUse))

				postToolUse, ok := event.Data.(*domain.PostToolUseEvent)
				Expect(ok).To(BeTrue())
				Expect(postToolUse.SessionID).To(Equal("abc123"))
				Expect(postToolUse.ToolName).To(Equal("Write"))
				Expect(postToolUse.ToolInput).To(HaveKeyWithValue("file_path", "/path/to/file.txt"))
				Expect(postToolUse.ToolResponse).To(HaveKeyWithValue("success", true))
				Expect(postToolUse.ToolUseID).To(Equal("toolu_01ABC123"))
			})
		})

		Context("when parsing Notification events", func() {
			It("parses valid Notification JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"cwd": "/Users/test/project",
					"permission_mode": "default",
					"hook_event_name": "Notification",
					"message": "Claude needs your permission to use Bash",
					"notification_type": "permission_prompt"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeNotification))

				notification, ok := event.Data.(*domain.NotificationEvent)
				Expect(ok).To(BeTrue())
				Expect(notification.SessionID).To(Equal("abc123"))
				Expect(notification.Message).To(Equal("Claude needs your permission to use Bash"))
				Expect(notification.NotificationType).To(Equal("permission_prompt"))
			})
		})

		Context("when parsing UserPromptSubmit events", func() {
			It("parses valid UserPromptSubmit JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"cwd": "/Users/test/project",
					"permission_mode": "default",
					"hook_event_name": "UserPromptSubmit",
					"prompt": "Write a function to calculate the factorial of a number"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeUserPromptSubmit))

				userPrompt, ok := event.Data.(*domain.UserPromptSubmitEvent)
				Expect(ok).To(BeTrue())
				Expect(userPrompt.SessionID).To(Equal("abc123"))
				Expect(userPrompt.Prompt).To(Equal("Write a function to calculate the factorial of a number"))
			})
		})

		Context("when parsing Stop events", func() {
			It("parses valid Stop JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "Stop",
					"stop_hook_active": true
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeStop))

				stopEvent, ok := event.Data.(*domain.StopEvent)
				Expect(ok).To(BeTrue())
				Expect(stopEvent.SessionID).To(Equal("abc123"))
				Expect(stopEvent.StopHookActive).To(BeTrue())
			})

			It("parses Stop with stop_hook_active false", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "Stop",
					"stop_hook_active": false
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				stopEvent, ok := event.Data.(*domain.StopEvent)
				Expect(ok).To(BeTrue())
				Expect(stopEvent.StopHookActive).To(BeFalse())
			})
		})

		Context("when parsing SubagentStop events", func() {
			It("parses valid SubagentStop JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "SubagentStop",
					"stop_hook_active": false,
					"agent_id": "a3bec5d",
					"agent_transcript_path": "/Users/test/.claude/projects/test/agent-a3bec5d.jsonl"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSubagentStop))

				subagentStop, ok := event.Data.(*domain.SubagentStopEvent)
				Expect(ok).To(BeTrue())
				Expect(subagentStop.SessionID).To(Equal("abc123"))
				Expect(subagentStop.StopHookActive).To(BeFalse())
				Expect(subagentStop.AgentID).To(Equal("a3bec5d"))
			})
		})

		Context("when parsing PreCompact events", func() {
			It("parses valid PreCompact JSON from Claude Code hook (manual)", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "PreCompact",
					"trigger": "manual",
					"custom_instructions": "Focus on the authentication flow"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypePreCompact))

				preCompact, ok := event.Data.(*domain.PreCompactEvent)
				Expect(ok).To(BeTrue())
				Expect(preCompact.SessionID).To(Equal("abc123"))
				Expect(preCompact.Trigger).To(Equal("manual"))
				Expect(preCompact.CustomInstructions).To(Equal("Focus on the authentication flow"))
			})

			It("parses valid PreCompact JSON from Claude Code hook (auto)", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "PreCompact",
					"trigger": "auto",
					"custom_instructions": ""
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				preCompact, ok := event.Data.(*domain.PreCompactEvent)
				Expect(ok).To(BeTrue())
				Expect(preCompact.Trigger).To(Equal("auto"))
				Expect(preCompact.CustomInstructions).To(BeEmpty())
			})
		})

		Context("when parsing SessionStart events", func() {
			It("parses valid SessionStart JSON from Claude Code hook (startup)", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"permission_mode": "default",
					"hook_event_name": "SessionStart",
					"source": "startup"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionStart))

				sessionStart, ok := event.Data.(*domain.SessionStartEvent)
				Expect(ok).To(BeTrue())
				Expect(sessionStart.SessionID).To(Equal("abc123"))
				Expect(sessionStart.Source).To(Equal("startup"))
			})

			It("parses SessionStart with different sources", func() {
				sources := []string{"startup", "resume", "clear", "compact"}

				for _, source := range sources {
					jsonData := []byte(`{
						"session_id": "abc123",
						"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
						"hook_event_name": "SessionStart",
						"source": "` + source + `"
					}`)

					var event domain.Event
					err := json.Unmarshal(jsonData, &event)
					Expect(err).NotTo(HaveOccurred())

					sessionStart, ok := event.Data.(*domain.SessionStartEvent)
					Expect(ok).To(BeTrue())
					Expect(sessionStart.Source).To(Equal(source))
				}
			})
		})

		Context("when parsing SessionEnd events", func() {
			It("parses valid SessionEnd JSON from Claude Code hook", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
					"cwd": "/Users/test/project",
					"permission_mode": "default",
					"hook_event_name": "SessionEnd",
					"reason": "exit"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionEnd))

				sessionEnd, ok := event.Data.(*domain.SessionEndEvent)
				Expect(ok).To(BeTrue())
				Expect(sessionEnd.SessionID).To(Equal("abc123"))
				Expect(sessionEnd.Reason).To(Equal("exit"))
			})

			It("parses SessionEnd with different reasons", func() {
				reasons := []string{"clear", "logout", "prompt_input_exit", "other"}

				for _, reason := range reasons {
					jsonData := []byte(`{
						"session_id": "abc123",
						"transcript_path": "/Users/test/.claude/projects/test/00893aaf.jsonl",
						"hook_event_name": "SessionEnd",
						"reason": "` + reason + `"
					}`)

					var event domain.Event
					err := json.Unmarshal(jsonData, &event)
					Expect(err).NotTo(HaveOccurred())

					sessionEnd, ok := event.Data.(*domain.SessionEndEvent)
					Expect(ok).To(BeTrue())
					Expect(sessionEnd.Reason).To(Equal(reason))
				}
			})
		})

		Context("when parsing unknown event types", func() {
			It("stores unknown type as map[string]any", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"hook_event_name": "FutureEventType",
					"custom_field": "custom_value"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventType("FutureEventType")))

				mapData, ok := event.Data.(map[string]any)
				Expect(ok).To(BeTrue())
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

		Context("backward compatibility with type field", func() {
			It("supports legacy type field format", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"type": "SessionStart",
					"source": "startup"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionStart))
			})

			It("prefers hook_event_name over type when both present", func() {
				jsonData := []byte(`{
					"session_id": "abc123",
					"type": "Stop",
					"hook_event_name": "SessionStart",
					"source": "startup"
				}`)

				var event domain.Event
				err := json.Unmarshal(jsonData, &event)
				Expect(err).NotTo(HaveOccurred())

				Expect(event.Type).To(Equal(domain.EventTypeSessionStart))
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when marshaling typed events", func() {
			It("produces flat JSON with hook_event_name field", func() {
				sessionStart := &domain.SessionStartEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "test-session-123",
						PermissionMode: "default",
					},
					Source: "startup",
				}

				event := domain.Event{
					Type: domain.EventTypeSessionStart,
					Data: sessionStart,
				}

				jsonData, err := json.Marshal(event)
				Expect(err).NotTo(HaveOccurred())

				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				Expect(result).To(HaveKeyWithValue("hook_event_name", "SessionStart"))
				Expect(result).To(HaveKeyWithValue("session_id", "test-session-123"))
				Expect(result).To(HaveKeyWithValue("source", "startup"))
			})
		})

		Context("when marshaling nil Data", func() {
			It("produces JSON with only hook_event_name field", func() {
				event := domain.Event{
					Type: domain.EventTypeSessionStart,
					Data: nil,
				}

				jsonData, err := json.Marshal(event)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(jsonData)).To(Equal(`{"hook_event_name":"SessionStart"}`))
			})
		})
	})

	Describe("Round-trip serialization", func() {
		It("preserves PostToolUse event through marshal/unmarshal cycle", func() {
			originalEvent := domain.Event{
				Type: domain.EventTypePostToolUse,
				Data: &domain.PostToolUseEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "test-session-123",
						PermissionMode: "acceptEdits",
					},
					ToolName:     "Bash",
					ToolInput:    map[string]any{"command": "ls -la"},
					ToolResponse: map[string]any{"stdout": "file list", "stderr": ""},
					ToolUseID:    "toolu_123",
				},
			}

			jsonData, err := json.Marshal(originalEvent)
			Expect(err).NotTo(HaveOccurred())

			var roundTrippedEvent domain.Event
			err = json.Unmarshal(jsonData, &roundTrippedEvent)
			Expect(err).NotTo(HaveOccurred())

			Expect(roundTrippedEvent.Type).To(Equal(originalEvent.Type))

			postToolUse, ok := roundTrippedEvent.Data.(*domain.PostToolUseEvent)
			Expect(ok).To(BeTrue())
			Expect(postToolUse.SessionID).To(Equal("test-session-123"))
			Expect(postToolUse.ToolName).To(Equal("Bash"))
			Expect(postToolUse.ToolUseID).To(Equal("toolu_123"))
		})

		It("preserves SessionStart event through cycle", func() {
			originalEvent := domain.Event{
				Type: domain.EventTypeSessionStart,
				Data: &domain.SessionStartEvent{
					HookEventBase: domain.HookEventBase{
						SessionID:      "test-session-456",
						PermissionMode: "default",
					},
					Source: "resume",
				},
			}

			jsonData, err := json.Marshal(originalEvent)
			Expect(err).NotTo(HaveOccurred())

			var roundTrippedEvent domain.Event
			err = json.Unmarshal(jsonData, &roundTrippedEvent)
			Expect(err).NotTo(HaveOccurred())

			sessionStart, ok := roundTrippedEvent.Data.(*domain.SessionStartEvent)
			Expect(ok).To(BeTrue())
			Expect(sessionStart.SessionID).To(Equal("test-session-456"))
			Expect(sessionStart.Source).To(Equal("resume"))
		})
	})
})
