# Implementation Plan: Custom JSON/BSON Marshaling for UserMessage.Content

## Overview

This implementation adds custom JSON and BSON marshaling to the `UserMessage` struct to support polymorphic content types. The `Content` field can be either:
1. A simple string (e.g., `"hello"`)
2. An array of `*UserMessageBlockContent` (containing text, image, or tool_result blocks)

The implementation follows the discriminated union pattern already established in the codebase (see `AssistantMessageContent` in `record_assistant.go` and `Record` in `record.go`). Custom marshaling methods will detect the content type during unmarshaling and preserve the original type during marshaling.

## Package Changes

None required. All necessary packages are already available:
- `encoding/json` - Standard library for JSON marshaling
- `go.mongodb.org/mongo-driver/v2/bson` - Already in go.mod for BSON marshaling

## Step 1: Add Import Statements

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Check existing imports

### `/Users/jayce/team-attention/cops/shared/domain/record_user.go`

**Description**:
Add necessary imports for JSON and BSON marshaling.

```go
package domain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)
```

## Step 2: Fix UserMessageBlockContentToolResult JSON Tag

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Check current JSON tag for tool_use_id field
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl`: Verify actual JSON field name in test data (line 10)

### `/Users/jayce/team-attention/cops/shared/domain/record_user.go`

**Description**:
The JSONL test data (line 10) shows `tool_use_id` as the JSON field name (with underscore), but the current struct has `json:"toolUseId"` (camelCase). The JSON tag must match the actual data format.

**Current code (line 50-53)**:
```go
type UserMessageBlockContentToolResult struct {
	ToolUseID string `json:"toolUseId" bson:"toolUseId"`
	Content   any    `json:"content" bson:"content"`
}
```

**Updated code**:
```go
type UserMessageBlockContentToolResult struct {
	ToolUseID string `json:"tool_use_id" bson:"toolUseId"`
	Content   any    `json:"content" bson:"content"`
}
```

**Note**: The JSON tag uses snake_case (`tool_use_id`) to match the Claude Code JSONL format, while BSON tag uses camelCase (`toolUseId`) for MongoDB storage consistency.

## Step 3: Implement Custom JSON Marshaling for UserMessage

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`: Understand pointer vs value type rules for discriminated unions
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Reference existing custom marshaling pattern in `Record.UnmarshalJSON` and `Record.MarshalJSON`
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Target file containing `UserMessage` struct

### `/Users/jayce/team-attention/cops/shared/domain/record_user.go`

**Description**:
Add `UnmarshalJSON` and `MarshalJSON` methods to `UserMessage` struct to handle polymorphic Content field. Add these after the `UserMessage` struct definition (after line 64).

