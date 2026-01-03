# Implementation Plan: User Feedback Iteration 1

## Overview

This plan addresses two critical issues identified in user review:

1. **Architecture Violation**: The `api/internal/service/auth/devicecode/` package violates hexagonal architecture rules. According to `.agent/rules/go/go-service.md`, service directories should only contain `{domain}_service.go`, `inbound/`, and `outbound/` subdirectories. Service-specific utility functions must be private helper functions within the service file, not a separate package.

2. **Missing Web Implementation**: The Web authentication flow (Steps 11-16 from the original plan) was not implemented. Users cannot approve device codes through the C-Ops web application, which is a critical acceptance criteria.

This plan will refactor the backend architecture to comply with project rules and implement all missing Web components.

---

## Step 1: Move devicecode Helpers to Private Functions

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service structure rules
- `.agent/rules/go/go-backend.md`: Function and parameter rules
- `api/internal/service/auth/devicecode/devicecode.go`: Functions to move
- `api/internal/service/auth/auth_service.go`: Target file

### Update `api/internal/service/auth/auth_service.go`

**Description**: Move the three helper functions from the devicecode package to private (unexported) functions at the bottom of the auth service file. Update all function calls to use the new private functions.

```go
// ADD at the bottom of the file (after all public methods):

// userCodeChars contains characters for human-friendly codes.
// Excludes ambiguous characters: 0, O, I, 1, L
const userCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateDeviceCodeID generates a cryptographically secure device code ID.
// Returns a 32-character hex string (16 bytes of randomness).
func generateDeviceCodeID() (string, error) {
	// Implementation outline:
	// 1. Create byte slice of length 16.
	// 2. Fill with crypto/rand.Read.
	// 3. If error, return error.
	// 4. Return hex.EncodeToString(bytes).
}

// generateUserCode generates a human-friendly 8-character code with hyphen.
// Format: XXXX-XXXX (e.g., "ABCD-EFGH")
func generateUserCode() (string, error) {
	// Implementation outline:
	// 1. Create byte slice of length 8.
	// 2. Fill with crypto/rand.Read.
	// 3. If error, return error.
	// 4. Create strings.Builder with capacity 9 (8 chars + 1 hyphen).
	// 5. Iterate through bytes:
	//    a. If index is 4, write hyphen character.
	//    b. Map byte to character from userCodeChars using modulo.
	//    c. Write character to builder.
	// 6. Return builder.String().
}

// normalizeUserCode normalizes user code input by removing hyphens and converting to uppercase.
func normalizeUserCode(code string) string {
	// Implementation outline:
	// 1. Convert code to uppercase using strings.ToUpper.
	// 2. Remove all hyphens using strings.ReplaceAll.
	// 3. Return normalized string.
}
```

**Update Import Section**:

```go
// REMOVE this import:
// "github.com/team-attention/cops/api/internal/service/auth/devicecode"

// ADD these imports if not already present:
import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)
```

**Update Method Calls**:

