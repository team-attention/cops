# Research Report

## Mode
General Research

## Request Summary
Fix the messageContent parsing and storage issue where the `toDocument` function in `adapter.go` only stores text-based content (`IsBlocks=false`) and ignores block-based content (`IsBlocks=true`), resulting in empty `messageContent` fields for assistant messages containing tool_use blocks.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | Contains the `toDocument` function that needs modification (lines 105-107) |
| `/Users/jayce/team-attention/cops/shared/domain/message.go` | Defines `MessageContent` struct with `MarshalJSON()` method that handles both text and blocks |
| `/Users/jayce/team-attention/cops/shared/domain/content_block.go` | Defines content block types: `text`, `tool_use`, `tool_result` |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:460-466` | Shows how `messageContent` is currently read back (text-only assumption) |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md` | Rules for outbound adapter implementation |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md` | Rules for struct field types |

## Log Data Analysis

### Message Content Types Observed

Based on analysis of JSONL log files in `/Users/jayce/.claude/projects/-Users-jayce-team-attention-cops`:

| Role | Content Type | Content Structure | Example |
|------|--------------|-------------------|---------|
| `user` | String | Plain text | `"content": "Clarify requirements for..."` |
| `user` | Array | `tool_result` blocks | `[{"type": "tool_result", "tool_use_id": "...", "content": "..."}]` |
| `user` | Array | `text` blocks | `[{"type": "text", "text": "..."}]` |
| `assistant` | Array | `text` blocks | `[{"type": "text", "text": "..."}]` |
| `assistant` | Array | `tool_use` blocks | `[{"type": "tool_use", "id": "...", "name": "...", "input": {...}}]` |
| `assistant` | Array | `thinking` blocks | `[{"type": "thinking", "thinking": "...", "signature": "..."}]` |

### Specific Log Examples

**1. User message with string content:**
```json
{
  "message": {
    "role": "user",
    "content": "Clarify requirements for the following request..."
  }
}
```
- `IsBlocks = false`, `Text` is set
- Currently stored correctly

**2. Assistant message with tool_use block:**
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
          "file_path": "/Users/jayce/team-attention/cops/.agent/artifacts/..."
        }
      }
    ]
  }
}
```
- `IsBlocks = true`, `Blocks` contains `ToolUseContentBlock`
- **Currently NOT stored** (messageContent field is empty)

**3. User message with tool_result block:**
```json
{
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "toolu_01NCGXvdFwVH9GWLFSb6Fxmq",
        "type": "tool_result",
        "content": "     1->package mongodb\n     2->..."
      }
    ]
  }
}
```
- `IsBlocks = true`, `Blocks` contains `ToolResultContentBlock`
- **Currently NOT stored**

**4. Assistant message with thinking block:**
```json
{
  "message": {
    "role": "assistant",
    "content": [
      {
        "type": "thinking",
        "thinking": "...",
        "signature": "..."
      }
    ]
  }
}
```
- Note: `thinking` block type is NOT currently handled in `content_block.go` (skipped for forward compatibility)

## Current Implementation Analysis

### Problem Location: `adapter.go:105-107`

```go
if msg.Content != nil && !msg.Content.IsBlocks && msg.Content.Text != nil {
    doc[mongoschema.SessionRecordMessageContentField] = *msg.Content.Text
}
```

**Issue**: This condition only handles text-based content (`IsBlocks = false`). When `IsBlocks = true`, nothing is stored.

### MessageContent.MarshalJSON() Already Works

From `/Users/jayce/team-attention/cops/shared/domain/message.go:76-84`:

```go
func (c MessageContent) MarshalJSON() ([]byte, error) {
    if c.IsBlocks {
        return json.Marshal(c.Blocks)
    }
    if c.Text != nil {
        return json.Marshal(*c.Text)
    }
    return json.Marshal("")
}
```

The `MarshalJSON` method correctly handles both cases:
- `IsBlocks = true`: Marshals the `Blocks` slice
- `IsBlocks = false`: Marshals the `Text` string

### Dashboard Read Side: `dashboard_repo.go:460-466`

```go
// Reconstruct content if available (only text content is stored)
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    msg.Content = &shareddomain.MessageContent{
        Text:     &content,
        IsBlocks: false,
    }
}
```

