# Implementation Plan

## Overview

Fix the empty `messageContent` storage issue in MongoDB by creating comprehensive integration tests with real JSONL data to reproduce and diagnose the issue, then implement the fix and add unit tests for parsing logic.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| --- | --- | --- | --- |
| JSON parsing | bytedance/sonic | `/bytedance/sonic` | Already in use, compatible with encoding/json interfaces |
| BDD testing framework | github.com/onsi/ginkgo/v2 | `/onsi/ginkgo` | Expressive DSL for clear and organized test specifications |
| Matcher library | github.com/onsi/gomega | `/onsi/gomega` | Rich set of assertion matchers, pairs with Ginkgo |

## Architecture Decisions

### Decision 1: Integration Test Strategy

**Choice**: Create integration tests that read actual JSONL files from `~/.claude/projects/` and verify end-to-end parsing works correctly.

**Rationale**: Real data testing will immediately reveal if parsing logic works correctly. The research report claims parsing is working, so integration tests with real data will either confirm this or expose the actual failure point.

### Decision 2: Test Location

**Choice**: Create tests in `shared/domain/` for unit tests and in `daemon/internal/service/logwatcher/` for integration tests.

**Rationale**: Unit tests belong with the code they test. Integration tests should be near the service that performs the actual file parsing.

### Decision 3: Thinking Block Support

**Choice**: Add `ThinkingContentBlock` type since 1,492 occurrences exist in real JSONL files and content is lost when skipped.

**Rationale**: The research shows significant `thinking` block usage. Silently skipping them means data loss. Adding support follows the existing pattern and preserves all message content.

### Decision 4: Debugging Approach

**Choice**: Add structured logging during storage to capture the actual content being serialized, then compare with source JSONL.

**Rationale**: If the research is correct that parsing works, the issue must be in serialization or storage. Logging at adapter.go line 107 will reveal what content is being marshaled.

### Decision 5: Test Framework

**Choice**: Use Ginkgo/Gomega BDD-style testing framework instead of standard Go testing.

**Rationale**: User explicitly requested BDD style with Ginkgo and Gomega. This provides expressive DSL with Describe/Context/It blocks and rich matchers.

## Implementation Steps

### Step 1: Add Unit Tests for MessageContent Parsing

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/shared/domain/message_suite_test.go` (create)
- `/Users/jayce/team-attention/cops/shared/domain/message_test.go` (create)

**Functions**:

```go
// message_suite_test.go - Ginkgo test suite bootstrap
package domain_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestDomain(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Domain Suite")
}
```

```go
// message_test.go - BDD specs for MessageContent
package domain_test

import (
    "encoding/json"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/team-attention/cops/shared/domain"
)

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
})
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| User message string | `"Hello world"` | Text="Hello world", IsBlocks=false | String branch |
| Empty string | `""` | Text="", IsBlocks=false | String empty case |
| Text block | `[{"type":"text","text":"Hi"}]` | Blocks[0].Text="Hi", IsBlocks=true | Text block branch |
| Tool use block | `[{"type":"tool_use","id":"x","name":"Read","input":{"file":"/a"}}]` | ToolUseContentBlock with nested input | Tool use branch |
| Tool result block | `[{"type":"tool_result","tool_use_id":"x","content":"ok","is_error":false}]` | ToolResultContentBlock parsed | Tool result branch |
| Unknown block type | `[{"type":"unknown","data":"x"}]` | Empty Blocks (skipped) | Default case |
| Mixed blocks | `[{"type":"text","text":"a"},{"type":"tool_use",...}]` | Two blocks parsed | Multiple block types |
| Invalid JSON | `{not json}` | Error returned | Error handling |

### Step 2: Add ThinkingContentBlock Type

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/shared/domain/content_block.go` (modify)
- `/Users/jayce/team-attention/cops/shared/domain/message.go` (modify)

**Functions**:

```go
// In content_block.go:
const ContentBlockTypeThinking ContentBlockType = "thinking"

// ThinkingContentBlock represents a thinking content block from extended thinking models.
type ThinkingContentBlock struct {
    Type      ContentBlockType `json:"type"`
    Thinking  string           `json:"thinking"`
    Signature string           `json:"signature,omitempty"`
}

// BlockType implements ContentBlock interface.
func (b *ThinkingContentBlock) BlockType() ContentBlockType { return ContentBlockTypeThinking }
```

```go
// In message.go UnmarshalJSON, add case in switch:
case ContentBlockTypeThinking:
    var tb ThinkingContentBlock
    if err := json.Unmarshal(raw, &tb); err != nil {
        return fmt.Errorf("failed to parse thinking block %d: %w", i, err)
    }
    block = &tb
```

**Additional Tests for message_test.go**:

```go
// Add to message_test.go within the "when content is an array of blocks" Context:
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| Thinking block with signature | `{"type":"thinking","thinking":"...","signature":"..."}` | ThinkingContentBlock parsed | New thinking case |
| Thinking block without signature | `{"type":"thinking","thinking":"..."}` | ThinkingContentBlock, Signature="" | Omitempty field |

### Step 3: Create Integration Test with Real JSONL Data

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/logwatcher_suite_test.go` (create)
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service_test.go` (create)

**Functions**:

```go
// logwatcher_suite_test.go - Ginkgo test suite bootstrap
package logwatcher_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestLogwatcher(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Logwatcher Suite")
}
```

