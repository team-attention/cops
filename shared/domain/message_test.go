package domain_test

import (
	"bufio"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

const testDataFile = "log_data.jsonl"

var _ = Describe("MessageContent", func() {
	Describe("UnmarshalJSON", func() {
		Context("when content is a string", func() {
			It("parses user message content correctly", func() {
				input := []byte(`"Hello world"`)
				var content domain.MessageContent
				err := json.Unmarshal(input, &content)

				Expect(err).NotTo(HaveOccurred())
				Expect(content.IsBlocks).To(BeFalse())
				Expect(content.Text).NotTo(BeNil())
				Expect(*content.Text).To(Equal("Hello world"))
			})

			It("handles empty string content", func() {
				input := []byte(`""`)
				var content domain.MessageContent
				err := json.Unmarshal(input, &content)

				Expect(err).NotTo(HaveOccurred())
				Expect(content.IsBlocks).To(BeFalse())
				Expect(content.Text).NotTo(BeNil())
				Expect(*content.Text).To(BeEmpty())
			})
		})

		Context("when content is an array of blocks", func() {
			Context("with text blocks", func() {
				It("parses text content block correctly", func() {
					input := []byte(`[{"type":"text","text":"Hi there"}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					textBlock, ok := content.Blocks[0].(*domain.TextContentBlock)
					Expect(ok).To(BeTrue())
					Expect(textBlock.Text).To(Equal("Hi there"))
				})
			})

			Context("with tool_use blocks", func() {
				It("parses tool use block with nested input", func() {
					input := []byte(`[{"type":"tool_use","id":"toolu_123","name":"Read","input":{"file_path":"/a/b.go"}}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					toolUseBlock, ok := content.Blocks[0].(*domain.ToolUseContentBlock)
					Expect(ok).To(BeTrue())
					Expect(toolUseBlock.ID).To(Equal("toolu_123"))
					Expect(toolUseBlock.Name).To(Equal("Read"))
					Expect(toolUseBlock.Input).To(HaveKeyWithValue("file_path", "/a/b.go"))
				})
			})

			Context("with tool_result blocks", func() {
				It("parses tool result block correctly", func() {
					input := []byte(`[{"type":"tool_result","tool_use_id":"toolu_123","content":"success","is_error":false}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					toolResultBlock, ok := content.Blocks[0].(*domain.ToolResultContentBlock)
					Expect(ok).To(BeTrue())
					Expect(toolResultBlock.ToolUseID).To(Equal("toolu_123"))
					Expect(toolResultBlock.Content).To(Equal("success"))
					Expect(toolResultBlock.IsError).To(BeFalse())
				})

				It("parses error tool result correctly", func() {
					input := []byte(`[{"type":"tool_result","tool_use_id":"toolu_456","content":"file not found","is_error":true}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					toolResultBlock, ok := content.Blocks[0].(*domain.ToolResultContentBlock)
					Expect(ok).To(BeTrue())
					Expect(toolResultBlock.IsError).To(BeTrue())
				})
			})

			Context("with thinking blocks", func() {
				It("parses thinking block with signature", func() {
					input := []byte(`[{"type":"thinking","thinking":"Let me analyze...","signature":"abc123"}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					thinkingBlock, ok := content.Blocks[0].(*domain.ThinkingContentBlock)
					Expect(ok).To(BeTrue())
					Expect(thinkingBlock.Thinking).To(Equal("Let me analyze..."))
					Expect(thinkingBlock.Signature).To(Equal("abc123"))
				})

				It("parses thinking block without signature", func() {
					input := []byte(`[{"type":"thinking","thinking":"Reasoning here"}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					thinkingBlock, ok := content.Blocks[0].(*domain.ThinkingContentBlock)
					Expect(ok).To(BeTrue())
					Expect(thinkingBlock.Thinking).To(Equal("Reasoning here"))
					Expect(thinkingBlock.Signature).To(BeEmpty())
				})
			})

			Context("with unknown block types", func() {
				It("skips unknown block types for forward compatibility", func() {
					input := []byte(`[{"type":"unknown","data":"something"}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(BeEmpty())
				})
			})

			Context("with mixed block types", func() {
				It("parses multiple block types in sequence", func() {
					input := []byte(`[{"type":"text","text":"Starting"},{"type":"tool_use","id":"t1","name":"Bash","input":{"cmd":"ls"}}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.Blocks).To(HaveLen(2))
					Expect(content.Blocks[0].BlockType()).To(Equal(domain.ContentBlockTypeText))
					Expect(content.Blocks[1].BlockType()).To(Equal(domain.ContentBlockTypeToolUse))
				})
			})

			Context("with real Claude Code session data", func() {
				It("parses actual assistant message with tool_use block from session logs", func() {
					// Real data from Claude Code JSONL session log
					input := []byte(`[{"type":"tool_use","id":"toolu_01MWqXkvw67AkXFwtGfUcr4J","name":"Skill","input":{"skill":"full-cycle","args":"Project를 등록할 때 Project 이름하고 Git 프로젝트인지, Remote URL이 뭔지 조사해서 올리는 것 같은데, 실제 저장되는 데이터에는 Remote URL 밖에 없어. 이 문제를 해결해줘."}}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					toolUseBlock, ok := content.Blocks[0].(*domain.ToolUseContentBlock)
					Expect(ok).To(BeTrue())
					Expect(toolUseBlock.ID).To(Equal("toolu_01MWqXkvw67AkXFwtGfUcr4J"))
					Expect(toolUseBlock.Name).To(Equal("Skill"))
					Expect(toolUseBlock.Input).To(HaveKeyWithValue("skill", "full-cycle"))
					Expect(toolUseBlock.Input).To(HaveKey("args"))
				})

				It("parses assistant message with text content from session logs", func() {
					// Real data showing text content
					input := []byte(`[{"type":"text","text":"I'll invoke the full-cycle skill to address this issue where Project registration collects name, Git info, and remote URL but only the remote URL is being stored."}]`)
					var content domain.MessageContent
					err := json.Unmarshal(input, &content)

					Expect(err).NotTo(HaveOccurred())
					Expect(content.IsBlocks).To(BeTrue())
					Expect(content.Blocks).To(HaveLen(1))

					textBlock, ok := content.Blocks[0].(*domain.TextContentBlock)
					Expect(ok).To(BeTrue())
					Expect(textBlock.Text).To(ContainSubstring("full-cycle skill"))
				})
			})
		})

		Context("when content is invalid", func() {
			It("returns error for invalid JSON", func() {
				input := []byte(`{not json}`)
				var content domain.MessageContent
				err := json.Unmarshal(input, &content)

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when content is text", func() {
			It("serializes text content correctly", func() {
				text := "Hello"
				content := domain.MessageContent{Text: &text, IsBlocks: false}
				result, err := json.Marshal(content)

				Expect(err).NotTo(HaveOccurred())
				Expect(string(result)).To(Equal(`"Hello"`))
			})
		})

		Context("when content is blocks", func() {
			It("serializes blocks array correctly", func() {
				content := domain.MessageContent{
					IsBlocks: true,
					Blocks: []domain.ContentBlock{
						&domain.TextContentBlock{Type: domain.ContentBlockTypeText, Text: "Hi"},
					},
				}
				result, err := json.Marshal(content)

				Expect(err).NotTo(HaveOccurred())
				Expect(string(result)).To(ContainSubstring(`"type":"text"`))
				Expect(string(result)).To(ContainSubstring(`"text":"Hi"`))
			})
		})

		Context("when content is uninitialized", func() {
			It("handles zero value gracefully", func() {
				content := domain.MessageContent{}
				result, err := json.Marshal(content)

				Expect(err).NotTo(HaveOccurred())
				// Current behavior returns "", but this test documents it
				Expect(result).NotTo(BeNil())
			})
		})

		Describe("edge cases", func() {
			Context("when IsBlocks is true but Blocks is nil", func() {
				It("returns empty array instead of null", func() {
					content := domain.MessageContent{IsBlocks: true, Blocks: nil}
					result, err := json.Marshal(content)

					Expect(err).NotTo(HaveOccurred())
					Expect(string(result)).To(Equal("[]"))
				})
			})

			Context("when content is completely uninitialized", func() {
				It("returns null instead of empty string", func() {
					content := domain.MessageContent{}
					result, err := json.Marshal(content)

					Expect(err).NotTo(HaveOccurred())
					Expect(string(result)).To(Equal("null"))
				})
			})
		})

		Describe("round-trip serialization", func() {
			It("preserves text content through marshal/unmarshal", func() {
				original := []byte(`"Test message"`)
				var content domain.MessageContent
				Expect(json.Unmarshal(original, &content)).To(Succeed())

				marshaled, err := json.Marshal(content)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(marshaled)).To(Equal(string(original)))
			})

			It("preserves block content through marshal/unmarshal", func() {
				original := []byte(`[{"type":"text","text":"Hello"}]`)
				var content domain.MessageContent
				Expect(json.Unmarshal(original, &content)).To(Succeed())

				marshaled, err := json.Marshal(content)
				Expect(err).NotTo(HaveOccurred())

				var restored domain.MessageContent
				Expect(json.Unmarshal(marshaled, &restored)).To(Succeed())
				Expect(restored.IsBlocks).To(Equal(content.IsBlocks))
				Expect(restored.Blocks).To(HaveLen(len(content.Blocks)))
			})
		})
	})

	Describe("Integration Test with Real JSONL Data", func() {
		var sessionRecords []map[string]any

		BeforeEach(func() {
			// Read test data file
			file, err := os.Open(testDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			sessionRecords = []map[string]any{}
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				var record map[string]any
				err := json.Unmarshal(scanner.Bytes(), &record)
				Expect(err).NotTo(HaveOccurred())
				sessionRecords = append(sessionRecords, record)
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())
		})

		It("should have exactly 8 session records", func() {
			Expect(sessionRecords).To(HaveLen(8))
		})

		Context("Line 1: text block in array", func() {
			It("parses text content block from user message", func() {
				record := sessionRecords[0]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeTrue())
				Expect(content.Blocks).To(HaveLen(1))

				textBlock, ok := content.Blocks[0].(*domain.TextContentBlock)
				Expect(ok).To(BeTrue())
				Expect(textBlock.Type).To(Equal(domain.ContentBlockTypeText))
				Expect(textBlock.Text).To(ContainSubstring("Save all artifacts"))
			})
		})

		Context("Line 2: plain string content", func() {
			It("parses string content from user message", func() {
				record := sessionRecords[1]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeFalse())
				Expect(content.Text).NotTo(BeNil())
				Expect(*content.Text).To(ContainSubstring("Caveat: The messages"))
			})
		})

		Context("Line 3: XML-like string content", func() {
			It("parses XML content as string", func() {
				record := sessionRecords[2]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeFalse())
				Expect(content.Text).NotTo(BeNil())
				Expect(*content.Text).To(ContainSubstring("<command-name>/clear</command-name>"))
			})
		})

		Context("Line 4: HTML tag string content", func() {
			It("parses HTML content as string", func() {
				record := sessionRecords[3]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeFalse())
				Expect(content.Text).NotTo(BeNil())
				Expect(*content.Text).To(ContainSubstring("<local-command-stdout>"))
			})
		})

		Context("Line 5: thinking block in array (CRITICAL)", func() {
			It("parses thinking content block from assistant message", func() {
				record := sessionRecords[4]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeTrue())
				Expect(content.Blocks).To(HaveLen(1))

				thinkingBlock, ok := content.Blocks[0].(*domain.ThinkingContentBlock)
				Expect(ok).To(BeTrue(), "Content block should be ThinkingContentBlock")
				Expect(thinkingBlock.Type).To(Equal(domain.ContentBlockTypeThinking))
				Expect(thinkingBlock.Thinking).To(ContainSubstring("사용자가"))
				Expect(thinkingBlock.Thinking).To(ContainSubstring("Project를 등록할 때"))
				Expect(thinkingBlock.Signature).To(Equal("signature-string"))
			})
		})

		Context("Line 6: assistant text response", func() {
			It("parses assistant text content block explaining action", func() {
				record := sessionRecords[5]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeTrue(), "Assistant message should have content blocks")
				Expect(content.Blocks).To(HaveLen(1), "Should have exactly 1 content block")

				textBlock, ok := content.Blocks[0].(*domain.TextContentBlock)
				Expect(ok).To(BeTrue(), "Content block should be TextContentBlock")
				Expect(textBlock.Type).To(Equal(domain.ContentBlockTypeText))
				Expect(textBlock.Text).To(ContainSubstring("I'll invoke the full-cycle skill"))
				Expect(textBlock.Text).To(ContainSubstring("Project registration"))
				Expect(textBlock.Text).To(ContainSubstring("remote URL"))
			})

			It("preserves complete message metadata", func() {
				record := sessionRecords[5]
				message := record["message"].(map[string]any)

				Expect(message).To(HaveKey("model"))
				Expect(message["model"]).To(Equal("claude-sonnet-4-5-20250929"))
				Expect(message).To(HaveKey("role"))
				Expect(message["role"]).To(Equal("assistant"))
				Expect(message).To(HaveKey("type"))
				Expect(message["type"]).To(Equal("message"))
			})
		})

		Context("Line 7: tool_use block with complex input (CRITICAL)", func() {
			It("parses tool_use block with Korean text in input", func() {
				record := sessionRecords[6]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeTrue())
				Expect(content.Blocks).To(HaveLen(1))

				toolUseBlock, ok := content.Blocks[0].(*domain.ToolUseContentBlock)
				Expect(ok).To(BeTrue(), "Content block should be ToolUseContentBlock")
				Expect(toolUseBlock.Type).To(Equal(domain.ContentBlockTypeToolUse))
				Expect(toolUseBlock.ID).To(Equal("toolu_01MWqXkvw67AkXFwtGfUcr4J"))
				Expect(toolUseBlock.Name).To(Equal("Skill"))
			})

			It("correctly parses nested input object with skill and args", func() {
				record := sessionRecords[6]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				toolUseBlock := content.Blocks[0].(*domain.ToolUseContentBlock)

				Expect(toolUseBlock.Input).To(HaveKey("skill"))
				Expect(toolUseBlock.Input["skill"]).To(Equal("full-cycle"))

				Expect(toolUseBlock.Input).To(HaveKey("args"))
				args, ok := toolUseBlock.Input["args"].(string)
				Expect(ok).To(BeTrue(), "args should be a string")
				Expect(args).To(ContainSubstring("Project를 등록할 때"))
				Expect(args).To(ContainSubstring("Remote URL"))
				Expect(args).To(ContainSubstring("이 문제를 해결해줘"))
			})

			It("handles Unicode (Korean) characters in input fields", func() {
				record := sessionRecords[6]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				toolUseBlock := content.Blocks[0].(*domain.ToolUseContentBlock)
				args := toolUseBlock.Input["args"].(string)

				// Verify Korean text is preserved correctly
				Expect(args).To(ContainSubstring("프로젝트"))
				Expect(args).To(ContainSubstring("저장"))
			})
		})

		Context("Line 8: tool_result block (CRITICAL)", func() {
			It("parses tool_result content block from user message", func() {
				record := sessionRecords[7]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				Expect(content.IsBlocks).To(BeTrue(), "User message with tool_result should have content blocks")
				Expect(content.Blocks).To(HaveLen(1), "Should have exactly 1 tool_result block")

				toolResultBlock, ok := content.Blocks[0].(*domain.ToolResultContentBlock)
				Expect(ok).To(BeTrue(), "Content block should be ToolResultContentBlock")
				Expect(toolResultBlock.Type).To(Equal(domain.ContentBlockTypeToolResult))
			})

			It("correctly extracts tool_use_id and content from tool_result", func() {
				record := sessionRecords[7]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				toolResultBlock := content.Blocks[0].(*domain.ToolResultContentBlock)

				Expect(toolResultBlock.ToolUseID).To(Equal("toolu_01MWqXkvw67AkXFwtGfUcr4J"))
				Expect(toolResultBlock.Content).To(Equal("Launching skill: full-cycle"))
			})

			It("correctly identifies non-error tool results", func() {
				record := sessionRecords[7]
				message := record["message"].(map[string]any)
				contentJSON, err := json.Marshal(message["content"])
				Expect(err).NotTo(HaveOccurred())

				var content domain.MessageContent
				err = json.Unmarshal(contentJSON, &content)
				Expect(err).NotTo(HaveOccurred())

				toolResultBlock := content.Blocks[0].(*domain.ToolResultContentBlock)

				// This tool_result doesn't have is_error field, should default to false
				// Note: The actual JSONL doesn't have is_error, so we need to check the struct default
				Expect(toolResultBlock.IsError).To(BeFalse(), "Missing is_error should default to false")
			})

			It("preserves toolUseResult metadata in record", func() {
				record := sessionRecords[7]

				Expect(record).To(HaveKey("toolUseResult"))
				toolUseResult := record["toolUseResult"].(map[string]any)
				Expect(toolUseResult).To(HaveKeyWithValue("success", true))
				Expect(toolUseResult).To(HaveKeyWithValue("commandName", "full-cycle"))
			})

			It("verifies tool_result links back to correct tool_use", func() {
				// Line 7 has tool_use, Line 8 has tool_result - they should match
				toolUseRecord := sessionRecords[6]
				toolResultRecord := sessionRecords[7]

				// Extract tool_use ID
				toolUseMessage := toolUseRecord["message"].(map[string]any)
				toolUseContentJSON, _ := json.Marshal(toolUseMessage["content"])
				var toolUseContent domain.MessageContent
				json.Unmarshal(toolUseContentJSON, &toolUseContent)
				toolUseBlock := toolUseContent.Blocks[0].(*domain.ToolUseContentBlock)

				// Extract tool_result tool_use_id
				toolResultMessage := toolResultRecord["message"].(map[string]any)
				toolResultContentJSON, _ := json.Marshal(toolResultMessage["content"])
				var toolResultContent domain.MessageContent
				json.Unmarshal(toolResultContentJSON, &toolResultContent)
				toolResultBlock := toolResultContent.Blocks[0].(*domain.ToolResultContentBlock)

				// Verify they match
				Expect(toolResultBlock.ToolUseID).To(Equal(toolUseBlock.ID),
					"tool_result.tool_use_id should match tool_use.id")
			})
		})

		Context("Round-trip serialization", func() {
			It("preserves all content through marshal/unmarshal cycle", func() {
				for i, record := range sessionRecords {
					message := record["message"].(map[string]any)
					contentJSON, err := json.Marshal(message["content"])
					Expect(err).NotTo(HaveOccurred(), "Line %d: failed to marshal original content", i+1)

					var content domain.MessageContent
					err = json.Unmarshal(contentJSON, &content)
					Expect(err).NotTo(HaveOccurred(), "Line %d: failed to unmarshal content", i+1)

					remarshaled, err := json.Marshal(content)
					Expect(err).NotTo(HaveOccurred(), "Line %d: failed to remarshal content", i+1)
					Expect(remarshaled).NotTo(BeEmpty(), "Line %d: remarshaled content should not be empty", i+1)
					Expect(string(remarshaled)).NotTo(Equal(`""`), "Line %d: remarshaled content should not be empty string", i+1)

					// Verify round-trip by unmarshaling again
					var restored domain.MessageContent
					err = json.Unmarshal(remarshaled, &restored)
					Expect(err).NotTo(HaveOccurred(), "Line %d: failed to restore content", i+1)

					if content.IsBlocks {
						Expect(restored.IsBlocks).To(BeTrue(), "Line %d: IsBlocks should be preserved", i+1)
						Expect(restored.Blocks).To(HaveLen(len(content.Blocks)), "Line %d: Block count should match", i+1)
					} else {
						Expect(restored.IsBlocks).To(BeFalse(), "Line %d: IsBlocks should be false", i+1)
						if content.Text != nil {
							Expect(restored.Text).NotTo(BeNil(), "Line %d: Text should not be nil", i+1)
							Expect(*restored.Text).To(Equal(*content.Text), "Line %d: Text content should match", i+1)
						}
					}
				}
			})
		})
	})
})
