# Implementation Plan: Rewrite Parsing Code to Use aggregation.v1.Record

## Overview

The API returns `aggregation.v1.Record[]` via `DashboardService.GetSession`, but the UI parsing code was written expecting `daemon.v1.SessionRecord[]`. Rather than changing the API, we will rewrite the parsing code to correctly handle the `aggregation.v1.Record` type with its discriminated union structure.

**Key Structural Differences:**

| Aspect | `daemon.v1.SessionRecord` | `aggregation.v1.Record` |
|--------|---------------------------|-------------------------|
| Type enum | `SessionType` (USER, ASSISTANT, SYSTEM, SUMMARY, FILE_HISTORY_SNAPSHOT, QUEUE_OPERATION) | `RecordType` (USER, ASSISTANT, FILE_HISTORY_SNAPSHOT) |
| Data access | Flat structure (`record.uuid`, `record.message`) | Discriminated union (`record.data.case`, `record.data.value`) |
| Metadata location | Top-level fields | Nested in `data.value.metadata` |
| Usage type | `daemon.v1.Usage` | `aggregation.v1.AssistantMessageUsage` |
| Content format | JSON string in `message.content` | Structured `AssistantMessageContent[]` array |

## Package Changes

None required.

---

## Step 1: Update Type Definitions in `content-block.ts`

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts`: Reference for `AssistantMessageUsage` type

### `/Users/jayce/team-attention/cops/web/src/feature/session/type/content-block.ts`

**Description**:
Update the `Usage` import to use `AssistantMessageUsage` from aggregation module.

**Current Code (line 2)**:
```typescript
import type { Usage } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Target Code**:
```typescript
import type { AssistantMessageUsage } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Additional Change - Update `ParsedMessage` interface (line 34)**:

**Current Code**:
```typescript
  usage?: Usage
```

**Target Code**:
```typescript
  usage?: AssistantMessageUsage
```

---

## Step 2: Rewrite `parse-content.ts` to Use `aggregation.v1.Record`

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts`: Reference for `Record`, `RecordType`, `UserRecordData`, `AssistantRecordData`, `AssistantMessageContent` types
- `/Users/jayce/team-attention/cops/web/src/feature/session/type/content-block.ts`: Reference for `ParsedMessage`, `ContentBlock` types

### `/Users/jayce/team-attention/cops/web/src/feature/session/util/parse-content.ts`

**Description**:
Complete rewrite of the parsing logic to handle the discriminated union structure of `aggregation.v1.Record`.

