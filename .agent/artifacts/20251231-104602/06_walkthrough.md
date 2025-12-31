# Development Walkthrough

## Summary

Fixed critical MongoDB aggregation pipeline errors (Location40235) in the C-Ops Dashboard API by correcting field naming in aggregation stages, and updated Protobuf schema to support polymorphic UserMessage content with proper type safety.

## Problem Statement

The C-Ops Dashboard API was failing with MongoDB error `(Location40235) The field name 'message.usage.outputTokens' cannot contain '.'` when executing aggregation pipelines for project and session statistics. This occurred because dotted field paths (e.g., `"message.usage.inputTokens"`) were being used as field **names** in `$group` and `$addFields` stages, which violates MongoDB's requirement that field names cannot contain dots.

Additionally, a secondary issue emerged during implementation: the domain model `UserMessage.Content` field was changed from `string` to `any` type to support polymorphic content (either string or array of blocks), but the Protobuf schema still defined it as a simple string field, causing type mismatch compilation errors.

## Code Overview

### New Components

#### Aggregation Field Name Constants

- **Location**: `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` (lines 21-33)
- **Purpose**: Define simple field names (without dots) for use in MongoDB aggregation output stages
- **Constants**:
  - `aggInputTokensField = "inputTokens"`: Input token sums aggregation field
  - `aggOutputTokensField = "outputTokens"`: Output token sums aggregation field
  - `aggCacheReadTokensField = "cacheReadTokens"`: Cache read token sums aggregation field
  - `aggCacheCreationTokensField = "cacheCreationTokens"`: Cache creation token sums aggregation field

These constants replace dotted paths like `"message.usage.inputTokens"` when creating new fields in aggregation stages, while source data references still use the full dotted path with `$` prefix.

#### Protobuf Schema Extensions

- **Location**: `idl/protobuf/aggregation/v1/aggregation.proto` (lines 29-62)
- **Purpose**: Support polymorphic UserMessage.Content field (string or array of blocks)
- **New Messages**:
  - `UserMessageBlockContentSource`: Image source information (type, media_type, data)
  - `UserMessageBlockContentToolResult`: Tool execution result information (tool_use_id, content)
  - `UserMessageBlockContent`: Single content block with type discrimination (text, image, or tool_result)
  - `UserMessageBlockContentList`: Wrapper for repeated UserMessageBlockContent blocks
- **Modified Message**:
  - `UserMessage`: Changed from `string content = 2` to `oneof content { string text = 2; UserMessageBlockContentList blocks = 3; }`

#### Converter Helper Function

- **Location**: `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` (lines 204-248)
- **Function**: `toProtoUserMessageBlockContent(block *shareddomain.UserMessageBlockContent) *aggregationv1.UserMessageBlockContent`
- **Purpose**: Convert domain UserMessageBlockContent to protobuf representation
- **Logic**:
  1. Creates base protobuf block with type field
  2. Copies text field if present (for text blocks)
  3. Converts source field if present (for image blocks)
  4. Converts tool_result field if present, marshaling non-string content to JSON

### Modified Components

#### MongoDB Aggregation Pipelines

