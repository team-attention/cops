# Development Walkthrough: Google OAuth Authentication System

## Summary

Implemented a comprehensive Google OAuth 2.0 authentication system with JWT token management across the C-Ops platform. The system introduces User and Account domain models, provides CLI device flow authentication, and enables secure session management for API access. The implementation follows hexagonal architecture with clear separation between domain models, service logic, and infrastructure adapters.

## Code Overview

### Domain Models

#### `User` Domain Model
- **Location**: `shared/domain/user.go`
- **Purpose**: Represents authenticated users in the system with support for multiple OAuth provider accounts
- **Key Fields**:
  - `ID`: Unique user identifier
  - `Email`: User's email address
  - `Name`: Display name
  - `ProfileImageURL`: Profile picture URL (optional)
  - `Accounts`: Array of linked OAuth accounts (embedded)

**Design Decision**: Accounts are embedded within the User document rather than stored in a separate collection. This simplifies queries and reduces database round-trips since users typically have 1-2 linked accounts.

#### `Account` Domain Model
- **Location**: `shared/domain/account.go`
- **Purpose**: Represents OAuth provider credentials linked to a user
- **Key Fields**:
  - `Provider`: OAuth provider identifier (e.g., "google")
  - `ProviderID`: User ID from the OAuth provider
- **Provider Support**: Currently supports Google OAuth with extensible design for future providers

**Architecture Note**: The Account model is never queried directly - it's always accessed through the User document using MongoDB's `$elemMatch` operator.

#### `Organization` and `OrganizationMember` Domain Models
- **Location**: `shared/domain/organization.go`
- **Purpose**: Define organizational structure for future multi-tenant support
- **Status**: Domain models and MongoSchema defined, but service implementation out of scope
- **Key Fields**:
  - Organization: `ID`, `Name`, `Slug`
  - OrganizationMember: `OrganizationID`, `UserID`, `Role`

### MongoSchema Implementations

#### `User` MongoSchema
- **Location**: `shared/domain/mongoschema/user.go`
- **Purpose**: MongoDB-specific schema for User with BSON ObjectID conversion
- **Field Constants**: Defined for all fields to avoid magic strings in queries
  - `UserIDField = "_id"`
  - `UserEmailField = "email"`
  - `UserAccountsField = "accounts"`
  - `AccountProviderField = "provider"` (embedded type)
  - `AccountProviderIDField = "providerId"` (embedded type)
- **Methods**:
  - `FromDomain(*domain.User)`: Converts domain model to MongoDB document
  - `ToDomain() *domain.User`: Converts MongoDB document to domain model

**Implementation Pattern**: Each struct type (User, Account) has its own set of field constants with the struct name as prefix. This follows the project standard for avoiding magic strings while maintaining clarity.

### Authentication Service (API)

#### `auth.Service`
- **Location**: `api/internal/service/auth/auth_service.go`
- **Purpose**: Core authentication business logic orchestrating OAuth flows and token generation
- **Dependencies**:
  - `jwtutil.Config`: JWT token configuration
  - `oauth.GoogleOAuthPort`: Google OAuth operations interface
  - `repository.UserRepositoryPort`: User persistence interface

**Key Methods**:

1. **`GoogleAuth(ctx, params)`**: Web-based OAuth flow
   - Exchanges Google authorization code for tokens
   - Fetches user profile from Google
   - Finds or creates user in database
   - Generates JWT token pair
   - Returns tokens only (no user data in response)

2. **`DeviceCode(ctx)`**: Initiates CLI device flow
   - Requests device code from Google
   - Returns verification URL and user code for CLI display
   - Typically used by `cops auth login` command

3. **`DevicePoll(ctx, deviceCode)`**: Polls for CLI authentication completion
   - Checks if user completed browser authentication
   - Returns pending status or tokens when complete
   - Creates new user if first-time login

4. **`RefreshToken(ctx, refreshToken)`**: Token renewal
   - Validates refresh token
   - Verifies user still exists
   - Generates new token pair
   - Returns fresh tokens

**Error Handling Strategy**: All errors are logged at the service layer with structured context (userID, email, etc.) but generic error messages are returned to prevent information leakage.

### JWT Utilities

