# Implementation Plan: Fix Import Errors for SessionType and SessionRecord

## Overview

The web frontend has incorrect imports in the session feature. Files are importing `SessionType` and `SessionRecord` from `@/gen/grpcstub/aggregation/v1/aggregation_pb`, but these types are only exported from `@/gen/grpcstub/daemon/v1/daemon_pb`. The `aggregation_pb` module exports `RecordType` and `Record` (different types with different structures), not `SessionType` and `SessionRecord`.

This is a straightforward import path correction that requires no logic changes - only the import source paths need to be updated.

## Package Changes

None required. This fix only involves correcting import paths within existing generated protobuf modules.

---

## Step 1: Update Imports in `parse-content.ts`

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/daemon/v1/daemon_pb.ts`: Verify exports of `SessionType` and `SessionRecord`

### `/Users/jayce/team-attention/cops/web/src/feature/session/util/parse-content.ts`

**Description**:
Change import source from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` for both `SessionType` enum and `SessionRecord` type.

**Current Code (lines 1-2)**:
```typescript
import { SessionType } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type { SessionRecord } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Target Code**:
```typescript
import { SessionType } from '@/gen/grpcstub/daemon/v1/daemon_pb'
import type { SessionRecord } from '@/gen/grpcstub/daemon/v1/daemon_pb'
```

**Verification**:
The `daemon_pb.ts` exports:
- `SessionType` enum with values: `UNSPECIFIED`, `USER`, `ASSISTANT`, `SYSTEM`, `SUMMARY`, `FILE_HISTORY_SNAPSHOT`, `QUEUE_OPERATION`
- `SessionRecord` type with fields: `uuid`, `parentUuid`, `sessionId`, `type` (SessionType), `timestamp`, `cwd`, `gitBranch`, `version`, `userType`, `isSidechain`, `isMeta`, `slug`, `requestId`, `message`

All `SessionType` enum values used in `parse-content.ts` are valid:
- `SessionType.USER` (line 32)
- `SessionType.ASSISTANT` (lines 44, 85, 105)
- `SessionType.SUMMARY` (line 132)
- `SessionType.FILE_HISTORY_SNAPSHOT` (line 132)
- `SessionType.QUEUE_OPERATION` (line 136)

---

## Step 2: Update Imports in `chat-view.tsx`

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/daemon/v1/daemon_pb.ts`: Verify `SessionRecord` export

### `/Users/jayce/team-attention/cops/web/src/feature/session/component/chat-view.tsx`

**Description**:
Change import source from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` for `SessionRecord` type.

**Current Code (line 6)**:
```typescript
import type { SessionRecord } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Target Code**:
```typescript
import type { SessionRecord } from '@/gen/grpcstub/daemon/v1/daemon_pb'
```

**Verification**:
The `ChatViewProps` interface uses `SessionRecord[]` for the `records` prop. The component passes records to `filterRecordsForChat()` and `parseMessageContent()` from `parse-content.ts`, which will now use the correct `SessionRecord` type from `daemon_pb`.

---

## Step 3: Update Imports in `content-block.ts` (Usage Type)

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/daemon/v1/daemon_pb.ts`: Verify `Usage` export

### `/Users/jayce/team-attention/cops/web/src/feature/session/type/content-block.ts`

**Description**:
Change import source from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` for `Usage` type. The `aggregation_pb` module does NOT export a `Usage` type - it only exports `AssistantMessageUsage`. The `Usage` type is only available from `daemon_pb`.

**Current Code (line 2)**:
```typescript
import type { Usage } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

**Target Code**:
```typescript
import type { Usage } from '@/gen/grpcstub/daemon/v1/daemon_pb'
```

**Verification**:
The `daemon_pb.ts` exports `Usage` type with fields: `inputTokens`, `outputTokens`, `cacheCreationInputTokens`, `cacheReadInputTokens`, `serviceTier`.

The `ParsedMessage` interface uses `usage?: Usage` which is compatible with the `daemon.v1.Usage` type from `daemon_pb`.

---

## Summary of Changes

| File | Line(s) | Change |
|------|---------|--------|
| `parse-content.ts` | 1 | Change import path from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` |
| `parse-content.ts` | 2 | Change import path from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` |
| `chat-view.tsx` | 6 | Change import path from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` |
| `content-block.ts` | 2 | Change import path from `aggregation/v1/aggregation_pb` to `daemon/v1/daemon_pb` |

---

## Verification Steps

After implementing the changes:

1. **TypeScript Compilation**: Run `npm run build` or `tsc --noEmit` to verify no type errors
2. **Development Server**: Start the dev server with `npm run dev`
3. **Manual Test**: Navigate to `http://localhost:3000/sessions/{id}` and verify the session detail page loads without import errors
4. **Console Check**: Verify no runtime errors in browser console related to missing exports
