# Requirements

## Request Summary

Fix the import error in the web frontend where `parse-content.ts` is trying to import `SessionType` from the wrong protobuf module. The file currently imports from `aggregation/v1/aggregation_pb.ts` which only exports `RecordType`, but needs to import from `daemon/v1/daemon_pb.ts` which exports `SessionType` with values like `USER`, `ASSISTANT`, `SYSTEM`, `SUMMARY`, `FILE_HISTORY_SNAPSHOT`, and `QUEUE_OPERATION`.

## Acceptance Criteria

- [ ] `parse-content.ts` imports `SessionType` from the correct module (`daemon/v1/daemon_pb.ts`)
- [ ] `parse-content.ts` imports `SessionRecord` from the correct module (`daemon/v1/daemon_pb.ts`)
- [ ] All references to `SessionType` enum values work correctly (USER, ASSISTANT, SYSTEM, SUMMARY, FILE_HISTORY_SNAPSHOT, QUEUE_OPERATION)
- [ ] The session detail page at `http://localhost:3000/sessions/{id}` loads without import errors
- [ ] No other files importing from `aggregation/v1/aggregation_pb.ts` are affected

## Scope

### In Scope
- Update import statements in `/Users/jayce/team-attention/cops/web/src/feature/session/util/parse-content.ts`
- Verify that `SessionType` enum values used in the code match those exported from `daemon/v1/daemon_pb.ts`
- Test that the session detail page loads correctly

### Out of Scope
- Modifying generated protobuf TypeScript files
- Changing the protobuf schema definitions
- Refactoring other parts of the session feature
- Updating other files that may use `SessionRecord` correctly

## Constraints

- Files under `web/src/gen/` are auto-generated and must NOT be edited manually
- The fix must only involve changing import statements, not the protobuf generation process
- Must maintain compatibility with existing code that uses `SessionRecord` type

## Additional Context

**Current Error:**
```
The requested module '/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts' does not provide an export named 'SessionType'
```

**Root Cause:**
The file `/Users/jayce/team-attention/cops/web/src/feature/session/util/parse-content.ts` has incorrect imports:
```typescript
import { SessionType } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
import type { SessionRecord } from '@/gen/grpcstub/aggregation/v1/aggregation_pb'
```

The `aggregation/v1/aggregation_pb.ts` module exports:
- `RecordType` enum (UNSPECIFIED, USER, ASSISTANT, FILE_HISTORY_SNAPSHOT)
- `Record` type (not `SessionRecord`)

The `daemon/v1/daemon_pb.ts` module exports:
- `SessionType` enum (UNSPECIFIED, USER, ASSISTANT, SYSTEM, SUMMARY, FILE_HISTORY_SNAPSHOT, QUEUE_OPERATION)
- `SessionRecord` type

**Correct Import Location:**
Both `SessionType` and `SessionRecord` should be imported from `@/gen/grpcstub/daemon/v1/daemon_pb`

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Are there other files importing from the wrong module? | Yes, but based on grep results, only `parse-content.ts` imports `SessionType` from aggregation. Other files (`chat-view.tsx`) import only `SessionRecord` from aggregation, which also needs fixing. The file `content-block.ts` imports `Usage` from aggregation which may be correct. |
| Should we also update `chat-view.tsx`? | Yes, it imports `SessionRecord` from `aggregation/v1/aggregation_pb` which should be from `daemon/v1/daemon_pb` |
| Is this a simple find-replace fix? | Yes, it's an import path correction from `@/gen/grpcstub/aggregation/v1/aggregation_pb` to `@/gen/grpcstub/daemon/v1/daemon_pb` for `SessionType` and `SessionRecord` |
