# Development Walkthrough

## Summary

Implemented custom JSON and BSON marshaling for the `UserMessage.Content` field to support polymorphic content types (string or array of content blocks), enabling proper serialization and deserialization of Claude Code JSONL session logs. Also added a Makefile to the shared module for convenient test execution.

## Code Overview

### Modified Components

#### `UserMessage` - Polymorphic Content Marshaling
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_user.go` (lines 69-221)
- **Purpose**: Support polymorphic content field that can be either a simple string or an array of block content
- **Changes**:
  - Changed `Content` field type from `string` to `any` (line 71)
  - Added custom JSON marshaling methods (lines 74-153)
  - Added custom BSON marshaling methods (lines 155-221)

**Key Methods**:

- `UnmarshalJSON(*UserMessage) error` (lines 87-138): Detects content type by inspecting the first character of raw JSON (`"` for string, `[` for array) and unmarshals accordingly. Handles null and empty values gracefully.

- `MarshalJSON(UserMessage) ([]byte, error)` (lines 140-153): Preserves the original content type during marshaling using an alias pattern to prevent infinite recursion.

- `UnmarshalBSON(*UserMessage) error` (lines 161-206): Uses BSON type detection system (`bson.TypeString` vs `bson.TypeArray`) to correctly unmarshal polymorphic content from MongoDB.

- `MarshalBSON(UserMessage) ([]byte, error)` (lines 208-221): Marshals to BSON using `bson.D` document structure, preserving content type.

#### `UserMessageBlockContentToolResult` - JSON Tag Fix
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_user.go` (line 59)
- **Changes**: Fixed JSON tag from `json:"toolUseId"` to `json:"tool_use_id"` to match Claude Code JSONL format (snake_case for JSON, camelCase for BSON)

**Why**: The Claude Code JSONL files use snake_case field names (`tool_use_id`), but our struct had camelCase, causing unmarshaling failures for tool_result blocks.

### New Components

#### Helper Types for Custom Marshaling
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_user.go`

**`userMessageAlias`** (lines 74-79): Alias struct with identical fields but no custom marshal methods, used to prevent infinite recursion during JSON marshaling.

**`userMessageRaw`** (lines 81-85): Used for initial JSON unmarshaling to capture `Content` as `json.RawMessage` for type inspection.

**`userMessageBSONRaw`** (lines 155-159): Used for initial BSON unmarshaling to capture `Content` as `bson.RawValue` for type detection.

#### Makefile for Shared Module
- **Location**: `/Users/jayce/team-attention/cops/shared/Makefile`
- **Purpose**: Provide convenient test command for the shared module
- **Target**: `make test` - Runs `go test -v ./domain/...`

## Testing

### Test Updates

**Updated Test** (lines 235-281 in `record_test.go`):
- Modified "parses user record with array content from JSONL file" to expect successful unmarshaling
- Added comprehensive assertions for array content with text and image blocks
- Validates block types, text content, and image source metadata

**New Tests Added** (lines 283-472 in `record_test.go`):

1. **String content parsing** (lines 283-319): Validates unmarshaling of line 4 from JSONL file with simple string content ("hellop")

2. **Tool result content parsing** (lines 321-367): Validates unmarshaling of line 10 with tool_result blocks, verifying `tool_use_id` field is correctly read

3. **Round-trip array content** (lines 369-422): Ensures unmarshal → marshal → unmarshal preserves array content and block types

4. **Round-trip string content** (lines 424-472): Ensures unmarshal → marshal → unmarshal preserves string content exactly

### Test Coverage

All 32 specs pass successfully:
```bash
cd /Users/jayce/team-attention/cops/shared
make test
# Output: SUCCESS! -- 32 Passed | 0 Failed | 0 Pending | 0 Skipped
```

### Verification Commands Run

```bash
# Test shared domain package
cd /Users/jayce/team-attention/cops/shared
go test -v ./domain/...  # Result: 32/32 PASS

# Build verification
cd /Users/jayce/team-attention/cops
go build ./shared/...    # Result: SUCCESS

# Using new Makefile
cd /Users/jayce/team-attention/cops/shared
make test                # Result: 32/32 PASS
```

## Technical Decisions

### Decision 1: Use `any` Type for Content Field

**Rationale**: The `UserMessage.Content` field needs to support two fundamentally different types (string and array). While Go doesn't have native union types, using `any` with custom marshaling provides type-safe handling at serialization boundaries while maintaining flexibility.

**Alternative Considered**: Creating separate `UserMessageWithString` and `UserMessageWithBlocks` structs was rejected because:
- It would complicate the `Record` unmarshaling logic
- The polymorphic nature is inherent to Claude Code's JSON format
- Custom marshaling is a proven pattern already used in `record.go` for the `Record` type

### Decision 2: First-Character Detection for JSON

**Rationale**: JSON type detection uses first-character inspection (`"` for string, `[` for array) rather than attempting to unmarshal both types.

**Why**: This approach is:
- Efficient: O(1) type detection before unmarshaling
- Reliable: JSON strings always start with `"`, arrays with `[`
- Consistent: Matches the pattern used in `Record.UnmarshalJSON` (line 87 in `record.go`)

### Decision 3: BSON Type System for MongoDB

**Rationale**: BSON unmarshaling uses the built-in type system (`bson.TypeString`, `bson.TypeArray`) instead of inspecting raw bytes.

**Why**: BSON provides explicit type information through `bson.RawValue.Type`, making it more reliable than string inspection. This follows mongo-driver v2 best practices.

### Decision 4: Snake_case for JSON, CamelCase for BSON

