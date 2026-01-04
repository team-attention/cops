# Review Result

**Status**: Changes Required

## Request Summary

Code review identified TypeScript build errors that need to be addressed. The frontend build fails with 4 errors related to unused imports and non-existent properties on protobuf types. Please address the violations listed below.

## Acceptance Criteria

- [ ] Remove unused `ToolUseContentBlock` import from `message-bubble.tsx`
- [ ] Remove references to non-existent `cwd` property from `session-header.tsx` (field was removed from `SessionDetail` proto message)
- [ ] Fix `role` property access in `use-user.ts` - the `Organization` proto type does not have a `role` field; roles are stored in `OrganizationMember.members`

## Scope

### In Scope
- Fix identified TypeScript errors that cause build failure
- Ensure frontend build passes successfully

### Out of Scope
- Any other refactoring or improvements not related to fixing these build errors
- Feature additions beyond fixing the errors

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
| :--- | :--- | :--- | :---- | :------------ |
| `/Users/jayce/team-attention/cops/web/src/feature/session/component/message-bubble.tsx` | 2 | TypeScript (TS6196) | `ToolUseContentBlock` is imported but never used | Remove `ToolUseContentBlock` from the import statement on line 2 |
| `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx` | 128 | TypeScript (TS2339) | Property `cwd` does not exist on type `SessionDetail` - the `cwd` field was removed from the proto definition | Remove or comment out the `title={session.cwd}` attribute on line 128 |
| `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx` | 130 | TypeScript (TS2339) | Property `cwd` does not exist on type `SessionDetail` | Replace `{session.cwd \|\| '/'}` with a placeholder like `'/'` or display a different field from the session |
| `/Users/jayce/team-attention/cops/web/src/shared/hook/use-user.ts` | 62 | TypeScript (TS2339) | Property `role` does not exist on type `Organization` - the proto `Organization` type has `id`, `name`, `slug`, and `members` fields, but no direct `role` field | The role must be retrieved from the `OrganizationMember` within `org.members` array. Find the member matching the current user's ID and extract the role from there. |

## Additional Context

### Root Cause Analysis

**Error 1 - Unused Import (`message-bubble.tsx`)**:
The `ToolUseContentBlock` type was imported but is not used in the component code. This is a simple cleanup.

**Error 2 & 3 - Missing `cwd` Property (`session-header.tsx`)**:
Looking at the generated `SessionDetail` type in `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts`, the `cwd` (current working directory) field is not present. The proto file appears to have reserved this field (field 4) based on the comment `J4 08 04 10 05` and `// cwd field was removed`. The component references this non-existent property.

Available fields on `SessionDetail`:
- `id: string`
- `projectId: string`
- `gitBranch: string`
- `version: string`
- `usage?: TokenUsageSummary`
- `startedAt?: Timestamp`
- `endedAt?: Timestamp`
- `records: Record[]`

**Error 4 - Missing `role` Property (`use-user.ts`)**:
The `Organization` type from domain proto (`/Users/jayce/team-attention/cops/web/src/gen/grpcstub/domain/v1/domain_pb.ts`) does not have a `role` field. The organization structure is:

```typescript
type Organization = {
  id: string
  name: string
  slug: string
  members: OrganizationMember[]
}

type OrganizationMember = {
  userId: string
  role: string  // "admin" or "member"
}
```

The code at line 62 tries to access `org.role`, but the role is actually stored per-member in `org.members[].role`. To get the current user's role in an organization, you need to:
1. Get the current user's ID
2. Find the matching member in `org.members` array
3. Return that member's `role`

### Suggested Fix for use-user.ts

Instead of:
```typescript
const orgs = data.organizations.map((org) => ({
  id: org.id,
  name: org.name,
  role: org.role as 'admin' | 'member',  // ERROR: org.role doesn't exist
}))
```

Fix should be:
```typescript
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
```

## Files Reviewed

- `/Users/jayce/team-attention/cops/web/src/feature/session/component/message-bubble.tsx`
- `/Users/jayce/team-attention/cops/web/src/feature/session/component/session-header.tsx`
- `/Users/jayce/team-attention/cops/web/src/shared/hook/use-user.ts`
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts` (reference)
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/domain/v1/domain_pb.ts` (reference)

## Rules Applied

- `.agent/rules/common.md`
- `.agent/rules/react/react-web.md`
- `.agent/rules/react/react-web-src.md`

## Related Documents

- Requirements: `/Users/jayce/team-attention/cops/.agent/artifacts/20260104-201920/01_clarify.md`
- Plan: `/Users/jayce/team-attention/cops/.agent/artifacts/20260104-201920/02_plan.md`