```go
// userMessageAlias is used to prevent infinite recursion during JSON marshaling.
// It has the same fields as UserMessage but without custom marshal methods.
type userMessageAlias struct {
	Role    UserMessageRole `json:"role"`
	Content any             `json:"content"`
}

// userMessageRaw is used for initial JSON unmarshaling to capture Content as raw bytes.
type userMessageRaw struct {
	Role    UserMessageRole `json:"role"`
	Content json.RawMessage `json:"content"`
}

// UnmarshalJSON deserializes JSON into UserMessage with polymorphic Content handling.
// It detects whether Content is a string or array and unmarshals accordingly.
func (m *UserMessage) UnmarshalJSON(data []byte) error {
	// Implementation outline:
	// 1. Unmarshal data into userMessageRaw to capture Role and raw Content bytes.
	var raw userMessageRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal UserMessage: %w", err)
	}

	// 2. Set m.Role from the raw struct.
	m.Role = raw.Role

	// 3. If raw.Content is empty or null, set m.Content to nil and return.
	if len(raw.Content) == 0 || bytes.Equal(raw.Content, []byte("null")) {
		m.Content = nil
		return nil
	}

	// 4. Trim whitespace from raw.Content to detect first character.
	trimmed := bytes.TrimSpace(raw.Content)
	if len(trimmed) == 0 {
		m.Content = nil
		return nil
	}

	// 5. If first character is '"' (string):
	if trimmed[0] == '"' {
		//    a. Unmarshal raw.Content into a string variable.
		var str string
		if err := json.Unmarshal(raw.Content, &str); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as string: %w", err)
		}
		//    b. Set m.Content to the string value.
		m.Content = str
		return nil
	}

	// 6. Else if first character is '[' (array):
	if trimmed[0] == '[' {
		//    a. Unmarshal raw.Content into []*UserMessageBlockContent.
		var blocks []*UserMessageBlockContent
		if err := json.Unmarshal(raw.Content, &blocks); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as array: %w", err)
		}
		//    b. Set m.Content to the slice value.
		m.Content = blocks
		return nil
	}

	// 7. Else: Return error for unexpected type.
	return fmt.Errorf("UserMessage.Content must be string or array, got unexpected JSON type starting with '%c'", trimmed[0])
}

// MarshalJSON serializes UserMessage to JSON, preserving the Content type.
// String content is marshaled as JSON string, array content as JSON array.
func (m UserMessage) MarshalJSON() ([]byte, error) {
	// Implementation outline:
	// 1. Create userMessageAlias from m to use default struct marshaling.
	alias := userMessageAlias{
		Role:    m.Role,
		Content: m.Content,
	}
	// 2. Marshal the alias to JSON bytes.
	// 3. Return the marshaled bytes.
	// Note: Since Content is `any` type, json.Marshal will correctly serialize
	//       string as JSON string and []*UserMessageBlockContent as JSON array.
	return json.Marshal(alias)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unmarshal string content | `{"role":"user","content":"hello"}` | Content = "hello" (string) | String detection branch |
| Unmarshal array with text block | `{"role":"user","content":[{"type":"text","text":"msg"}]}` | Content = []*UserMessageBlockContent with text | Array detection branch |
| Unmarshal array with image block | `{"role":"user","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"..."}}]}` | Content = []*UserMessageBlockContent with image | Array detection branch |
| Unmarshal array with tool_result block | `{"role":"user","content":[{"type":"tool_result","tool_use_id":"id","content":"result"}]}` | Content = []*UserMessageBlockContent with tool_result | Array detection branch |
| Unmarshal mixed array | `{"role":"user","content":[{"type":"text","text":"msg"},{"type":"image","source":{...}}]}` | Content = []*UserMessageBlockContent with both | Array with multiple elements |
| Unmarshal null content | `{"role":"user","content":null}` | Content = nil | Null content handling |
| Unmarshal missing content | `{"role":"user"}` | Content = nil | Empty content handling |
| Unmarshal invalid content type (object) | `{"role":"user","content":{"invalid":"object"}}` | Error: "must be string or array" | Error branch |
| Unmarshal invalid content type (number) | `{"role":"user","content":123}` | Error: "must be string or array" | Error branch |
| Marshal string content | UserMessage{Content: "hello"} | `{"role":"user","content":"hello"}` | String marshal |
| Marshal array content | UserMessage{Content: []*UserMessageBlockContent{...}} | `{"role":"user","content":[...]}` | Array marshal |
| Marshal nil content | UserMessage{Content: nil} | `{"role":"user","content":null}` | Nil marshal |
| Round-trip string | Unmarshal -> Marshal -> Unmarshal | Identical result | Data integrity |
| Round-trip array | Unmarshal -> Marshal -> Unmarshal | Identical result | Data integrity |

## Step 4: Implement Custom BSON Marshaling for UserMessage

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Target file (same as Step 3)
- Context7 documentation for mongo-driver v2 BSON interfaces (already researched)

### `/Users/jayce/team-attention/cops/shared/domain/record_user.go`

**Description**:
Add `UnmarshalBSON` and `MarshalBSON` methods to `UserMessage` struct. These follow the same logic as JSON methods but use BSON encoding. Add these after the JSON marshaling methods.

```go
// userMessageBSONRaw is used for initial BSON unmarshaling to capture Content as raw value.
type userMessageBSONRaw struct {
	Role    UserMessageRole `bson:"role"`
	Content bson.RawValue   `bson:"content"`
}

// UnmarshalBSON deserializes BSON into UserMessage with polymorphic Content handling.
// It detects whether Content is a string or array and unmarshals accordingly.
func (m *UserMessage) UnmarshalBSON(data []byte) error {
	// Implementation outline:
	// 1. Unmarshal data into userMessageBSONRaw to capture Role and raw Content.
	var raw userMessageBSONRaw
	if err := bson.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal UserMessage BSON: %w", err)
	}

	// 2. Set m.Role from the raw struct.
	m.Role = raw.Role

	// 3. If raw.Content is empty (Type == 0) or null, set m.Content to nil and return.
	if raw.Content.Type == 0 || raw.Content.Type == bson.TypeNull {
		m.Content = nil
		return nil
	}

	// 4. Check raw.Content.Type:
	switch raw.Content.Type {
	case bson.TypeString:
		//    a. If bson.TypeString: Unmarshal raw.Content into a string variable.
		var str string
		if err := raw.Content.Unmarshal(&str); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as string from BSON: %w", err)
		}
		//       Set m.Content to the string value.
		m.Content = str

	case bson.TypeArray:
		//    b. If bson.TypeArray: Unmarshal raw.Content into []*UserMessageBlockContent.
		var blocks []*UserMessageBlockContent
		if err := raw.Content.Unmarshal(&blocks); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as array from BSON: %w", err)
		}
		//       Set m.Content to the slice value.
		m.Content = blocks

	default:
		//    c. Else: Return error for unexpected type.
		return fmt.Errorf("UserMessage.Content must be string or array in BSON, got type %v", raw.Content.Type)
	}

	// 5. Return nil on success.
	return nil
}

// MarshalBSON serializes UserMessage to BSON, preserving the Content type.
// String content is marshaled as BSON string, array content as BSON array.
func (m UserMessage) MarshalBSON() ([]byte, error) {
	// Implementation outline:
	// 1. Create a bson.D document with Role and Content fields.
	doc := bson.D{
		{Key: "role", Value: m.Role},
		{Key: "content", Value: m.Content},
	}
	// 2. Marshal the document to BSON bytes using bson.Marshal.
	// 3. Return the marshaled bytes.
	// Note: bson.Marshal will correctly serialize string as BSON string
	//       and []*UserMessageBlockContent as BSON array.
	return bson.Marshal(doc)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unmarshal BSON string content | BSON doc with string content | Content = string | String type branch |
| Unmarshal BSON array content | BSON doc with array content | Content = []*UserMessageBlockContent | Array type branch |
| Unmarshal BSON null content | BSON doc with null content | Content = nil | Null type branch |
| Unmarshal BSON empty content | BSON doc with Type=0 | Content = nil | Empty type branch |
| Marshal BSON string content | UserMessage{Content: "hello"} | Valid BSON with string | String marshal |
| Marshal BSON array content | UserMessage{Content: []*...} | Valid BSON with array | Array marshal |
| Round-trip BSON string | Unmarshal -> Marshal -> Unmarshal | Identical result | Data integrity |
| Round-trip BSON array | Unmarshal -> Marshal -> Unmarshal | Identical result | Data integrity |

## Step 5: Update Existing Test Case

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record_test.go`: Test file containing the test at lines 235-267

### `/Users/jayce/team-attention/cops/shared/domain/record_test.go`

**Description**:
Update the test case "parses user record with array content from JSONL file" (lines 235-267) to expect successful unmarshaling instead of error handling.

**Current code (lines 235-267)**:
```go
It("parses user record with array content from JSONL file", func() {
	// Read line 9 from record_user.jsonl (array content with image)
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 9 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())
	Expect(line).NotTo(BeEmpty())

	// Unmarshal into domain.Record
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)

	// Assert unmarshal result (may fail due to schema mismatch)
	// UserMessage.Content is string but JSON has array
	if err != nil {
		// Expected failure: content type mismatch
		Expect(err).To(HaveOccurred())
	} else {
		// If unmarshal succeeds, verify the record
		Expect(record.Type).To(Equal(domain.RecordTypeUser))
	}
})
```

**Updated code**:
```go
It("parses user record with array content from JSONL file", func() {
	// Read line 9 from record_user.jsonl (array content with image)
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 9 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())
	Expect(line).NotTo(BeEmpty())

	// Unmarshal into domain.Record
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)

	// Assert successful unmarshaling
	Expect(err).NotTo(HaveOccurred())
	Expect(record.Type).To(Equal(domain.RecordTypeUser))

	// Assert Data is *UserRecord
	userRecord, ok := record.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())

	// Assert Content is []*UserMessageBlockContent
	blocks, ok := userRecord.Message.Content.([]*domain.UserMessageBlockContent)
	Expect(ok).To(BeTrue())
	Expect(blocks).To(HaveLen(2))

	// Assert first block is text type
	Expect(blocks[0].Type).To(Equal("text"))
	Expect(blocks[0].Text).NotTo(BeNil())
	Expect(*blocks[0].Text).To(Equal("이렇게 보내면 어떻게 되는거지"))

	// Assert second block is image type
	Expect(blocks[1].Type).To(Equal("image"))
	Expect(blocks[1].Source).NotTo(BeNil())
	Expect(blocks[1].Source.Type).To(Equal("base64"))
	Expect(blocks[1].Source.Media_type).To(Equal("image/png"))
})
```

## Step 6: Add New Test Cases for Content Types

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record_test.go`: Test file to add new tests
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl`: Test data file

### `/Users/jayce/team-attention/cops/shared/domain/record_test.go`

**Description**:
Add comprehensive test cases for all content type scenarios. Add these tests inside the `Context("when unmarshaling user records", ...)` block, after the existing tests.

```go
It("parses user record with string content from JSONL file", func() {
	// Implementation outline:
	// 1. Read line 4 from record_user.jsonl (string content "hellop").
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 4 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())
	Expect(line).NotTo(BeEmpty())

	// 2. Unmarshal into domain.Record.
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)

	// 3. Assert no error occurred.
	Expect(err).NotTo(HaveOccurred())

	// 4. Assert record.Data is *domain.UserRecord.
	userRecord, ok := record.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())

	// 5. Assert userRecord.Message.Content is a string.
	content, ok := userRecord.Message.Content.(string)
	Expect(ok).To(BeTrue())

	// 6. Assert the string value equals "hellop".
	Expect(content).To(Equal("hellop"))
})

It("parses user record with tool_result content from JSONL file", func() {
	// Implementation outline:
	// 1. Read line 10 from record_user.jsonl (tool_result content).
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 10 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())
	Expect(line).NotTo(BeEmpty())

	// 2. Unmarshal into domain.Record.
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)

	// 3. Assert no error occurred.
	Expect(err).NotTo(HaveOccurred())

	// 4. Assert record.Data is *domain.UserRecord.
	userRecord, ok := record.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())

	// 5. Assert userRecord.Message.Content is []*domain.UserMessageBlockContent.
	blocks, ok := userRecord.Message.Content.([]*domain.UserMessageBlockContent)
	Expect(ok).To(BeTrue())

	// 6. Assert the slice has 1 element.
	Expect(blocks).To(HaveLen(1))

	// 7. Assert element Type is "tool_result".
	Expect(blocks[0].Type).To(Equal("tool_result"))

	// 8. Assert element ToolUseID equals "toolu_01K1DDVNnv3oFCVTTQkGVtJm".
	Expect(blocks[0].UserMessageBlockContentToolResult).NotTo(BeNil())
	Expect(blocks[0].ToolUseID).To(Equal("toolu_01K1DDVNnv3oFCVTTQkGVtJm"))

	// 9. Assert element Content equals "tool-content".
	Expect(blocks[0].UserMessageBlockContentToolResult.Content).To(Equal("tool-content"))
})

It("round-trips user record with array content", func() {
	// Implementation outline:
	// 1. Read line 9 from record_user.jsonl.
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 9 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())

	// 2. Unmarshal into domain.Record.
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)
	Expect(err).NotTo(HaveOccurred())

	// 3. Marshal back to JSON.
	jsonData, err := json.Marshal(record)
	Expect(err).NotTo(HaveOccurred())

	// 4. Unmarshal again into new domain.Record.
	var record2 domain.Record
	err = json.Unmarshal(jsonData, &record2)
	Expect(err).NotTo(HaveOccurred())

	// 5. Assert both records have equal Type.
	Expect(record2.Type).To(Equal(record.Type))

	// 6. Assert both UserRecord.Message.Content are []*UserMessageBlockContent.
	userRecord1, ok := record.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())
	userRecord2, ok := record2.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())

	blocks1, ok := userRecord1.Message.Content.([]*domain.UserMessageBlockContent)
	Expect(ok).To(BeTrue())
	blocks2, ok := userRecord2.Message.Content.([]*domain.UserMessageBlockContent)
	Expect(ok).To(BeTrue())

	// 7. Assert both slices have same length.
	Expect(blocks2).To(HaveLen(len(blocks1)))

	// 8. Assert corresponding block types match.
	for i := range blocks1 {
		Expect(blocks2[i].Type).To(Equal(blocks1[i].Type))
	}
})

It("round-trips user record with string content", func() {
	// Implementation outline:
	// 1. Read line 4 from record_user.jsonl.
	file, err := os.Open(userTestDataFile)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var line string
	for scanner.Scan() {
		lineNum++
		if lineNum == 4 {
			line = scanner.Text()
			break
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())

	// 2. Unmarshal into domain.Record.
	var record domain.Record
	err = json.Unmarshal([]byte(line), &record)
	Expect(err).NotTo(HaveOccurred())

	// 3. Marshal back to JSON.
	jsonData, err := json.Marshal(record)
	Expect(err).NotTo(HaveOccurred())

	// 4. Unmarshal again into new domain.Record.
	var record2 domain.Record
	err = json.Unmarshal(jsonData, &record2)
	Expect(err).NotTo(HaveOccurred())

	// 5. Assert both records have equal Type.
	Expect(record2.Type).To(Equal(record.Type))

	// 6. Assert both UserRecord.Message.Content are strings.
	userRecord1, ok := record.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())
	userRecord2, ok := record2.Data.(*domain.UserRecord)
	Expect(ok).To(BeTrue())

	content1, ok := userRecord1.Message.Content.(string)
	Expect(ok).To(BeTrue())
	content2, ok := userRecord2.Message.Content.(string)
	Expect(ok).To(BeTrue())

	// 7. Assert both string values are equal.
	Expect(content2).To(Equal(content1))
})
```

**Test Scenarios Summary**:

| Test Case | Line | Content Type | Blocks | Validation |
| :-------- | :--- | :----------- | :----- | :--------- |
| String content | 4 | string | N/A | Content equals "hellop" |
| Array with text+image | 9 | array | text, image | 2 blocks, types match |
| Array with tool_result | 10 | array | tool_result | ToolUseID and Content match |
| Round-trip string | 4 | string | N/A | Values preserved |
| Round-trip array | 9 | array | text, image | All blocks preserved |

## Implementation Order

The implementation must follow this order due to dependencies:

1. **Step 1**: Add imports to `record_user.go`
2. **Step 2**: Fix `UserMessageBlockContentToolResult` JSON tag (required for correct unmarshaling)
3. **Step 3**: Implement JSON marshaling methods for `UserMessage`
4. **Step 4**: Implement BSON marshaling methods for `UserMessage`
5. **Step 5**: Update existing test case
6. **Step 6**: Add new test cases

## Error Messages

All error messages must be descriptive for debugging:

| Error Condition | Error Message |
| :-------------- | :------------ |
| JSON unmarshal failure | `"failed to unmarshal UserMessage: %w"` |
| Invalid JSON content type | `"UserMessage.Content must be string or array, got unexpected JSON type starting with '%c'"` |
| JSON unmarshal failure (string) | `"failed to unmarshal UserMessage.Content as string: %w"` |
| JSON unmarshal failure (array) | `"failed to unmarshal UserMessage.Content as array: %w"` |
| BSON unmarshal failure | `"failed to unmarshal UserMessage BSON: %w"` |
| Invalid BSON content type | `"UserMessage.Content must be string or array in BSON, got type %v"` |
| BSON unmarshal failure (string) | `"failed to unmarshal UserMessage.Content as string from BSON: %w"` |
| BSON unmarshal failure (array) | `"failed to unmarshal UserMessage.Content as array from BSON: %w"` |

## Quality Checklist

- [x] Every function has a concrete signature
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (no candidates)
- [x] Execution order is clear and dependencies are explicit
