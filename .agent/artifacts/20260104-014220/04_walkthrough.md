# Development Walkthrough: Daemon Token Refresh Logic

## Summary

Implemented automatic token refresh capability for the daemon's API client to handle expired access tokens transparently. When the daemon receives a 401 Unauthorized error from the API server, it automatically uses the refresh token to obtain new credentials, persists them to `~/.cops/auth.json`, and retries the original request without requiring manual re-authentication.

## Code Overview

### New Components

#### `AuthAPIPort` Interface
- **Location**: `daemon/internal/service/auth/outbound/api/auth_port.go`
- **Purpose**: Defines the port interface for auth API operations
- **Key Methods**:
  - `RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)`: Exchanges a refresh token for a new access token and refresh token pair

#### `AuthAPIClient` Adapter
- **Location**: `daemon/internal/service/auth/outbound/api/connectrpc/auth_client.go`
- **Purpose**: ConnectRPC implementation of the AuthAPIPort interface
- **Key Methods**:
  - `NewAuthAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config)`: Creates the auth API client using the daemon's shared HTTP client
  - `RefreshToken(ctx context.Context, refreshToken string) (*api.TokenResult, error)`: Calls the RefreshToken gRPC endpoint and returns the new token pair

#### `AuthInterceptor`
- **Location**: `daemon/internal/platform/interceptor/auth_interceptor.go`
- **Purpose**: ConnectRPC interceptor that transparently injects authentication headers and handles token refresh on 401 errors
- **Key Methods**:
  - `WrapUnary(next connect.UnaryFunc)`: Intercepts unary RPC calls to:
    1. Inject Authorization header with Bearer token from auth service
    2. Detect 401 Unauthenticated errors
    3. Automatically refresh token using RefreshAccessToken
    4. Retry the original request with new token (single retry, no infinite loop)
  - `WrapStreamingClient(next connect.StreamingClientFunc)`: Injects auth header for streaming requests (no retry support for streams)
  - `WrapStreamingHandler(next connect.StreamingHandlerFunc)`: No-op for client-side interceptor

#### `module_auth.go` Module
- **Location**: `daemon/cmd/internal/container/module_auth.go`
- **Purpose**: fx module that wires up auth service dependencies
- **Providers**:
  - Home directory provider for auth file path resolution
  - AuthAPIClient as AuthAPIPort interface
  - Auth service with injected dependencies

### Modified Components

#### `Auth Service`
- **Location**: `daemon/internal/service/auth/auth_service.go`
- **Changes**:
  - **Added** `authAPI AuthAPIPort` field: Injected port for calling the RefreshToken API
  - **Added** `refreshMu sync.Mutex` field: Prevents concurrent token refresh attempts
  - **Added** `GetRefreshToken()`: Returns the current refresh token from cached state
  - **Added** `RefreshAccessToken(ctx context.Context)`: Core token refresh logic that:
    1. Acquires refresh mutex to prevent concurrent refreshes
    2. Retrieves current refresh token from cached state
    3. Calls authAPI.RefreshToken to get new token pair
    4. Saves new tokens to `~/.cops/auth.json` via saveAuthState
    5. Updates in-memory cached state with new tokens
    6. Returns the new access token
  - **Added** `InvalidateCache()`: Forces reload of auth state from file on next access
  - **Added** `saveAuthState(state *AuthState)`: Persists auth state to filesystem with secure permissions (0700 for directory, 0600 for file)

#### `APIClient` Setup
- **Location**: `daemon/internal/platform/setup/copsapi.go`
- **Changes**:
  - **Added** `interceptor connect.Interceptor` field: Stores the auth interceptor
  - **Added** `SetInterceptor(interceptor connect.Interceptor)`: Sets the interceptor to be used for ConnectRPC clients
  - **Added** `ConnectOptions() []connect.ClientOption`: Returns ConnectRPC client options with the auth interceptor if set