- **Location**: `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- **Changes**: Replaced all dotted field names in aggregation stages with simple constants

**Affected Methods:**

1. **`ListProjects`** (lines 154-251)
   - `$group` stage in `$lookup` sub-pipeline: Changed field names from dotted paths to simple constants
   - `$project` stage in `$lookup` sub-pipeline: Updated field references to match new names
   - Outer `$project` stage: Updated `$ifNull` references to use new field names
   - Result extraction: Updated `mongoutil.Get` calls to use new field names

2. **`ListSessions`** (lines 321-392)
   - `$group` stage: Changed token field names from dotted paths to simple constants
   - Result extraction: Updated `mongoutil.Get` calls to use new field names

3. **`getRecentProjects`** (lines 528-569)
   - `$addFields` stage: Changed token field names from dotted paths to simple constants
   - Result extraction: Updated `mongoutil.Get` calls to use new field names

4. **`getRecentSessions`** (lines 585-627)
   - `$group` stage: Changed token field names from dotted paths to simple constants
   - Result extraction: Updated `mongoutil.Get` calls to use new field names

5. **`getProjectStats`** (lines 648-684)
   - `$group` stage: Changed token field names from dotted paths to simple constants
   - `$project` stage: Updated field references to match new names
   - Result extraction: Updated `mongoutil.Get` calls to use new field names

**Pattern Example:**

```go
// BEFORE (incorrect - dotted field name)
bson.M{"$group": bson.M{
    mongoschema.MessageUsageInputTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    // Evaluates to: "message.usage.inputTokens": {"$sum": "$message.usage.inputTokens"}
    // Error: field name cannot contain dots
}}

// AFTER (correct - simple field name)
bson.M{"$group": bson.M{
    aggInputTokensField: bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    // Evaluates to: "inputTokens": {"$sum": "$message.usage.inputTokens"}
    // Field name "inputTokens" is valid, source reference "$message.usage.inputTokens" is valid
}}
```

#### Domain Model Extensions

- **Location**: `shared/domain/record_user.go`
- **Changes**:
  1. Added new types to support block-based content:
     - `UserRecordToolUseResult`: Tool execution result container
     - `UserRecordToolUseResultFile`: File information from tool results
     - `UserMessageBlockContent`: Content block with type discrimination
     - `UserMessageBlockContentToolResult`: Embedded tool result data
     - `UserMessageBlockContentSource`: Image source data
  2. Changed `UserMessage.Content` from `string` to `any` type to support polymorphic content
  3. Implemented custom JSON/BSON marshaling:
     - `UnmarshalJSON`: Detects content type (string vs array) and unmarshals appropriately
     - `MarshalJSON`: Preserves content type during serialization
     - `UnmarshalBSON`: Handles BSON deserialization with type detection
     - `MarshalBSON`: Handles BSON serialization

#### ConnectRPC Converter

- **Location**: `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`
- **Changes**: Updated `toProtoUserRecordData` function to handle polymorphic UserMessage.Content
- **Logic**:
  1. Creates base UserRecordData with metadata
  2. Sets ParentUuid if present
  3. **Converts Content based on runtime type**:
     - If `string`: Sets `Message.Content = &UserMessage_Text{Text: content}`
     - If `[]*UserMessageBlockContent`: Converts each block to protobuf and sets `Message.Content = &UserMessage_Blocks{Blocks: ...}`
  4. Converts ThinkingMetadata if present
  5. Converts Todos if present

#### Assistant Record BSON Tags

- **Location**: `shared/domain/record_assistant.go` (lines 44-45)
- **Changes**: Fixed BSON tag naming for ephemeral token fields to use camelCase instead of snake_case
- **Reason**: Maintains consistency between JSON (snake_case for wire format) and BSON (camelCase for MongoDB storage)

## Testing

### Build Verification

```bash
# Build all modules
cd /Users/jayce/team-attention/cops && go build ./...
# Result: SUCCESS (all compilation errors resolved)

# Build specific modules
cd api && go build ./...
cd shared && go build ./...
# Result: SUCCESS
```

### Protobuf Code Generation

```bash
# Regenerate protobuf stubs
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
# Result: Generated new types in shared/gen/grpcstub/aggregation/v1/
```

### API Endpoint Verification Commands

After deploying changes, verify the following endpoints work correctly:

```bash
# 1. Dashboard Overview Stats
curl http://localhost:8080/api/v1/dashboard/overview

# 2. List Projects with Pagination
curl http://localhost:8080/api/v1/dashboard/projects?page=1&pageSize=10

# 3. List Sessions
curl http://localhost:8080/api/v1/dashboard/sessions?page=1&pageSize=10

# 4. List Sessions for Specific Project
curl http://localhost:8080/api/v1/dashboard/sessions?projectId=<project-id>

