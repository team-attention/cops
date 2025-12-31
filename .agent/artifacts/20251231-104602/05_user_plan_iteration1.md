# Implementation Plan: Fix Protobuf Schema Mismatch for UserMessage.Content

## Overview

The domain model `UserMessage.Content` uses an `any` type to support polymorphic content (either `string` or `[]*UserMessageBlockContent`). However, the protobuf schema defines `UserMessage.content` as `string content = 2;`, causing a type mismatch and compilation error in the converter.

This plan updates the protobuf schema to use the `oneof` pattern to properly represent the polymorphic nature of the Content field, regenerates the protobuf code, and updates the converter to handle the new schema.

## Package Changes

No new packages are required. This implementation uses existing protobuf capabilities.

---

## Step 1: Update Protobuf Schema

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Protobuf naming conventions and patterns
- `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`: Current schema to modify
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Domain model structure for reference

### `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`

**Description**:
Add new message types for `UserMessageBlockContent` and its nested types, add a wrapper message `UserMessageBlockContentList`, and modify `UserMessage` to use `oneof content` pattern.

#### New Message: `UserMessageBlockContentSource`

Add after line 27 (after `MessageMetadata`):

```protobuf
// UserMessageBlockContentSource contains image source information.
message UserMessageBlockContentSource {
  string type = 1;
  string media_type = 2;
  string data = 3;
}
```

#### New Message: `UserMessageBlockContentToolResult`

Add after `UserMessageBlockContentSource`:

```protobuf
// UserMessageBlockContentToolResult contains tool result information.
message UserMessageBlockContentToolResult {
  string tool_use_id = 1;
  string content = 2;
}
```

#### New Message: `UserMessageBlockContent`

Add after `UserMessageBlockContentToolResult`:

```protobuf
// UserMessageBlockContent contains a single content block (text, image, or tool_result).
message UserMessageBlockContent {
  string type = 1;
  string text = 2;
  UserMessageBlockContentSource source = 3;
  UserMessageBlockContentToolResult tool_result = 4;
}
```

#### New Message: `UserMessageBlockContentList`

Add after `UserMessageBlockContent`:

```protobuf
// UserMessageBlockContentList wraps repeated UserMessageBlockContent for oneof usage.
message UserMessageBlockContentList {
  repeated UserMessageBlockContent blocks = 1;
}
```

#### Modified Message: `UserMessage`

Replace the existing `UserMessage` message (lines 29-33) with:

```protobuf
// UserMessage contains user message content.
message UserMessage {
  string role = 1;
  oneof content {
    string text = 2;
    UserMessageBlockContentList blocks = 3;
  }
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Schema validation passes | Run `buf lint` | No errors | Lint validation |
| Code generation succeeds | Run `buf generate` | Generated Go files | Code generation |

---

## Step 2: Regenerate Protobuf Code

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`: Code generation command reference

**Command**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Description**:
Run the buf generate command to regenerate Go protobuf stubs. This will create new types in `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`:

- `UserMessageBlockContentSource` struct
- `UserMessageBlockContentToolResult` struct
- `UserMessageBlockContent` struct
- `UserMessageBlockContentList` struct
- `UserMessage` struct with `oneof` interface:
  - `isUserMessage_Content` interface
  - `UserMessage_Text` struct (wrapping `string`)
  - `UserMessage_Blocks` struct (wrapping `*UserMessageBlockContentList`)
  - `GetText()` method returning `string`
  - `GetBlocks()` method returning `*UserMessageBlockContentList`

---

## Step 3: Update Converter to Handle oneof Content

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md`: Converter patterns
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`: Converter to modify
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: Domain model types

### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`

**Description**:
Replace the current workaround implementation in `toProtoUserRecordData` with proper schema-driven conversion that maps domain `any` type to protobuf `oneof` field.

#### Replace Function: `toProtoUserRecordData`

Remove lines 122-188 and replace with:

```go
// toProtoUserRecordData converts UserRecord to proto UserRecordData.
func toProtoUserRecordData(u *shareddomain.UserRecord) *aggregationv1.UserRecordData {
	// 1. Create base UserRecordData with metadata and other fields.
	data := &aggregationv1.UserRecordData{
		Metadata: &aggregationv1.MessageMetadata{
			ParentUuid:  "",
			IsSidechain: u.IsSidechain,
			UserType:    string(u.UserType),
			SessionId:   u.SessionID,
			Version:     u.Version,
			GitBranch:   u.GitBranch,
			Uuid:        u.UUID,
			Timestamp:   timestamppb.New(u.Timestamp),
		},
		Message: &aggregationv1.UserMessage{
			Role: string(u.Message.Role),
		},
		IsMeta: u.IsMeta,
	}

	// 2. Set ParentUuid if present.
	if u.ParentUUID != nil {
		data.Metadata.ParentUuid = *u.ParentUUID
	}

	// 3. Convert Content based on its type (string or []*UserMessageBlockContent).
	if u.Message.Content != nil {
		switch content := u.Message.Content.(type) {
		case string:
			// a. If string: Set Message.Content to UserMessage_Text.
			data.Message.Content = &aggregationv1.UserMessage_Text{
				Text: content,
			}
		case []*shareddomain.UserMessageBlockContent:
			// b. If []*UserMessageBlockContent: Convert to protobuf blocks.
			protoBlocks := make([]*aggregationv1.UserMessageBlockContent, len(content))
			for i, block := range content {
				protoBlocks[i] = toProtoUserMessageBlockContent(block)
			}
			// Set Message.Content to UserMessage_Blocks.
			data.Message.Content = &aggregationv1.UserMessage_Blocks{
				Blocks: &aggregationv1.UserMessageBlockContentList{
					Blocks: protoBlocks,
				},
			}
		}
	}

	// 4. Convert ThinkingMetadata if present.
	if u.ThinkingMetadata != nil {
		data.ThinkingMetadata = &aggregationv1.UserRecordThinkingMetadata{
			Level:    u.ThinkingMetadata.Level,
			Disabled: u.ThinkingMetadata.Disabled,
		}
		if len(u.ThinkingMetadata.Triggers) > 0 {
			data.ThinkingMetadata.Triggers = make([]*aggregationv1.UserRecordThinkingMetadataTrigger, len(u.ThinkingMetadata.Triggers))
			for i, trigger := range u.ThinkingMetadata.Triggers {
				data.ThinkingMetadata.Triggers[i] = &aggregationv1.UserRecordThinkingMetadataTrigger{
					Start: int32(trigger.Start),
					End:   int32(trigger.End),
					Text:  trigger.Text,
				}
			}
		}
	}

	// 5. Convert Todos if present.
	if len(u.Todos) > 0 {
		data.Todos = make([]*aggregationv1.UserRecordTodo, len(u.Todos))
		for i, todo := range u.Todos {
			data.Todos[i] = &aggregationv1.UserRecordTodo{
				Content:    todo.Content,
				Status:     todo.Status,
				ActiveForm: todo.ActiveForm,
			}
		}
	}

	// 6. Return the fully constructed data.
	return data
}
```

