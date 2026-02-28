package opencodeadapter_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	session "github.com/team-attention/cops/shared/domain/v2"
	"github.com/team-attention/cops/shared/domain/v2/opencodeadapter"
)

func TestOpenCodeAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenCodeAdapter Suite")
}

// newTestMessage creates an OpenCodeMessage with common defaults for testing.
// Only role and parts are required; other fields are set to sensible defaults.
func newTestMessage(role, parts string) *opencodeadapter.OpenCodeMessage {
	return &opencodeadapter.OpenCodeMessage{
		ID:        "msg_test",
		SessionID: "sess_test",
		Role:      role,
		Parts:     parts,
		Model:     "claude-sonnet-4-20250514",
		CreatedAt: 1700000000,
		UpdatedAt: 1700000005,
	}
}

var _ = Describe("Converter", func() {
	var adapter *opencodeadapter.Adapter

	BeforeEach(func() {
		adapter = opencodeadapter.NewAdapter()
	})

	Describe("AdaptMessage", func() {
		Context("when processing user messages", func() {
			It("converts text part to HumanMessage", func() {
				msg := &opencodeadapter.OpenCodeMessage{
					ID:        "msg_1",
					SessionID: "sess_1",
					Role:      "user",
					Parts:     `[{"type":"text","text":"hello"}]`,
					CreatedAt: 1700000000,
					UpdatedAt: 1700000005,
				}

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeHuman))

				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.Content).To(HaveLen(1))
				Expect(human.Content[0].Type).To(Equal(session.HumanContentBlockTypeText))
				Expect(*human.Content[0].Text).To(Equal("hello"))
				Expect(human.TreeNodeMeta.Provider).To(Equal("open_code"))
				Expect(human.TreeNodeMeta.SessionID).To(Equal("sess_1"))
				Expect(human.TreeNodeMeta.UUID).To(Equal("msg_1"))
				Expect(human.TreeNodeMeta.Timestamp).To(Equal(time.Unix(1700000000, 0)))
			})

			It("converts tool results to ToolExecution", func() {
				msg := newTestMessage("user", `[{"type":"tool-result","toolInvocationId":"t1","toolName":"Bash","result":"ok"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(2)) // HumanMessage + ToolExecution

				// First session is HumanMessage (empty content)
				Expect(sessions[0].Type).To(Equal(session.SessionTypeHuman))

				// Second session is ToolExecution
				Expect(sessions[1].Type).To(Equal(session.SessionTypeToolExecution))
				toolExec := sessions[1].Data.(*session.ToolExecution)
				Expect(toolExec.ID).To(Equal("t1"))
				Expect(toolExec.ToolName).To(Equal("Bash"))
				Expect(toolExec.Result).NotTo(BeNil())
				Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusSuccess))
				Expect(toolExec.Result.Content).To(Equal("ok"))
			})

			It("handles mixed text and tool-result parts", func() {
				msg := newTestMessage("user", `[{"type":"text","text":"done"},{"type":"tool-result","toolInvocationId":"t1","result":"output"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(2)) // HumanMessage with text + ToolExecution

				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.Content).To(HaveLen(1))
				Expect(*human.Content[0].Text).To(Equal("done"))

				toolExec := sessions[1].Data.(*session.ToolExecution)
				Expect(toolExec.ID).To(Equal("t1"))
			})
		})

		Context("when processing assistant messages", func() {
			It("converts text part to AgentMessage", func() {
				msg := &opencodeadapter.OpenCodeMessage{
					ID:        "msg_2",
					SessionID: "sess_1",
					Role:      "assistant",
					Parts:     `[{"type":"text","text":"response"}]`,
					Model:     "claude-sonnet-4-20250514",
					CreatedAt: 1700000000,
					UpdatedAt: 1700000005,
				}

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))
				Expect(sessions[0].Type).To(Equal(session.SessionTypeAgent))

				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeText))
				Expect(*agent.Content[0].Text).To(Equal("response"))
				Expect(agent.Model).To(Equal("claude-sonnet-4-20250514"))
				Expect(agent.Provider).To(Equal("open_code"))
			})

			It("converts tool invocations to ToolExecution", func() {
				msg := newTestMessage("assistant",
					`[{"type":"tool-invocation","toolInvocationId":"t1","toolName":"Bash","args":{"command":"ls"},"state":"result","result":"file.txt"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(2)) // AgentMessage + ToolExecution

				// Check AgentMessage has ToolCallRef
				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(1))
				Expect(agent.Content[0].Type).To(Equal(session.AgentContentBlockTypeToolCallRef))
				Expect(agent.Content[0].ToolCallRef.ToolExecutionID).To(Equal("t1"))
				Expect(agent.Content[0].ToolCallRef.ToolName).To(Equal("Bash"))

				// Check ToolExecution
				toolExec := sessions[1].Data.(*session.ToolExecution)
				Expect(toolExec.ID).To(Equal("t1"))
				Expect(toolExec.ToolName).To(Equal("Bash"))
				Expect(toolExec.Input).To(HaveKeyWithValue("command", "ls"))
				Expect(toolExec.SourceAgentUUID).To(Equal("msg_test"))
				Expect(toolExec.Result).NotTo(BeNil())
				Expect(toolExec.Result.Status).To(Equal(session.ToolResultStatusSuccess))
				Expect(toolExec.Result.Content).To(Equal("file.txt"))
			})

			It("skips partial-call tool invocations", func() {
				msg := newTestMessage("assistant",
					`[{"type":"tool-invocation","toolInvocationId":"t1","state":"partial-call"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1)) // Only AgentMessage, no ToolExecution

				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(BeEmpty())
			})

			It("handles tool invocation with call state (no result)", func() {
				msg := newTestMessage("assistant",
					`[{"type":"tool-invocation","toolInvocationId":"t1","toolName":"Read","args":{"file_path":"/test.txt"},"state":"call"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(2)) // AgentMessage + ToolExecution

				toolExec := sessions[1].Data.(*session.ToolExecution)
				Expect(toolExec.ID).To(Equal("t1"))
				Expect(toolExec.ToolName).To(Equal("Read"))
				Expect(toolExec.Result).To(BeNil()) // No result for "call" state
			})

			It("handles nil usage (no tokens in schema)", func() {
				msg := newTestMessage("assistant", `[{"type":"text","text":"hi"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())

				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Usage).To(BeNil())
			})

			It("sets Provider to open_code", func() {
				msg := newTestMessage("assistant", `[{"type":"text","text":"hi"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())

				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Provider).To(Equal("open_code"))
			})

			It("handles multiple text parts", func() {
				msg := newTestMessage("assistant", `[{"type":"text","text":"a"},{"type":"text","text":"b"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())

				agent := sessions[0].Data.(*session.AgentMessage)
				Expect(agent.Content).To(HaveLen(2))
				Expect(*agent.Content[0].Text).To(Equal("a"))
				Expect(*agent.Content[1].Text).To(Equal("b"))
			})

			It("handles result as JSON object", func() {
				msg := newTestMessage("assistant",
					`[{"type":"tool-invocation","toolInvocationId":"t1","toolName":"Test","args":{},"state":"result","result":{"key":"val"}}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())

				toolExec := sessions[1].Data.(*session.ToolExecution)
				resultMap, ok := toolExec.Result.Content.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(resultMap["key"]).To(Equal("val"))
			})
		})

		Context("when processing unknown roles", func() {
			It("returns nil for unrecognized role", func() {
				msg := newTestMessage("system", `[{"type":"text","text":"system msg"}]`)

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(BeNil())
			})
		})

		Context("when parts JSON is malformed", func() {
			It("returns error for invalid parts JSON", func() {
				msg := newTestMessage("user", "not-json")

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).To(HaveOccurred())
				Expect(sessions).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("parse parts JSON"))
			})

			It("returns error for invalid parts JSON in assistant message", func() {
				msg := newTestMessage("assistant", "{broken")

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).To(HaveOccurred())
				Expect(sessions).To(BeNil())
			})
		})

		Context("when parts is empty", func() {
			It("returns HumanMessage with empty content for user", func() {
				msg := newTestMessage("user", "")

				sessions, err := adapter.AdaptMessage(msg)
				Expect(err).NotTo(HaveOccurred())
				Expect(sessions).To(HaveLen(1))

				human := sessions[0].Data.(*session.HumanMessage)
				Expect(human.Content).To(BeEmpty())
			})
		})
	})

	Describe("AdaptBatch", func() {
		It("converts multiple messages", func() {
			messages := []*opencodeadapter.OpenCodeMessage{
				newTestMessage("user", `[{"type":"text","text":"hello"}]`),
				newTestMessage("assistant", `[{"type":"text","text":"hi"}]`),
				newTestMessage("user", `[{"type":"text","text":"bye"}]`),
			}

			sessions, err := adapter.AdaptBatch(messages)
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).To(HaveLen(3))
			Expect(sessions[0].Type).To(Equal(session.SessionTypeHuman))
			Expect(sessions[1].Type).To(Equal(session.SessionTypeAgent))
			Expect(sessions[2].Type).To(Equal(session.SessionTypeHuman))
		})

		It("stops on first error and returns it", func() {
			messages := []*opencodeadapter.OpenCodeMessage{
				newTestMessage("user", `[{"type":"text","text":"hello"}]`),
				newTestMessage("user", "not-json"), // This will fail
				newTestMessage("user", `[{"type":"text","text":"bye"}]`),
			}

			sessions, err := adapter.AdaptBatch(messages)
			Expect(err).To(HaveOccurred())
			Expect(sessions).To(BeNil())
		})
	})

	Describe("ParseParts", func() {
		It("returns nil, nil for empty parts string", func() {
			msg := &opencodeadapter.OpenCodeMessage{Parts: ""}
			parts, err := msg.ParseParts()
			Expect(err).NotTo(HaveOccurred())
			Expect(parts).To(BeNil())
		})

		It("returns nil, error for invalid JSON", func() {
			msg := &opencodeadapter.OpenCodeMessage{Parts: "not-json"}
			parts, err := msg.ParseParts()
			Expect(err).To(HaveOccurred())
			Expect(parts).To(BeNil())
		})

		It("parses valid parts JSON", func() {
			msg := &opencodeadapter.OpenCodeMessage{Parts: `[{"type":"text","text":"hi"}]`}
			parts, err := msg.ParseParts()
			Expect(err).NotTo(HaveOccurred())
			Expect(parts).To(HaveLen(1))
			Expect(parts[0].Type).To(Equal("text"))
			Expect(*parts[0].Text).To(Equal("hi"))
		})

		It("returns nil, nil for nil receiver", func() {
			var msg *opencodeadapter.OpenCodeMessage
			parts, err := msg.ParseParts()
			Expect(err).NotTo(HaveOccurred())
			Expect(parts).To(BeNil())
		})
	})

	Describe("createdTime (tested indirectly)", func() {
		It("sets correct timestamp for non-zero CreatedAt", func() {
			msg := newTestMessage("user", `[{"type":"text","text":"hi"}]`)
			msg.CreatedAt = 1700000000

			sessions, err := adapter.AdaptMessage(msg)
			Expect(err).NotTo(HaveOccurred())

			human := sessions[0].Data.(*session.HumanMessage)
			Expect(human.TreeNodeMeta.Timestamp).To(Equal(time.Unix(1700000000, 0)))
		})

		It("sets zero timestamp for zero CreatedAt", func() {
			msg := newTestMessage("user", `[{"type":"text","text":"hi"}]`)
			msg.CreatedAt = 0

			sessions, err := adapter.AdaptMessage(msg)
			Expect(err).NotTo(HaveOccurred())

			human := sessions[0].Data.(*session.HumanMessage)
			Expect(human.TreeNodeMeta.Timestamp).To(Equal(time.Time{}))
		})
	})
})
