# Research Report

## Mode
General Research

## Request Summary
Investigate the JSONL message content parsing issue where certain MessageContent types (specifically `tool_use` and potentially others) may not be parsed correctly. The investigation covers the current UnmarshalJSON implementation, bytedance/sonic compatibility, and identifies any missing content block types.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/shared/domain/message.go` | Contains custom `UnmarshalJSON` implementation for `MessageContent` (lines 16-73) |
| `/Users/jayce/team-attention/cops/shared/domain/content_block.go` | Defines `ContentBlock` interface and concrete types (`text`, `tool_use`, `tool_result`) |
| `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go:134` | Shows how `sonic.Unmarshal` is used to parse JSONL records |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:106-117` | Shows serialization using `sonic.Marshal` |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:466-478` | Shows deserialization using `sonic.Unmarshal` with fallback |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md` | Rules for struct field types |

## Analysis Summary

### What's Working

1. **Basic JSONL Parsing**: The daemon successfully parses JSONL files using `sonic.Unmarshal` (line 134 in `log_service.go`).

2. **Sonic/encoding/json Compatibility**: According to Sonic documentation, `sonic.Unmarshal` correctly calls custom `json.Unmarshaler` interfaces. The `MessageContent.UnmarshalJSON` method uses `encoding/json` internally, which is compatible with sonic.

3. **Text Content Blocks**: `TextContentBlock` parsing works correctly - the `type`, `text` fields are properly mapped.

4. **Tool Use Content Blocks**: `ToolUseContentBlock` parsing works correctly - verified by examining actual JSONL files where `tool_use` blocks with `id`, `name`, and `input` (map[string]any) are present.

5. **Tool Result Content Blocks**: `ToolResultContentBlock` parsing works correctly - `tool_use_id`, `type`, `content`, `is_error` fields are handled.

6. **API Write Side (Fixed)**: The `adapter.go` now uses `sonic.Marshal(msg.Content)` (line 107), which correctly serializes both text and block content.

7. **API Read Side (Fixed)**: The `dashboard_repo.go` (lines 466-478) now uses `sonic.Unmarshal` with fallback for backward compatibility with legacy plain-text content.

### What's NOT Working / Missing

1. **`thinking` Content Block Type NOT Handled**:
   - Observed in JSONL logs: 1,492 occurrences of `"type":"thinking"` blocks
   - NOT defined in `content_block.go`
   - Currently silently skipped via `default` case (line 65-67 in `message.go`)
   - Structure observed:
   ```json
   {
     "type": "thinking",
     "thinking": "...",
     "signature": "..."
   }
   ```

2. **No Unit Tests**: There are no unit tests for `MessageContent.UnmarshalJSON` or `ContentBlock` types. This makes it harder to verify parsing correctness and catch regressions.

### Content Block Types in JSONL Files

Analysis of JSONL files in `/Users/jayce/.claude/projects/-Users-jayce-team-attention-cops/`:

| Type | Count | Status |
|------|-------|--------|
| `text` | 3,803 | Handled |
| `tool_use` | 4,035 | Handled |
| `tool_result` | 4,032 | Handled |
| `thinking` | 1,492 | **NOT Handled** (skipped) |

## Technical Investigation

### UnmarshalJSON Implementation Analysis

The current implementation in `message.go:16-73`:

```go
func (c *MessageContent) UnmarshalJSON(data []byte) error {
    // Try string first
    var text string
    if err := json.Unmarshal(data, &text); err == nil {
        c.Text = &text
        c.IsBlocks = false
        return nil
    }

    // Otherwise, try to unmarshal as an array of content blocks
    var rawBlocks []json.RawMessage
    if err := json.Unmarshal(data, &rawBlocks); err != nil {
        return fmt.Errorf("content must be string or array: %w", err)
    }

    // ... type discrimination and parsing
}
```

**Analysis**:
- Correctly handles polymorphic content (string vs array)
- Uses `json.RawMessage` for type discrimination before parsing
- Unknown block types are silently skipped (forward compatibility)
- Uses `encoding/json` internally (compatible with sonic)

### Sonic Compatibility Verification

From Sonic documentation and testing:
- Sonic supports `json.Marshaler` and `json.Unmarshaler` interfaces
- When encountering a type with custom `UnmarshalJSON`, sonic delegates to it
- The `MessageContent.UnmarshalJSON` uses `encoding/json` internally, which is safe

### ToolUseContentBlock.Input Field

```go
type ToolUseContentBlock struct {
    Type  ContentBlockType `json:"type"`
    ID    string           `json:"id"`
    Name  string           `json:"name"`
    Input map[string]any   `json:"input"`  // Generic map for arbitrary input
}
```