#### `jwtutil` Package
- **Location**: `api/internal/platform/util/jwtutil/jwtutil.go`
- **Purpose**: JWT token generation and validation using HS256 signing
- **Key Types**:
  - `Config`: JWT configuration (secret, durations, issuer)
  - `TokenPair`: Contains access token, refresh token, and expiry time

**Key Functions**:

1. **`GenerateTokenPair(cfg, userID)`**:
   - Creates both access and refresh tokens in a single call
   - Access token: 30 minutes (configurable)
   - Refresh token: 30 days (configurable)
   - Uses only `jwt.RegisteredClaims` with UserID in Subject field
   - No custom claims to keep tokens simple

2. **`ValidateAccessToken(cfg, tokenString)`**:
   - Parses and validates access token
   - Checks signing method is HS256
   - Verifies signature and expiry
   - Returns userID from Subject claim

3. **`ValidateRefreshToken(cfg, tokenString)`**:
   - Similar to access token validation
   - Longer expiry time
   - Returns userID for token refresh flow

**Security Notes**:
- Tokens are signed with HS256 (HMAC with SHA-256)
- Secret key must be at least 256 bits
- Token expiry times are enforced at validation
- Invalid tokens return generic errors (no timing attacks)

### Authentication Middleware

#### `AuthMiddleware`
- **Location**: `api/internal/platform/middleware/auth.go`
- **Purpose**: Fiber middleware for JWT authentication on protected endpoints
- **Flow**:
  1. Extract `Authorization` header
  2. Validate `Bearer {token}` format
  3. Validate token using `jwtutil.ValidateAccessToken`
  4. Store userID in Fiber locals with key `"userId"`
  5. Continue to next handler or return 401 Unauthorized

**Context Key Convention**: Uses camelCase `"userId"` following project standards (not `"user_id"`).

#### `GetUserID(c *fiber.Ctx)`
- Helper function to extract userID from Fiber context
- Returns empty string if not authenticated
- Type-safe extraction with nil checks

### Outbound Adapters

#### Google OAuth Adapter
- **Location**: `api/internal/service/auth/outbound/oauth/google/google_oauth.go`
- **Purpose**: Implements Google OAuth 2.0 protocol operations
- **Port Interface**: `oauth.GoogleOAuthPort`

**Key Methods**:

1. **`ExchangeCode(ctx, code, redirectURI)`**: Web OAuth flow
   - Exchanges authorization code for Google tokens
   - Returns tokens with expiry calculation
   - **Bug Fix Applied**: Token expiry now correctly calculated using `time.Until()` instead of `token.Expiry.Sub(token.Expiry)` which always returned 0

2. **`GetUserInfo(ctx, accessToken)`**: Fetch user profile
   - Calls Google's userinfo v2 API
   - Returns user ID, email, name, picture, email verification status
   - Handles API errors gracefully

3. **`InitiateDeviceFlow(ctx)`**: Device code request
   - POSTs to Google's device code endpoint
   - Returns device code, user code, verification URL, interval
   - Used by CLI for headless authentication

4. **`PollDeviceCode(ctx, deviceCode)`**: Device flow polling
   - Polls Google's token endpoint
   - Returns `nil` for pending state (`authorization_pending`, `slow_down`)
   - Returns tokens when user completes authentication
   - Returns error for denied/expired states

**Configuration**: OAuth credentials (client ID, secret, scopes) are injected via main config, not hardcoded.

#### User MongoDB Repository
- **Location**: `api/internal/service/auth/outbound/repository/mongodb/user_repo.go`
- **Purpose**: User persistence with MongoDB
- **Port Interface**: `repository.UserRepositoryPort`

**Key Methods**:

1. **`Create(ctx, user)`**: Insert new user
   - Converts domain.User to mongoschema.User
   - Inserts into `users` collection
   - Sets ID from MongoDB's generated ObjectID
   - Returns created user

2. **`GetByID(ctx, userID)`**: Retrieve user by ID
   - Converts string ID to ObjectID
   - Finds by `_id` field
   - Returns NotFoundError if missing
   - Converts to domain model

3. **`FindByAccountProvider(ctx, provider, providerID)`**: Find by OAuth account
   - Uses `$elemMatch` to search embedded accounts array
   - Query pattern:
     ```go
     filter := bson.M{
         "accounts": bson.M{
             "$elemMatch": bson.M{
                 "provider":   provider,
                 "providerId": providerID,
             },
         },
     }
     ```
   - Returns `nil` (not error) if user not found
   - Used to check if OAuth account already linked