```go
// IN DeviceCode method, CHANGE:
// userCode, err := devicecode.GenerateUserCode()
// TO:
userCode, err := generateUserCode()

// CHANGE:
// UserCode: devicecode.NormalizeUserCode(userCode),
// TO:
UserCode: normalizeUserCode(userCode),

// IN DeviceCodeApprove method, CHANGE:
// normalizedCode := devicecode.NormalizeUserCode(params.UserCode)
// TO:
normalizedCode := normalizeUserCode(params.UserCode)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| generateUserCode success | - | 9-char string (XXXX-XXXX format) | Happy path |
| generateUserCode random bytes | - | Different codes on each call | Randomness |
| normalizeUserCode lowercase | "abcd-efgh" | "ABCDEFGH" | Uppercase conversion |
| normalizeUserCode with hyphen | "ABCD-EFGH" | "ABCDEFGH" | Hyphen removal |
| normalizeUserCode without hyphen | "ABCDEFGH" | "ABCDEFGH" | No-op |

---

## Step 2: Delete devicecode Package

**Files to Delete**:
- `api/internal/service/auth/devicecode/devicecode.go`
- `api/internal/service/auth/devicecode/` (entire directory)

**Description**: After moving all functions to auth_service.go, delete the entire devicecode package directory.

Use bash command:
```bash
rm -rf /Users/jayce/team-attention/cops/api/internal/service/auth/devicecode
```

---

## Step 3: Create Web Auth Feature Directory Structure

**Files to Read**:
- `.agent/rules/react/react-web-src.md`: Web source directory structure rules

### Create Directory Structure

**Description**: Create the auth feature directory structure following FDD architecture.

Create directories:
```
web/src/feature/auth/
├── component/
├── hook/
└── type/
```

Use bash commands:
```bash
mkdir -p /Users/jayce/team-attention/cops/web/src/feature/auth/component
mkdir -p /Users/jayce/team-attention/cops/web/src/feature/auth/hook
mkdir -p /Users/jayce/team-attention/cops/web/src/feature/auth/type
```

---

## Step 4: Implement Device Code Type Definitions

**Files to Read**:
- `.agent/rules/react/react-web.md`: TypeScript type rules (discriminated unions)

### `web/src/feature/auth/type/device-code.ts`

**Description**: Define discriminated union types for device approval states. This follows the rule of using specific discriminated unions instead of optional properties.

```typescript
// DeviceApprovalPending represents the initial state before approval attempt.
interface DeviceApprovalPending {
  status: 'pending';
}

// DeviceApprovalSuccess represents successful device approval.
interface DeviceApprovalSuccess {
  status: 'success';
  message: string;
}

// DeviceApprovalError represents an error during device approval.
interface DeviceApprovalError {
  status: 'error';
  errorCode: DeviceApprovalErrorCode;
  message: string;
}

// DeviceApprovalErrorCode enumerates possible error conditions.
type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNAUTHORIZED'
  | 'UNKNOWN';

// DeviceApprovalState is a discriminated union representing all possible states.
export type DeviceApprovalState =
  | DeviceApprovalPending
  | DeviceApprovalSuccess
  | DeviceApprovalError;
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Type check pending | `{ status: 'pending' }` | Valid DeviceApprovalPending | Pending discriminator |
| Type check success | `{ status: 'success', message: 'OK' }` | Valid DeviceApprovalSuccess | Success discriminator |
| Type check error | `{ status: 'error', errorCode: 'NOT_FOUND', message: 'err' }` | Valid DeviceApprovalError | Error discriminator |

---

## Step 5: Implement Device Approval Hook

**Files to Read**:
- `.agent/rules/react/react-web-src.md`: gRPC API integration pattern
- `web/src/feature/project/hook/use-get-project.ts`: Example hook pattern for reference

### `web/src/feature/auth/hook/use-approve-device.ts`

**Description**: Create a mutation hook wrapping the DeviceCodeApprove RPC method using ConnectRPC and TanStack Query.

```typescript
import { useMutation } from '@connectrpc/connect-query';
import { deviceCodeApprove } from '@/gen/grpcstub/auth/v1/auth-AuthService_connectquery';
import { transport } from '@/shared/service/connect-transport';

// useApproveDevice provides a mutation hook for approving device codes.
// Returns a TanStack Query mutation object with mutate/mutateAsync functions.
export const useApproveDevice = () => {
  // Implementation outline:
  // 1. Call useMutation with deviceCodeApprove function.
  // 2. Pass transport as option.
  // 3. Return the mutation object.
  return useMutation(deviceCodeApprove, { transport });
};
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Hook initialization | - | Mutation object with mutate function | Hook creation |
| Mutation call | `{ userCode: 'ABCD-EFGH' }` | API request sent | Mutation execution |

---

## Step 6: Update Web Transport with Authentication Interceptor

**Files to Read**:
- `.agent/rules/react/react-web-src.md`: Service layer guidelines
- `web/src/shared/service/connect-transport.ts`: Current transport implementation

### Update `web/src/shared/service/connect-transport.ts`

**Description**: Add an interceptor to the ConnectRPC transport that includes the JWT access token from localStorage in the Authorization header for all requests.

```typescript
import { createConnectTransport } from '@connectrpc/connect-web';
import type { Interceptor } from '@connectrpc/connect';

