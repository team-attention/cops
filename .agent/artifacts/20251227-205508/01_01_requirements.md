# Requirements

## Request Summary
Fix the messageContent parsing and storage issue where the database messageContent field remains empty because the log data contains a nested structure (message.content[].input) that requires proper extraction and serialization before being stored in MongoDB.

## Acceptance Criteria

- [ ] Criterion 1: When message.content contains tool_use blocks with input fields, the input JSON data is properly extracted and serialized to the messageContent database field
- [ ] Criterion 2: The messageContent field in MongoDB contains a valid JSON string representation of the content blocks (not empty) for messages with tool_use content
- [ ] Criterion 3: Existing text-based content continues to work correctly (user messages with simple string content are still stored properly)
- [ ] Criterion 4: The solution handles all content block types (text, tool_use, tool_result) appropriately when serializing to messageContent

## Scope

### In Scope
- Modify the `toDocument` function in `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` to handle block-based content (lines 105-107)
- Serialize the entire MessageContent structure (including tool_use blocks with input fields) to JSON for storage in the messageContent field
- Ensure proper handling of both text content (IsBlocks=false) and block content (IsBlocks=true)
- Maintain backward compatibility with existing text-based message storage

### Out of Scope
- Changes to the domain model structure in `shared/domain/message.go` or `shared/domain/content_block.go`
- Modifications to the JSONL parsing logic in the daemon service
- Changes to the MessageContent UnmarshalJSON/MarshalJSON implementation (already correct)
- Database migration for existing empty messageContent fields
- Dashboard UI changes to display the new messageContent format

## Constraints
- Must use the existing MessageContent.MarshalJSON() method which already handles proper serialization
- Cannot modify the shared domain models as they are used across multiple services
- Must maintain type safety and error handling patterns used in the codebase
- Should use the bytedance/sonic JSON library consistently with the rest of the codebase

## Additional Context
- The issue is in lines 105-107 of `adapter.go` which only stores text content when `IsBlocks=false`
- The MessageContent struct already has proper MarshalJSON implementation that handles both text and blocks
- When content is blocks (`IsBlocks=true`), the current code doesn't store anything in messageContent
- Example log data shows tool_use blocks with nested input field:
  ```json
  {
    "message": {
      "content": [
        {
          "type": "tool_use",
          "id": "toolu_01JtApkeeftAiktVUCG4XWS3",
          "name": "Task",
          "input": {
            "subagent_type": "Explore",
            "prompt": "Explore the web frontend...",
            "description": "Explore web frontend"
          }
        }
      ]
    }
  }
  ```

## Questions Resolved

| Question                                                                                     | Answer                                                                                                                                                         |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Should we serialize the entire MessageContent (including blocks) or just extract input data? | Serialize the entire MessageContent structure as JSON, preserving all block types and their fields (type, id, name, input, etc.)                             |
| Should we maintain backward compatibility with text-based content?                          | Yes, both text content (IsBlocks=false) and block content (IsBlocks=true) should be properly stored                                                          |
| What JSON library should be used for serialization?                                          | Use bytedance/sonic for consistency with the rest of the codebase                                                                                             |
| Should we handle nil Content gracefully?                                                     | Yes, maintain the existing nil check and only serialize when msg.Content is not nil                                                                           |
| Do we need to modify the database schema?                                                    | No, the messageContent field already exists and stores string data - we just need to populate it correctly                                                    |