### ConnectRPC Handler

#### `AuthGRPCHandler`
- **Location**: `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`
- **Purpose**: ConnectRPC endpoints for authentication operations
- **Interface**: Implements `authv1connect.AuthServiceHandler`

**RPC Methods**:

1. **`GoogleAuth(ctx, req)`**: Web OAuth endpoint
   - Accepts authorization code and redirect URI
   - Calls service.GoogleAuth
   - Returns tokens in protobuf format

2. **`DeviceCode(ctx, req)`**: Device flow initiation
   - No parameters (empty request)
   - Returns device code info for CLI display

3. **`DevicePoll(ctx, req)`**: Device flow polling
   - Accepts device code
   - Returns pending status or tokens

4. **`RefreshToken(ctx, req)`**: Token refresh
   - Accepts refresh token
   - Returns new token pair

**Response Design**: All responses return tokens only, no user information. This simplifies the protocol and reduces data transfer.

### Protobuf Definitions

#### `auth.proto`
- **Location**: `idl/protobuf/auth/v1/auth.proto`
- **Messages**:
  - `TokenPair`: access_token, refresh_token, expires_at (Unix timestamp)
  - `GoogleAuthReq`: authorization_code, redirect_uri
  - `DeviceCodeReq`: Empty message
  - `DeviceCodeRes`: device_code, user_code, verification_url, expires_in, interval
  - `DevicePollReq`: device_code
  - `DevicePollRes`: pending (bool), tokens (optional)
  - `RefreshTokenReq`: refresh_token

**Naming Convention**: Request/response messages use `Req`/`Res` suffixes (not `Request`/`Response`).

### CLI Authentication Service

#### `auth.Service` (CLI)
- **Location**: `cli/internal/service/auth/auth_service.go`
- **Purpose**: CLI-side authentication state management
- **Storage**: `~/.cops/auth.json` with 0600 permissions

**Key Types**:

```go
type AuthState struct {
    Tokens *TokenInfo `json:"tokens"`
}

type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    int64  `json:"expiresAt"` // Unix timestamp
}
```

**Key Methods**:

1. **`InitiateLogin(ctx)`**: Start device flow
   - Calls API DeviceCode endpoint
   - Returns device code info for display

2. **`PollLogin(ctx, deviceCode)`**: Poll for completion
   - Calls API DevicePoll endpoint
   - Saves tokens to `~/.cops/auth.json` when complete
   - Returns boolean indicating completion

3. **`Logout(ctx)`**: Remove credentials
   - Deletes `~/.cops/auth.json` file
   - Safe to call when not logged in

4. **`GetAccessToken(ctx)`**: Get valid token
   - Reads auth state from file
   - Checks if token expires within 5 minutes
   - Automatically refreshes if near expiry
   - Saves new tokens after refresh
   - Returns error if not logged in

5. **`IsLoggedIn()`**: Check authentication status
   - Reads auth state
   - Returns true if valid tokens exist

**Security**: Auth file created with 0700 directory permissions and 0600 file permissions to prevent unauthorized access.

### CLI Commands

#### Login Command
- **Location**: `cli/internal/service/auth/inbound/cli/cobra/login.go`
- **Flow**:
  1. Call `InitiateLogin` to get device code
  2. Display verification URL and user code
  3. Poll every `interval` seconds
  4. Show success message when complete
  5. 10-minute timeout for user action

**User Experience**:
```
To sign in, open this URL in your browser:
  https://www.google.com/device

Then enter this code:
  ABCD-EFGH

Waiting for authentication...
```

#### Logout Command
- **Location**: `cli/internal/service/auth/inbound/cli/cobra/logout.go`
- **Purpose**: Remove stored credentials
- **Flow**: Call `Logout()` service method

#### Status Command
- **Location**: `cli/internal/service/auth/inbound/cli/cobra/status.go`
- **Purpose**: Show current authentication state
- **Output**: Indicates if logged in or not

### Daemon Authentication Service

#### `auth.Service` (Daemon)
- **Location**: `daemon/internal/service/auth/auth_service.go`
- **Purpose**: Shared authentication state reading for daemon
- **Storage**: Reads from same `~/.cops/auth.json` as CLI

