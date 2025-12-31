# Requirements

## Request Summary

Implement custom JSON and BSON marshaling for the UserMessage.Content field to support polymorphic content types. The Content field currently uses the `any` type but needs type-safe handling for two distinct forms: simple string messages and arrays of content blocks. Content blocks use UserMessageBlockContent which is a unified struct that handles text, image, and tool_result block types through discriminated union pattern with embedded fields. This implementation will enable proper serialization/deserialization of Claude Code JSONL session logs.

## Acceptance Criteria

- [ ] UserMessage.Content field supports two content types:
  - String (simple text messages like "hello")
  - Array of *UserMessageBlockContent (unified blocks supporting text/image/tool_result via embedded fields)
- [ ] UnmarshalJSON correctly detects content type and unmarshals:
  - String JSON → string value
  - Array JSON → []*UserMessageBlockContent (handles all block types: text, image, tool_result)
- [ ] MarshalJSON preserves original content type:
  - string → JSON string
  - []*UserMessageBlockContent → JSON array with blocks (all three types: text/image/tool_result)
- [ ] UnmarshalBSON and MarshalBSON implement same logic as JSON methods for BSON format
- [ ] Returns descriptive error when content is neither string nor valid array type
- [ ] Test case at line 235-267 in record_test.go passes (unmarshaling array content from JSONL line 9)
- [ ] All existing tests continue to pass
- [ ] Round-trip serialization preserves data integrity (unmarshal → marshal → unmarshal produces identical result)

## Scope

### In Scope
- Implement UnmarshalJSON for UserMessage.Content field
- Implement MarshalJSON for UserMessage.Content field
- Implement UnmarshalBSON for UserMessage.Content field
- Implement MarshalBSON for UserMessage.Content field
- Add type detection logic to distinguish between string and block array
- Update existing test at lines 235-267 to verify successful unmarshaling (remove error expectation)
- Add new test cases for:
  - Unmarshaling string content
  - Unmarshaling array content with text blocks (text field)
  - Unmarshaling array content with image blocks (source field)
  - Unmarshaling array content with tool_result blocks (embedded ToolResult fields)
  - Round-trip serialization for all content types

### Out of Scope
- Modifying UserMessageBlockContent struct (already updated at lines 42-53 with unified structure):
  - Type, Text, Source fields for text/image blocks
  - Embedded *UserMessageBlockContentToolResult for tool_result blocks
- Modifying UserRecordToolUseResult struct (handles separate metadata, not Content field)
- Changing the `any` type of Content field (polymorphic nature requires `any` with custom marshaling)
- Adding validation for block content structure (assume JSONL data is valid)
- Supporting other content types beyond string and UserMessageBlockContent arrays

## Constraints

- Must follow Go struct definition rules from .agent/rules/go/go-struct.md:
  - Use pointer types for discriminated union members in block structs
  - Use value types for required fields
- Must maintain compatibility with existing JSONL test data files:
  - record_user.jsonl (10 lines)
  - All existing records must continue to parse correctly
- Custom marshaling must follow the same architectural pattern used elsewhere in the codebase
- Error messages must be descriptive enough to debug content type mismatches
- Implementation must work with both encoding/json and MongoDB BSON encoding

## Additional Context

### Example Content Types from JSONL

1. String content (line 4):
```json
"content":"hellop"
```

2. Array content with text and image blocks (line 9):
```json
"content":[
  {"type":"text","text":"이렇게 보내면 어떻게 되는거지"},
  {"type":"image","source":{"type":"base64","media_type":"image/png","data":"data-string"}}
]
```

3. Array content with tool_result blocks (line 10):
```json
"content":[
  {"tool_use_id":"toolu_01K1DDVNnv3oFCVTTQkGVtJm","type":"tool_result","content":"tool-content"}
]
```

**All block types use the unified UserMessageBlockContent struct:**
- Text blocks: `type="text"` with `text` field populated
- Image blocks: `type="image"` with `source` field populated (UserMessageBlockContentSource)
- Tool result blocks: `type="tool_result"` with embedded `*UserMessageBlockContentToolResult` fields (toolUseId, content)

### Related Files
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go` - Target file for implementation
- `/Users/jayce/team-attention/cops/shared/domain/record_test.go` - Test file to update
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl` - Test data file

### Dependencies
- `encoding/json` - For JSON marshaling/unmarshaling
- `go.mongodb.org/mongo-driver/bson` - For BSON marshaling/unmarshaling (if used elsewhere in codebase)

## Questions Resolved

| Question                                               | Answer                                                                                                                                                                  |
| ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| What should the Content field type be?                 | Discriminated union supporting: (1) string, (2) []UserMessageBlockContent (unified struct handling text/image/tool_result via embedded fields)                          |
| What is the expected behavior for tool_result content? | Tool results appear in Content as []UserMessageBlockContent with type="tool_result", using embedded *UserMessageBlockContentToolResult for toolUseId and content fields |
| Should implementation follow existing patterns?        | Yes, follow same pattern as other custom marshaling in the codebase - detect type in UnmarshalJSON/BSON, preserve type in MarshalJSON/BSON                              |
| What should happen on type mismatch?                   | Return descriptive error if content is neither string nor array                                                                                                         |
| Do we need a new struct for tool results?              | No, UserMessageBlockContent handles all block types via embedded *UserMessageBlockContentToolResult (already implemented at lines 42-53)                                |
| Should the test at line 235-267 be updated?            | Yes, update to expect successful unmarshaling instead of error, since we're now supporting array content                                                                |
