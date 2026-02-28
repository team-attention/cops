package codexcliadapter_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	session "github.com/team-attention/cops/shared/domain/v2"
	"github.com/team-attention/cops/shared/domain/v2/codexcliadapter"
)

func TestCodexCLIAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CodexCLIAdapter Suite")
}

// makeEntry creates a raw JSONL line from type, timestamp, and payload.
func makeEntry(entryType string, ts time.Time, payload any) string {
	raw := map[string]any{
		"type":      entryType,
		"timestamp": ts.Format(time.RFC3339Nano),
		"payload":   payload,
	}
	b, err := json.Marshal(raw)
	Expect(err).NotTo(HaveOccurred())
	return string(b)
}

// makeEntryNoTimestamp creates a raw JSONL line without a timestamp field.
func makeEntryNoTimestamp(entryType string, payload any) string {
	raw := map[string]any{
		"type":    entryType,
		"payload": payload,
	}
	b, err := json.Marshal(raw)
	Expect(err).NotTo(HaveOccurred())
	return string(b)
}

var _ = Describe("Adapter", func() {
	var adapter *codexcliadapter.Adapter
	ts := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	BeforeEach(func() {
		adapter = codexcliadapter.NewAdapter(slog.Default())
	})

	Describe("AdaptEntry", func() {
		Context("when processing metadata entries", func() {
			It("caches session ID from session_meta", func() {
				line := makeEntry("session_meta", ts, map[string]any{
					"id":  "test-session-123",
					"cwd": "/some/path",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())

				// Process a user_message to verify cached sessionID
				msgLine := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				sessions, err = adapter.AdaptEntry(msgLine)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.TreeNodeMeta.SessionID).To(Equal("test-session-123"))
			})

			It("caches model provider from session_meta", func() {
				mp := "openai"
				line := makeEntry("session_meta", ts, map[string]any{
					"id":             "sess-1",
					"cwd":            "/path",
					"model_provider": mp,
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())

				// Process an agent_message to verify cached provider
				msgLine := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions, err = adapter.AdaptEntry(msgLine)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Provider).To(Equal("openai"))
			})

			It("caches model from turn_context", func() {
				line := makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t1",
					"cwd":     "/path",
					"model":   "o3-pro",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())

				// Process an agent_message to verify cached model
				msgLine := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions, err = adapter.AdaptEntry(msgLine)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Model).To(Equal("o3-pro"))
			})

			It("returns nil sessions for session_meta", func() {
				line := makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-1",
					"cwd": "/path",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil sessions for turn_context", func() {
				line := makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t1",
					"cwd":     "/path",
					"model":   "gpt-4o",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("handles session_meta without model_provider (older format)", func() {
				line := makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-old",
					"cwd": "/path",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())

				// Process an agent_message to verify default provider
				msgLine := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions, err = adapter.AdaptEntry(msgLine)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Provider).To(Equal(codexcliadapter.DefaultModelProvider))
			})

			It("overrides cached state when later session_meta arrives", func() {
				line1 := makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-A",
					"cwd": "/path",
				})
				_, err := adapter.AdaptEntry(line1)
				Expect(err).NotTo(HaveOccurred())

				line2 := makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-B",
					"cwd": "/path",
				})
				_, err = adapter.AdaptEntry(line2)
				Expect(err).NotTo(HaveOccurred())

				msgLine := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				sessions, err := adapter.AdaptEntry(msgLine)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.TreeNodeMeta.SessionID).To(Equal("sess-B"))
			})
		})

		Context("when processing event_msg entries", func() {
			It("converts user_message to HumanMessage", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello world",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeHuman))
				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.Content).To(HaveLen(1))
				Expect(human.Content[0].Type).To(Equal(session.HumanContentBlockTypeText))
				Expect(*human.Content[0].Text).To(Equal("hello world"))
				Expect(human.TreeNodeMeta.Provider).To(Equal("codex_cli"))
			})

			It("converts agent_message to AgentMessage with text", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "agent response text",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeAgent))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeText))
				Expect(*agent.Content[0].Text).To(Equal("agent response text"))
				Expect(agent.Provider).To(Equal(codexcliadapter.DefaultModelProvider))
			})

			It("converts agent_reasoning to AgentMessage with thinking", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type": "agent_reasoning",
					"text": "**thinking about the problem**",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeAgent))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeThinking))
				Expect(agent.Content[0].Thinking.Content).To(Equal("**thinking about the problem**"))
			})

			It("returns nil for token_count", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type": "token_count",
					"info": map[string]any{
						"total_token_usage": map[string]any{
							"input_tokens":        100,
							"cached_input_tokens": 50,
							"output_tokens":       200,
							"total_tokens":        350,
						},
					},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil for task_started", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type":    "task_started",
					"turn_id": "t1",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil for task_complete", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type":    "task_complete",
					"turn_id": "t1",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})
		})

		Context("when processing response_item entries", func() {
			It("converts assistant message to AgentMessage", func() {
				role := "assistant"
				line := makeEntry("response_item", ts, map[string]any{
					"type": "message",
					"role": role,
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "answer text",
						},
					},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeAgent))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeText))
				Expect(*agent.Content[0].Text).To(Equal("answer text"))
			})

			It("converts reasoning to AgentMessage with thinking", func() {
				line := makeEntry("response_item", ts, map[string]any{
					"type": "reasoning",
					"summary": []any{
						map[string]any{
							"type": "summary_text",
							"text": "reasoning part 1",
						},
						map[string]any{
							"type": "summary_text",
							"text": "reasoning part 2",
						},
					},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeAgent))
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeThinking))
				Expect(agent.Content[0].Thinking.Content).To(Equal("reasoning part 1\nreasoning part 2"))
			})

			It("returns nil for developer role", func() {
				role := "developer"
				line := makeEntry("response_item", ts, map[string]any{
					"type": "message",
					"role": role,
					"content": []any{
						map[string]any{"type": "input_text", "text": "system prompt"},
					},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil for user role", func() {
				role := "user"
				line := makeEntry("response_item", ts, map[string]any{
					"type": "message",
					"role": role,
					"content": []any{
						map[string]any{"type": "input_text", "text": "user context"},
					},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil for reasoning with empty Summary slice", func() {
				line := makeEntry("response_item", ts, map[string]any{
					"type":    "reasoning",
					"summary": []any{},
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})
		})

		Context("error handling", func() {
			It("returns error for malformed JSON", func() {
				sessions, err := adapter.AdaptEntry("{invalid json")
				Expect(err).To(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("returns nil for empty line", func() {
				sessions, err := adapter.AdaptEntry("")
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})

			It("uses time.Now() fallback when timestamp is missing", func() {
				before := time.Now()
				line := makeEntryNoTimestamp("event_msg", map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				sessions, err := adapter.AdaptEntry(line)
				after := time.Now()
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.TreeNodeMeta.Timestamp).To(BeTemporally(">=", before))
				Expect(human.TreeNodeMeta.Timestamp).To(BeTemporally("<=", after))
			})
		})

		Context("field verification", func() {
			It("sets Provider to codex_cli in all TreeNodeMeta", func() {
				// Test user_message
				line1 := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				sessions1, err := adapter.AdaptEntry(line1)
				Expect(err).NotTo(HaveOccurred())
				human := sessions1[0].Data.(*session.HumanMessage)
				Expect(human.TreeNodeMeta.Provider).To(Equal("codex_cli"))

				// Test agent_message
				line2 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions2, err := adapter.AdaptEntry(line2)
				Expect(err).NotTo(HaveOccurred())
				agent := sessions2[0].Data.(*session.AgentMessage)
				Expect(agent.TreeNodeMeta.Provider).To(Equal("codex_cli"))
			})

			It("generates unique UUID for each session", func() {
				line1 := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				line2 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions1, _ := adapter.AdaptEntry(line1)
				sessions2, _ := adapter.AdaptEntry(line2)
				uuid1 := sessions1[0].Data.(*session.HumanMessage).TreeNodeMeta.UUID
				uuid2 := sessions2[0].Data.(*session.AgentMessage).TreeNodeMeta.UUID
				Expect(uuid1).NotTo(BeEmpty())
				Expect(uuid2).NotTo(BeEmpty())
				Expect(uuid1).NotTo(Equal(uuid2))
			})

			It("populates SessionID from cached session_meta", func() {
				metaLine := makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-1",
					"cwd": "/path",
				})
				_, err := adapter.AdaptEntry(metaLine)
				Expect(err).NotTo(HaveOccurred())

				// user_message
				line1 := makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				})
				sessions1, _ := adapter.AdaptEntry(line1)
				human := sessions1[0].Data.(*session.HumanMessage)
				Expect(human.TreeNodeMeta.SessionID).To(Equal("sess-1"))

				// agent_message
				line2 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions2, _ := adapter.AdaptEntry(line2)
				agent := sessions2[0].Data.(*session.AgentMessage)
				Expect(agent.TreeNodeMeta.SessionID).To(Equal("sess-1"))

				// response_item assistant
				role := "assistant"
				line3 := makeEntry("response_item", ts, map[string]any{
					"type": "message",
					"role": role,
					"content": []any{
						map[string]any{"type": "output_text", "text": "answer"},
					},
				})
				sessions3, _ := adapter.AdaptEntry(line3)
				agentRI := sessions3[0].Data.(*session.AgentMessage)
				Expect(agentRI.TreeNodeMeta.SessionID).To(Equal("sess-1"))
			})

			It("populates Model from cached turn_context", func() {
				tcLine := makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t1",
					"cwd":     "/path",
					"model":   "o3-pro",
				})
				_, err := adapter.AdaptEntry(tcLine)
				Expect(err).NotTo(HaveOccurred())

				// agent_message
				line1 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions1, _ := adapter.AdaptEntry(line1)
				agent1 := sessions1[0].Data.(*session.AgentMessage)
				Expect(agent1.Model).To(Equal("o3-pro"))

				// response_item assistant
				role := "assistant"
				line2 := makeEntry("response_item", ts, map[string]any{
					"type": "message",
					"role": role,
					"content": []any{
						map[string]any{"type": "output_text", "text": "answer"},
					},
				})
				sessions2, _ := adapter.AdaptEntry(line2)
				agent2 := sessions2[0].Data.(*session.AgentMessage)
				Expect(agent2.Model).To(Equal("o3-pro"))
			})

			It("uses default provider when session_meta not yet processed", func() {
				line := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				})
				sessions, err := adapter.AdaptEntry(line)
				Expect(err).NotTo(HaveOccurred())
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Provider).To(Equal(codexcliadapter.DefaultModelProvider))
			})

			It("updates model when turn_context changes", func() {
				// First turn_context
				tc1 := makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t1",
					"cwd":     "/path",
					"model":   "o3-pro",
				})
				_, _ = adapter.AdaptEntry(tc1)

				msg1 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response1",
				})
				sessions1, _ := adapter.AdaptEntry(msg1)
				Expect(sessions1[0].Data.(*session.AgentMessage).Model).To(Equal("o3-pro"))

				// Second turn_context
				tc2 := makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t2",
					"cwd":     "/path",
					"model":   "o4-mini",
				})
				_, _ = adapter.AdaptEntry(tc2)

				msg2 := makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response2",
				})
				sessions2, _ := adapter.AdaptEntry(msg2)
				Expect(sessions2[0].Data.(*session.AgentMessage).Model).To(Equal("o4-mini"))
			})
		})
	})

	Describe("AdaptBatch", func() {
		It("converts multiple lines to combined sessions", func() {
			lines := []string{
				makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				}),
				makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				}),
			}
			sessions := adapter.AdaptBatch(lines)
			Expect(sessions).To(HaveLen(2))
		})

		It("skips invalid lines and continues processing", func() {
			lines := []string{
				makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				}),
				"{invalid json",
				makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				}),
			}
			sessions := adapter.AdaptBatch(lines)
			Expect(sessions).To(HaveLen(2))
		})

		It("returns nil for empty input", func() {
			sessions := adapter.AdaptBatch([]string{})
			Expect(sessions).To(BeNil())
		})

		It("processes metadata before content (two-pass)", func() {
			// Metadata placed AFTER content entries in input order
			lines := []string{
				makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				}),
				makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-abc",
					"cwd": "/path",
				}),
				makeEntry("turn_context", ts, map[string]any{
					"turn_id": "t1",
					"cwd":     "/path",
					"model":   "o3-pro",
				}),
				makeEntry("event_msg", ts, map[string]any{
					"type":    "agent_message",
					"message": "response",
				}),
			}
			sessions := adapter.AdaptBatch(lines)
			// Should have 2 content sessions (user_message + agent_message)
			Expect(sessions).To(HaveLen(2))

			// All sessions should have the cached SessionID from session_meta
			// even though the event_msg at index 0 was before session_meta at index 1
			human := sessions[0].Data.(*session.HumanMessage)
			Expect(human.TreeNodeMeta.SessionID).To(Equal("sess-abc"))

			agent := sessions[1].Data.(*session.AgentMessage)
			Expect(agent.TreeNodeMeta.SessionID).To(Equal("sess-abc"))
			Expect(agent.Model).To(Equal("o3-pro"))
		})

		It("isolates state between separate adapter instances", func() {
			adapter1 := codexcliadapter.NewAdapter(slog.Default())
			adapter2 := codexcliadapter.NewAdapter(slog.Default())

			// adapter1 processes session_meta
			lines1 := []string{
				makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-A",
					"cwd": "/path",
				}),
			}
			adapter1.AdaptBatch(lines1)

			// adapter2 processes event_msg without prior session_meta
			lines2 := []string{
				makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				}),
			}
			sessions := adapter2.AdaptBatch(lines2)
			Expect(sessions).To(HaveLen(1))
			human := sessions[0].Data.(*session.HumanMessage)
			Expect(human.TreeNodeMeta.SessionID).To(BeEmpty())
		})

		It("handles multiple session_meta entries (later overrides)", func() {
			lines := []string{
				makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-A",
					"cwd": "/path",
				}),
				makeEntry("session_meta", ts, map[string]any{
					"id":  "sess-B",
					"cwd": "/path",
				}),
				makeEntry("event_msg", ts, map[string]any{
					"type":    "user_message",
					"message": "hello",
				}),
			}
			sessions := adapter.AdaptBatch(lines)
			Expect(sessions).To(HaveLen(1))
			human := sessions[0].Data.(*session.HumanMessage)
			Expect(human.TreeNodeMeta.SessionID).To(Equal("sess-B"))
		})
	})
})

