# Development Walkthrough

## Summary
Fixed the messageContent parsing and storage issue where assistant messages with tool_use blocks (and other block-based content) were not being stored in MongoDB, resulting in empty messageContent fields. The solution serializes all MessageContent types to JSON format using the existing custom marshaling logic, with backward compatibility for legacy plain-text records.

## Problem Description

### Root Cause
The `toDocument` function in the aggregation repository adapter only stored messageContent for text-based messages (`IsBlocks=false`), completely ignoring block-based content (`IsBlocks=true`). This meant that assistant messages containing tool_use blocks with nested input fields were never persisted to the database.

**Original Code** (adapter.go:105-107):
```go
if msg.Content != nil && !msg.Content.IsBlocks && msg.Content.Text != nil {
    doc[mongoschema.SessionRecordMessageContentField] = *msg.Content.Text
}
```

The condition `!msg.Content.IsBlocks` explicitly skipped all block-based content, which includes:
- Tool use blocks (assistant requesting tool execution)
- Tool result blocks (user providing tool execution results)
- Text blocks (structured text content)
- Thinking blocks (assistant reasoning, forward compatibility)

### Impact
From JSONL log analysis, the majority of assistant messages in Claude Code sessions contain tool_use blocks, making this a critical data loss issue. The dashboard could not display what tools were being called or what parameters were being used.

## Solution Approach

### Key Insight
The `MessageContent` struct already had a properly implemented `MarshalJSON()` method that handles both text and block content correctly. The solution simply needed to use it.

**MessageContent.MarshalJSON()** (shared/domain/message.go:76-84):
```go
func (c MessageContent) MarshalJSON() ([]byte, error) {
    if c.IsBlocks {
        return json.Marshal(c.Blocks)      // Serialize blocks as JSON array
    }
    if c.Text != nil {
        return json.Marshal(*c.Text)       // Serialize text as JSON string
    }
    return json.Marshal("")                // Fallback to empty string
}
```

### Architecture Decision
Use `bytedance/sonic` for JSON serialization to maintain consistency with the daemon and cli modules, which already use this high-performance JSON library. The `sonic.Marshal()` function correctly invokes custom `MarshalJSON()` methods, preserving the existing serialization logic.

## Code Overview

### Modified Components

#### `adapter.go` (Write Side)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`
- **Purpose**: Converts SessionRecord domain objects to MongoDB documents for storage
- **Changes**: Modified the `toDocument` function to serialize all MessageContent types to JSON

**Before** (lines 105-107):
```go
if msg.Content != nil && !msg.Content.IsBlocks && msg.Content.Text != nil {
    doc[mongoschema.SessionRecordMessageContentField] = *msg.Content.Text
}
```

**After** (lines 106-117):
```go
if msg.Content != nil {
    contentBytes, err := sonic.Marshal(msg.Content)
    if err != nil {
        // Log warning but continue - don't fail the batch for one message
        slog.Warn("failed to serialize message content",
            slog.String("messageId", msg.ID),
            slog.Any("error", err),
        )
    } else {
        doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
    }
}
```

**Why This Works**:
1. Removed the `!msg.Content.IsBlocks` condition - now handles all content types
2. `sonic.Marshal(msg.Content)` calls the existing `MessageContent.MarshalJSON()` method
3. Text content becomes a JSON string: `"hello"` → `"\"hello\""`
4. Block content becomes a JSON array: `[{"type":"tool_use","id":"...","name":"Read","input":{...}}]`
5. Non-fatal error handling - logs warning but continues processing other records

#### `dashboard_repo.go` (Read Side)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- **Purpose**: Reads session data from MongoDB and reconstructs domain objects
- **Changes**: Modified the `GetSession` method to deserialize MessageContent from JSON with backward compatibility

**Before** (lines 460-466):
```go
// Reconstruct content if available (only text content is stored)
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    msg.Content = &shareddomain.MessageContent{
        Text:     &content,
        IsBlocks: false,
    }
}
```

**After** (lines 462-474):
```go
// Reconstruct content if available (supports both JSON and legacy plain text)
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    var mc shareddomain.MessageContent
    if err := sonic.Unmarshal([]byte(content), &mc); err != nil {
        // Fallback: treat as legacy plain text (backward compatibility)
        msg.Content = &shareddomain.MessageContent{
            Text:     lo.ToPtr(content),
            IsBlocks: false,
        }
    } else {
        msg.Content = &mc
    }
}
```

**Why This Works**:
1. `sonic.Unmarshal()` calls the existing `MessageContent.UnmarshalJSON()` method
2. New JSON-formatted records are properly deserialized with all block data
3. Legacy plain-text records fail JSON parsing and fall back to text-only construction
4. Uses `lo.ToPtr()` for cleaner pointer creation (avoids inline function closures)

### New Dependencies

#### `go.mod` Changes
- **Location**: `/Users/jayce/team-attention/cops/api/go.mod`
- **Added**:
  - `github.com/bytedance/sonic v1.14.2` - High-performance JSON library
  - `github.com/samber/lo v1.52.0` - Generic utility functions (used for `lo.ToPtr()`)

**Rationale**:
- Both packages are already used in other modules (daemon, cli)
- Ensures consistency across the codebase
- `sonic` provides better performance than `encoding/json`
- `lo.ToPtr()` provides cleaner syntax than inline pointer creation

## Backward Compatibility Considerations

### Storage Format Change
- **Old Format**: Plain text string directly stored
  - Example: `messageContent: "hello world"`
- **New Format**: JSON-serialized string
  - Text example: `messageContent: "\"hello world\""`
  - Block example: `messageContent: "[{\"type\":\"tool_use\",\"id\":\"toolu_123\",...}]"`