```go
// log_service_test.go - BDD specs for log service integration tests
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| Parse user message | Real JSONL user line | Content.Text populated | String content |
| Parse assistant with tool_use | Real JSONL assistant line | Content.Blocks populated | Block array content |
| Parse assistant with thinking | Real JSONL with thinking | ThinkingContentBlock in Blocks | New thinking type |
| Serialize and restore | Any real SessionRecord | Marshal/Unmarshal roundtrip succeeds | Storage path |
| Empty file | Empty JSONL | No records, no error | Edge case |

### Step 4: Debug Storage Path

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` (modify temporarily for debugging, then revert)

**Functions**:

```go
// In toDocument function, add debug logging:
if msg.Content != nil {
    contentBytes, err := sonic.Marshal(msg.Content)
    if err != nil {
        slog.Warn("failed to serialize message content",
            slog.String("messageId", msg.ID),
            slog.Any("error", err),
        )
    } else {
        // DEBUG: Log what we're actually storing
        slog.Debug("storing message content",
            slog.String("messageId", msg.ID),
            slog.Bool("isBlocks", msg.Content.IsBlocks),
            slog.Int("contentLen", len(contentBytes)),
            slog.String("contentPreview", string(contentBytes[:min(100, len(contentBytes))])),
        )
        doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
    }
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| Content is nil | msg.Content == nil | Field not added | nil check |
| Content is empty text | msg.Content.Text == "" | `""` stored | Empty string |
| Content has blocks | msg.Content.IsBlocks == true | JSON array stored | Block serialization |

### Step 5: Investigate MarshalJSON for Empty String Issue

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/shared/domain/message.go` (investigate and potentially fix)

**Analysis**:

Looking at the current `MarshalJSON` implementation:

```go
func (c MessageContent) MarshalJSON() ([]byte, error) {
    if c.IsBlocks {
        return json.Marshal(c.Blocks)
    }
    if c.Text != nil {
        return json.Marshal(*c.Text)
    }
    return json.Marshal("")  // <-- THIS could be the issue
}
```

**Root Cause Hypothesis**: If `IsBlocks` is `false` and `Text` is `nil`, the function returns `json.Marshal("")` which produces `""`. This could happen if:
1. Parsing fails silently and neither Text nor Blocks is set
2. There's a zero-value `MessageContent` being serialized

**Fix**:

```go
func (c MessageContent) MarshalJSON() ([]byte, error) {
    if c.IsBlocks {
        if c.Blocks == nil {
            return json.Marshal([]ContentBlock{})  // Empty array, not null
        }
        return json.Marshal(c.Blocks)
    }
    if c.Text != nil {
        return json.Marshal(*c.Text)
    }
    // Return null instead of empty string for uninitialized content
    return []byte("null"), nil
}
```

**Additional Tests for MarshalJSON edge cases** (add to message_test.go):

```go
// Add to message_test.go within the "MarshalJSON" Describe:
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
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| IsBlocks=true with blocks | Content with 2 blocks | JSON array with blocks | Normal block case |
| IsBlocks=true with nil blocks | Content{IsBlocks: true, Blocks: nil} | `[]` | Edge case |
| IsBlocks=false with text | Content with Text="hello" | `"hello"` | Normal text case |
| Zero value | Content{} | `null` | Uninitialized content |

### Step 6: Add Debug Script for Data Flow Verification

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/scripts/debug_content_storage.go` (create, delete after fix)

**Functions**:

```go
// main parses a JSONL file and traces content through the data flow.
func main() {
    // 1. Read one line from a JSONL file
    // 2. Unmarshal into SessionRecord
    // 3. Print Content state: IsBlocks, Text pointer, Blocks slice
    // 4. Marshal Content with sonic.Marshal
    // 5. Print marshaled result
    // 6. Unmarshal back into MessageContent
    // 7. Print final state
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| --- | --- | --- | --- |
| Trace user message | User JSONL line | Shows Text populated | String content path |
| Trace assistant message | Assistant JSONL line | Shows Blocks populated | Block content path |
| Trace tool_use specifically | Line with tool_use | Shows ToolUseContentBlock | Tool use parsing |

## Execution Order

1. Step 1: Add unit tests for MessageContent (validates parsing logic in isolation)
2. Step 2: Add ThinkingContentBlock type (addresses missing content type)
3. Step 3: Create integration tests with real JSONL (reproduces issue with real data)
4. Step 5: Investigate and fix MarshalJSON empty string issue (likely root cause fix)
5. Step 4: Add debug logging temporarily (only if Step 3/5 don't reveal issue)
6. Step 6: Create debug script (only if issue still unclear)

## Notes for Execute Agent

- **Install Ginkgo/Gomega**: Run `go get github.com/onsi/ginkgo/v2` and `go get github.com/onsi/gomega` in both `shared/` and `daemon/` modules
- **Run unit tests first**: `ginkgo ./shared/domain/...` after Step 1 (or `go test ./shared/domain/...`)
- **Test with real JSONL**: Integration tests require `~/.claude/projects/` directory with JSONL files
- **Check existing records**: After fix, query MongoDB to verify new records have non-empty `messageContent`
- **Backward compatibility**: The MarshalJSON fix should return `null` instead of `""` for uninitialized content, which is a breaking change for any code expecting empty string. Verify dashboard_repo.go fallback handles this.
- **Debug logging is temporary**: Remove debug logging from adapter.go after issue is resolved
- **Delete debug script**: The debug script in Step 6 should be deleted after investigation is complete
- **Interface verification**: After adding ThinkingContentBlock, ensure `var _ ContentBlock = (*ThinkingContentBlock)(nil)` compile-time check is added
- **Ginkgo CLI**: Optionally install `ginkgo` CLI with `go install github.com/onsi/ginkgo/v2/ginkgo@latest` for enhanced test running features