**Important**: The dashboard read side currently assumes `messageContent` is always a text string. If we store JSON for blocks, this needs to be updated too.

## Technical Investigation

### JSON Library Usage

| Module | Library | Usage |
|--------|---------|-------|
| `api/` | `encoding/json` (via domain) | Implicit through `MessageContent.MarshalJSON()` |
| `daemon/` | `bytedance/sonic` | Explicit in `log_service.go`, `configwatcher_service.go` |
| `cli/` | `bytedance/sonic` | Explicit in `jsonl_parser.go`, `filesystem_config.go` |
| `shared/` | `encoding/json` | Used in `message.go` for custom marshal/unmarshal |

**Recommendation**: The API module currently doesn't import `sonic` directly. Since `MessageContent.MarshalJSON()` uses `encoding/json` internally, we should either:
1. Use `encoding/json` for consistency with the domain model, OR
2. Add `sonic` dependency to API module for consistency with other modules

Given that `MessageContent.MarshalJSON()` already uses `encoding/json`, option 1 is simpler and maintains existing behavior.

### Error Handling Patterns

From codebase analysis, error handling follows these patterns:

```go
// Wrap errors with context
return fmt.Errorf("failed to marshal content: %w", err)

// Log errors at service layer
r.logger.Error("failed to serialize content",
    slog.String("messageId", msg.ID),
    slog.Any("error", err),
)
```

## Package Candidates

### Problem: JSON Serialization for messageContent

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| encoding/json | (stdlib) | Already used by `MessageContent.MarshalJSON()`, no new dependency needed, consistent with domain model |

**Decision**: Use `encoding/json` via `MessageContent.MarshalJSON()` - no new packages needed.

## Technical Constraints

1. **Must not modify shared domain models** - `MessageContent`, `ContentBlock` types are shared across services
2. **MessageContent.MarshalJSON() uses encoding/json** - Must be compatible with this
3. **Database schema unchanged** - `messageContent` field already stores string data
4. **Dashboard read side needs update** - If we store JSON, the read side must parse it back

## Similar Implementations Found

### Example 1: Using MarshalJSON in similar context
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go:82`
- **Relevance**: Shows how to use `sonic.Unmarshal` for JSONL parsing with domain types

### Example 2: Config file serialization
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go:86`
- **Relevance**: Shows pattern for serializing complex types to JSON string using `sonic.MarshalIndent`

## Proposed Solution

### Approach: Unified JSON Serialization

The fix is straightforward since `MessageContent.MarshalJSON()` already handles both cases:

```go
// Replace lines 105-107 in adapter.go with:
if msg.Content != nil {
    contentBytes, err := json.Marshal(msg.Content)
    if err != nil {
        // Log error but don't fail the entire batch
        r.logger.Warn("failed to serialize message content",
            slog.String("messageId", msg.ID),
            slog.Any("error", err),
        )
    } else {
        doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
    }
}
```

### Read Side Update Required

The dashboard read side (`dashboard_repo.go:460-466`) must be updated to:
1. Try to parse `messageContent` as JSON array (blocks)
2. If that fails, treat it as plain text (backward compatibility)

```go
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    var mc shareddomain.MessageContent
    if err := json.Unmarshal([]byte(content), &mc); err != nil {
        // Fallback: treat as plain text (backward compatibility)
        msg.Content = &shareddomain.MessageContent{
            Text:     &content,
            IsBlocks: false,
        }
    } else {
        msg.Content = &mc
    }
}
```

## Additional Information for Planning

### Scope Note from Requirements
The requirements document states: "Out of Scope: Dashboard UI changes to display the new messageContent format". However, the database read side (dashboard_repo.go) is **data access layer**, not UI. It must be updated to correctly reconstruct `MessageContent` from the stored JSON.

### Content Block Types Not Currently Handled
The `thinking` block type observed in logs is not handled in `content_block.go`. It is skipped via the `default` case for forward compatibility. This is acceptable behavior and should not be changed.

### Performance Consideration
Using `json.Marshal` for every message adds minimal overhead. The `MessageContent.MarshalJSON()` method is already implemented and tested.

### Backward Compatibility
Old records with plain text `messageContent` will be read correctly due to the fallback logic in the read side update.