# 5. Get Project Details
curl http://localhost:8080/api/v1/dashboard/projects/<project-id>
```

**Expected Results:**
- No `Location40235` MongoDB errors in logs
- Token usage values correctly populated (non-zero for projects/sessions with actual usage)
- Session counts match expected values
- Pagination works correctly
- All response fields present and properly formatted

## Issues & Resolutions

| Issue | Resolution |
| ----- | --------- |
| MongoDB aggregation error: "The field name 'message.usage.outputTokens' cannot contain '.'" | Created local constants (`aggInputTokensField`, etc.) for use as aggregation output field names, while keeping dotted paths with `$` prefix for source data references |
| Compilation error: "cannot use u.Message.Content (type any) as string value" | Updated Protobuf schema to use `oneof content { string text; UserMessageBlockContentList blocks }` pattern and modified converter to handle type discrimination |
| BSON tag inconsistency for ephemeral token fields | Changed BSON tags from snake_case to camelCase to maintain consistency with MongoDB field naming conventions |
| Type mismatch between domain model and protobuf schema | Regenerated protobuf code after schema update and implemented proper `oneof` handling in converter with runtime type detection |

## Key Design Decisions

### Why Not Modify mongoschema Constants?

The constants in `shared/domain/mongoschema/session_record.go` correctly represent the actual nested field paths in MongoDB documents (`"message.usage.inputTokens"`). These are used for:
1. **Source data references** in aggregation pipelines (with `$` prefix)
2. **Query filters** for matching documents
3. **Documentation** of the actual document structure

Changing these would break existing queries and misrepresent the data schema. The solution was to create separate constants specifically for aggregation output field names.

### Why Use `oneof` Pattern Instead of `google.protobuf.Any`?

The `oneof` pattern was chosen for the polymorphic content field because:
1. **Type Safety**: Provides compile-time type checking and clear API semantics
2. **Clarity**: Explicitly documents the two valid content types (string or blocks)
3. **Efficiency**: No runtime type URL resolution like `Any` requires
4. **Simplicity**: Easier for API consumers to understand and use

### Why Custom JSON/BSON Marshaling for UserMessage?

Custom marshaling ensures:
1. **Backward Compatibility**: Existing string content works without changes
2. **Forward Compatibility**: New block-based content is properly typed
3. **Type Preservation**: Content type (string vs array) is preserved during serialization/deserialization
4. **MongoDB Compatibility**: BSON marshaling handles both types correctly

## Related Documentation

- **Architecture**: See `/Users/jayce/team-attention/cops/CLAUDE.md` for system overview
- **MongoDB Schema**: See `shared/domain/mongoschema/session_record.go` for field path constants
- **Protobuf Conventions**: See `.agent/rules/idl/protobuf.md` for schema design patterns
- **Go Struct Rules**: See `.agent/rules/go/go-struct.md` for pointer vs value type guidelines
- **ConnectRPC Patterns**: See `.agent/rules/go/go-inbound-grpc-connectrpc.md` for converter patterns

## Future Considerations

1. **Cache Creation Tokens**: The `aggCacheCreationTokensField` constant is defined but not currently used in aggregation pipelines. Consider adding cache creation token tracking if needed.

2. **Additional Content Block Types**: The protobuf schema and domain models now support extensible content blocks. New block types can be added by:
   - Extending `UserMessageBlockContent` with new optional fields
   - Updating `toProtoUserMessageBlockContent` converter
   - No changes to aggregation pipelines required

3. **Performance Optimization**: If aggregation performance becomes an issue with large datasets, consider:
   - Adding indexes on `projectId`, `sessionId`, and `timestamp` fields
   - Using `$hint` to force index usage in complex pipelines
   - Implementing materialized views for frequently-accessed statistics

4. **Error Monitoring**: Monitor production logs for any remaining MongoDB errors or type conversion issues in the converter layer.