```typescript
import { RecordType } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type {
  Record,
  UserRecordData,
  AssistantRecordData,
  AssistantMessageContent,
} from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type {
  ParsedMessage,
  ContentBlock,
  LinkedToolCall,
  ToolUseContentBlock,
  ToolResultContentBlock,
} from '../type/content-block'

// Helper to extract user message text content from UserRecordData
const extractUserMessageText = (userData: UserRecordData): string => {
  // Implementation outline:
  // 1. Check if message exists on userData
  // 2. Check message.content.case to determine content type:
  //    a. If case === 'text': return message.content.value directly
  //    b. If case === 'blocks': extract text from blocks array
  //       - Iterate through message.content.value.blocks
  //       - Concatenate text fields from blocks with type 'text'
  // 3. Return empty string if no content found
}

// Helper to convert AssistantMessageContent[] to ContentBlock[]
const convertAssistantContent = (content: AssistantMessageContent[]): ContentBlock[] => {
  // Implementation outline:
  // 1. Map over the content array
  // 2. For each AssistantMessageContent item:
  //    a. If type === 'text': return TextContentBlock { type: 'text', text: item.text }
  //    b. If type === 'thinking': return TextContentBlock { type: 'text', text: item.thinking }
  //    c. If type === 'tool_use': return ToolUseContentBlock {
  //         type: 'tool_use',
  //         id: item.toolUseId,
  //         name: item.toolUseName,
  //         input: JSON.parse(item.toolUseInputJson) or {}
  //       }
  //    d. If type === 'tool_result': return ToolResultContentBlock {
  //         type: 'tool_result',
  //         tool_use_id: item.toolUseId,
  //         content: item.text
  //       }
  // 3. Filter out any undefined/invalid blocks
  // 4. Return the resulting ContentBlock array
}

// Parse a single Record into a renderable ParsedMessage
export const parseMessageContent = (record: Record): ParsedMessage => {
  // Implementation outline:
  // 1. Handle USER record type:
  //    a. Check if record.type === RecordType.USER
  //    b. Extract userData from record.data (case === 'userData')
  //    c. Get metadata from userData.metadata
  //    d. Extract text content using extractUserMessageText()
  //    e. Return ParsedMessage with:
  //       - uuid: metadata.uuid
  //       - type: 'user'
  //       - timestamp: metadata.timestamp
  //       - isMeta: userData.isMeta
  //       - isSidechain: metadata.isSidechain
  //       - content: [{ type: 'text', text }]
  //
  // 2. Handle ASSISTANT record type:
  //    a. Check if record.type === RecordType.ASSISTANT
  //    b. Extract assistantData from record.data (case === 'assistantData')
  //    c. Get metadata from assistantData.metadata
  //    d. Convert message content using convertAssistantContent()
  //    e. Return ParsedMessage with:
  //       - uuid: metadata.uuid
  //       - type: 'assistant'
  //       - timestamp: metadata.timestamp
  //       - isMeta: false (assistant records don't have isMeta)
  //       - isSidechain: metadata.isSidechain
  //       - usage: assistantData.message?.usage
  //       - content: converted content blocks
  //
  // 3. Handle FILE_HISTORY_SNAPSHOT record type:
  //    a. Check if record.type === RecordType.FILE_HISTORY_SNAPSHOT
  //    b. Return ParsedMessage with type: 'system' and empty content
  //       (these are filtered out in filterRecordsForChat)
  //
  // 4. Fallback for UNSPECIFIED or unknown types:
  //    a. Return ParsedMessage with type: 'system' and empty content
}

// Extract and link tool calls from records
export const extractToolCalls = (records: Record[]): LinkedToolCall[] => {
  // Implementation outline:
  // 1. Create Map to store tool_use blocks with metadata
  // 2. Create Map to store tool_result blocks by tool_use_id
  //
  // 3. First pass - collect tool_use blocks from assistant records:
  //    a. Filter records where type === RecordType.ASSISTANT
  //    b. For each assistant record:
  //       - Extract assistantData from record.data
  //       - Get message.content array
  //       - For each content block with type === 'tool_use':
  //         * Store in toolUseMap with key = toolUseId
  //         * Value = { block: ToolUseContentBlock, sourceUuid: metadata.uuid, timestamp }
  //
  // 4. Second pass - collect tool_result blocks:
  //    a. Filter records where type === RecordType.ASSISTANT
  //    b. For each assistant record:
  //       - For each content block with type === 'tool_result':
  //         * Store in toolResults map with key = toolUseId
  //
  // 5. Link tool uses with results:
  //    a. Map over toolUseMap entries
  //    b. For each entry, create LinkedToolCall with:
  //       - toolUse: the stored ToolUseContentBlock
  //       - toolResult: matching result from toolResults map (if exists)
  //       - sourceMessageUuid: stored sourceUuid
  //       - timestamp: stored timestamp
  //
  // 6. Return array of LinkedToolCall objects
}

// Filter records for chat view display
export const filterRecordsForChat = (records: Record[]): Record[] => {
  // Implementation outline:
  // 1. Filter out FILE_HISTORY_SNAPSHOT records:
  //    a. Return records.filter(record => record.type !== RecordType.FILE_HISTORY_SNAPSHOT)
  //
  // Note: RecordType only has USER, ASSISTANT, FILE_HISTORY_SNAPSHOT
  // There is no SYSTEM, SUMMARY, or QUEUE_OPERATION in aggregation schema
  // So we only need to filter out FILE_HISTORY_SNAPSHOT
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Parse USER record with text content | Record with type=USER, data.case='userData', message.content.case='text' | ParsedMessage with type='user', content=[{type:'text', text:'...'}] | User text branch |
| Parse USER record with blocks content | Record with type=USER, data.case='userData', message.content.case='blocks' | ParsedMessage with type='user', concatenated text from blocks | User blocks branch |
| Parse ASSISTANT record | Record with type=ASSISTANT, data.case='assistantData' | ParsedMessage with type='assistant', converted content blocks, usage | Assistant branch |
| Parse ASSISTANT with tool_use | Record with type=ASSISTANT, content has type='tool_use' | ParsedMessage with ToolUseContentBlock in content | Tool use conversion |
| Parse FILE_HISTORY_SNAPSHOT | Record with type=FILE_HISTORY_SNAPSHOT | ParsedMessage with type='system', empty content | Snapshot branch |
| Parse UNSPECIFIED type | Record with type=UNSPECIFIED | ParsedMessage with type='system', empty content | Fallback branch |
| Filter records | Array with USER, ASSISTANT, FILE_HISTORY_SNAPSHOT | Array with only USER and ASSISTANT | Filter logic |
| Extract tool calls | Records with tool_use and tool_result | LinkedToolCall[] with matched pairs | Tool extraction |
| Extract tool calls - unmatched | Records with tool_use but no tool_result | LinkedToolCall[] with toolResult=undefined | Unmatched tool |

---

## Step 3: Update `chat-view.tsx` to Use `Record` Type

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts`: Reference for `Record` type

