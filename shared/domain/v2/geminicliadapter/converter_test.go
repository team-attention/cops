package geminicliadapter_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	session "github.com/team-attention/cops/shared/domain/v2"
	"github.com/team-attention/cops/shared/domain/v2/geminicliadapter"
)

func TestGeminiCLIAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GeminiCLIAdapter Suite")
}

var _ = Describe("Converter", func() {
	var adapter *geminicliadapter.Adapter

	BeforeEach(func() {
		adapter = geminicliadapter.NewAdapter()
	})

	Describe("AdaptSession", func() {
		Context("when session has multiple messages", func() {
			It("converts all messages in order", func() {
				now := time.Now()
				sess := &geminicliadapter.GeminiSession{
					SessionID:   "session-1",
					ProjectHash: "abc123",
					StartTime:   now,
					LastUpdated: now,
					Messages: []*geminicliadapter.GeminiMessage{
						{
							ID:        "msg-1",
							Timestamp: now,
							Type:      "user",
							Content:   "Hello",
						},
						{
							ID:        "msg-2",
							Timestamp: now.Add(time.Second),
							Type:      "gemini",
							Content:   "Hi there",
							Model:     "gemini-2.5-pro",
						},
						{
							ID:        "msg-3",
							Timestamp: now.Add(2 * time.Second),
							Type:      "info",
							Content:   "Context loaded",
						},
					},
				}

				results, err := adapter.AdaptSession(sess)
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(3))
				Expect(results[0].Type).To(Equal(session.SessionTypeHuman))
				Expect(results[1].Type).To(Equal(session.SessionTypeAgent))
				Expect(results[2].Type).To(Equal(session.SessionTypeSystem))
			})
		})

		Context("when session has no messages", func() {
			It("returns empty slice", func() {
				sess := &geminicliadapter.GeminiSession{
					SessionID:   "session-empty",
					ProjectHash: "abc123",
					StartTime:   time.Now(),
					LastUpdated: time.Now(),
					Messages:    []*geminicliadapter.GeminiMessage{},
				}

				results, err := adapter.AdaptSession(sess)
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})
	})

	Describe("AdaptMessage", func() {
		Context("user message", func() {
			It("converts to HumanMessage with text content", func() {
				now := time.Now()
				msg := &geminicliadapter.GeminiMessage{
					ID:        "user-1",
					Timestamp: now,
					Type:      "user",
					Content:   "What is Go?",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].Type).To(Equal(session.SessionTypeHuman))

				human := results[0].Data.(*session.HumanMessage)
				Expect(human.UUID).To(Equal("user-1"))
				Expect(human.SessionID).To(Equal("session-1"))
				Expect(human.Provider).To(Equal("gemini_cli"))
				Expect(human.Timestamp).To(Equal(now))
				Expect(human.Content).To(HaveLen(1))
				Expect(human.Content[0].Type).To(Equal(session.HumanContentBlockTypeText))
				Expect(*human.Content[0].Text).To(Equal("What is Go?"))
			})
		})

		Context("gemini message with text only", func() {
			It("converts to AgentMessage with text content block", func() {
				now := time.Now()
				msg := &geminicliadapter.GeminiMessage{
					ID:        "gemini-1",
					Timestamp: now,
					Type:      "gemini",
					Content:   "Go is a programming language.",
					Model:     "gemini-2.5-pro",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].Type).To(Equal(session.SessionTypeAgent))

				agent := results[0].Data.(*session.AgentMessage)
				Expect(agent.UUID).To(Equal("gemini-1"))
				Expect(agent.SessionID).To(Equal("session-1"))
				Expect(agent.Provider).To(Equal("google"))
				Expect(agent.Model).To(Equal("gemini-2.5-pro"))
				Expect(agent.TreeNodeMeta.Provider).To(Equal("gemini_cli"))
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeText))
				Expect(*agent.Content[0].Text).To(Equal("Go is a programming language."))
			})
		})

		Context("gemini message with empty content", func() {
			It("does not add text block for empty content", func() {
				msg := &geminicliadapter.GeminiMessage{
					ID:        "gemini-2",
					Timestamp: time.Now(),
					Type:      "gemini",
					Content:   "",
					Model:     "gemini-2.5-pro",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(1))

				agent := results[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(BeEmpty())
			})
		})

		Context("gemini message with thinking", func() {
			It("converts thoughts to thinking content blocks", func() {
				msg := &geminicliadapter.GeminiMessage{
					ID:        "gemini-3",
					Timestamp: time.Now(),
					Type:      "gemini",
					Content:   "The answer is 42.",
					Model:     "gemini-2.5-pro",
					Thoughts: []*geminicliadapter.GeminiThought{
						{
							Subject:     "Analyzing the question",
							Description: "The user is asking about the meaning of life.",
							Timestamp:   time.Now(),
						},
					},
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())

				agent := results[0].Data.(*session.AgentMessage)
				// Thinking block comes first, then text block
				Expect(agent.Content).To(HaveLen(2))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeThinking))
				Expect(agent.Content[0].Thinking.Content).To(Equal("Analyzing the question\n\nThe user is asking about the meaning of life."))
				Expect(agent.Content[1].Type).To(Equal(session.AgentContentBlockTypeText))
			})

			It("handles thought with subject only", func() {
				msg := &geminicliadapter.GeminiMessage{
					ID:        "gemini-4",
					Timestamp: time.Now(),
					Type:      "gemini",
					Content:   "",
					Model:     "gemini-2.5-pro",
					Thoughts: []*geminicliadapter.GeminiThought{
						{
							Subject:     "Processing request",
							Description: "",
							Timestamp:   time.Now(),
						},
					},
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())

				agent := results[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeThinking))
				Expect(agent.Content[0].Thinking.Content).To(Equal("Processing request"))
			})
		})

		Context("gemini message with tool calls", func() {
			It("creates AgentMessage with refs and ToolExecution sessions", func() {
				now := time.Now()
				msg := &geminicliadapter.GeminiMessage{
					ID:        "gemini-5",
					Timestamp: now,
					Type:      "gemini",
					Content:   "",
					Model:     "gemini-2.5-pro",
					ToolCalls: []*geminicliadapter.GeminiToolCall{
						{
							ID:        "tc-1",
							Name:      "read_file",
							Args:      map[string]any{"path": "/tmp/test.txt"},
							Result:    []any{"file contents here"},
							Status:    "success",
							Timestamp: now,
						},
						{
							ID:        "tc-2",
							Name:      "write_file",
							Args:      map[string]any{"path": "/tmp/out.txt", "content": "data"},
							Result:    []any{"written successfully"},
							Status:    "success",
							Timestamp: now,
						},
					},
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				// 1 AgentMessage + 2 ToolExecutions
				Expect(results).To(HaveLen(3))

				Expect(results[0].Type).To(Equal(session.SessionTypeAgent))
				agent := results[0].Data.(*session.AgentMessage)
				// 2 tool_call_ref blocks (no text block since content is empty)
				Expect(agent.Content).To(HaveLen(2))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeToolCallRef))
				Expect(agent.Content[0].ToolCallRef.ToolExecutionID).To(Equal("tc-1"))
				Expect(agent.Content[0].ToolCallRef.ToolName).To(Equal("read_file"))
				Expect(agent.Content[1].Type).To(Equal(session.AgentContentBlockTypeToolCallRef))
				Expect(agent.Content[1].ToolCallRef.ToolExecutionID).To(Equal("tc-2"))

				Expect(results[1].Type).To(Equal(session.SessionTypeToolExecution))
				toolExec1 := results[1].Data.(*session.ToolExecution)
				Expect(toolExec1.ID).To(Equal("tc-1"))
				Expect(toolExec1.ToolName).To(Equal("read_file"))
				Expect(toolExec1.Input).To(Equal(map[string]any{"path": "/tmp/test.txt"}))
				Expect(toolExec1.SourceAgentUUID).To(Equal("gemini-5"))
				Expect(toolExec1.Result.Status).To(Equal(session.ToolResultStatusSuccess))
				Expect(toolExec1.Result.Content).To(Equal("file contents here"))

				Expect(results[2].Type).To(Equal(session.SessionTypeToolExecution))
				toolExec2 := results[2].Data.(*session.ToolExecution)
				Expect(toolExec2.ID).To(Equal("tc-2"))
				Expect(toolExec2.ToolName).To(Equal("write_file"))
			})
		})

		Context("info message", func() {
			It("converts to SystemMessage with info subtype and content", func() {
				now := time.Now()
				msg := &geminicliadapter.GeminiMessage{
					ID:        "info-1",
					Timestamp: now,
					Type:      "info",
					Content:   "Loading context...",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].Type).To(Equal(session.SessionTypeSystem))

				sys := results[0].Data.(*session.SystemMessage)
				Expect(sys.UUID).To(Equal("info-1"))
				Expect(sys.SessionID).To(Equal("session-1"))
				Expect(sys.Provider).To(Equal("gemini_cli"))
				Expect(sys.Subtype).To(Equal(session.SystemMessageSubtypeInfo))
				Expect(sys.Info).NotTo(BeNil())
				Expect(sys.Info.Content).To(Equal("Loading context..."))
			})

			It("returns nil Info when content is empty", func() {
				msg := &geminicliadapter.GeminiMessage{
					ID:        "info-2",
					Timestamp: time.Now(),
					Type:      "info",
					Content:   "",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(HaveLen(1))

				sys := results[0].Data.(*session.SystemMessage)
				Expect(sys.Subtype).To(Equal(session.SystemMessageSubtypeInfo))
				Expect(sys.Info).To(BeNil())
			})
		})

		Context("unknown message type", func() {
			It("returns nil with no error", func() {
				msg := &geminicliadapter.GeminiMessage{
					ID:        "unknown-1",
					Timestamp: time.Now(),
					Type:      "unknown_type",
					Content:   "something",
				}

				results, err := adapter.AdaptMessage(msg, "session-1")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).To(BeNil())
			})
		})
	})

	Describe("Tool Status Conversion", func() {
		It("maps 'success' to ToolResultStatusSuccess", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "ts-1",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-s",
						Name:      "tool",
						Args:      map[string]any{},
						Result:    []any{"ok"},
						Status:    "success",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusSuccess))
		})

		It("maps 'error' to ToolResultStatusError", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "ts-2",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:            "tc-e",
						Name:          "tool",
						Args:          map[string]any{},
						Status:        "error",
						ResultDisplay: "something went wrong",
						Timestamp:     time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusError))
			Expect(toolExec.Result.Error).NotTo(BeNil())
			Expect(*toolExec.Result.Error).To(Equal("something went wrong"))
		})

		It("maps 'cancelled' to ToolResultStatusSkipped", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "ts-3",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-c",
						Name:      "tool",
						Args:      map[string]any{},
						Status:    "cancelled",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusSkipped))
		})

		It("maps unknown status to ToolResultStatusSuccess", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "ts-4",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-u",
						Name:      "tool",
						Args:      map[string]any{},
						Result:    []any{"ok"},
						Status:    "xyz",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusSuccess))
		})
	})

	Describe("Token Usage Conversion", func() {
		It("returns nil usage when tokens is nil", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "tok-1",
				Timestamp: time.Now(),
				Type:      "gemini",
				Content:   "text",
				Model:     "gemini-2.5-pro",
				Tokens:    nil,
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			agent := results[0].Data.(*session.AgentMessage)
			Expect(agent.Usage).To(BeNil())
		})

		It("maps token fields correctly", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "tok-2",
				Timestamp: time.Now(),
				Type:      "gemini",
				Content:   "text",
				Model:     "gemini-2.5-pro",
				Tokens: &geminicliadapter.GeminiTokens{
					Input:    100,
					Output:   50,
					Cached:   25,
					Thoughts: 10,
					Tool:     5,
					Total:    190,
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			agent := results[0].Data.(*session.AgentMessage)
			Expect(agent.Usage).NotTo(BeNil())
			Expect(agent.Usage.InputTokens).To(Equal(100))
			Expect(agent.Usage.OutputTokens).To(Equal(50))
			Expect(agent.Usage.CacheReadInputTokens).To(Equal(25))
		})
	})

	Describe("Tool Result Content Handling", func() {
		It("unwraps single-element result array", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "trc-1",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-single",
						Name:      "tool",
						Args:      map[string]any{},
						Result:    []any{"single value"},
						Status:    "success",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Content).To(Equal("single value"))
		})

		It("keeps multi-element result as array", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "trc-2",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-multi",
						Name:      "tool",
						Args:      map[string]any{},
						Result:    []any{"a", "b"},
						Status:    "success",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			arr, ok := toolExec.Result.Content.([]any)
			Expect(ok).To(BeTrue())
			Expect(arr).To(Equal([]any{"a", "b"}))
		})

		It("returns nil content for empty result", func() {
			msg := &geminicliadapter.GeminiMessage{
				ID:        "trc-3",
				Timestamp: time.Now(),
				Type:      "gemini",
				Model:     "gemini-2.5-pro",
				ToolCalls: []*geminicliadapter.GeminiToolCall{
					{
						ID:        "tc-empty",
						Name:      "tool",
						Args:      map[string]any{},
						Result:    nil,
						Status:    "success",
						Timestamp: time.Now(),
					},
				},
			}

			results, _ := adapter.AdaptMessage(msg, "s")
			toolExec := results[1].Data.(*session.ToolExecution)
			Expect(toolExec.Result.Content).To(BeNil())
		})
	})

	Describe("ParseSessionFile", func() {
		It("parses valid session JSON file", func() {
			tmpDir, err := os.MkdirTemp("", "gemini-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			jsonContent := `{
				"sessionId": "sess-123",
				"projectHash": "hash-abc",
				"startTime": "2026-02-28T10:00:00Z",
				"lastUpdated": "2026-02-28T10:05:00Z",
				"messages": [
					{
						"id": "msg-1",
						"timestamp": "2026-02-28T10:00:01Z",
						"type": "user",
						"content": "Hello"
					}
				]
			}`
			filePath := filepath.Join(tmpDir, "session-test.json")
			err = os.WriteFile(filePath, []byte(jsonContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			sess, err := geminicliadapter.ParseSessionFile(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(sess.SessionID).To(Equal("sess-123"))
			Expect(sess.ProjectHash).To(Equal("hash-abc"))
			Expect(sess.Messages).To(HaveLen(1))
			Expect(sess.Messages[0].ID).To(Equal("msg-1"))
			Expect(sess.Messages[0].Type).To(Equal("user"))
			Expect(sess.Messages[0].Content).To(Equal("Hello"))
		})

		It("returns error for non-existent file", func() {
			_, err := geminicliadapter.ParseSessionFile("/nonexistent/path/session.json")
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("returns error for invalid JSON", func() {
			tmpDir, err := os.MkdirTemp("", "gemini-test-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			filePath := filepath.Join(tmpDir, "session-bad.json")
			err = os.WriteFile(filePath, []byte("not json {{{"), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = geminicliadapter.ParseSessionFile(filePath)
			Expect(err).To(HaveOccurred())
		})
	})
})