**Rationale**: `UserMessageBlockContentToolResult.ToolUseID` uses different JSON and BSON tags:
- JSON: `json:"tool_use_id"` (snake_case)
- BSON: `bson:"toolUseId"` (camelCase)

**Why**:
- Claude Code JSONL files use snake_case field naming
- MongoDB collections in this project use camelCase for consistency
- This dual-tag approach maintains compatibility with both systems

### Decision 5: Add Makefile to Shared Module

**Rationale**: Created `/Users/jayce/team-attention/cops/shared/Makefile` with a `test` target.

**Why**:
- Provides consistent interface across modules (cli, api, daemon all have Makefiles)
- User explicitly requested convenient `make test` command
- Follows project conventions for developer tooling

## Issues & Resolutions

| Issue | Resolution |
|-------|-----------|
| Test at line 235-267 failing due to array content | Updated test to expect successful unmarshaling and added assertions for block content validation |
| `tool_use_id` field not unmarshaling from JSONL | Fixed JSON tag from camelCase to snake_case to match Claude Code's JSONL format |
| BSON marshaling pattern unclear for polymorphic types | Researched mongo-driver v2 via Context7, implemented using `bson.RawValue` for type detection |
| No convenient way to run tests in shared module | Created Makefile with `test` target following project conventions |

## Data Flow

### JSON Unmarshaling Flow

```
JSONL line → json.Unmarshal → UserMessage.UnmarshalJSON
  ↓
userMessageRaw (captures raw Content bytes)
  ↓
First character inspection: '"' or '['?
  ↓
String path: json.Unmarshal → string
Array path: json.Unmarshal → []*UserMessageBlockContent
  ↓
m.Content = (string or []*UserMessageBlockContent)
```

### BSON Unmarshaling Flow

```
BSON bytes → bson.Unmarshal → UserMessage.UnmarshalBSON
  ↓
userMessageBSONRaw (captures Content as bson.RawValue)
  ↓
Type detection: raw.Content.Type?
  ↓
bson.TypeString → raw.Content.Unmarshal(&str)
bson.TypeArray → raw.Content.Unmarshal(&blocks)
  ↓
m.Content = (string or []*UserMessageBlockContent)
```

## Content Block Types Supported

The `UserMessageBlockContent` struct (lines 50-56) uses a discriminated union pattern with embedded fields to handle three block types:

### Text Blocks
```json
{"type":"text","text":"message content"}
```
- Uses `Text *string` field (optional pointer for discriminated union)

### Image Blocks
```json
{"type":"image","source":{"type":"base64","media_type":"image/png","data":"..."}}
```
- Uses `Source *UserMessageBlockContentSource` field
- Contains nested source metadata (type, media type, base64 data)

### Tool Result Blocks
```json
{"type":"tool_result","tool_use_id":"toolu_xxx","content":"result"}
```
- Uses embedded `*UserMessageBlockContentToolResult` field
- Fields: `tool_use_id` (string), `content` (any)

## Future Considerations

### Validation
Current implementation assumes JSONL data is valid. Future enhancements could add:
- Block type validation (ensure `type` field matches populated fields)
- Required field validation for each block type
- Content format validation (e.g., base64 data format)

### Performance
For large JSONL files with many block content messages:
- Consider streaming unmarshaling instead of loading entire files
- Profile memory usage for sessions with many image blocks
- Evaluate need for content deduplication (base64 images can be large)

### Type Safety
While custom marshaling provides type safety at serialization boundaries, runtime type assertions are still needed when accessing `Content`. Consider:
- Helper methods like `GetStringContent()` and `GetBlockContent()` with error returns
- A `ContentType` enum field to avoid reflection-based type checking

## Related Commits

This work is part of a larger effort to support polymorphic content in the C-Ops system:

- **Commit 6debadb**: "fix(api): fix MongoDB aggregation field naming errors" - Current work
- **Related to**: Support for Claude Code JSONL session log parsing with block-based content

## How to Use the New Features

### Running Tests
```bash
# From shared directory
cd /Users/jayce/team-attention/cops/shared
make test

# Or from project root
make -C shared test
```

### Working with Polymorphic Content

**Reading string content**:
```go
var record domain.Record
json.Unmarshal(jsonData, &record)

userRecord := record.Data.(*domain.UserRecord)
if content, ok := userRecord.Message.Content.(string); ok {
    fmt.Println("String content:", content)
}
```

**Reading block content**:
```go
if blocks, ok := userRecord.Message.Content.([]*domain.UserMessageBlockContent); ok {
    for _, block := range blocks {
        switch block.Type {
        case "text":
            fmt.Println("Text:", *block.Text)
        case "image":
            fmt.Println("Image:", block.Source.Media_type)
        case "tool_result":
            fmt.Println("Tool result:", block.ToolUseID)
        }
    }
}
```

**Creating messages programmatically**:
```go
// String content
msg := domain.UserMessage{
    Role: domain.UserMessageRoleUser,
    Content: "Hello, Claude!",
}

// Block content
msg := domain.UserMessage{
    Role: domain.UserMessageRoleUser,
    Content: []*domain.UserMessageBlockContent{
        {
            Type: "text",
            Text: ptr("Message with blocks"),
        },
    },
}

// Helper function for pointer
func ptr(s string) *string { return &s }
```

### BSON Compatibility

The same marshaling logic works seamlessly with MongoDB:
```go
// Insert into MongoDB
collection.InsertOne(ctx, userRecord)

// Query from MongoDB
var result domain.UserRecord
collection.FindOne(ctx, filter).Decode(&result)

// Content field is correctly unmarshaled as string or []*UserMessageBlockContent
```
