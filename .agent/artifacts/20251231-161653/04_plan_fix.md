# Implementation Plan: Fix OAuth Token Expiry Calculation Bug

## Overview

This plan addresses a critical bug identified in the Google OAuth adapter's `ExchangeCode` method. The bug causes the `ExpiresIn` field to always return 0 seconds because the code incorrectly subtracts `token.Expiry` from itself instead of calculating the duration until the token expires.

**Bug Location**: `api/internal/service/auth/outbound/oauth/google/google_oauth.go:58`

**Current (Buggy) Code**:
```go
ExpiresIn: int(token.Expiry.Sub(token.Expiry).Seconds()),
```

**Impact**: All OAuth tokens appear to expire immediately, breaking authentication flows that rely on token expiry information.

## Package Changes

None required. The fix uses the existing `time` package already imported in the file.

## Implementation Steps

### Step 1: Fix Token Expiry Calculation in ExchangeCode Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/oauth/google/google_oauth.go`: Contains the buggy code to be fixed
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Rules for outbound adapters

#### `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/oauth/google/google_oauth.go`

**Description**:
Fix the `ExpiresIn` calculation in the `ExchangeCode` method by using `time.Until()` to calculate the duration from now until the token expires.

**Change Location**: Line 58

**Current Code**:
```go
return &oauthport.TokenResponse{
    AccessToken:  token.AccessToken,
    RefreshToken: token.RefreshToken,
    ExpiresIn:    int(token.Expiry.Sub(token.Expiry).Seconds()),
}, nil
```

**Fixed Code**:
```go
return &oauthport.TokenResponse{
    AccessToken:  token.AccessToken,
    RefreshToken: token.RefreshToken,
    ExpiresIn:    int(time.Until(token.Expiry).Seconds()),
}, nil
```

**Explanation**:
- `time.Until(token.Expiry)` returns the duration from `time.Now()` until `token.Expiry`
- This is equivalent to `token.Expiry.Sub(time.Now())` but more idiomatic
- Converting to `int` after `.Seconds()` provides the expected integer seconds value

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Token expires in 1 hour | `token.Expiry = time.Now().Add(1 * time.Hour)` | `ExpiresIn = 3600` (approximately) | Normal token exchange |
| Token expires in 30 minutes | `token.Expiry = time.Now().Add(30 * time.Minute)` | `ExpiresIn = 1800` (approximately) | Normal token exchange |
| Token already expired | `token.Expiry = time.Now().Add(-5 * time.Minute)` | `ExpiresIn = -300` (negative value) | Edge case - expired token |
| Token expires immediately | `token.Expiry = time.Now()` | `ExpiresIn = 0` | Edge case - immediate expiry |

**Notes**:
- The `time.Until()` function is preferred over `token.Expiry.Sub(time.Now())` as it is more readable and idiomatic Go
- If `token.Expiry` is before `time.Now()`, the result will be negative. This is acceptable behavior as it indicates the token is already expired
- No additional error handling is needed since `golang.org/x/oauth2` already handles token exchange errors

## Verification Steps

After implementing the fix:

1. **Build Check**: Verify the code compiles without errors
   ```bash
   cd /Users/jayce/team-attention/cops/api && go build ./...
   ```

2. **Unit Test** (if exists): Run any existing tests for the Google OAuth adapter
   ```bash
   cd /Users/jayce/team-attention/cops/api && go test ./internal/service/auth/outbound/oauth/google/...
   ```

3. **Manual Verification**: The fix can be verified by checking that:
   - `ExpiresIn` returns a positive integer (in seconds) for valid tokens
   - The value approximately matches the expected token lifetime (typically 3600 seconds for Google OAuth access tokens)

## Quality Checklist

- [x] Fix uses idiomatic Go (`time.Until()` instead of manual subtraction)
- [x] No new dependencies required
- [x] No changes to function signatures
- [x] No changes to interfaces or ports
- [x] Single line change minimizes risk of regressions
- [x] Test scenarios cover normal and edge cases