**Key Features**:

1. **Caching**: 30-second cache TTL to reduce file I/O
2. **Thread Safety**: Uses `sync.RWMutex` for concurrent access
3. **Automatic Reload**: Refreshes cache when stale

**Key Methods**:

1. **`GetAccessToken()`**: Get current token
   - Reads from cached state or file
   - Checks token expiry
   - Returns error if expired or missing
   - **No automatic refresh** (daemon doesn't call API)

2. **`IsAuthenticated()`**: Check if valid auth exists
   - Returns true if valid token available

**Daemon Behavior**: If authentication is missing or expired, daemon operations will fail with an error. The user must run `cops auth login` to re-authenticate. This is intentional - unauthenticated requests are not allowed.

### HTTP Client Pattern

#### CLI HTTP Client
- **Location**: `cli/internal/platform/setup/httpclient/httpclient.go`
- **Pattern**: Uses `imroc/req/v3` library
- **Key Methods**:
  - `StandardHTTPClient()`: Returns `*http.Client` for ConnectRPC
  - `WithAuth(accessToken)`: Returns cloned client with Bearer token

**Usage Pattern**:
```go
// Service makes authenticated request
func (s *Service) DoWork(ctx context.Context) error {
    token, err := s.authSvc.GetAccessToken(ctx)
    if err != nil {
        return err
    }

    authClient := s.httpClient.WithAuth(token)
    // Use authClient for authenticated requests
}
```

**Design Decision**: Authentication is handled per-request by services, not globally injected at client initialization. This allows services to control token refresh timing.

#### Daemon API Client
- **Location**: `daemon/internal/platform/setup/copsapi.go`
- **Pattern**: Same as CLI - `imroc/req/v3` with `WithAuth()`
- **Usage**: Daemon services get token from auth service before each API call

## Configuration Requirements

### API Server Configuration

**Environment Variables Required**:

```bash
# JWT Configuration
JWT_SECRET_KEY=your-256-bit-secret-key-here  # Required
JWT_ACCESS_TOKEN_DURATION=30m                 # Optional, default: 30m
JWT_REFRESH_TOKEN_DURATION=720h              # Optional, default: 30 days
JWT_ISSUER=cops                               # Optional, default: cops

# Google OAuth Configuration
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com      # Required
GOOGLE_CLIENT_SECRET=your-client-secret                          # Required
GOOGLE_SCOPES=email,profile                                      # Optional, default: email,profile
```

**Configuration File**: `api/.meta/.env.example` has been updated with these variables.

**Obtaining Google OAuth Credentials**:
1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project or select existing
3. Enable Google+ API
4. Go to Credentials > Create Credentials > OAuth 2.0 Client ID
5. Configure consent screen
6. Add authorized redirect URIs (for web flow)
7. For device flow, no redirect URI needed
8. Copy Client ID and Client Secret to environment

### CLI/Daemon Configuration

**No configuration required** - authentication state is stored in `~/.cops/auth.json` after login.

**File Locations**:
- Auth file: `~/.cops/auth.json` (0600 permissions)
- Directory: `~/.cops/` (0700 permissions)

## Testing Considerations

### Unit Tests

**JWT Utilities**:
- Test token generation with various user IDs
- Test validation with valid tokens
- Test validation with expired tokens
- Test validation with malformed tokens
- Test validation with wrong signing method (should fail)

**Auth Service**:
- Mock `GoogleOAuthPort` and `UserRepositoryPort`
- Test new user creation flow
- Test existing user login flow
- Test device flow polling states
- Test token refresh with valid/invalid tokens

**Repositories**:
- Test user creation and retrieval
- Test finding user by OAuth provider account
- Test `$elemMatch` query for embedded accounts
- Test ObjectID conversion

### Integration Tests

**OAuth Flow**:
- Mock Google OAuth endpoints
- Test full authorization code exchange
- Test device code flow (initiate, poll pending, poll complete)
- Test user info fetching

**Database Operations**:
- Test user CRUD with real MongoDB (test container)
- Test account linking and searching
- Test concurrent access scenarios

### E2E Tests

**CLI Login Flow**:
1. Mock API server responding with device code
2. Mock API server responding with pending state
3. Mock API server responding with tokens
4. Verify auth.json file created with correct permissions
5. Verify token refresh on next command

**Daemon Authentication**:
1. Start daemon with valid auth.json
2. Verify daemon can read tokens
3. Remove auth.json mid-operation
4. Verify daemon fails gracefully with auth error

## Architecture Decisions

### 1. Embedded Accounts vs Separate Collection

**Decision**: Embed accounts within User document

**Rationale**:
- Users typically have 1-2 linked accounts
- Queries always need user + accounts together
- Reduces database round-trips
- Simplifies schema (no foreign keys)
- MongoDB excels at embedded documents

**Tradeoff**: If users have many accounts (>10), separate collection would be better. Current design assumes low account count.

### 2. JWT-Only Authentication (No Sessions)

**Decision**: Use JWT tokens instead of server-side sessions

**Rationale**:
- Stateless - no session storage needed
- Scales horizontally without shared state
- Works across multiple API servers
- Standard industry practice for APIs
- Easy to verify without database lookup

**Tradeoff**: Cannot invalidate tokens before expiry. Mitigation: short access token lifetime (30 min).

### 3. Device Flow for CLI

**Decision**: Use OAuth device flow instead of localhost callback

**Rationale**:
- Works in SSH/headless environments
- No need to open local server port
- Simple user experience (enter code in browser)
- Standardized protocol (RFC 8628)
- Secure (device code != user code)

**Tradeoff**: Requires polling. Mitigation: respect Google's interval parameter.

### 4. Tokens-Only Response

**Decision**: Auth endpoints return only tokens, not user information

**Rationale**:
- Reduces response size
- Prevents information leakage
- User info available via separate endpoint if needed
- Simplifies protocol
- Focus on authentication, not profile data

**Implementation**: User info can be added later via `GetCurrentUser` RPC without breaking existing clients.

### 5. Per-Request Authentication

**Decision**: Services add auth tokens per-request, not at client initialization

**Rationale**:
- Allows token refresh without recreating clients
- Services control when to refresh
- Clean separation of concerns
- Testable (can inject different tokens)
- Follows OAuth best practices

**Pattern**: `httpClient.WithAuth(token)` clones client with Bearer header.

### 6. Shared Auth State (CLI/Daemon)

**Decision**: Daemon reads CLI's auth.json instead of separate config

**Rationale**:
- Single source of truth
- User only logs in once (via CLI)
- No config sync issues
- Simpler user experience
- Daemon has no independent auth flow

**Tradeoff**: Daemon requires CLI login first. This is acceptable since CLI is primary interface.

### 7. Mandatory Authentication

**Decision**: Daemon fails operations if not authenticated (no fallback)

**Rationale**:
- Security-first design
- Prevents accidental data leaks
- Clear user feedback (must login)
- No ambiguity in behavior
- Aligns with production security model

**User Impact**: If token expires, daemon logs error and stops. User must run `cops auth login`.

## Key Implementation Patterns

### Hexagonal Architecture

**Layers**:
```
Inbound (CLI/HTTP/gRPC) → Service → Outbound (MongoDB/OAuth)
```

**Dependencies Flow Inward**:
- Inbound adapters depend on service
- Service depends on outbound ports (interfaces)
- Outbound adapters implement ports
- No cross-layer dependencies

**Benefits**:
- Service logic independent of infrastructure
- Easy to swap implementations (Postgres instead of MongoDB)
- Testable with mocks
- Clear separation of concerns

### Port/Adapter Pattern

**Port (Interface)**:
```go
type UserRepositoryPort interface {
    Create(ctx context.Context, user *domain.User) (*domain.User, error)
    GetByID(ctx context.Context, userID string) (*domain.User, error)
    FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error)
}
```

**Adapter (Implementation)**:
```go
type MongoUserRepository struct {
    logger    *slog.Logger
    usersColl *mongo.Collection
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    // MongoDB-specific implementation
}

var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil) // Compile-time check
```

**Dependency Injection**:
```go
type Service struct {
    userRepo repository.UserRepositoryPort // Port type
}

// Container injects concrete adapter
fx.Provide(
    fx.Annotate(
        mongodb.NewMongoUserRepository,
        fx.As(new(repository.UserRepositoryPort)), // Binds adapter to port
    ),
)
```

### Error Handling Strategy

**Principle**: Log at service layer, return generic errors

```go
result, err := s.userRepo.GetByID(ctx, userID)
if err != nil {
    s.logger.Error("failed to get user",
        slog.String("userID", userID),
        slog.Any("error", err),
    )
    return nil, fmt.Errorf("user not found: %w", err)
}
```

**Benefits**:
- Detailed logs for debugging
- Generic errors prevent information leakage
- Consistent error messages across services
- Security-conscious design

### Structured Logging

**Pattern**: Bind logger in constructor

```go
func NewService(l *slog.Logger, ...) *Service {
    return &Service{
        logger: l.With(slog.String("name", "auth.service")),
        // ...
    }
}
```

**Usage**:
```go
s.logger.Info("user logged in",
    slog.String("userID", string(user.ID)),
    slog.String("email", user.Email),
)
```

**Benefits**:
- Structured logs (not string formatting)
- Searchable fields
- Consistent component naming
- Easy log aggregation

## Issues Encountered and Resolutions

### Issue 1: OAuth Token Expiry Calculation Bug

**Problem**: In initial implementation, token expiry was calculated as:
```go
ExpiresIn: int(token.Expiry.Sub(token.Expiry).Seconds())
```
This always returned 0 because it subtracted the expiry time from itself.

**Resolution**: Changed to:
```go
ExpiresIn: int(time.Until(token.Expiry).Seconds())
```
This correctly calculates seconds from now until expiry.

**Impact**: Without fix, all OAuth tokens would appear to expire immediately, breaking refresh logic.

**File**: `api/internal/service/auth/outbound/oauth/google/google_oauth.go:59`

## Testing the Authentication System

### Manual Testing Steps

**1. Configure API Server**:
```bash
cd api
cp .meta/.env.example .env
# Edit .env with your Google OAuth credentials
export $(cat .env | xargs)
```

**2. Start API Server**:
```bash
cd api
make dev  # Starts with Docker hot reload
```

**3. Test CLI Login**:
```bash
cd cli
go build -o cops ./cmd
./cops auth login
# Follow displayed instructions to authenticate
```

**4. Verify Authentication**:
```bash
./cops auth status
# Should show "Logged in"

cat ~/.cops/auth.json
# Should show tokens
```

**5. Test Daemon**:
```bash
cd daemon
go build -o daemon ./cmd
./daemon  # Should read CLI auth and connect successfully
```

**6. Test Logout**:
```bash
./cops auth logout
./cops auth status
# Should show "Not logged in"
```

### Verification Checklist

- [ ] API server starts without errors
- [ ] Device flow displays Google verification URL
- [ ] Browser authentication redirects correctly
- [ ] CLI saves tokens to `~/.cops/auth.json`
- [ ] File permissions are 0600 (verify with `ls -la ~/.cops/`)
- [ ] Token refresh works when near expiry
- [ ] Daemon reads CLI auth state
- [ ] Daemon operations fail gracefully when not authenticated
- [ ] Logout removes auth file
- [ ] Re-login works after logout

## Related Files

**Domain Models**:
- `shared/domain/user.go`
- `shared/domain/account.go`
- `shared/domain/organization.go`

**MongoSchema**:
- `shared/domain/mongoschema/user.go`
- `shared/domain/mongoschema/organization.go`
- `shared/domain/mongoschema/organization_member.go`

**API Implementation**:
- `api/internal/service/auth/auth_service.go`
- `api/internal/platform/util/jwtutil/jwtutil.go`
- `api/internal/platform/middleware/auth.go`
- `api/internal/service/auth/outbound/oauth/google/google_oauth.go`
- `api/internal/service/auth/outbound/repository/mongodb/user_repo.go`
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**CLI Implementation**:
- `cli/internal/service/auth/auth_service.go`
- `cli/internal/service/auth/inbound/cli/cobra/login.go`
- `cli/internal/service/auth/inbound/cli/cobra/logout.go`
- `cli/internal/service/auth/inbound/cli/cobra/status.go`

**Daemon Implementation**:
- `daemon/internal/service/auth/auth_service.go`

**Protobuf Definitions**:
- `idl/protobuf/auth/v1/auth.proto`

**Configuration**:
- `api/internal/platform/setup/config/config.go`
- `api/.meta/.env.example`
