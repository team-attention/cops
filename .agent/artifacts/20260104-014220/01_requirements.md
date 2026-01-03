# Requirements

## Request Summary

The daemon currently sends logs to the API server but lacks token refresh logic when the access token expires. When the daemon receives an authentication error (401 Unauthorized) from the API server, it should automatically use the refresh token to obtain a new access token, persist it to the local filesystem (~/.cops/auth.json), and retry the original request. This ensures continuous operation without requiring manual re-authentication.

## Acceptance Criteria

- [ ] Daemon detects 401 Unauthorized errors from API server when sending logs
- [ ] On 401 error, daemon retrieves refresh token from existing auth service
- [ ] Daemon calls the RefreshToken API endpoint to get new access token
- [ ] New access token and refresh token are persisted to ~/.cops/auth.json
- [ ] Original failed request is retried with the new access token
- [ ] If refresh token is also invalid/expired, daemon logs appropriate error without infinite retry loop
- [ ] Access token is injected into API client requests via Authorization header
- [ ] Token refresh logic is centralized and reusable (not scattered across multiple files)
- [ ] Existing auth service cache is invalidated/updated after token refresh

## Scope

### In Scope
- Add authentication header injection to daemon's ConnectRPC API client
- Implement 401 error detection in log sending flow
- Integrate refresh token logic using existing auth service
- Update auth service to support token refresh via API
- Retry original request after successful token refresh
- Update local auth.json file after token refresh

### Out of Scope
- Changing the auth.json file format or location
- Adding new authentication methods (OAuth, etc.)
- Implementing token refresh for CLI commands (already exists in CLI)
- Adding metrics/monitoring for token refresh operations
- Implementing exponential backoff for retries (simple retry is sufficient)

## Constraints

- Must use existing RefreshToken gRPC endpoint at api/internal/service/auth
- Must reuse existing daemon auth service at daemon/internal/service/auth
- Must not break existing token refresh functionality in CLI
- Must follow hexagonal architecture patterns (service layer, outbound adapters)
- Token refresh must be transparent to log watching business logic
- Must handle concurrent requests during token refresh gracefully

## Additional Context

### Current Implementation

**Daemon Auth Service** (`daemon/internal/service/auth/auth_service.go`):
- Reads auth state from ~/.cops/auth.json
- Caches auth state with 30-second TTL
- Checks token expiry before returning access token
- Returns error if token is expired (no refresh logic)

**API Client** (`daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`):
- Uses ConnectRPC to call aggregation service
- No authentication header injection
- No error handling for 401 responses

**CLI Auth Service** (`cli/internal/service/auth/auth_service.go`):
- Has working token refresh implementation
- Uses auth API port to call RefreshToken endpoint
- Updates local auth.json after refresh
- Can serve as reference implementation

**API Server Auth Endpoint** (`idl/protobuf/auth/v1/auth.proto`):
- RefreshToken RPC exists: `rpc RefreshToken(RefreshTokenReq) returns (RefreshTokenRes)`
- Returns new access token, refresh token, and expiry

### Architecture

```
LogWatcher Service
    ↓ (calls)
API Client Adapter (ConnectRPC)
    ↓ (needs auth)
Auth Service ← (get access token, refresh if needed)
    ↓ (calls RefreshToken API)
Auth API Client Adapter
```

### Related Files

- `/Users/jayce/team-attention/cops/daemon/internal/service/auth/auth_service.go` - Daemon auth service
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` - API client
- `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/copsapi.go` - API client setup
- `/Users/jayce/team-attention/cops/cli/internal/service/auth/auth_service.go` - Reference implementation
- `/Users/jayce/team-attention/cops/idl/protobuf/auth/v1/auth.proto` - Auth service proto definition

## Questions Resolved

| Question | Answer |
|----------|--------|
| Should we handle 401 errors only, or other auth errors (403, etc.)? | Focus on 401 Unauthorized only. 403 Forbidden indicates insufficient permissions, not expired tokens. |
| Where should the auth token be injected into requests? | At the API client level (ConnectRPC adapter), using ConnectRPC interceptors or per-request headers. |
| Should we implement retry logic with backoff? | Simple one-time retry after refresh is sufficient. No need for exponential backoff. |
| What happens if refresh token is also expired? | Log error and fail gracefully. User will need to re-authenticate using CLI. Do not retry infinitely. |
| Should we add a new outbound adapter for auth API calls? | Yes, daemon's auth service needs an outbound adapter to call the RefreshToken API endpoint. Follow the pattern in CLI's auth service. |
| How should concurrent requests during token refresh be handled? | Use mutex in auth service to ensure only one refresh happens at a time. Other requests wait for the refresh to complete. |