// createAuthInterceptor creates an interceptor that adds JWT to requests.
const createAuthInterceptor = (): Interceptor => {
  return (next) => async (req) => {
    // Implementation outline:
    // 1. Get access token from localStorage using key 'cops_access_token'.
    // 2. If token exists and is not empty:
    //    a. Set Authorization header to `Bearer ${token}`.
    // 3. Call next(req) to proceed with the request.
    // 4. Return the response.
  };
};

// REPLACE existing transport export:
export const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080',
  interceptors: [createAuthInterceptor()],
});
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Token exists | localStorage has token | Authorization header added | Token present |
| Token missing | localStorage empty | No Authorization header | Token absent |
| Request processing | Any request | Request proceeds normally | Interceptor passthrough |

---

## Step 7: Implement Device Approval Component

**Files to Read**:
- `.agent/rules/react/react-web.md`: Component rules (named exports, arrow functions)
- `.agent/rules/react/react-web-src.md`: Component patterns
- `web/src/route/dashboard.tsx`: Example component pattern for reference

### `web/src/feature/auth/component/device-approval.tsx`

**Description**: Create a component that displays the device code and provides an approve button. The component handles loading, success, and error states using discriminated unions.

```typescript
import { useState } from 'react';
import { CheckCircle, XCircle, Loader2, Terminal, Shield } from 'lucide-react';
import { Button } from '@/gen/shadcn/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert';
import { useApproveDevice } from '../hook/use-approve-device';
import type { DeviceApprovalState } from '../type/device-code';

interface DeviceApprovalProps {
  userCode: string;
}

export const DeviceApproval = ({ userCode }: DeviceApprovalProps) => {
  // Implementation outline:
  // 1. Initialize state with DeviceApprovalState (initial: { status: 'pending' }).
  // 2. Get mutation object from useApproveDevice hook.
  // 3. Define handleApprove async function:
  //    a. Call mutation.mutateAsync with { userCode }.
  //    b. On success:
  //       - Set state to { status: 'success', message: 'Device approved successfully!' }.
  //    c. On error:
  //       - Parse error from mutation error object.
  //       - Map ConnectRPC error codes to DeviceApprovalErrorCode:
  //         * CodeNotFound -> 'NOT_FOUND'
  //         * CodeDeadlineExceeded -> 'EXPIRED'
  //         * CodeAlreadyExists -> 'ALREADY_APPROVED'
  //         * CodeUnauthenticated -> 'UNAUTHORIZED'
  //         * Default -> 'UNKNOWN'
  //       - Set state to { status: 'error', errorCode, message: error.message }.
  // 4. Render based on state.status:
  //    a. If 'pending':
  //       - Card with dark zinc-900 background.
  //       - Terminal icon in header.
  //       - Display user code in large monospace font with letter-spacing.
  //       - Show approve button (disabled during mutation.isPending).
  //       - Show loading spinner if mutation.isPending.
  //    b. If 'success':
  //       - Card with green-tinted background.
  //       - CheckCircle icon.
  //       - Success message.
  //       - Instruction: "You can return to your terminal."
  //    c. If 'error':
  //       - Card with red-tinted background.
  //       - XCircle icon.
  //       - Error message with specific text based on errorCode:
  //         * NOT_FOUND: "Device code not found. It may have expired."
  //         * EXPIRED: "This device code has expired. Please generate a new one."
  //         * ALREADY_APPROVED: "This device code has already been approved."
  //         * UNAUTHORIZED: "You must be logged in to approve devices."
  //         * UNKNOWN: Generic error message.
};
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Initial render | Valid userCode | Pending state with approve button | Pending UI |
| Approve click | User clicks approve | Mutation called, loading state shown | Loading UI |
| Approve success | Mutation succeeds | Success state with checkmark | Success UI |
| Approve error NOT_FOUND | Mutation fails with CodeNotFound | Error state with not found message | Error UI - NOT_FOUND |
| Approve error EXPIRED | Mutation fails with CodeDeadlineExceeded | Error state with expired message | Error UI - EXPIRED |
| Approve error ALREADY_APPROVED | Mutation fails with CodeAlreadyExists | Error state with already approved message | Error UI - ALREADY_APPROVED |
| Approve error UNAUTHORIZED | Mutation fails with CodeUnauthenticated | Error state with unauthorized message | Error UI - UNAUTHORIZED |

---

## Step 8: Create Device Approval Route

**Files to Read**:
- `.agent/rules/react/react-web-src.md`: Route patterns
- `web/src/route/dashboard.tsx`: Example route pattern for reference
- `web/src/route/__root.tsx`: Root layout for understanding routing structure

### `web/src/route/auth/device.tsx`

**Description**: Create a route component for `/auth/device` that handles authentication checks, extracts the code from query parameters, and renders the DeviceApproval component.

```typescript
import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect } from 'react';
import { Shield, AlertCircle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/gen/shadcn/ui/card';
import { Alert, AlertDescription } from '@/gen/shadcn/ui/alert';
import { DeviceApproval } from '@/feature/auth/component/device-approval';

// DeviceSearchParams defines the search params type for this route.
interface DeviceSearchParams {
  code?: string;
}

// Route configuration using TanStack Router's createFileRoute.
export const Route = createFileRoute('/auth/device')({
  component: DeviceApprovalPage,
  validateSearch: (search: Record<string, unknown>): DeviceSearchParams => {
    // Implementation outline:
    // 1. Validate that search.code is a string if present.
    // 2. Return object with code property (undefined if not valid string).
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
    };
  },
});

