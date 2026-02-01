package claudecodeadapter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
	session "github.com/team-attention/cops/shared/domain/v2"
	"github.com/team-attention/cops/shared/domain/v2/claudecodeadapter"
)

func TestClaudeCodeAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClaudeCodeAdapter Suite")
}

var _ = Describe("Converter", func() {
	Describe("SubAgent Metadata Conversion", func() {
		var adapter *claudecodeadapter.Adapter

		BeforeEach(func() {
			adapter = claudecodeadapter.NewAdapter()
		})

		Context("when processing assistant transcripts with tool_use", func() {
			It("populates Metadata for SubAgent tool execution", func() {
				agentID := "test-agent-id"
				toolUseID := "toolu_123"
				toolName := "Bash"

				ast := &domain.AssistantTranscript{
					RequestID: "req_123",
					Message: domain.AssistantTranscriptMessage{
						Model: "claude-opus-4-5-20251101",
						Content: []*domain.AssistantContentBlock{
							{
								Type:  "tool_use",
								ID:    &toolUseID,
								Name:  &toolName,
								Input: map[string]any{"command": "ls"},
							},
						},
					},
				}
				ast.TreeNodeMeta.UUID = "uuid-123"
				ast.TreeNodeMeta.AgentID = &agentID

				transcript := &domain.Transcript{
					Type: domain.TranscriptTypeAssistant,
					Data: ast,
				}

				sessions, err := adapter.Adapt(transcript)
				Expect(err).NotTo(HaveOccurred())

				// Find ToolExecution session
				var toolExec *session.ToolExecution
				for _, s := range sessions {
					if s.Type == session.SessionTypeToolExecution {
						toolExec = s.Data.(*session.ToolExecution)
						break
					}
				}

				Expect(toolExec).NotTo(BeNil())
				Expect(toolExec.Metadata).NotTo(BeNil())
				Expect(toolExec.Metadata.IsSubagent).To(BeTrue())
				Expect(toolExec.Metadata.SubagentID).To(Equal("test-agent-id"))
			})

			It("leaves Metadata nil for main session tool execution", func() {
				toolUseID := "toolu_456"
				toolName := "Read"

				ast := &domain.AssistantTranscript{
					RequestID: "req_456",
					Message: domain.AssistantTranscriptMessage{
						Model: "claude-opus-4-5-20251101",
						Content: []*domain.AssistantContentBlock{
							{
								Type:  "tool_use",
								ID:    &toolUseID,
								Name:  &toolName,
								Input: map[string]any{"file_path": "/test.txt"},
							},
						},
					},
				}
				ast.TreeNodeMeta.UUID = "uuid-456"
				ast.TreeNodeMeta.AgentID = nil // Main session

				transcript := &domain.Transcript{
					Type: domain.TranscriptTypeAssistant,
					Data: ast,
				}

				sessions, err := adapter.Adapt(transcript)
				Expect(err).NotTo(HaveOccurred())

				var toolExec *session.ToolExecution
				for _, s := range sessions {
					if s.Type == session.SessionTypeToolExecution {
						toolExec = s.Data.(*session.ToolExecution)
						break
					}
				}

				Expect(toolExec).NotTo(BeNil())
				Expect(toolExec.Metadata).To(BeNil())
			})

			It("leaves Metadata nil when AgentID is empty string", func() {
				emptyAgentID := ""
				toolUseID := "toolu_789"
				toolName := "Write"

				ast := &domain.AssistantTranscript{
					RequestID: "req_789",
					Message: domain.AssistantTranscriptMessage{
						Model: "claude-opus-4-5-20251101",
						Content: []*domain.AssistantContentBlock{
							{
								Type:  "tool_use",
								ID:    &toolUseID,
								Name:  &toolName,
								Input: map[string]any{"file_path": "/test.txt"},
							},
						},
					},
				}
				ast.TreeNodeMeta.UUID = "uuid-789"
				ast.TreeNodeMeta.AgentID = &emptyAgentID

				transcript := &domain.Transcript{
					Type: domain.TranscriptTypeAssistant,
					Data: ast,
				}

				sessions, err := adapter.Adapt(transcript)
				Expect(err).NotTo(HaveOccurred())

				var toolExec *session.ToolExecution
				for _, s := range sessions {
					if s.Type == session.SessionTypeToolExecution {
						toolExec = s.Data.(*session.ToolExecution)
						break
					}
				}

				Expect(toolExec).NotTo(BeNil())
				Expect(toolExec.Metadata).To(BeNil())
			})
		})

		Context("when processing user transcripts with tool_result", func() {
			It("populates Metadata for SubAgent tool result", func() {
				agentID := "test-agent-id"

				u := &domain.UserTranscript{
					Message: domain.UserTranscriptMessage{
						Content: []any{
							map[string]any{
								"type":        "tool_result",
								"tool_use_id": "toolu_123",
								"content":     "success",
							},
						},
					},
				}
				u.TreeNodeMeta.UUID = "uuid-user-123"
				u.TreeNodeMeta.AgentID = &agentID

				transcript := &domain.Transcript{
					Type: domain.TranscriptTypeUser,
					Data: u,
				}

				sessions, err := adapter.Adapt(transcript)
				Expect(err).NotTo(HaveOccurred())

				var toolExec *session.ToolExecution
				for _, s := range sessions {
					if s.Type == session.SessionTypeToolExecution {
						toolExec = s.Data.(*session.ToolExecution)
						break
					}
				}

				Expect(toolExec).NotTo(BeNil())
				Expect(toolExec.Metadata).NotTo(BeNil())
				Expect(toolExec.Metadata.IsSubagent).To(BeTrue())
				Expect(toolExec.Metadata.SubagentID).To(Equal("test-agent-id"))
			})

			It("leaves Metadata nil for main session tool result", func() {
				u := &domain.UserTranscript{
					Message: domain.UserTranscriptMessage{
						Content: []any{
							map[string]any{
								"type":        "tool_result",
								"tool_use_id": "toolu_456",
								"content":     "result data",
							},
						},
					},
				}
				u.TreeNodeMeta.UUID = "uuid-user-456"
				u.TreeNodeMeta.AgentID = nil // Main session

				transcript := &domain.Transcript{
					Type: domain.TranscriptTypeUser,
					Data: u,
				}

				sessions, err := adapter.Adapt(transcript)
				Expect(err).NotTo(HaveOccurred())

				var toolExec *session.ToolExecution
				for _, s := range sessions {
					if s.Type == session.SessionTypeToolExecution {
						toolExec = s.Data.(*session.ToolExecution)
						break
					}
				}

				Expect(toolExec).NotTo(BeNil())
				Expect(toolExec.Metadata).To(BeNil())
			})
		})
	})
})
