# Implementation Plan: Fix Device Login Redirect URL Bug

## Overview

This plan addresses a bug in the Device Login Flow where the redirect URL is malformed. When unauthenticated users access `/auth/device?code=ABC123`, they should be redirected to `/auth?redirect=/auth/device?code=ABC123`. However, the current implementation produces `/auth/device[object Object]` because `location.search` (a TanStack Router SearchParams object) is concatenated directly to a string.

The fix is straightforward: TanStack Router's `ParsedLocation` interface provides a `searchStr` property that contains the search parameters as a properly formatted string (e.g., `?code=ABC123`). We need to use `location.searchStr` instead of `location.search`.

## Package Changes

None required. This fix uses existing TanStack Router functionality.

## Step 1: Fix Redirect URL Construction in beforeLoad Hook

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md`: TypeScript and React coding conventions
- `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md`: Project structure and import rules
- `/Users/jayce/team-attention/cops/web/src/route/auth/index.tsx`: Reference for how redirect URLs are handled in the auth flow

### `/Users/jayce/team-attention/cops/web/src/route/auth/device.tsx`

**Description**:
Replace `location.search` with `location.searchStr` in the beforeLoad hook to construct a valid redirect URL string.

**Current Code (Lines 26-27)**:
```typescript
// Build redirect URL with full path and search params
const redirectUrl = location.pathname + location.search
```

**Fixed Code**:
```typescript
// Build redirect URL with full path and search params
const redirectUrl = location.pathname + location.searchStr
```

**Explanation**:

According to TanStack Router's `ParsedLocation` interface:

```typescript
interface ParsedLocation {
  href: string           // Full URL
  pathname: string       // Path portion (e.g., "/auth/device")
  search: TFullSearchSchema  // Parsed search params as object
  searchStr: string      // Search params as string (e.g., "?code=ABC123")
  state: ParsedHistoryState
  hash: string
  maskedLocation?: ParsedLocation
  unmaskOnReload?: boolean
}
```

The `searchStr` property:
- Contains the search parameters as a properly formatted query string
- Includes the leading `?` when search params exist
- Is an empty string `""` when no search params exist
- Handles URL encoding automatically

**Edge Cases Handled**:

| Scenario | `pathname` | `searchStr` | Result |
|----------|------------|-------------|--------|
| With code param | `/auth/device` | `?code=ABC123` | `/auth/device?code=ABC123` |
| No search params | `/auth/device` | `""` | `/auth/device` |
| Multiple params | `/auth/device` | `?code=ABC123&foo=bar` | `/auth/device?code=ABC123&foo=bar` |
| Special characters | `/auth/device` | `?code=ABC%20123` | `/auth/device?code=ABC%20123` |

**Test Scenarios**:

| Scenario | Input URL | Expected `redirectUrl` | Branch Covered |
|----------|-----------|------------------------|----------------|
| Standard device code | `/auth/device?code=ABC123` | `/auth/device?code=ABC123` | Normal flow with search params |
| No search params | `/auth/device` | `/auth/device` | Edge case: empty searchStr |
| URL-encoded characters | `/auth/device?code=ABC%20XYZ` | `/auth/device?code=ABC%20XYZ` | Special character handling |
| Multiple search params | `/auth/device?code=ABC&extra=value` | `/auth/device?code=ABC&extra=value` | Multiple params preserved |

## Verification Steps

After implementing the fix, verify the following:

1. **Manual Testing**:
   - Clear localStorage to simulate unauthenticated state
   - Navigate to `http://localhost:3000/auth/device?code=TESTCODE`
   - Verify redirect goes to `/auth?redirect=/auth/device?code=TESTCODE`
   - Complete authentication and verify redirect back to `/auth/device?code=TESTCODE`

2. **No Regression**:
   - Authenticated users accessing `/auth/device?code=X` should see the device approval page
   - The device approval flow should work correctly after the fix

## Summary

| File | Change | Lines Affected |
|------|--------|----------------|
| `web/src/route/auth/device.tsx` | Replace `location.search` with `location.searchStr` | Line 27 |

This is a one-line fix that leverages TanStack Router's built-in `searchStr` property to correctly construct the redirect URL.