#### New Function: `toProtoUserMessageBlockContent`

Add after `toProtoUserRecordData`:

```go
// toProtoUserMessageBlockContent converts a single domain UserMessageBlockContent to protobuf.
func toProtoUserMessageBlockContent(block *shareddomain.UserMessageBlockContent) *aggregationv1.UserMessageBlockContent {
	// 1. Create base protobuf block with type field.
	protoBlock := &aggregationv1.UserMessageBlockContent{
		Type: block.Type,
	}

	// 2. If Text is set, copy to proto.
	if block.Text != nil {
		protoBlock.Text = *block.Text
	}

	// 3. If Source is set (image block), convert to proto.
	if block.Source != nil {
		protoBlock.Source = &aggregationv1.UserMessageBlockContentSource{
			Type:      block.Source.Type,
			MediaType: block.Source.Media_type,
			Data:      block.Source.Data,
		}
	}

	// 4. If UserMessageBlockContentToolResult is set (tool_result block), convert to proto.
	if block.UserMessageBlockContentToolResult != nil {
		// Convert Content to string (may be string or other type).
		contentStr := ""
		if block.UserMessageBlockContentToolResult.Content != nil {
			switch c := block.UserMessageBlockContentToolResult.Content.(type) {
			case string:
				contentStr = c
			default:
				// Marshal non-string content to JSON.
				if jsonBytes, err := sonic.Marshal(c); err == nil {
					contentStr = string(jsonBytes)
				}
			}
		}
		protoBlock.ToolResult = &aggregationv1.UserMessageBlockContentToolResult{
			ToolUseId: block.UserMessageBlockContentToolResult.ToolUseID,
			Content:   contentStr,
		}
	}

	// 5. Return the converted block.
	return protoBlock
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| String content | UserRecord with string Content | Message.Content = UserMessage_Text | String type assertion |
| Block array content with text | UserRecord with []*UserMessageBlockContent containing text block | Message.Content = UserMessage_Blocks with text | Array type assertion, text block |
| Block array content with image | UserRecord with []*UserMessageBlockContent containing image block | Message.Content = UserMessage_Blocks with source | Array type assertion, image block |
| Block array content with tool_result | UserRecord with []*UserMessageBlockContent containing tool_result block | Message.Content = UserMessage_Blocks with tool_result | Array type assertion, tool_result block |
| Nil content | UserRecord with nil Content | Message.Content = nil | Nil check |
| Mixed block array | UserRecord with multiple block types | Message.Content = UserMessage_Blocks with all types | Multiple block conversion |

---

## Step 4: Remove Unused Import

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`: Check if sonic import is still needed

### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`

**Description**:
After the changes, the `sonic` import is still needed for marshaling non-string tool result content. No import changes required.

---

## Step 5: Verify Build

**Command**:
```bash
cd /Users/jayce/team-attention/cops && go build ./...
```

**Description**:
Build all modules to verify there are no compilation errors.

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Build succeeds | Run go build | Exit code 0, no errors | Full build |

---

## Summary of Changes

### Files Modified

1. `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`
   - Add `UserMessageBlockContentSource` message
   - Add `UserMessageBlockContentToolResult` message
   - Add `UserMessageBlockContent` message
   - Add `UserMessageBlockContentList` message
   - Modify `UserMessage` to use `oneof content` pattern

2. `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go` (regenerated)
   - Generated types for new messages
   - Generated `oneof` interface and wrapper types for `UserMessage.Content`

3. `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`
   - Replace `toProtoUserRecordData` function with proper `oneof` handling
   - Add `toProtoUserMessageBlockContent` helper function

### Execution Order

1. Update protobuf schema (Step 1)
2. Regenerate protobuf code (Step 2)
3. Update converter (Step 3)
4. Verify build (Step 5)

### Dependencies

- Step 2 depends on Step 1 (schema must be updated before regenerating)
- Step 3 depends on Step 2 (generated types must exist before updating converter)
- Step 5 depends on Steps 1-3 (all changes must be in place for build to succeed)
