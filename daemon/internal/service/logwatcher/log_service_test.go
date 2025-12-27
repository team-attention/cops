package logwatcher_test

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("LogService Integration", func() {
	var (
		claudeProjectsDir string
		jsonlFiles        []string
	)

	BeforeEach(func() {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		claudeProjectsDir = filepath.Join(homeDir, ".claude", "projects")

		// Find JSONL files
		matches, err := filepath.Glob(filepath.Join(claudeProjectsDir, "*", "*.jsonl"))
		if err != nil || len(matches) == 0 {
			Skip("No JSONL files found in ~/.claude/projects/")
		}
		jsonlFiles = matches
	})

	Describe("parsing real JSONL files", func() {
		Context("when processing user messages", func() {
			It("parses string content correctly", func() {
				for _, jsonlFile := range jsonlFiles {
					file, err := os.Open(jsonlFile)
					if err != nil {
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							continue
						}

						var record shareddomain.SessionRecord
						if err := sonic.Unmarshal([]byte(line), &record); err != nil {
							continue
						}

						if record.Type == shareddomain.SessionTypeUser && record.Message != nil {
							// User messages should have Text content
							if record.Message.Content != nil && !record.Message.Content.IsBlocks {
								Expect(record.Message.Content.Text).NotTo(BeNil(),
									"User message should have Text field populated")
							}
							return // Found and validated at least one
						}
					}
				}
			})
		})

		Context("when processing assistant messages with tool_use", func() {
			It("parses block content with tool_use correctly", func() {
				foundToolUse := false

				for _, jsonlFile := range jsonlFiles {
					file, err := os.Open(jsonlFile)
					if err != nil {
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							continue
						}

						var record shareddomain.SessionRecord
						if err := sonic.Unmarshal([]byte(line), &record); err != nil {
							continue
						}

						if record.Type == shareddomain.SessionTypeAssistant && record.Message != nil {
							content := record.Message.Content
							if content != nil && content.IsBlocks {
								for _, block := range content.Blocks {
									if block.BlockType() == shareddomain.ContentBlockTypeToolUse {
										toolUse, ok := block.(*shareddomain.ToolUseContentBlock)
										Expect(ok).To(BeTrue())
										Expect(toolUse.ID).NotTo(BeEmpty())
										Expect(toolUse.Name).NotTo(BeEmpty())
										foundToolUse = true
										break
									}
								}
							}
						}
						if foundToolUse {
							break
						}
					}
					if foundToolUse {
						break
					}
				}

				if !foundToolUse {
					Skip("No tool_use blocks found in JSONL files")
				}
			})
		})
	})

	Describe("serialization round-trip", func() {
		It("preserves content through sonic.Marshal/Unmarshal", func() {
			for _, jsonlFile := range jsonlFiles {
				file, err := os.Open(jsonlFile)
				if err != nil {
					continue
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 1024*1024)

				testedCount := 0
				for scanner.Scan() && testedCount < 10 {
					line := scanner.Text()
					if line == "" {
						continue
					}

					var record shareddomain.SessionRecord
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						continue
					}

					if record.Message != nil && record.Message.Content != nil {
						// Marshal the content
						contentBytes, err := sonic.Marshal(record.Message.Content)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(contentBytes)).NotTo(Equal(`""`),
							"Content should not serialize to empty string")

						// Unmarshal back
						var restored shareddomain.MessageContent
						err = sonic.Unmarshal(contentBytes, &restored)
						Expect(err).NotTo(HaveOccurred())

						// Verify structure preserved
						Expect(restored.IsBlocks).To(Equal(record.Message.Content.IsBlocks))
						if restored.IsBlocks {
							Expect(restored.Blocks).To(HaveLen(len(record.Message.Content.Blocks)))
						}

						testedCount++
					}
				}

				if testedCount > 0 {
					return // Successfully tested
				}
			}
		})
	})
})