var _ = Describe("Types", func() {
	Describe("SessionMetaPayload", func() {
		It("parses JSON with all fields", func() {
			raw := `{"id":"abc","cwd":"/path","model_provider":"openai","originator":"codex_cli_rs","cli_version":"0.106.0","source":"cli"}`
			var payload codexcliadapter.SessionMetaPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.ID).To(Equal("abc"))
			Expect(payload.CWD).To(Equal("/path"))
			Expect(payload.ModelProvider).NotTo(BeNil())
			Expect(*payload.ModelProvider).To(Equal("openai"))
			Expect(payload.Originator).NotTo(BeNil())
			Expect(*payload.Originator).To(Equal("codex_cli_rs"))
			Expect(payload.CLIVersion).NotTo(BeNil())
			Expect(*payload.CLIVersion).To(Equal("0.106.0"))
		})

		It("parses JSON missing optional fields", func() {
			raw := fmt.Sprintf(`{"id":"abc","timestamp":"%s","cwd":"/path"}`, time.Now().Format(time.RFC3339Nano))
			var payload codexcliadapter.SessionMetaPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.ID).To(Equal("abc"))
			Expect(payload.ModelProvider).To(BeNil())
			Expect(payload.Originator).To(BeNil())
			Expect(payload.CLIVersion).To(BeNil())
		})

		It("parses JSON with object Source", func() {
			raw := `{"id":"abc","cwd":"/path","source":{"subagent":{"thread_spawn":{}}}}`
			var payload codexcliadapter.SessionMetaPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.Source).NotTo(BeNil())
			sourceMap, ok := payload.Source.(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(sourceMap).To(HaveKey("subagent"))
		})
	})

	Describe("EventMsgPayload", func() {
		It("parses valid user_message JSON", func() {
			raw := `{"type":"user_message","message":"hello"}`
			var payload codexcliadapter.EventMsgPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.Type).To(Equal("user_message"))
			Expect(payload.Message).To(Equal("hello"))
		})
	})

	Describe("ResponseItemPayload", func() {
		It("parses valid assistant message JSON", func() {
			raw := `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]}`
			var payload codexcliadapter.ResponseItemPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.Type).To(Equal("message"))
			Expect(*payload.Role).To(Equal("assistant"))
			Expect(payload.Content).To(HaveLen(1))
			Expect(payload.Content[0].Type).To(Equal("output_text"))
			Expect(payload.Content[0].Text).To(Equal("hi"))
		})
	})

	Describe("TurnContextPayload", func() {
		It("parses valid turn_context JSON", func() {
			raw := `{"turn_id":"t1","model":"o3-pro","cwd":"/path"}`
			var payload codexcliadapter.TurnContextPayload
			err := json.Unmarshal([]byte(raw), &payload)
			Expect(err).NotTo(HaveOccurred())
			Expect(payload.TurnID).To(Equal("t1"))
			Expect(payload.Model).To(Equal("o3-pro"))
		})
	})

	Describe("TokenUsageDetail", func() {
		It("parses without ReasoningOutputTokens", func() {
			raw := `{"input_tokens":100,"cached_input_tokens":50,"output_tokens":200,"total_tokens":350}`
			var detail codexcliadapter.TokenUsageDetail
			err := json.Unmarshal([]byte(raw), &detail)
			Expect(err).NotTo(HaveOccurred())
			Expect(detail.InputTokens).To(Equal(100))
			Expect(detail.CachedInputTokens).To(Equal(50))
			Expect(detail.OutputTokens).To(Equal(200))
			Expect(detail.TotalTokens).To(Equal(350))
			Expect(detail.ReasoningOutputTokens).To(BeNil())
		})
	})
})