**Verified Working**: The `map[string]any` type correctly captures nested input objects. Example from JSONL:
```json
{
  "type": "tool_use",
  "id": "toolu_018iucND3FFrSrfuvehv93bf",
  "name": "Read",
  "input": {
    "file_path": "/Users/jayce/team-attention/cops/.agent/artifacts/..."
  }
}
```

## Package Candidates

### Problem 1: Missing `thinking` Block Type

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| (no external package needed) | N/A | Simple struct addition following existing pattern |

### Problem 2: JSON Parsing (Current Implementation)

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| bytedance/sonic | `/bytedance/sonic` | Already in use, high performance, compatible with encoding/json interfaces |
| encoding/json | (stdlib) | Already used in UnmarshalJSON, no dependency needed for custom logic |

## Technical Constraints

1. **Backward Compatibility**: New content block types should not break parsing of existing data
2. **Forward Compatibility**: Unknown block types should be silently skipped (current behavior)
3. **Sonic Compatibility**: Custom unmarshaler uses `encoding/json` internally, which is compatible with sonic
4. **No Tests**: Must add unit tests when making changes

## Similar Implementations Found

### Example 1: Existing ContentBlock implementations
- **File**: `/Users/jayce/team-attention/cops/shared/domain/content_block.go:17-46`
- **Relevance**: Shows exact pattern to follow for adding `ThinkingContentBlock`

```go
// TextContentBlock represents a text content block.
type TextContentBlock struct {
    Type ContentBlockType `json:"type"`
    Text string           `json:"text"`
}

// BlockType implements ContentBlock interface.
func (b *TextContentBlock) BlockType() ContentBlockType { return ContentBlockTypeText }
```

### Example 2: Type discrimination in UnmarshalJSON
- **File**: `/Users/jayce/team-attention/cops/shared/domain/message.go:44-68`
- **Relevance**: Shows pattern for adding new case in switch statement

## Root Cause Analysis

### Is there a parsing failure?

**NO** - The core parsing logic is working correctly. The investigation found:

1. **Sonic + custom UnmarshalJSON works**: sonic.Unmarshal correctly delegates to MessageContent.UnmarshalJSON
2. **All defined types parse correctly**: `text`, `tool_use`, `tool_result` blocks are parsed
3. **Previous issue was fixed**: The storage issue in `adapter.go` (only storing text content) has been fixed to use `sonic.Marshal(msg.Content)`

### What appears to be "not working"

The `thinking` content block type is present in JSONL files (1,492 occurrences) but is not parsed - it's silently skipped. This is by design (forward compatibility), but may cause data loss if thinking blocks are needed.

## Proposed Investigation/Fix

### If `thinking` blocks are needed:

1. Add `ContentBlockTypeThinking` constant to `content_block.go`
2. Add `ThinkingContentBlock` struct:
   ```go
   type ThinkingContentBlock struct {
       Type      ContentBlockType `json:"type"`
       Thinking  string           `json:"thinking"`
       Signature string           `json:"signature,omitempty"`
   }
   ```
3. Add case to switch in `message.go:46-68`
4. Add unit tests

### If current behavior is acceptable:

Document that unknown block types are silently skipped for forward compatibility.

## Additional Information for Planning

### Data Flow Verification

```
JSONL file (Claude Code)
    -> daemon/log_service.go (sonic.Unmarshal to SessionRecord)
    -> shared/domain/message.go (UnmarshalJSON for MessageContent)
    -> api/adapter.go (sonic.Marshal for storage)
    -> MongoDB (stored as JSON string)
    -> api/dashboard_repo.go (sonic.Unmarshal for retrieval)
```

All stages verified working for defined content block types.

### Missing Test Coverage

The following should have unit tests:
- `MessageContent.UnmarshalJSON` with various content types
- `MessageContent.MarshalJSON` round-trip
- Each `ContentBlock` type parsing
- Edge cases: empty content, null content, malformed JSON

### Performance Consideration

Sonic is significantly faster than `encoding/json` for the outer unmarshal, but `MessageContent.UnmarshalJSON` uses `encoding/json` internally. This is acceptable because:
1. MessageContent is a small portion of the overall SessionRecord
2. The polymorphic parsing logic requires json.RawMessage which sonic handles differently
3. Mixing sonic/encoding is officially supported

## Conclusion

The core JSONL message content parsing is **working correctly** for all defined content block types (`text`, `tool_use`, `tool_result`). The previous issue where block content wasn't stored has been fixed.

The only gap is the `thinking` content block type which exists in JSONL files but is intentionally not parsed (forward compatibility design). If this data is needed, it requires adding the `ThinkingContentBlock` type following the existing pattern.

**Recommendation**: If `thinking` blocks contain valuable data for the dashboard, add support for them. Otherwise, the current implementation is correct and no changes are needed.
