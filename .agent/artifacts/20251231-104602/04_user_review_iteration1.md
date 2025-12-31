# Review Result

**Status**: Changes Required

## Request Summary

Build error due to type mismatch between domain model and protobuf schema. The domain model `UserMessage.Content` was changed to `any` type (supporting both `string` and `[]*UserMessageBlockContent` array for polymorphic content), but the protobuf definition still declares it as `string content = 2;`. This causes a compilation error in the converter.

The implementation violates protobuf schema design principles by attempting to use type assertions in the converter to work around the schema mismatch. Instead, the protobuf schema must be updated to properly represent the polymorphic nature of the Content field.

## Acceptance Criteria

- [ ] Update protobuf schema `UserMessage.content` to support polymorphic content (string or array)
- [ ] Regenerate protobuf code using `buf generate`
- [ ] Update converter to handle the new protobuf schema without type assertions
- [ ] Ensure build succeeds without errors

## Scope

### In Scope
- Update `idl/protobuf/aggregation/v1/aggregation.proto` to support polymorphic UserMessage.content
- Regenerate protobuf stubs
- Fix converter implementation to match new schema

### Out of Scope
- Changes to domain model (already correctly implemented)
- Changes to other parts of the codebase unrelated to this type issue

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto` | 32 | `.agent/rules/idl/protobuf.md` | `UserMessage.content` field type mismatch with domain model. Protobuf defines `string content = 2` but domain model uses `any` type (string or array). | Change field definition to use `oneof` pattern to support polymorphic content: `oneof content { string text = 2; UserMessageBlockContentList blocks = 3; }` |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` | 124-150 | `.agent/rules/go/go-inbound-grpc-connectrpc.md` | Converter uses type assertion workaround (`if str, ok := ...`) instead of proper schema-driven conversion. This is a symptom of the protobuf schema mismatch. | After updating protobuf schema, implement proper conversion that maps domain `any` type to protobuf `oneof` field |

## Additional Context

### Current Protobuf Schema (Incorrect)

```protobuf
// UserMessage contains user message content.
message UserMessage {
  string role = 1;
  string content = 2;  // ❌ This should support polymorphic types
}
```

### Domain Model (Correct)

```go
type UserMessage struct {
    Role    UserMessageRole `json:"role" bson:"role"`
    Content any             `json:"content" bson:"content"` // ✅ Can be string or []*UserMessageBlockContent
}
```

### Proposed Protobuf Schema Fix

Option 1 - Using `oneof` (Recommended):
```protobuf
message UserMessageBlockContentList {
  repeated UserMessageBlockContent blocks = 1;
}

message UserMessage {
  string role = 1;
  oneof content {
    string text = 2;
    UserMessageBlockContentList blocks = 3;
  }
}
```

Option 2 - Using `google.protobuf.Any`:
```protobuf
import "google/protobuf/any.proto";

message UserMessage {
  string role = 1;
  google.protobuf.Any content = 2;
}
```

**Recommendation**: Use Option 1 (`oneof`) as it provides type safety and clearer API semantics. The converter can then map:
- Domain `string` → Protobuf `text` field
- Domain `[]*UserMessageBlockContent` → Protobuf `blocks` field

### Build Error

```
api-1  | internal/service/dashboard/inbound/grpc/connectrpc/converter.go:150:13: cannot use u.Message.Content (variable of interface type any) as string value in struct literal: need type assertion
```

### Files to Modify

1. `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto` - Update schema
2. Run: `cd idl/protobuf && buf generate` - Regenerate stubs
3. `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` - Update converter logic

## Rules References

The following rules were applied during this review:
- [`.agent/rules/idl/protobuf.md`](.agent/rules/idl/protobuf.md) - Protobuf schema design conventions
- [`.agent/rules/go/go-inbound-grpc-connectrpc.md`](.agent/rules/go/go-inbound-grpc-connectrpc.md) - ConnectRPC converter patterns
- [`.agent/rules/go/go-struct.md`](.agent/rules/go/go-struct.md) - Go struct type conventions
- [`.agent/rules/common.md`](.agent/rules/common.md) - General coding standards