#### `Aggregation API Client`
- **Location**: `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- **Changes**:
  - **Modified** `NewAPIClient` constructor to pass `apiClient.ConnectOptions()...` when creating the AggregationServiceClient
  - This enables the auth interceptor for all aggregation service calls
  - Also added `OrganizationID` field to the SendLogs request (separate unrelated change)

#### `Platform Module`
- **Location**: `daemon/cmd/internal/container/module_platform.go`
- **Changes**:
  - **Added** auth interceptor provider: `fx.Provide(interceptor.NewAuthInterceptor)`
  - **Added** fx.Invoke hook to wire the interceptor to the API client: `apiClient.SetInterceptor(authInterceptor)`

#### `Application Composition`
- **Location**: `daemon/cmd/internal/container/application.go`
- **Changes**:
  - **Added** `newAuthModule()` to the module list
  - **Ordering**: Auth module must come before platform module to ensure auth.Service is available for the AuthInterceptor

## Architecture & Flow

### Component Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│                      Daemon Application                      │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ Auth Module  │      │ Platform     │      │ Log Module   │
│              │      │ Module       │      │              │
└──────┬───────┘      └──────┬───────┘      └──────┬───────┘
       │                     │                     │
       │ Provides            │ Uses                │ Uses
       ▼                     ▼                     ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ Auth         │◄─────│ Auth         │      │ Aggregation  │
│ Service      │      │ Interceptor  │      │ API Client   │
└──────┬───────┘      └──────┬───────┘      └──────┬───────┘
       │                     │                     │
       │ Uses                │ Injected into       │ Uses
       ▼                     ▼                     ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ AuthAPIClient├─────►│ APIClient    │◄─────│ ConnectRPC   │
│ (ConnectRPC) │      │ (setup)      │      │ Clients      │
└──────────────┘      └──────────────┘      └──────────────┘
```

### Token Refresh Flow

```
1. LogWatcher sends batch via Aggregation API Client
                    │
                    ▼
2. AuthInterceptor intercepts request
   ├─ GetAccessToken() from Auth Service
   │  ├─ Success: Set Authorization header
   │  └─ Fail: Send without auth (user not logged in)
   └─ Execute request via ConnectRPC
                    │
                    ▼
3. API Server validates token
   ├─ Valid: Process request ✓
   └─ Expired: Return 401 Unauthenticated
                    │
                    ▼
4. AuthInterceptor detects 401
   └─ Call RefreshAccessToken(ctx)
                    │
                    ▼
5. Auth Service refreshes token
   ├─ GetRefreshToken() from cached state
   ├─ Call AuthAPIClient.RefreshToken()
   ├─ AuthAPIClient calls RefreshToken gRPC endpoint
   ├─ Receive new access token + refresh token
   ├─ Save to ~/.cops/auth.json (secure permissions)
   └─ Update in-memory cached state
                    │
                    ▼
6. AuthInterceptor retries request
   ├─ Update Authorization header with new token
   └─ Execute request again (single retry)
                    │
                    ▼
7. API Server processes request with new token ✓
```

### Error Handling Flow

```
User Not Logged In (no auth.json):
  GetAccessToken() → Error
  → Send request without Authorization header
  → API returns 401
  → Log warning: "user may need to re-authenticate via CLI"
  → Return original 401 error (NO RETRY)

Token Expired, Refresh Token Valid:
  GetAccessToken() → Expired token
  → Send request with expired token
  → API returns 401
  → RefreshAccessToken() → Success
  → Retry with new token → Success ✓

Token Expired, Refresh Token Invalid:
  GetAccessToken() → Expired token
  → Send request with expired token
  → API returns 401
  → RefreshAccessToken() → Error from API
  → Log warning: "token refresh failed"
  → Return original 401 error (NO INFINITE RETRY)
```

## Testing

### Manual Verification Commands

The implementation can be verified by testing the following scenarios:

#### 1. Build the Project
```bash
cd /Users/jayce/team-attention/cops
go build ./daemon/...
# Expected: Clean build with no errors
```

#### 2. Test Daemon Startup
```bash
cd /Users/jayce/team-attention/cops/daemon
./daemon
# Expected: Daemon starts successfully
# Expected: Auth module loads before platform module
```

#### 3. Test Token Refresh (Simulated Expiry)
To properly test token refresh, you would need to:
1. Authenticate using the CLI: `cops auth login`
2. Manually edit `~/.cops/auth.json` to set `expiresAt` to a past timestamp
3. Trigger a log event that causes the daemon to send logs
4. Observe daemon logs for "received 401, attempting token refresh"
5. Verify successful retry with new token

#### 4. Test User Not Logged In
```bash
# Remove auth file
rm ~/.cops/auth.json

# Start daemon and trigger log event
# Expected: Request sent without Authorization header
# Expected: Log message: "no valid access token available"
# Expected: Warning on 401: "user may need to re-authenticate via CLI"
```

### Unit Test Coverage

