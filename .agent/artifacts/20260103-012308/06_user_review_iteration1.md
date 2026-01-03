# Review Result

**Status**: Changes Required

## Request Summary

User feedback identified implementation issues that deviate from project architecture rules and planned implementation steps. The feedback highlights two main issues:

1. **Architecture Violation**: The `api/internal/service/auth/devicecode/` package violates the hexagonal architecture pattern. According to `.agent/rules/go/go-service.md`, services should follow the structure `{domain}_service.go` with `inbound/` and `outbound/` subdirectories only. Utility functions specific to a service should be private helper functions within the service file, not a separate package.

2. **Missing Web Implementation**: Steps 11-16 from the implementation plan were not implemented. The Web authentication flow is a critical acceptance criteria that enables users to approve device codes through the C-Ops web application.

## Acceptance Criteria

- [ ] Remove `api/internal/service/auth/devicecode/` package and move functions as private helpers in `api/internal/service/auth/auth_service.go`
- [ ] Create `web/src/feature/auth/` directory with device approval components
- [ ] Create `web/src/feature/auth/type/device-code.ts` with type definitions
- [ ] Create `web/src/feature/auth/hook/use-approve-device.ts` with mutation hook
- [ ] Create `web/src/feature/auth/component/device-approval.tsx` with approval UI
- [ ] Create `web/src/route/auth/device.tsx` route for `/auth/device` page
- [ ] Update `web/src/shared/service/connect-transport.ts` with authentication interceptor

## Scope

### In Scope
- Refactor devicecode package into private helper functions
- Implement all Web authentication components (Steps 11-16 from plan)
- Ensure transport includes authentication headers

### Out of Scope
- Any API changes (backend implementation is complete)
- CLI changes (already implemented correctly)
- Any functionality beyond what was specified in Steps 11-16

## Violations Found

| File | Line | Rule | Issue | Suggested Fix |
|------|------|------|-------|---------------|
| `api/internal/service/auth/devicecode/devicecode.go` | 1 | `.agent/rules/go/go-service.md` | Separate package for service-specific utility functions violates hexagonal architecture | Move `GenerateDeviceCodeID`, `GenerateUserCode`, and `NormalizeUserCode` as unexported (private) functions in `auth_service.go` |
| `web/src/feature/auth/` | N/A | `02_plan.md` Step 11 | Directory does not exist | Create directory structure: `web/src/feature/auth/{component,hook,type}/` |
| `web/src/feature/auth/type/device-code.ts` | N/A | `02_plan.md` Step 12 | File does not exist | Create type definitions for `DeviceApprovalState` discriminated union |
| `web/src/feature/auth/hook/use-approve-device.ts` | N/A | `02_plan.md` Step 13 | File does not exist | Create hook using `useMutation` from `@connectrpc/connect-query` |
| `web/src/feature/auth/component/device-approval.tsx` | N/A | `02_plan.md` Step 14 | File does not exist | Create component with approval UI, loading states, and error handling |
| `web/src/shared/service/connect-transport.ts` | 1-5 | `02_plan.md` Step 15 | Missing authentication interceptor | Add interceptor to include `Authorization: Bearer {token}` header from localStorage |
| `web/src/route/auth/device.tsx` | N/A | `02_plan.md` Step 16 | File does not exist | Create route component for `/auth/device` with authentication check and redirect logic |

## Detailed Violation Analysis

### 1. Devicecode Package Architecture Violation

**Current Implementation:**
```
api/internal/service/auth/
├── auth_service.go
├── devicecode/              <- WRONG: Should not exist
│   └── devicecode.go
├── inbound/
└── outbound/
```

**Expected Structure per `.agent/rules/go/go-service.md`:**
```
{domain}/
├── {domain}_service.go         # Core service implementation (includes private helpers)
├── inbound/                    # Inbound adapters only
└── outbound/                   # Outbound adapters only
```

**Specific Code to Move:**

The following functions from `/Users/jayce/team-attention/cops/api/internal/service/auth/devicecode/devicecode.go` should be moved to `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go` as unexported (private) functions:

- `GenerateDeviceCodeID()` -> `generateDeviceCodeID()`
- `GenerateUserCode()` -> `generateUserCode()`
- `NormalizeUserCode()` -> `normalizeUserCode()`

After moving, delete the entire `api/internal/service/auth/devicecode/` directory.

### 2. Missing Web Implementation (Steps 11-16)

**Step 11 - Directory Structure:**
Missing: `web/src/feature/auth/` with subdirectories `component/`, `hook/`, `type/`

**Step 12 - Type Definitions:**
Missing: `web/src/feature/auth/type/device-code.ts`

Should contain discriminated union types per `.agent/rules/react/react-web.md`:
```typescript
interface DeviceApprovalPending {
  status: 'pending';
}

interface DeviceApprovalSuccess {
  status: 'success';
  message: string;
}

interface DeviceApprovalError {
  status: 'error';
  errorCode: DeviceApprovalErrorCode;
  message: string;
}

export type DeviceApprovalState =
  | DeviceApprovalPending
  | DeviceApprovalSuccess
  | DeviceApprovalError;
```

**Step 13 - Approval Hook:**
Missing: `web/src/feature/auth/hook/use-approve-device.ts`

Should wrap the `deviceCodeApprove` mutation from generated stubs.

**Step 14 - Device Approval Component:**
Missing: `web/src/feature/auth/component/device-approval.tsx`

Should display user code, approve button, loading state, and success/error messages.

**Step 15 - Transport Authentication:**
Current `/Users/jayce/team-attention/cops/web/src/shared/service/connect-transport.ts`:
```typescript
import { createConnectTransport } from '@connectrpc/connect-web'

export const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})
```

Missing: Authentication interceptor to include JWT token in requests.

**Step 16 - Device Route:**
Missing: `web/src/route/auth/device.tsx`

Should handle `/auth/device?code=XXX` route with:
- Authentication check
- Redirect to login if not authenticated
- Device approval component rendering

## Additional Context

- Requirements document: `/Users/jayce/team-attention/cops/.agent/artifacts/20260103-012308/01_requirements.md`
- Plan document: `/Users/jayce/team-attention/cops/.agent/artifacts/20260103-012308/02_plan.md` (see Steps 11-16 for Web implementation details)
- User feedback provided specific issues that align with the planned but unimplemented steps

## Rules References

The following rules were applied during this review:
- [`.agent/rules/common.md`](/Users/jayce/team-attention/cops/.agent/rules/common.md) - General coding rules
- [`.agent/rules/go/go-service.md`](/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md) - Service package structure rules
- [`.agent/rules/go/go-hexagonal-layout.md`](/Users/jayce/team-attention/cops/.agent/rules/go/go-hexagonal-layout.md) - Hexagonal architecture rules
- [`.agent/rules/react/react-web.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md) - React/TypeScript rules
- [`.agent/rules/react/react-web-src.md`](/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md) - Web source directory structure rules