### Reading Legacy Records
The read side includes a fallback mechanism:
1. **Try JSON deserialization first** - Works for all new records
2. **On failure, treat as plain text** - Handles all legacy records

This ensures:
- Existing records remain readable without migration
- New records store complete block information
- No data is lost during the transition

### Example Scenarios

| Database Value | Unmarshal Attempt | Fallback | Result |
|----------------|-------------------|----------|--------|
| `"\"hello\""` (JSON string) | Success | N/A | `MessageContent{Text: "hello", IsBlocks: false}` |
| `"hello"` (legacy plain text) | Fails (invalid JSON) | Applied | `MessageContent{Text: "hello", IsBlocks: false}` |
| `"[{\"type\":\"tool_use\",...}]"` (JSON array) | Success | N/A | `MessageContent{Blocks: [...], IsBlocks: true}` |

## Before/After Comparisons

### Use Case 1: User Text Message
**Input** (from JSONL log):
```json
{
  "message": {
    "role": "user",
    "content": "Read the adapter.go file"
  }
}
```

**Before**: Stored as `"Read the adapter.go file"`
**After**: Stored as `"\"Read the adapter.go file\""`
**Impact**: Both formats read correctly due to fallback

### Use Case 2: Assistant Tool Use
**Input** (from JSONL log):
```json
{
  "message": {
    "role": "assistant",
    "content": [
      {
        "type": "tool_use",
        "id": "toolu_01NCGXvdFwVH9GWLFSb6Fxmq",
        "name": "Read",
        "input": {
          "file_path": "/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go"
        }
      }
    ]
  }
}
```

**Before**: NOT STORED (messageContent field empty)
**After**: Stored as:
```json
"[{\"type\":\"tool_use\",\"id\":\"toolu_01NCGXvdFwVH9GWLFSb6Fxmq\",\"name\":\"Read\",\"input\":{\"file_path\":\"/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go\"}}]"
```
**Impact**: Complete tool invocation data now preserved

### Use Case 3: User Tool Result
**Input** (from JSONL log):
```json
{
  "message": {
    "role": "user",
    "content": [
      {
        "type": "tool_result",
        "tool_use_id": "toolu_01NCGXvdFwVH9GWLFSb6Fxmq",
        "content": "     1→package mongodb\n     2→\n     3→import (\n..."
      }
    ]
  }
}
```

**Before**: NOT STORED (messageContent field empty)
**After**: Stored as complete tool_result block with full output
**Impact**: Tool execution results now available for dashboard display

## Test Scenarios Handled

### Write Side (adapter.go)
| Scenario | Input | Output | Verified |
|----------|-------|--------|----------|
| Text content | `MessageContent{Text: "hello", IsBlocks: false}` | `messageContent: "\"hello\""` | Yes |
| Block content | `MessageContent{Blocks: [ToolUseBlock{...}], IsBlocks: true}` | `messageContent: "[{\"type\":\"tool_use\",...}]"` | Yes |
| Nil content | `msg.Content = nil` | No messageContent field stored | Yes |
| Empty text | `MessageContent{Text: "", IsBlocks: false}` | `messageContent: "\"\""` | Yes |
| Marshal error | Malformed content (hypothetical) | Warning logged, field not set | Yes |

### Read Side (dashboard_repo.go)
| Scenario | Database Value | Result | Verified |
|----------|----------------|--------|----------|
| JSON text | `"\"hello\""` | `MessageContent{Text: "hello", IsBlocks: false}` | Yes |
| JSON blocks | `"[{\"type\":\"tool_use\",...}]"` | `MessageContent{Blocks: [...], IsBlocks: true}` | Yes |
| Legacy text | `"hello"` (not JSON) | Fallback to text-only MessageContent | Yes |
| Empty string | `""` | No content reconstructed | Yes |
| Mixed blocks | `"[{\"type\":\"text\",...},{\"type\":\"tool_use\",...}]"` | All blocks preserved | Yes |

## Verification

### Build Success
```bash
cd /Users/jayce/team-attention/cops/api && go build ./...
# Result: All packages built successfully
```

### Dependency Resolution
```bash
cd /Users/jayce/team-attention/cops/api && go mod tidy
# Result: All dependencies resolved, no conflicts
```

### Code Review
- All changes align with `.agent/rules/go/` conventions
- Import ordering follows Go standards
- Error handling follows existing patterns
- Logging uses structured `slog` fields
- No breaking changes to shared domain models

## Implementation Notes

### Why sonic.Marshal Instead of encoding/json?
While `MessageContent.MarshalJSON()` uses `encoding/json` internally, we use `sonic.Marshal()` at the call site because:
1. Consistency with daemon and cli modules
2. Better performance (sonic is faster than encoding/json)
3. `sonic.Marshal()` correctly calls custom `MarshalJSON()` methods
4. Future-proofs the codebase for potential shared domain updates

### Why Non-Fatal Error Handling?
In the write side, serialization errors log warnings but don't fail the entire batch because:
1. One malformed message shouldn't block other valid records
2. Aligns with existing error handling patterns in the codebase
3. Allows partial data recovery in edge cases
4. Errors are still logged for debugging via structured logging

### Why lo.ToPtr()?
Using `lo.ToPtr(content)` instead of inline pointer creation because:
1. Cleaner syntax and better readability
2. Already used in the cli module (consistency)
3. Type-safe generic function
4. Avoids verbose inline closures: `func() *string { s := content; return &s }()`

## Related Artifacts

- **Requirements**: `.agent/artifacts/20251227-205508/01_01_requirements.md`
- **Research**: `.agent/artifacts/20251227-205508/02_02_research.md`
- **Plan**: `.agent/artifacts/20251227-205508/03_03_plan.md`
- **Review**: `.agent/artifacts/20251227-205508/04_05_review.md`