Key test scenarios to cover (not implemented in this workflow):

| Scenario | Component | Expected Behavior |
| -------- | --------- | ----------------- |
| RefreshToken success | AuthAPIClient | Returns TokenResult with new tokens |
| RefreshToken API error | AuthAPIClient | Returns error, logs error |
| RefreshAccessToken with valid refresh token | Auth Service | Calls AuthAPI, saves to file, updates cache, returns new token |
| RefreshAccessToken with invalid refresh token | Auth Service | Returns error from API, does not update cache |
| RefreshAccessToken concurrent calls | Auth Service | Only one refresh executes, others wait |
| WrapUnary with valid token | AuthInterceptor | Injects header, executes request |
| WrapUnary with 401, refresh succeeds | AuthInterceptor | Refreshes token, retries once, succeeds |
| WrapUnary with 401, refresh fails | AuthInterceptor | Returns original 401, logs warning |
| WrapUnary when not authenticated | AuthInterceptor | Sends without header, warns on 401 |
| SaveAuthState file permissions | Auth Service | Creates directory (0700) and file (0600) |
| InvalidateCache | Auth Service | Clears cached state |

## Implementation Notes for Future Developers

### Thread Safety

1. **Concurrent Token Refresh**: The `refreshMu` mutex in Auth Service ensures only one token refresh operation occurs at a time. Other goroutines attempting to refresh will block until the current refresh completes.

2. **Cache Access**: The `mu` read-write mutex protects the cached auth state. Read operations (GetAccessToken, GetRefreshToken) acquire read locks, while write operations (cache invalidation, state updates) acquire write locks.

3. **Interceptor Retry**: The AuthInterceptor only retries once after a successful token refresh. This prevents infinite retry loops if the refresh token is also invalid.

### Security Considerations

1. **File Permissions**: Auth state is saved with 0600 permissions (owner read/write only) to protect sensitive tokens. The `.cops` directory is created with 0700 permissions (owner access only).

2. **No Token Logging**: The implementation carefully avoids logging actual token values. Only errors and operation results are logged.

3. **Refresh Token Expiry**: If the refresh token is expired or invalid, the daemon gracefully fails and logs a warning message directing the user to re-authenticate via the CLI.

### Dependency Injection Order

The `newAuthModule()` must be registered **before** `newPlatformModule()` in the fx application composition because:
- The AuthInterceptor (created in platform module) depends on auth.Service (provided by auth module)
- fx resolves dependencies in module registration order
- This dependency ordering is documented in the application.go file

### Streaming Requests

The current implementation only supports automatic retry for unary RPC calls. Streaming requests receive the Authorization header but do NOT support automatic retry on 401 errors because:
- Reconnecting a stream requires more complex state management
- Streams are not currently used in the daemon's API communication
- This can be enhanced in the future if streaming endpoints are added

### Token Cache TTL

The auth service caches the auth state for 30 seconds to avoid excessive file I/O. After 30 seconds, the state is reloaded from `~/.cops/auth.json`. This TTL is hardcoded and can be made configurable if needed.

### Error Types

The implementation uses ConnectRPC error codes (`connect.CodeUnauthenticated`) to detect 401 errors. This is the standard way to handle gRPC error codes in ConnectRPC applications.

### Configuration

No new configuration options were added. The implementation uses existing configuration:
- `~/.cops/auth.json` location (existing)
- `cfg.API.URL` for the auth API endpoint (existing)
- Auth service cache TTL is hardcoded to 30 seconds

## Related Documentation

- Original CLI auth service implementation: `cli/internal/service/auth/auth_service.go`
- RefreshToken gRPC proto definition: `idl/protobuf/auth/v1/auth.proto`
- Hexagonal architecture patterns: `.agent/rules/go/go-port-adapter-pattern.md`
- Outbound adapter guidelines: `.agent/rules/go/go-outbound.md`
- Platform setup patterns: `.agent/rules/go/go-platform-setup.md`

## Future Enhancements

1. **Metrics**: Add metrics for token refresh operations (success/failure counts, latency)
2. **Exponential Backoff**: Implement retry with exponential backoff for transient errors
3. **Streaming Support**: Add automatic retry support for streaming RPC calls
4. **Configurable Cache TTL**: Make the 30-second auth cache TTL configurable
5. **Token Preemptive Refresh**: Refresh token proactively before expiry (e.g., when <5 minutes remaining)
