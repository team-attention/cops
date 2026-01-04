# Requirements

## Request Summary

The Daemon currently cannot automatically refresh expired access tokens, unlike the CLI which has proactive token refresh logic. When an access token expires, the Daemon's auth interceptor sends requests without authentication, causing 401 errors for authenticated endpoints. This needs to be fixed so the Daemon can autonomously refresh tokens and handle authentication failures properly.

## Acceptance Criteria

- [ ] Daemon's `AuthService.GetAccessToken()` implements proactive token refresh with 5-minute buffer (matching CLI behavior)
- [ ] When access token is near expiry or expired, Daemon automatically calls `RefreshAccessToken()` before returning the token
- [ ] Auth interceptor does NOT send requests when no valid access token is available for authenticated endpoints
- [ ] Auth interceptor returns appropriate error when token cannot be obtained or refreshed
- [ ] Token refresh is thread-safe and prevents concurrent refresh attempts (already exists with `refreshMu`)
- [ ] Refreshed tokens are saved to `~/.cops/auth.json` (already implemented in `RefreshAccessToken`)
- [ ] All token refresh operations are properly logged (Info level for success, Error level for failures)

## Scope

### In Scope
- Modify `daemon/internal/service/auth/auth_service.go`:
  - Update `GetAccessToken()` to implement proactive token refresh logic (similar to CLI)
  - Add 5-minute refresh buffer to check token expiry before it actually expires
  - Call `RefreshAccessToken()` when token is near expiry or expired

- Modify `daemon/internal/platform/interceptor/auth_interceptor.go`:
  - Remove logic that sends requests without authentication when token is unavailable
  - Return error immediately when `GetAccessToken()` fails
  - Keep 401 retry logic with token refresh (lines 56-74) as-is

### Out of Scope
- Changing the token refresh API endpoint or protocol
- Modifying the CLI's authentication behavior
- Implementing background token refresh (scheduled refresh before expiry)
- Adding token refresh retry logic with exponential backoff
- Creating a shared auth service package between CLI and Daemon

## Constraints

- Must maintain backward compatibility with existing `~/.cops/auth.json` format
- Must maintain thread-safety for concurrent token refresh attempts
- Must use existing `AuthAPIPort.RefreshToken()` interface
- Must follow existing logging patterns using structured logging (slog)
- Implementation should mirror CLI's `GetAccessToken()` logic (lines 156-198 in `cli/internal/service/auth/auth_service.go`)

## Additional Context

### Current Behavior (Problematic)

**Daemon Auth Service:**
```go
// daemon/internal/service/auth/auth_service.go:48-76
func (s *Service) GetAccessToken() (string, error) {
    // Only checks if expired, returns error - NO automatic refresh
    if s.cachedState.Tokens.ExpiresAt <= now {
        s.logger.Warn("access token expired", ...)
        return "", fmt.Errorf("access token expired")
    }
    return s.cachedState.Tokens.AccessToken, nil
}
```

**Auth Interceptor:**
```go
// daemon/internal/platform/interceptor/auth_interceptor.go:34-46
token, err := i.authService.GetAccessToken()
if err != nil {
    // Sends request WITHOUT auth - problematic for authenticated endpoints
    i.logger.Debug("no valid access token available, sending request without auth")
    return next(ctx, req)
}
```

### Expected Behavior (CLI Reference)

**CLI Auth Service:**
```go
// cli/internal/service/auth/auth_service.go:156-198
func (s *Service) GetAccessToken(ctx context.Context) (string, error) {
    now := time.Now().Unix()
    tokenExpiry := state.Tokens.ExpiresAt
    refreshBuffer := int64(300) // 5 minutes

    // Proactive refresh BEFORE expiry
    if tokenExpiry-now > refreshBuffer {
        return state.Tokens.AccessToken, nil
    }

    // Automatically refresh when near expiry
    result, err := s.apiClient.RefreshToken(ctx, state.Tokens.RefreshToken)
    // ... save and return new token
}
```

### Log Evidence

```
time=2026-01-05T04:22:08.829+09:00 level=WARN msg="access token expired" name=daemon.auth.service expiresAt=1767553553 now=1767554528
time=2026-01-05T04:22:08.829+09:00 level=DEBUG msg="no valid access token available, sending request without auth" name=auth.interceptor
time=2026-01-05T04:22:08.833+09:00 level=WARN msg="request failed with 401, user may need to re-authenticate via CLI" name=auth.interceptor
```

### Related Files

- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go` - Main service to modify
- `/Users/jayce/team-attention/cops/daemon/internal/platform/interceptor/auth_interceptor.go` - Interceptor to modify
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go` - Reference implementation

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should Daemon's `GetAccessToken()` take a `context.Context` parameter like CLI? | Yes, it needs context to call `RefreshAccessToken(ctx)` for the API call |
| What should the refresh buffer be? | 5 minutes (300 seconds), matching CLI behavior |
| Should we keep the 401 retry logic in the interceptor? | Yes, keep lines 56-74 as-is - it handles cases where token becomes invalid mid-request |
| What should happen when token refresh fails? | Return the error to caller, log it, and do NOT send the request without auth |
