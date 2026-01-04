# Implementation Plan: Fix TypeScript Build Errors

## Overview

This plan addresses 4 TypeScript build errors in the frontend codebase:
1. Unused import `ToolUseContentBlock` in message-bubble.tsx
2. Non-existent `cwd` property access in session-header.tsx (lines 128, 130)
3. Non-existent `role` property access on `Organization` type in use-user.ts (line 62)

These errors prevent the frontend build from succeeding and must be fixed without changing any functionality beyond removing the errors.

## Package Changes

No package changes required.

## Implementation Steps

### Step 1: Remove Unused Import from message-bubble.tsx

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/session/component/message-bubble.tsx`: The file containing the unused import

#### `/Users/jayce/team-attention/cops/web/src/feature/session/component/message-bubble.tsx`

**Description**:
Remove the unused `ToolUseContentBlock` type from the import statement on line 2. The type is imported but never used in the component code.

**Change**:

```tsx
// BEFORE (line 2):
import type { ParsedMessage, ToolUseContentBlock } from '../type/content-block'

// AFTER (line 2):
import type { ParsedMessage } from '../type/content-block'
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Build without error | Run `npm run build` | Build succeeds without TS6196 error | Happy path |
| Component still renders | View session with messages | MessageBubble components render correctly | Functionality preserved |

---

### Step 2: Remove Non-existent `cwd` Property References from session-header.tsx

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx`: The file accessing non-existent `cwd` property
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts`: Generated protobuf types to confirm available fields on `SessionDetail`

#### `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx`

**Description**:
Remove references to `session.cwd` on lines 128 and 130. The `cwd` field was removed from the `SessionDetail` protobuf message (reserved field 4). Since there's no replacement field available in the proto, we'll remove the entire "working directory" display section (lines 124-132).

**Changes**:

```tsx
// REMOVE LINES 124-132:
// Delete this entire block:
              <div className="flex items-center gap-2 font-mono text-xs text-zinc-500">
                <Terminal className="h-3 w-3" />
                <span
                  className="max-w-xs truncate lg:max-w-md"
                  title={session.cwd}
                >
                  {session.cwd || '/'}
                </span>
              </div>
```

**Justification**:
- The `cwd` field no longer exists on `SessionDetail` (it was reserved/removed from proto)
- The proto has: `id`, `projectId`, `gitBranch`, `version`, `usage`, `startedAt`, `endedAt`, `records`
- No suitable replacement field exists for displaying working directory
- Removing the entire display element is the cleanest solution

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Build without error | Run `npm run build` | Build succeeds without TS2339 errors on lines 128, 130 | Happy path |
| Header still renders | View session detail page | SessionHeader displays all other information correctly | Functionality preserved |
| UI layout not broken | View session detail page | No visual gaps or layout issues from removed element | UI integrity |

---

### Step 3: Fix `role` Property Access in use-user.ts

**Files to Read**:
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-user.ts`: The file incorrectly accessing `org.role`
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/domain/v1/domain_pb.ts`: Generated protobuf types showing `Organization` structure

#### `/Users/jayce/team-attention/cops/web/src/shared/hook/use-user.ts`

**Description**:
Fix the mapping of protobuf `Organization` to `OrganizationData` in the `useEffect` hook. The `Organization` proto type does not have a direct `role` field. Instead, roles are stored per-member in `org.members[]` where each member has `userId` and `role`. We need to find the current user's membership in each organization and extract the role from there.

**Changes**:

```tsx
// BEFORE (lines 58-64):
      // Map protobuf Organizations to OrganizationData[]
      const orgs = data.organizations.map((org) => ({
        id: org.id,
        name: org.name,
        role: org.role as 'admin' | 'member',
      }))
      setOrganizations(orgs)

// AFTER (lines 58-66):
      // Map protobuf Organizations to OrganizationData[]
      const userId = data.user?.id
      const orgs = data.organizations.map((org) => {
        // Find the current user's membership in this organization
        const membership = org.members.find((m) => m.userId === userId)
        return {
          id: org.id,
          name: org.name,
          role: (membership?.role || 'member') as 'admin' | 'member',
        }
      })
      setOrganizations(orgs)
```

**Implementation Details**:
1. Extract the current user's ID from `data.user?.id` before mapping organizations
2. For each organization, find the member whose `userId` matches the current user
3. Extract the `role` from that membership
4. Default to `'member'` if no matching membership is found (defensive programming)
5. Cast to the expected union type `'admin' | 'member'`

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Build without error | Run `npm run build` | Build succeeds without TS2339 error on line 62 | Happy path |
| User is admin | User with admin role in org | `organizations` array contains org with `role: 'admin'` | Admin role path |
| User is member | User with member role in org | `organizations` array contains org with `role: 'member'` | Member role path |
| User not in members | Organization with no matching member | `organizations` array contains org with `role: 'member'` (fallback) | Defensive fallback |
| Multiple organizations | User belongs to 3+ orgs | All organizations mapped with correct roles | Multiple orgs path |

---

## Quality Checklist

- [x] All changes preserve existing functionality
- [x] No new dependencies added
- [x] All TypeScript errors addressed:
  - [x] TS6196: Unused import in message-bubble.tsx
  - [x] TS2339: Non-existent `cwd` property in session-header.tsx (line 128)
  - [x] TS2339: Non-existent `cwd` property in session-header.tsx (line 130)
  - [x] TS2339: Non-existent `role` property in use-user.ts (line 62)
- [x] Changes follow existing code patterns
- [x] No functional changes beyond error fixes
- [x] Test scenarios cover all modified code paths
- [x] Defensive programming applied (fallback role in use-user.ts)