function DeviceApprovalPage() {
  // Implementation outline:
  // 1. Get navigate function from useNavigate hook.
  // 2. Get search params from useSearch hook with type DeviceSearchParams.
  // 3. Get code from search params.
  // 4. Use useEffect to check authentication on mount:
  //    a. Get access token from localStorage (key: 'cops_access_token').
  //    b. If token is missing or empty:
  //       - Build returnUrl: `/auth/device?code=${code}`.
  //       - Navigate to `/auth/login?returnUrl=${encodeURIComponent(returnUrl)}`.
  // 5. Render layout:
  //    a. Container: min-h-screen flex items-center justify-center bg-zinc-950.
  //    b. If code is missing:
  //       - Card with AlertCircle icon.
  //       - Error message: "No device code provided".
  //       - Description: "Please use the link from your CLI."
  //    c. If code exists and authenticated:
  //       - Card with Shield icon.
  //       - Title: "Approve CLI Access".
  //       - Description: "Review and approve this device code to sign in."
  //       - Render DeviceApproval component with userCode prop.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Not authenticated | No token in localStorage | Redirect to `/auth/login?returnUrl=...` | Auth check |
| Authenticated, no code | Token exists, no code param | Error: "No device code provided" | Code validation |
| Authenticated, with code | Token exists, code=ABCD-EFGH | DeviceApproval component rendered | Happy path |

---

## Implementation Order

1. **Step 1**: Move devicecode package functions to private functions in auth_service.go
2. **Step 2**: Delete devicecode package directory
3. **Step 3**: Create web auth feature directory structure
4. **Step 4**: Implement device code type definitions
5. **Step 5**: Implement device approval hook
6. **Step 6**: Update transport with authentication interceptor
7. **Step 7**: Implement device approval component
8. **Step 8**: Create device approval route

---

## Testing Strategy

### Backend Testing

1. **Unit Tests**:
   - Test private helper functions (generateUserCode, normalizeUserCode) in auth_service_test.go
   - Verify user code format (XXXX-XXXX pattern)
   - Verify normalization logic

2. **Integration Tests**:
   - Test auth service with real MongoDB
   - Verify device code approval flow end-to-end

### Frontend Testing

1. **Component Tests**:
   - DeviceApproval component state transitions
   - Error handling for all error codes
   - Loading state display

2. **Route Tests**:
   - Authentication redirect logic
   - Query parameter extraction
   - Error state when code is missing

### E2E Testing

1. **Full Flow**:
   - Run `cops auth login` in CLI
   - Visit displayed URL (while not logged in)
   - Redirect to login page with returnUrl
   - Login with Google OAuth
   - Redirect back to device approval page
   - Approve device
   - Verify CLI receives tokens

2. **Error Scenarios**:
   - Try to approve expired code
   - Try to approve same code twice
   - Try to approve non-existent code

---

## Manual Testing Checklist

### Backend Refactoring
- [ ] Run `go build ./api/...` and verify no import errors
- [ ] Verify devicecode package no longer exists
- [ ] Run existing auth service tests
- [ ] Test device code generation produces valid format
- [ ] Test normalization function with various inputs

### Web Implementation
- [ ] Run `npm run build` and verify no TypeScript errors
- [ ] Navigate to `/auth/device` without token -> Redirects to login
- [ ] Navigate to `/auth/device` without code param -> Shows error
- [ ] Navigate to `/auth/device?code=TEST-CODE` with token -> Shows approval UI
- [ ] Click approve button -> Loading state appears
- [ ] Successful approval -> Success message appears
- [ ] Try invalid code -> Error message appears

### Integration
- [ ] Run full CLI login flow
- [ ] Verify C-Ops Web URL is displayed (not Google URL)
- [ ] Visit URL while logged out -> Redirect to login -> Redirect back
- [ ] Approve device -> CLI receives tokens
- [ ] Verify tokens stored in `~/.cops/auth.json`
- [ ] Try to approve same code again -> Error message

---

## Critical Implementation Notes

1. **No Import Changes in Other Files**: The only file that imports devicecode package is auth_service.go. After moving the functions, no other files need to be updated.

2. **Function Names Change to Lowercase**: When moving from exported functions (GenerateUserCode) to private functions (generateUserCode), the first letter becomes lowercase.

3. **Preserve Implementation Logic**: The implementation of the three helper functions should be moved exactly as-is. Only the function names change (from exported to unexported).

4. **Web Transport Must Include Auth**: The DeviceCodeApprove RPC requires authentication. The interceptor in Step 6 is critical - without it, all approval attempts will fail with UNAUTHORIZED.

5. **TanStack Router File-Based Routing**: The route file `web/src/route/auth/device.tsx` will automatically create the `/auth/device` route. No manual route registration needed.

6. **Error Code Mapping**: The component must map ConnectRPC error codes to user-friendly messages. Reference the handler implementation in `api/internal/service/auth/inbound/grpc/connectrpc/handler.go` for the error codes used.

7. **localStorage Key**: Both the transport interceptor and the route component use the same key `cops_access_token` to access the JWT token. This must be consistent.

8. **Return URL Encoding**: When redirecting to login, the returnUrl must be URL-encoded to preserve the code query parameter.

9. **shadcn/ui Components**: Use existing shadcn components from `@/gen/shadcn/ui/`. These should already be installed. If any are missing, install via `npx shadcn@latest add <component-name>`.

10. **Code Format Display**: The user code should be displayed in a large, monospace font with letter spacing for readability (e.g., "A B C D - E F G H").

---

## Acceptance Criteria Verification

After implementation, verify:

- [ ] `api/internal/service/auth/devicecode/` directory does not exist
- [ ] `api/internal/service/auth/auth_service.go` contains three private helper functions
- [ ] `web/src/feature/auth/` directory exists with component, hook, and type subdirectories
- [ ] `web/src/feature/auth/type/device-code.ts` defines DeviceApprovalState discriminated union
- [ ] `web/src/feature/auth/hook/use-approve-device.ts` exports useApproveDevice hook
- [ ] `web/src/feature/auth/component/device-approval.tsx` exports DeviceApproval component
- [ ] `web/src/shared/service/connect-transport.ts` includes authentication interceptor
- [ ] `web/src/route/auth/device.tsx` creates `/auth/device` route
- [ ] Full CLI login flow works end-to-end through web approval
- [ ] Authentication check works (redirect to login if not authenticated)
- [ ] Error states display appropriate messages
