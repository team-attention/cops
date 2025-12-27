# Requirements

## Request Summary
Fix JSONL message content parsing and storage issue where `messageContent` is being stored as empty string (`""`) in MongoDB. The current issue is that despite having parsing logic, the actual message content is not being persisted to the database. This needs to be fixed and verified with integration tests using real JSONL data.

## Acceptance Criteria

- [ ] Criterion 1: All ContentBlock types are correctly parsed from JSONL logs (text, tool_use, tool_result)
- [ ] Criterion 2: ToolUseContentBlock correctly extracts id, name, and input fields from tool_use entries
- [ ] Criterion 3: ToolResultContentBlock correctly extracts tool_use_id, content, and is_error fields from tool_result entries
- [ ] Criterion 4: TextContentBlock correctly extracts text field from text entries
- [ ] Criterion 5: String content (user messages) is correctly parsed and stored in MessageContent.Text field
- [ ] Criterion 6: Array content (assistant messages) is correctly parsed into MessageContent.Blocks slice
- [ ] Criterion 7: Unknown/future content block types are gracefully skipped without causing parsing failures
- [ ] Criterion 8: Daemon successfully processes JSONL files containing mixed content types without errors
- [ ] Criterion 9: Parsed data is correctly serialized to JSON and stored in MongoDB (NOT empty string)
- [ ] Criterion 10: Integration tests using real JSONL data verify end-to-end parsing and storage
- [ ] Criterion 11: Unit tests exist for all content type parsing scenarios

## Scope

### In Scope
- Investigate why messageContent is stored as empty string in MongoDB despite parsing logic existing
- Create integration tests using real JSONL data from `~/.claude/**/*.jsonl`
- Verification that MessageContent.UnmarshalJSON correctly handles all three ContentBlock types
- Verification that ToolUseContentBlock.Input (map[string]any) correctly deserializes complex nested JSON structures
- Verification that MarshalJSON correctly serializes parsed content back to original format
- Fix the storage issue in daemon → API → MongoDB data flow
- Adding/fixing unit tests for all ContentBlock types
- Verification that sonic.Unmarshal/Marshal works correctly with the custom unmarshaler
- Ensuring forward compatibility with unknown content block types

### Out of Scope
- Changes to the JSONL log format itself (read-only processing)
- Support for content types beyond text, tool_use, and tool_result (unless discovered in actual logs)
- Migration of existing database records
- Performance optimization beyond ensuring correct functionality
- Changes to the API server collector endpoint (focus on daemon parsing only)

## Constraints
- Must maintain backward compatibility with existing JSONL log format
- Must use bytedance/sonic for JSON parsing (already in use by daemon)
- Must follow Go struct pointer/value type rules from `.agent/rules/go/go-struct.md`
- Must maintain hexagonal architecture pattern in daemon service
- Cannot modify generated protobuf code
- Must handle large JSONL files efficiently (already using bufio.Scanner with 1MB buffer)

## Additional Context
- Current implementation already has ContentBlock types defined in `/Users/jayce/team-attention/cops/shared/domain/content_block.go`
- Custom UnmarshalJSON exists in `/Users/jayce/team-attention/cops/shared/domain/message.go` (lines 16-73)
- Daemon parses JSONL in `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` (line 134)
- Example JSONL shows tool_use with structure: `{"type":"tool_use","id":"toolu_...","name":"Task","input":{...}}`
- Example JSONL shows tool_result with structure: `{"type":"tool_result","content":"...","is_error":true,"tool_use_id":"toolu_..."}`
- MongoDB storage uses messageContent field (from `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`)
- The system must handle ~80+ tool_result entries in a typical session log file

## Questions Resolved

| Question | Answer |
|----------|--------|
| What specific error is occurring when parsing tool_use content? | User reported parsing failures but did not provide specific error messages. Need to investigate actual error during implementation. |
| Are all three ContentBlock types (text, tool_use, tool_result) required? | Yes - all three types appear in actual Claude Code JSONL logs based on the example data examined. |
| Should the ToolUseContentBlock.Input field preserve exact JSON structure? | Yes - it's defined as `map[string]any` to handle arbitrary nested structures that tools accept as input. |
| How should unknown content block types be handled? | Gracefully skip them (current code has `default: continue` case in UnmarshalJSON switch statement). |
| Is there a specific JSONL file causing the issue? | User mentioned "certain MessageContent types" but didn't specify a particular file. Implementation should test with real logs from `~/.claude/**/*.jsonl`. |
| Should we validate the content structure beyond parsing? | No - just parse and store. Validation happens at API/application layer if needed. |
| What happens if tool_use input contains invalid JSON? | The parsing should fail for that specific block, but current implementation continues to next block rather than failing entire message. |