### `/Users/jayce/team-attention/cops/web/src/feature/session/component/chat-view.tsx`

**Description**:
Update import and type annotation to use `Record` instead of `SessionRecord`.

**Current Code (line 6)**:
```typescript
import type { SessionRecord } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Target Code**:
```typescript
import type { Record } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Additional Change - Update `ChatViewProps` interface (line 11)**:

**Current Code**:
```typescript
  records: SessionRecord[]
```

**Target Code**:
```typescript
  records: Record[]
```

---

## Summary of Changes

| File | Change Type | Description |
|------|-------------|-------------|
| `content-block.ts` | Import change | `Usage` -> `AssistantMessageUsage` |
| `content-block.ts` | Type change | Update `ParsedMessage.usage` type |
| `parse-content.ts` | Complete rewrite | Rewrite all functions to use `Record` and `RecordType` with discriminated union handling |
| `chat-view.tsx` | Import change | `SessionRecord` -> `Record` |
| `chat-view.tsx` | Type change | Update `ChatViewProps.records` type |

---

## Type Mapping Reference

### RecordType to ParsedMessage.type Mapping

| RecordType | ParsedMessage.type |
|------------|-------------------|
| `USER` | `'user'` |
| `ASSISTANT` | `'assistant'` |
| `FILE_HISTORY_SNAPSHOT` | `'system'` (filtered out) |
| `UNSPECIFIED` | `'system'` (fallback) |

### Data Field Mapping

| ParsedMessage Field | USER Source | ASSISTANT Source |
|---------------------|-------------|------------------|
| `uuid` | `record.data.value.metadata.uuid` | `record.data.value.metadata.uuid` |
| `timestamp` | `record.data.value.metadata.timestamp` | `record.data.value.metadata.timestamp` |
| `isMeta` | `record.data.value.isMeta` | `false` |
| `isSidechain` | `record.data.value.metadata.isSidechain` | `record.data.value.metadata.isSidechain` |
| `usage` | N/A | `record.data.value.message.usage` |
| `content` | Extract from `message.content` | Convert from `message.content[]` |

### AssistantMessageContent.type to ContentBlock Mapping

| AssistantMessageContent.type | ContentBlock Type | Field Mapping |
|-----------------------------|-------------------|---------------|
| `'text'` | `TextContentBlock` | `text: item.text` |
| `'thinking'` | `TextContentBlock` | `text: item.thinking` |
| `'tool_use'` | `ToolUseContentBlock` | `id: item.toolUseId`, `name: item.toolUseName`, `input: JSON.parse(item.toolUseInputJson)` |
| `'tool_result'` | `ToolResultContentBlock` | `tool_use_id: item.toolUseId`, `content: item.text` |

---

## Verification Steps

After implementing the changes:

1. **TypeScript Compilation**: Run `npm run build` or `tsc --noEmit` to verify no type errors
2. **Development Server**: Start the dev server with `npm run dev`
3. **Manual Test**: Navigate to `http://localhost:3000/sessions/{id}` and verify:
   - Session detail page loads without import errors
   - User messages display correctly
   - Assistant messages display correctly with text content
   - Tool use blocks render properly
   - Token usage statistics display (if visible in UI)
4. **Console Check**: Verify no runtime errors in browser console
