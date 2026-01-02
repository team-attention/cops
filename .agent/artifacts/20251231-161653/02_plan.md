# Implementation Plan: User Authentication with Google OAuth

## Overview

This plan implements User and Account domain models with Google OAuth authentication for the C-Ops system. The implementation adds:

1. **Domain Models**: User (with embedded Accounts), Organization, OrganizationMember in the shared module
2. **Authentication**: Google OAuth 2.0 with JWT token generation/validation
3. **CLI Authentication**: Device flow OAuth with token storage
4. **Daemon Integration**: Shared authentication using CLI tokens

The implementation follows the existing hexagonal architecture patterns, ConnectRPC for gRPC, MongoDB for persistence, and fx/dig for dependency injection.

**Note**: Organization service implementation is out of scope for this plan. Only domain models and MongoSchema are defined for future use.

---

## Authentication Flow Architecture

### Flow Clarification

The authentication flow is designed as follows:

```
CLI -> C-Ops API -> Google OAuth -> C-Ops API -> JWT Tokens -> CLI
```

**How it works:**
1. CLI requests a device code from C-Ops API (not directly from Google)
2. C-Ops API acts as an intermediary and requests the device code from Google
3. User authenticates with Google in their browser
4. CLI polls C-Ops API for completion
5. When Google authentication completes, C-Ops API:
   - Receives Google tokens
   - Creates/updates user in C-Ops database
   - Generates C-Ops JWT tokens (separate from Google tokens)
6. CLI receives and stores C-Ops JWT tokens

**Important**: The CLI never directly communicates with Google. All OAuth communication goes through C-Ops API, which:
- Manages the Google OAuth client credentials
- Creates user records in the database
- Issues its own JWT tokens for API authentication

### Why This Architecture?

This design supports **multiple login providers** in the future:

1. **Provider abstraction**: The C-Ops API abstracts away the OAuth provider. CLI only knows about C-Ops authentication endpoints.
2. **Easy to extend**: Adding GitHub, Microsoft, or other OAuth providers only requires:
   - Adding a new OAuth adapter in the API
   - No CLI changes needed
3. **Consistent token format**: Regardless of the OAuth provider used, CLI always receives the same C-Ops JWT token format.

### Future: Web-Based Login Page

When a web-based login page is implemented:

```
CLI -> C-Ops API -> C-Ops Web Login Page -> (Google/GitHub/etc.) -> C-Ops API -> JWT
```

The CLI would:
1. Request a device code from C-Ops API
2. Display: "Open https://cops.example.com/device and enter code: ABCD-EFGH"
3. User logs in via C-Ops web page (which supports multiple OAuth providers)
4. CLI polls and receives tokens

**No CLI changes required** - the verification_url would simply point to C-Ops web instead of Google directly.

---

## Google OAuth Device Flow Explained

The Device Flow (RFC 8628) is designed for devices with limited input capabilities (like CLI tools) where users cannot easily enter credentials directly.

### How It Works

```
+----------+                                +----------------+                 +--------+
|          |  (1) Request Device Code       |                |                 |        |
|   CLI    | -----------------------------> |   C-Ops API    |                 | Google |
|          |                                |                |                 |        |
|          |  (2) Device Code + User Code   |                |                 |        |
|          | <----------------------------- |                |                 |        |
|          |                                |                |                 |        |
|          |  Display to user:              |                |                 |        |
|          |  "Go to: google.com/device"    |                |                 |        |
|          |  "Enter code: ABCD-EFGH"       |                |                 |        |
|          |                                |                |                 |        |
+----------+                                +----------------+                 +--------+
                                                   |
     User opens browser, enters code,              |  (3) API requests device code
     authenticates with Google                     |       from Google
                                                   |
+----------+                                +----------------+                 +--------+
|          |  (4) Poll for completion       |                |  (5) Poll      |        |
|   CLI    | -----------------------------> |   C-Ops API    | -------------> | Google |
|          |                                |                |                 |        |
|          |  (6) Pending / Tokens          |                |  (7) Tokens    |        |
|          | <----------------------------- |                | <------------- |        |
|          |                                |                |                 |        |
|          |  Save tokens to ~/.cops/auth.json              |                 |        |
|          |                                |                |                 |        |
+----------+                                +----------------+                 +--------+
```

### Step-by-Step CLI Authentication Flow

1. **User runs `cops auth login`**
   - CLI calls C-Ops API `DeviceCode` endpoint
   - API requests a device code from Google OAuth

2. **API returns device code info**
   - `device_code`: Internal code for polling (never shown to user)
   - `user_code`: Human-readable code like "ABCD-EFGH" (shown to user)
   - `verification_url`: URL where user enters the code
   - `interval`: How often to poll (typically 5 seconds)
   - `expires_in`: How long the code is valid (typically 30 minutes)

3. **CLI displays instructions**
   ```
   To sign in, open this URL in your browser:
     https://www.google.com/device

   Then enter this code:
     ABCD-EFGH

   Waiting for authentication...
   ```

4. **User authenticates in browser**
   - Opens the verification URL
   - Enters the user code
   - Signs in with Google account
   - Grants permission to the app

5. **CLI polls for completion**
   - Calls `DevicePoll` endpoint every `interval` seconds
   - If user hasn't completed: returns `pending: true`
   - If user completed: returns JWT tokens

6. **CLI saves tokens**
   - Stores tokens in `~/.cops/auth.json` with 0600 permissions
   - Displays success message with user info

### Why Device Flow?

- **No browser redirect needed**: Works in SSH sessions, containers, headless environments
- **No localhost server**: Doesn't require opening ports or running a local server
- **User-friendly**: Simple code entry is easier than pasting URLs
- **Secure**: Device code and user code are separate; only user code is displayed

---

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | JWT token generation and validation | `github.com/golang-jwt/jwt/v5` | Industry-standard Go JWT library with good community support and active maintenance |
| Add | Google OAuth2 integration | `golang.org/x/oauth2` | Official Google OAuth2 library for Go |
| Add | Google OAuth2 endpoints | `golang.org/x/oauth2/google` | Google-specific OAuth2 endpoints |

---

## Step 1: Define Domain Models in Shared Module

**Files to Read**:
- `.agent/rules/go/go-platform-domain.md`: Domain model guidelines
- `.agent/rules/go/go-struct.md`: Struct definition rules (pointer types for struct arrays)
- `shared/domain/common.go`: Existing ID type definition
- `shared/domain/project.go`: Example domain model pattern

### `shared/domain/account.go`

**Description**: Define Account domain model for OAuth provider accounts. Accounts are embedded in User.

```go
package domain

// AccountProvider represents supported OAuth providers.
type AccountProvider string

const (
    AccountProviderGoogle AccountProvider = "google"
)

// Account represents an OAuth provider account linked to a user.
// Accounts are embedded within the User document, not stored separately.
type Account struct {
    Provider   AccountProvider `json:"provider" bson:"provider"`
    ProviderID string          `json:"providerId" bson:"providerId"`
}
```

### `shared/domain/user.go`

**Description**: Define User domain model representing authenticated users with embedded accounts.

```go
package domain

// User represents an authenticated user in the system.
// A user can have multiple linked OAuth accounts embedded in the Accounts array.
type User struct {
    ID              ID         `json:"-" bson:"-"`
    Email           string     `json:"email" bson:"email"`
    Name            string     `json:"name" bson:"name"`
    ProfileImageURL string     `json:"profileImageUrl,omitempty" bson:"profileImageUrl,omitempty"`
    Accounts        []*Account `json:"accounts" bson:"accounts"`
}
```

### `shared/domain/organization.go`

**Description**: Define Organization and OrganizationMember domain models. Note: Organization service is out of scope; only domain models are defined.

```go
package domain

// MemberRole represents the role of a user within an organization.
type MemberRole string

const (
    MemberRoleAdmin  MemberRole = "admin"
    MemberRoleMember MemberRole = "member"
)

// Organization represents a group that owns projects.
type Organization struct {
    ID   ID     `json:"id" bson:"-"`
    Name string `json:"name" bson:"name"`
    Slug string `json:"slug" bson:"slug"`
}

// OrganizationMember represents membership relationship between user and organization.
type OrganizationMember struct {
    ID             ID         `json:"-" bson:"-"`
    OrganizationID ID         `json:"-" bson:"-"`
    UserID         ID         `json:"-" bson:"-"`
    Role           MemberRole `json:"role" bson:"role"`
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid MemberRole constants | `MemberRoleAdmin` | `"admin"` | Enum values |
| Valid MemberRole constants | `MemberRoleMember` | `"member"` | Enum values |
| Valid AccountProvider constants | `AccountProviderGoogle` | `"google"` | Enum values |

---

## Step 2: Define MongoSchema for Domain Models

**Files to Read**:
- `.agent/rules/go/go-platform-domain-mongoschema.md`: MongoSchema guidelines
- `shared/domain/mongoschema/project.go`: Example MongoSchema pattern

### `shared/domain/mongoschema/user.go`

**Description**: MongoDB schema for User with ID conversion. Accounts are embedded in the User document. Each struct type has its own field constants.

```go
package mongoschema

import (
    "github.com/team-attention/cops/shared/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    UserCollectionName = "users"
)

// User struct field constants
const (
    UserIDField              = "_id"
    UserEmailField           = "email"
    UserNameField            = "name"
    UserProfileImageURLField = "profileImageUrl"
    UserAccountsField        = "accounts"
)

// Account struct field constants (embedded type gets its own constants)
const (
    AccountProviderField   = "provider"
    AccountProviderIDField = "providerId"
)

type User struct {
    domain.User `bson:",inline"`
    ID          bson.ObjectID `bson:"_id,omitempty"`
}

func (s *User) FromDomain(d *domain.User) {
    // Implementation outline:
    // 1. Return early if d is nil.
    // 2. Copy the embedded User struct from domain.
    // 3. Convert domain.ID to bson.ObjectID if not empty.
}

func (s *User) ToDomain() *domain.User {
    // Implementation outline:
    // 1. Return nil if s is nil.
    // 2. Set s.User.ID from s.ID.Hex().
    // 3. Return pointer to embedded User.
}
```

### `shared/domain/mongoschema/organization.go`

**Description**: MongoDB schema for Organization with ID conversion. Note: Organization service is out of scope.

```go
package mongoschema

import (
    "github.com/team-attention/cops/shared/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    OrganizationCollectionName = "organizations"
)

const (
    OrganizationIDField   = "_id"
    OrganizationNameField = "name"
    OrganizationSlugField = "slug"
)

type Organization struct {
    domain.Organization `bson:",inline"`
    ID                  bson.ObjectID `bson:"_id,omitempty"`
}

func (s *Organization) FromDomain(d *domain.Organization) {
    // Implementation outline:
    // 1. Return early if d is nil.
    // 2. Copy the embedded Organization struct from domain.
    // 3. Convert d.ID to bson.ObjectID if not empty.
}

func (s *Organization) ToDomain() *domain.Organization {
    // Implementation outline:
    // 1. Return nil if s is nil.
    // 2. Set s.Organization.ID from s.ID.Hex().
    // 3. Return pointer to embedded Organization.
}
```

### `shared/domain/mongoschema/organization_member.go`

**Description**: MongoDB schema for OrganizationMember relationship. Note: Organization service is out of scope.

```go
package mongoschema

import (
    "github.com/team-attention/cops/shared/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    OrganizationMemberCollectionName = "organization_members"
)

const (
    OrganizationMemberIDField             = "_id"
    OrganizationMemberOrganizationIDField = "organizationId"
    OrganizationMemberUserIDField         = "userId"
    OrganizationMemberRoleField           = "role"
)

type OrganizationMember struct {
    domain.OrganizationMember `bson:",inline"`
    ID                        bson.ObjectID `bson:"_id,omitempty"`
    OrganizationID            bson.ObjectID `bson:"organizationId"`
    UserID                    bson.ObjectID `bson:"userId"`
}

func (s *OrganizationMember) FromDomain(d *domain.OrganizationMember) {
    // Implementation outline:
    // 1. Return early if d is nil.
    // 2. Copy the embedded OrganizationMember struct from domain.
    // 3. Convert d.ID to bson.ObjectID if not empty.
    // 4. Convert d.OrganizationID to bson.ObjectID if not empty.
    // 5. Convert d.UserID to bson.ObjectID if not empty.
}

func (s *OrganizationMember) ToDomain() *domain.OrganizationMember {
    // Implementation outline:
    // 1. Return nil if s is nil.
    // 2. Set s.OrganizationMember.ID from s.ID.Hex().
    // 3. Set s.OrganizationMember.OrganizationID from s.OrganizationID.Hex().
    // 4. Set s.OrganizationMember.UserID from s.UserID.Hex().
    // 5. Return pointer to embedded OrganizationMember.
}
```

---

## Step 3: Define Protobuf Services

**Files to Read**:
- `.agent/rules/idl/protobuf.md`: Protobuf conventions
- `idl/protobuf/project/v1/project.proto`: Example service definition

### `idl/protobuf/auth/v1/auth.proto`

**Description**: Authentication service protobuf definition. Simplified to return only tokens.

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1;authv1";

// TokenPair contains access and refresh tokens.
message TokenPair {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_at = 3;  // Unix timestamp when access token expires
}

// GoogleAuthReq contains Google OAuth authorization code.
message GoogleAuthReq {
  // authorization_code is the code received from Google OAuth callback
  string authorization_code = 1;
  // redirect_uri must match the URI used to obtain the code
  string redirect_uri = 2;
}

// GoogleAuthRes contains authentication result (tokens only).
message GoogleAuthRes {
  TokenPair tokens = 1;
}

// DeviceCodeReq initiates device flow authentication.
message DeviceCodeReq {}

// DeviceCodeRes contains device code for CLI authentication.
message DeviceCodeRes {
  string device_code = 1;
  string user_code = 2;
  string verification_url = 3;
  int32 expires_in = 4;
  int32 interval = 5;
}

// DevicePollReq polls for device authentication completion.
message DevicePollReq {
  string device_code = 1;
}

// DevicePollRes contains poll result (tokens only when complete).
message DevicePollRes {
  bool pending = 1;
  TokenPair tokens = 2;
}

// RefreshTokenReq contains refresh token for token renewal.
message RefreshTokenReq {
  string refresh_token = 1;
}

// RefreshTokenRes contains new token pair.
message RefreshTokenRes {
  TokenPair tokens = 1;
}

// AuthService handles authentication operations.
service AuthService {
  // GoogleAuth exchanges Google OAuth code for JWT tokens (web flow).
  rpc GoogleAuth(GoogleAuthReq) returns (GoogleAuthRes);

  // DeviceCode initiates device flow for CLI authentication.
  rpc DeviceCode(DeviceCodeReq) returns (DeviceCodeRes);

  // DevicePoll polls for device authentication completion.
  rpc DevicePoll(DevicePollReq) returns (DevicePollRes);

  // RefreshToken exchanges refresh token for new token pair.
  rpc RefreshToken(RefreshTokenReq) returns (RefreshTokenRes);
}
```

### Update `idl/protobuf/project/v1/project.proto`

**Description**: Add organization_id to project registration.

```protobuf
// Add to RegisterProjectReq message:
message RegisterProjectReq {
  // ... existing fields ...

  // organization_id is the organization this project belongs to
  string organization_id = 6;
}
```

---

## Step 4: Implement JWT Utility in API Module

**Files to Read**:
- `.agent/rules/go/go-platform.md`: Platform package guidelines
- `api/internal/platform/util/errutil/errutil.go`: Error utility pattern
- `api/internal/platform/setup/config/config.go`: Config location for JWT settings

### `api/internal/platform/util/jwtutil/jwtutil.go`

**Description**: JWT token generation and validation utilities. Uses only RegisteredClaims with UserID in the `sub` field.

```go
package jwtutil

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Config holds JWT configuration.
// This struct is populated from api/internal/platform/setup/config/config.go
type Config struct {
    SecretKey            string
    AccessTokenDuration  time.Duration
    RefreshTokenDuration time.Duration
    Issuer               string
}

// TokenPair contains access and refresh tokens.
type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

// GenerateTokenPair creates new access and refresh tokens.
// Uses only jwt.RegisteredClaims with UserID stored in Subject field.
func GenerateTokenPair(cfg *Config, userID string) (*TokenPair, error) {
    // Implementation outline:
    // 1. Calculate access token expiry time (now + AccessTokenDuration).
    // 2. Create access token claims using jwt.RegisteredClaims:
    //    - Subject: userID
    //    - Issuer: cfg.Issuer
    //    - IssuedAt: now
    //    - ExpiresAt: access token expiry
    // 3. Create JWT token with HS256 signing method.
    // 4. Sign access token with SecretKey.
    // 5. Calculate refresh token expiry time (now + RefreshTokenDuration).
    // 6. Create refresh token claims using jwt.RegisteredClaims:
    //    - Subject: userID
    //    - Issuer: cfg.Issuer
    //    - IssuedAt: now
    //    - ExpiresAt: refresh token expiry
    // 7. Sign refresh token with SecretKey.
    // 8. Return TokenPair with both tokens and access token expiry.
}

// ValidateAccessToken parses and validates an access token.
// Returns the userID from the Subject claim.
func ValidateAccessToken(cfg *Config, tokenString string) (string, error) {
    // Implementation outline:
    // 1. Create parser with jwt.WithValidMethods([]string{"HS256"}).
    // 2. Parse token string with jwt.RegisteredClaims.
    // 3. Provide key function that validates signing method is HMAC.
    // 4. If parsing fails, return appropriate error.
    // 5. Extract claims and return Subject (userID).
}

// ValidateRefreshToken parses and validates a refresh token.
// Returns the userID from the Subject claim.
func ValidateRefreshToken(cfg *Config, tokenString string) (string, error) {
    // Implementation outline:
    // 1. Create parser with jwt.WithValidMethods([]string{"HS256"}).
    // 2. Parse token string with jwt.RegisteredClaims.
    // 3. Validate signing method is HMAC.
    // 4. Return Subject (userID) if valid.
    // 5. Return appropriate error for expired, invalid, or malformed tokens.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Generate valid token pair | Valid userID | TokenPair with valid tokens | Happy path |
| Validate valid access token | Valid token string | userID string | Happy path |
| Validate expired access token | Expired token | Error: token expired | Error handling |
| Validate malformed token | Invalid string | Error: malformed token | Validation branch |
| Validate wrong signing method | RS256 signed token | Error: invalid signing method | Security branch |

---

## Step 5: Update API Configuration

**Files to Read**:
- `api/internal/platform/setup/config/config.go`: Existing config structure

### Update `api/internal/platform/setup/config/config.go`

**Description**: Add JWT and OAuth configuration to the main config file.

```go
// Add to Config struct:
type Config struct {
    // ... existing fields ...
    JWT   JWTConfig
    OAuth OAuthConfig
}

// JWTConfig holds JWT token configuration.
type JWTConfig struct {
    SecretKey            string        `env:"JWT_SECRET_KEY,required"`
    AccessTokenDuration  time.Duration `env:"JWT_ACCESS_TOKEN_DURATION" envDefault:"30m"`
    RefreshTokenDuration time.Duration `env:"JWT_REFRESH_TOKEN_DURATION" envDefault:"720h"` // 30 days
    Issuer               string        `env:"JWT_ISSUER" envDefault:"cops"`
}

// OAuthConfig holds OAuth provider configuration.
type OAuthConfig struct {
    GoogleClientID     string   `env:"GOOGLE_CLIENT_ID,required"`
    GoogleClientSecret string   `env:"GOOGLE_CLIENT_SECRET,required"`
    GoogleScopes       []string `env:"GOOGLE_SCOPES" envDefault:"email,profile"`
}
```

---

## Step 6: Implement Auth Middleware in API Module

**Files to Read**:
- `.agent/rules/go/go-inbound-http-fiber.md`: Fiber middleware patterns
- `api/internal/platform/setup/server/fiber.go`: Existing Fiber setup

### `api/internal/platform/middleware/auth.go`

**Description**: Authentication middleware for extracting and validating JWT from requests. Simplified to only extract UserID.

```go
package middleware

import (
    "log/slog"
    "strings"

    "github.com/gofiber/fiber/v2"

    "github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
    UserIDContextKey contextKey = "userId"
)

// AuthMiddleware creates a Fiber middleware for JWT authentication.
func AuthMiddleware(l *slog.Logger, jwtCfg *jwtutil.Config) fiber.Handler {
    // Implementation outline:
    // 1. Return fiber.Handler function.
    // 2. Extract Authorization header from request.
    // 3. If header is missing or doesn't start with "Bearer ", return 401.
    // 4. Extract token string after "Bearer " prefix.
    // 5. Call jwtutil.ValidateAccessToken with token.
    // 6. If validation fails, log error and return 401.
    // 7. Store userID in Fiber locals for handler access.
    // 8. Call c.Next() to continue request processing.
}

// OptionalAuthMiddleware extracts user ID if token is present, but doesn't require it.
func OptionalAuthMiddleware(l *slog.Logger, jwtCfg *jwtutil.Config) fiber.Handler {
    // Implementation outline:
    // 1. Return fiber.Handler function.
    // 2. Extract Authorization header from request.
    // 3. If header is missing, call c.Next() without setting context.
    // 4. If header present, validate token.
    // 5. If valid, store userID in Fiber locals.
    // 6. If invalid, log warning but continue without context.
    // 7. Call c.Next().
}

// GetUserID extracts userID from Fiber context.
func GetUserID(c *fiber.Ctx) string {
    // Implementation outline:
    // 1. Get value from Fiber locals using UserIDContextKey.
    // 2. Type assert to string.
    // 3. Return empty string if not found or wrong type.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid Bearer token | `Authorization: Bearer valid_token` | UserID set, next called | Happy path |
| Missing header | No Authorization header | 401 Unauthorized | Missing auth |
| Invalid token format | `Authorization: Basic token` | 401 Unauthorized | Format validation |
| Expired token | Expired Bearer token | 401 Unauthorized | Token validation |
| Optional with valid token | Valid Bearer token | UserID set | Optional happy path |
| Optional without token | No header | Next called, no context | Optional skip |

---

## Step 7: Implement Auth Service in API Module

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service implementation guidelines
- `api/internal/service/project/project_service.go`: Example service

### `api/internal/service/auth/auth_service.go`

**Description**: Core authentication service handling OAuth and token operations. Simplified to handle only user authentication without organization logic.

```go
package auth

import (
    "context"
    "log/slog"

    "github.com/team-attention/cops/api/internal/platform/util/jwtutil"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
    "github.com/team-attention/cops/shared/domain"
)

// GoogleAuthParams contains parameters for Google OAuth authentication.
type GoogleAuthParams struct {
    AuthorizationCode string
    RedirectURI       string
}

// DeviceCodeResult contains device code for CLI flow.
type DeviceCodeResult struct {
    DeviceCode      string
    UserCode        string
    VerificationURL string
    ExpiresIn       int
    Interval        int
}

// DevicePollResult contains result of device code polling.
type DevicePollResult struct {
    Pending bool
    Tokens  *jwtutil.TokenPair
}

// Service implements authentication business logic.
type Service struct {
    logger    *slog.Logger
    jwtCfg    *jwtutil.Config
    oauthPort oauth.GoogleOAuthPort
    userRepo  repository.UserRepositoryPort
}

// NewService creates a new auth service.
func NewService(
    l *slog.Logger,
    jwtCfg *jwtutil.Config,
    oauthPort oauth.GoogleOAuthPort,
    userRepo repository.UserRepositoryPort,
) *Service {
    return &Service{
        logger:    l.With(slog.String("name", "auth.service")),
        jwtCfg:    jwtCfg,
        oauthPort: oauthPort,
        userRepo:  userRepo,
    }
}

// GoogleAuth handles Google OAuth code exchange and user creation/lookup.
func (s *Service) GoogleAuth(ctx context.Context, params GoogleAuthParams) (*jwtutil.TokenPair, error) {
    // Implementation outline:
    // 1. Exchange authorization code for Google tokens via oauthPort.
    // 2. Fetch user info from Google using access token.
    // 3. Look up existing user by accounts.provider="google" and accounts.providerId=googleUserID.
    // 4. If user exists:
    //    a. Generate JWT token pair with userID.
    //    b. Return tokens.
    // 5. If user doesn't exist:
    //    a. Create new User with Google profile info.
    //    b. Add Account to user's Accounts array.
    //    c. Save user to database.
    //    d. Generate JWT token pair with new userID.
    //    e. Return tokens.
}

// DeviceCode initiates device flow authentication.
func (s *Service) DeviceCode(ctx context.Context) (*DeviceCodeResult, error) {
    // Implementation outline:
    // 1. Call oauthPort.InitiateDeviceFlow().
    // 2. Return DeviceCodeResult with device code, user code, verification URL.
}

// DevicePoll checks if device authentication is complete.
func (s *Service) DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error) {
    // Implementation outline:
    // 1. Call oauthPort.PollDeviceCode(deviceCode).
    // 2. If still pending (authorization_pending error), return DevicePollResult{Pending: true}.
    // 3. If complete:
    //    a. Fetch user info from Google using returned access token.
    //    b. Look up existing user by provider account.
    //    c. If not found, create new user with account.
    //    d. Generate JWT token pair.
    //    e. Return DevicePollResult with tokens.
}

// RefreshToken exchanges refresh token for new token pair.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*jwtutil.TokenPair, error) {
    // Implementation outline:
    // 1. Validate refresh token using jwtutil.ValidateRefreshToken.
    // 2. Extract userID from Subject claim.
    // 3. Fetch user by ID to ensure still valid.
    // 4. If user not found, return error.
    // 5. Generate new token pair with userID.
    // 6. Return new TokenPair.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| New user Google auth | Valid auth code, new Google account | Tokens, user created | New user path |
| Existing user Google auth | Valid auth code, existing account | Tokens | Existing user path |
| Invalid auth code | Invalid authorization code | Error: invalid code | OAuth error |
| Device code initiation | Empty request | Device code result | Happy path |
| Device poll pending | Device code not yet authorized | Pending=true | Pending state |
| Device poll complete | Authorized device code | Tokens | Complete state |
| Refresh valid token | Valid refresh token | New token pair | Happy path |
| Refresh expired token | Expired refresh token | Error: token expired | Expired token |
| Refresh with non-existent user | Valid token, deleted user | Error: user not found | User validation |

---

## Step 8: Implement Auth Repository Ports and Adapters

**Files to Read**:
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines
- `api/internal/service/project/outbound/repository/project_repo_port.go`: Example port
- `api/internal/service/project/outbound/repository/mongodb/project_repo.go`: Example adapter

### `api/internal/service/auth/outbound/repository/user_repo_port.go`

**Description**: User repository interface. Accounts are managed as embedded documents within User.

```go
package repository

import (
    "context"

    "github.com/team-attention/cops/shared/domain"
)

// UserRepositoryPort defines interface for user data persistence.
type UserRepositoryPort interface {
    // Create creates a new user with embedded accounts.
    Create(ctx context.Context, user *domain.User) (*domain.User, error)

    // GetByID retrieves user by ID.
    GetByID(ctx context.Context, userID string) (*domain.User, error)

    // FindByAccountProvider finds user by OAuth provider account.
    // Searches within the embedded accounts array.
    FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error)
}
```

### `api/internal/service/auth/outbound/oauth/google_port.go`

**Description**: Google OAuth interface.

```go
package oauth

import "context"

// GoogleUserInfo contains user profile from Google.
type GoogleUserInfo struct {
    ID            string
    Email         string
    Name          string
    Picture       string
    EmailVerified bool
}

// DeviceCodeResponse contains device code flow data.
type DeviceCodeResponse struct {
    DeviceCode      string
    UserCode        string
    VerificationURL string
    ExpiresIn       int
    Interval        int
}

// TokenResponse contains OAuth tokens from Google.
type TokenResponse struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
}

// GoogleOAuthPort defines interface for Google OAuth operations.
type GoogleOAuthPort interface {
    // ExchangeCode exchanges authorization code for tokens (web flow).
    ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)

    // GetUserInfo fetches user profile using access token.
    GetUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error)

    // InitiateDeviceFlow starts device code flow.
    InitiateDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error)

    // PollDeviceCode polls for device code authorization.
    // Returns nil TokenResponse if authorization is still pending.
    PollDeviceCode(ctx context.Context, deviceCode string) (*TokenResponse, error)
}
```

### `api/internal/service/auth/outbound/repository/mongodb/user_repo.go`

**Description**: MongoDB implementation of user repository. Users are stored with embedded accounts.

```go
package mongodb

import (
    "context"
    "log/slog"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"

    "github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
    "github.com/team-attention/cops/shared/domain"
    "github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoUserRepository struct {
    logger    *slog.Logger
    usersColl *mongo.Collection
}

func NewMongoUserRepository(l *slog.Logger, db *mongo.Database) *MongoUserRepository {
    return &MongoUserRepository{
        logger:    l.With(slog.String("name", "auth.repository.mongodb.user")),
        usersColl: db.Collection(mongoschema.UserCollectionName),
    }
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
    // Implementation outline:
    // 1. Create mongoschema.User from domain.User.
    // 2. Insert into users collection.
    // 3. Set user.ID from inserted ObjectID.
    // 4. Return user.
}

func (r *MongoUserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
    // Implementation outline:
    // 1. Convert userID to ObjectID.
    // 2. Find document by _id.
    // 3. Convert to domain.User using ToDomain().
    // 4. Return user or NotFoundError.
}

func (r *MongoUserRepository) FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error) {
    // Implementation outline:
    // 1. Build query to search embedded accounts array using field constants:
    //    filter := bson.M{
    //        mongoschema.UserAccountsField: bson.M{
    //            "$elemMatch": bson.M{
    //                mongoschema.AccountProviderField:   provider,
    //                mongoschema.AccountProviderIDField: providerID,
    //            },
    //        },
    //    }
    // 2. Find document matching the filter.
    // 3. If not found, return nil (not error).
    // 4. Convert to domain.User using ToDomain().
    // 5. Return user.
}

var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil)
```

### `api/internal/service/auth/outbound/oauth/google/google_oauth.go`

**Description**: Google OAuth implementation. OAuth config is injected from main config.

```go
package google

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"
    "net/url"
    "strings"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"

    "github.com/team-attention/cops/api/internal/platform/setup/config"
    oauthport "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
)

type GoogleOAuthAdapter struct {
    logger       *slog.Logger
    config       *oauth2.Config
    clientID     string
    clientSecret string
    httpClient   *http.Client
}

func NewGoogleOAuthAdapter(l *slog.Logger, cfg *config.Config) *GoogleOAuthAdapter {
    return &GoogleOAuthAdapter{
        logger: l.With(slog.String("name", "auth.oauth.google")),
        config: &oauth2.Config{
            ClientID:     cfg.OAuth.GoogleClientID,
            ClientSecret: cfg.OAuth.GoogleClientSecret,
            Scopes:       cfg.OAuth.GoogleScopes,
            Endpoint:     google.Endpoint,
        },
        clientID:     cfg.OAuth.GoogleClientID,
        clientSecret: cfg.OAuth.GoogleClientSecret,
        httpClient:   http.DefaultClient,
    }
}

func (a *GoogleOAuthAdapter) ExchangeCode(ctx context.Context, code, redirectURI string) (*oauthport.TokenResponse, error) {
    // Implementation outline:
    // 1. Create copy of config with provided redirect URI.
    // 2. Call config.Exchange(ctx, code).
    // 3. Convert oauth2.Token to TokenResponse.
    // 4. Return TokenResponse.
}

func (a *GoogleOAuthAdapter) GetUserInfo(ctx context.Context, accessToken string) (*oauthport.GoogleUserInfo, error) {
    // Implementation outline:
    // 1. Create request to https://www.googleapis.com/oauth2/v2/userinfo.
    // 2. Set Authorization: Bearer accessToken header.
    // 3. Execute request.
    // 4. Parse JSON response into GoogleUserInfo.
    // 5. Return user info.
}

func (a *GoogleOAuthAdapter) InitiateDeviceFlow(ctx context.Context) (*oauthport.DeviceCodeResponse, error) {
    // Implementation outline:
    // 1. Build form data with client_id and scope.
    // 2. POST to https://oauth2.googleapis.com/device/code.
    // 3. Parse JSON response containing:
    //    - device_code
    //    - user_code
    //    - verification_url
    //    - expires_in
    //    - interval
    // 4. Return DeviceCodeResponse.
}

func (a *GoogleOAuthAdapter) PollDeviceCode(ctx context.Context, deviceCode string) (*oauthport.TokenResponse, error) {
    // Implementation outline:
    // 1. Build form data with:
    //    - client_id
    //    - client_secret
    //    - device_code
    //    - grant_type=urn:ietf:params:oauth:grant-type:device_code
    // 2. POST to https://oauth2.googleapis.com/token.
    // 3. Parse response.
    // 4. If error "authorization_pending", return nil (still pending).
    // 5. If error "slow_down", return nil (still pending, caller should increase interval).
    // 6. If error "access_denied" or "expired_token", return error.
    // 7. If success, return TokenResponse with access_token.
}

var _ oauthport.GoogleOAuthPort = (*GoogleOAuthAdapter)(nil)
```

---

## Step 9: Implement Auth ConnectRPC Handler

**Files to Read**:
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler guidelines
- `api/internal/service/project/inbound/grpc/connectrpc/handler.go`: Example handler

### `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**: ConnectRPC handler for auth service. Simplified responses with tokens only.

```go
package connectrpc

import (
    "context"
    "log/slog"
    "net/http"

    "connectrpc.com/connect"

    authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
    "github.com/team-attention/cops/api/internal/service/auth"
)

type AuthGRPCHandler struct {
    svc    *auth.Service
    logger *slog.Logger
}

func NewAuthGRPCHandler(l *slog.Logger, svc *auth.Service) *AuthGRPCHandler {
    return &AuthGRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "auth.grpc.connectrpc")),
    }
}

// GetHandler implements ConnectHandler interface.
func (h *AuthGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
    return authv1connect.NewAuthServiceHandler(h, opts...)
}

func (h *AuthGRPCHandler) GoogleAuth(
    ctx context.Context,
    req *connect.Request[authv1.GoogleAuthReq],
) (*connect.Response[authv1.GoogleAuthRes], error) {
    // Implementation outline:
    // 1. Extract params from request message.
    // 2. Call h.svc.GoogleAuth(ctx, params).
    // 3. Convert TokenPair to protobuf TokenPair.
    // 4. Return connect.NewResponse with tokens only.
}

func (h *AuthGRPCHandler) DeviceCode(
    ctx context.Context,
    req *connect.Request[authv1.DeviceCodeReq],
) (*connect.Response[authv1.DeviceCodeRes], error) {
    // Implementation outline:
    // 1. Call h.svc.DeviceCode(ctx).
    // 2. Convert result to protobuf response.
    // 3. Return connect.NewResponse(res).
}

func (h *AuthGRPCHandler) DevicePoll(
    ctx context.Context,
    req *connect.Request[authv1.DevicePollReq],
) (*connect.Response[authv1.DevicePollRes], error) {
    // Implementation outline:
    // 1. Extract device code from request.
    // 2. Call h.svc.DevicePoll(ctx, deviceCode).
    // 3. Build response with pending flag and tokens (if complete).
    // 4. Return connect.NewResponse(res).
}

func (h *AuthGRPCHandler) RefreshToken(
    ctx context.Context,
    req *connect.Request[authv1.RefreshTokenReq],
) (*connect.Response[authv1.RefreshTokenRes], error) {
    // Implementation outline:
    // 1. Extract refresh token from request.
    // 2. Call h.svc.RefreshToken(ctx, refreshToken).
    // 3. Convert result to protobuf response.
    // 4. Return connect.NewResponse(res).
}

var _ authv1connect.AuthServiceHandler = (*AuthGRPCHandler)(nil)
```

---

## Step 10: Register Auth Module in API Container

**Files to Read**:
- `api/cmd/internal/container/module_project.go`: Example module registration
- `api/cmd/internal/container/application.go`: Application composition

### `api/cmd/internal/container/module_auth.go`

**Description**: Register auth service and handlers.

```go
package container

import (
    "go.uber.org/fx"

    "github.com/team-attention/cops/api/internal/platform/setup/config"
    "github.com/team-attention/cops/api/internal/platform/util/jwtutil"
    "github.com/team-attention/cops/api/internal/service/auth"
    "github.com/team-attention/cops/api/internal/service/auth/inbound/grpc/connectrpc"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth/google"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
    "github.com/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb"
)

func newAuthModule() fx.Option {
    return fx.Module("auth",
        // JWT config from main config
        fx.Provide(func(cfg *config.Config) *jwtutil.Config {
            return &jwtutil.Config{
                SecretKey:            cfg.JWT.SecretKey,
                AccessTokenDuration:  cfg.JWT.AccessTokenDuration,
                RefreshTokenDuration: cfg.JWT.RefreshTokenDuration,
                Issuer:               cfg.JWT.Issuer,
            }
        }),

        // OAuth adapter (config injected via constructor)
        fx.Provide(
            fx.Annotate(
                google.NewGoogleOAuthAdapter,
                fx.As(new(oauth.GoogleOAuthPort)),
            ),
        ),

        // User repository
        fx.Provide(
            fx.Annotate(
                mongodb.NewMongoUserRepository,
                fx.As(new(repository.UserRepositoryPort)),
            ),
        ),

        // Service
        fx.Provide(auth.NewService),

        // ConnectRPC handler
        fx.Provide(
            fx.Annotate(
                connectrpc.NewAuthGRPCHandler,
                fx.As(new(ConnectHandler)),
                fx.ResultTags(`group:"connect_handlers"`),
            ),
        ),
    )
}
```

### Update `api/cmd/internal/container/application.go`

**Description**: Add auth module to application.

```go
// Add to fx.New():
fx.New(
    // ... existing modules ...
    newAuthModule(),
    // ...
)
```

---

## Step 11: Implement CLI Auth Commands

**Files to Read**:
- `.agent/rules/go/go-dig-container.md`: Dig container guidelines
- `cli/internal/service/tracking/inbound/cli/cobra/handler.go`: Example CLI handler
- `cli/internal/platform/setup/config/config.go`: CLI config structure

### `cli/internal/service/auth/auth_service.go`

**Description**: CLI auth service for managing local authentication state. Stores only tokens.

```go
package auth

import (
    "context"
    "encoding/json"
    "log/slog"
    "os"
    "path/filepath"
    "time"

    "github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
)

// AuthState represents the local authentication state.
// Stores only token information.
type AuthState struct {
    Tokens *TokenInfo `json:"tokens"`
}

// TokenInfo contains token data.
type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    int64  `json:"expiresAt"` // Unix timestamp
}

// Service implements CLI authentication logic.
type Service struct {
    logger    *slog.Logger
    apiClient api.AuthAPIPort
    authPath  string
}

// NewService creates a new CLI auth service.
func NewService(l *slog.Logger, apiClient api.AuthAPIPort, homeDir string) *Service {
    return &Service{
        logger:    l.With(slog.String("name", "auth.service")),
        apiClient: apiClient,
        authPath:  filepath.Join(homeDir, ".cops", "auth.json"),
    }
}

// LoginResult contains the result of login flow for display.
type LoginResult struct {
    DeviceCode      string
    UserCode        string
    VerificationURL string
    Interval        int
}

// InitiateLogin starts the device flow and returns display info.
func (s *Service) InitiateLogin(ctx context.Context) (*LoginResult, error) {
    // Implementation outline:
    // 1. Call apiClient.DeviceCode(ctx) to get device code.
    // 2. Return LoginResult with device code info for display.
}

// PollLogin polls for authentication completion.
func (s *Service) PollLogin(ctx context.Context, deviceCode string) (bool, error) {
    // Implementation outline:
    // 1. Call apiClient.DevicePoll(ctx, deviceCode).
    // 2. If pending, return false, nil.
    // 3. If complete:
    //    a. Create AuthState with tokens.
    //    b. Save to auth.json with 0600 permissions.
    //    c. Return true, nil.
}

// Logout removes the local authentication state.
func (s *Service) Logout(ctx context.Context) error {
    // Implementation outline:
    // 1. Check if auth.json exists.
    // 2. If exists, remove the file.
    // 3. Log success.
}

// GetAuthState returns the current authentication state.
func (s *Service) GetAuthState() (*AuthState, error) {
    // Implementation outline:
    // 1. Check if auth.json exists.
    // 2. If not exists, return nil (not logged in).
    // 3. Read and parse auth.json.
    // 4. Return AuthState.
}

// IsLoggedIn checks if user is currently logged in.
func (s *Service) IsLoggedIn() bool {
    // Implementation outline:
    // 1. Try to get auth state.
    // 2. Return true if state exists with valid tokens.
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (s *Service) GetAccessToken(ctx context.Context) (string, error) {
    // Implementation outline:
    // 1. Get current auth state.
    // 2. If not logged in, return error.
    // 3. Check if token is expired or will expire in <5 minutes.
    // 4. If near expiry:
    //    a. Call apiClient.RefreshToken with refresh token.
    //    b. Update auth.json with new tokens.
    // 5. Return access token.
}

// saveAuthState writes auth state to file with secure permissions.
func (s *Service) saveAuthState(state *AuthState) error {
    // Implementation outline:
    // 1. Ensure ~/.cops directory exists (0700 permissions).
    // 2. Marshal state to JSON with indentation.
    // 3. Write to auth.json with os.WriteFile and 0600 permissions.
}
```

### `cli/internal/service/auth/outbound/api/auth_port.go`

**Description**: API client interface for auth operations. Simplified responses.

```go
package api

import "context"

// DeviceCodeResult contains device code flow data.
type DeviceCodeResult struct {
    DeviceCode      string
    UserCode        string
    VerificationURL string
    ExpiresIn       int
    Interval        int
}

// PollResult contains poll result.
type PollResult struct {
    Pending      bool
    AccessToken  string
    RefreshToken string
    ExpiresAt    int64
}

// TokenResult contains new tokens.
type TokenResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    int64
}

// AuthAPIPort defines the interface for auth API operations.
type AuthAPIPort interface {
    // DeviceCode initiates device flow.
    DeviceCode(ctx context.Context) (*DeviceCodeResult, error)

    // DevicePoll polls for device code completion.
    DevicePoll(ctx context.Context, deviceCode string) (*PollResult, error)

    // RefreshToken refreshes the access token.
    RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)
}
```

### `cli/internal/service/auth/outbound/api/connectrpc/auth_client.go`

**Description**: ConnectRPC client implementation for auth API.

```go
package connectrpc

import (
    "context"
    "log/slog"
    "net/http"

    "connectrpc.com/connect"

    "github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
    authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

type AuthAPIClient struct {
    logger *slog.Logger
    client authv1connect.AuthServiceClient
}

func NewAuthAPIClient(l *slog.Logger, httpClient *http.Client, baseURL string) *AuthAPIClient {
    return &AuthAPIClient{
        logger: l.With(slog.String("name", "auth.api.connectrpc")),
        client: authv1connect.NewAuthServiceClient(httpClient, baseURL),
    }
}

func (c *AuthAPIClient) DeviceCode(ctx context.Context) (*api.DeviceCodeResult, error) {
    // Implementation outline:
    // 1. Create DeviceCodeReq.
    // 2. Call client.DeviceCode(ctx, connect.NewRequest(req)).
    // 3. Convert response to DeviceCodeResult.
    // 4. Return result.
}

func (c *AuthAPIClient) DevicePoll(ctx context.Context, deviceCode string) (*api.PollResult, error) {
    // Implementation outline:
    // 1. Create DevicePollReq with device code.
    // 2. Call client.DevicePoll(ctx, connect.NewRequest(req)).
    // 3. Convert response to PollResult.
    // 4. Return result.
}

func (c *AuthAPIClient) RefreshToken(ctx context.Context, refreshToken string) (*api.TokenResult, error) {
    // Implementation outline:
    // 1. Create RefreshTokenReq with refresh token.
    // 2. Call client.RefreshToken(ctx, connect.NewRequest(req)).
    // 3. Convert response to TokenResult.
    // 4. Return result.
}

var _ api.AuthAPIPort = (*AuthAPIClient)(nil)
```

### `cli/internal/service/auth/inbound/cli/cobra/handler.go`

**Description**: CLI command handler for auth.

```go
package cobra

import (
    "log/slog"

    "github.com/spf13/cobra"

    "github.com/team-attention/cops/cli/internal/service/auth"
)

// AuthCLIHandler handles auth CLI commands.
type AuthCLIHandler struct {
    logger *slog.Logger
    svc    *auth.Service
}

// NewAuthCLIHandler creates a new auth CLI handler.
func NewAuthCLIHandler(l *slog.Logger, svc *auth.Service) *AuthCLIHandler {
    return &AuthCLIHandler{
        logger: l.With(slog.String("name", "auth.cli.cobra")),
        svc:    svc,
    }
}

// Commands implements CLICommandProvider interface.
func (h *AuthCLIHandler) Commands() []*cobra.Command {
    authCmd := &cobra.Command{
        Use:   "auth",
        Short: "Manage authentication",
    }

    authCmd.AddCommand(
        h.NewLoginCommand(),
        h.NewLogoutCommand(),
        h.NewStatusCommand(),
    )

    return []*cobra.Command{authCmd}
}
```

### `cli/internal/service/auth/inbound/cli/cobra/login.go`

**Description**: Login command implementation.

```go
package cobra

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
)

// NewLoginCommand creates the login command.
func (h *AuthCLIHandler) NewLoginCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "login",
        Short: "Log in with Google OAuth",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation outline:
            // 1. Create context with 10-minute timeout.
            // 2. Call svc.InitiateLogin(ctx) to get device code.
            // 3. Display instructions:
            //    fmt.Println("To sign in, open this URL in your browser:")
            //    fmt.Printf("  %s\n\n", result.VerificationURL)
            //    fmt.Println("Then enter this code:")
            //    fmt.Printf("  %s\n\n", result.UserCode)
            //    fmt.Println("Waiting for authentication...")
            // 4. Start polling loop:
            //    - Call svc.PollLogin(ctx, deviceCode) every interval seconds
            //    - If returns true, authentication complete
            //    - If context timeout, show error
            // 5. On success:
            //    fmt.Println("\nAuthentication successful!")
        },
    }
}
```

### `cli/internal/service/auth/inbound/cli/cobra/logout.go`

**Description**: Logout command implementation.

```go
package cobra

import "github.com/spf13/cobra"

// NewLogoutCommand creates the logout command.
func (h *AuthCLIHandler) NewLogoutCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "logout",
        Short: "Log out and remove stored credentials",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation outline:
            // 1. Call svc.Logout(cmd.Context()).
            // 2. If error, display error message.
            // 3. Display: "Logged out successfully"
        },
    }
}
```

### `cli/internal/service/auth/inbound/cli/cobra/status.go`

**Description**: Status command implementation.

```go
package cobra

import (
    "fmt"

    "github.com/spf13/cobra"
)

// NewStatusCommand creates the status command.
func (h *AuthCLIHandler) NewStatusCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show current authentication status",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation outline:
            // 1. Call svc.IsLoggedIn().
            // 2. If not logged in:
            //    fmt.Println("Not logged in")
            //    fmt.Println("Run 'cops auth login' to authenticate")
            // 3. If logged in:
            //    fmt.Println("Logged in")
            //    // Optionally show token expiry info
        },
    }
}
```

---

## Step 12: Update CLI to Include Auth Token in Requests

**Files to Read**:
- `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`: Example API client
- `cli/internal/platform/setup/httpclient/httpclient.go`: HTTP client setup (uses `imroc/req/v3`)

### Update CLI HTTP Client

**Description**: Use `imroc/req/v3` middleware to add auth token when making API calls. Auth token is passed at call time, not injected into client initialization.

```go
// cli/internal/platform/setup/httpclient/httpclient.go

import (
    "net/http"

    "github.com/imroc/req/v3"

    "github.com/team-attention/cops/cli/internal/platform/setup/config"
)

// APIHTTPClient is an HTTP client configured for the API server.
type APIHTTPClient struct {
    *req.Client
}

// InitAPIHTTPClient creates a new HTTP client for the API server.
// Note: Auth token is NOT injected here - it's added per-request by services.
func InitAPIHTTPClient(cfg *config.Config) *APIHTTPClient {
    client := req.C().
        SetBaseURL(cfg.API.URL).
        SetTimeout(cfg.API.Timeout)

    return &APIHTTPClient{Client: client}
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIHTTPClient) StandardHTTPClient() *http.Client {
    return c.Client.GetClient()
}

// WithAuth returns a cloned client with auth header set for authenticated requests.
// This should be called by service logic that needs authentication.
func (c *APIHTTPClient) WithAuth(accessToken string) *req.Client {
    // Implementation outline:
    // 1. Clone the underlying req.Client.
    // 2. Set Authorization header: "Bearer " + accessToken
    // 3. Return the cloned client.
    return c.Client.Clone().SetCommonBearerAuthToken(accessToken)
}
```

### Service Usage Pattern

When a service needs to make an authenticated request:

```go
// In service that needs auth:
func (s *SomeService) DoAuthenticatedCall(ctx context.Context) error {
    // 1. Get access token from auth service
    token, err := s.authSvc.GetAccessToken(ctx)
    if err != nil {
        return err
    }

    // 2. Create authenticated client for this request
    authClient := s.httpClient.WithAuth(token)

    // 3. Make the request
    // ...
}
```

---

## Step 13: Update Daemon for Authentication

**Files to Read**:
- `daemon/internal/platform/setup/copsapi.go`: Existing API client setup (uses `imroc/req/v3`)
- `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`: API client

### `daemon/internal/service/auth/auth_service.go`

**Description**: Daemon auth service that reads CLI auth state.

```go
package auth

import (
    "encoding/json"
    "log/slog"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// AuthState mirrors CLI auth state structure.
type AuthState struct {
    Tokens *TokenInfo `json:"tokens"`
}

type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    int64  `json:"expiresAt"`
}

// Service manages daemon authentication state.
type Service struct {
    logger      *slog.Logger
    authPath    string
    mu          sync.RWMutex
    cachedState *AuthState
    lastLoad    time.Time
}

// NewService creates a new daemon auth service.
func NewService(l *slog.Logger, homeDir string) *Service {
    return &Service{
        logger:   l.With(slog.String("name", "daemon.auth.service")),
        authPath: filepath.Join(homeDir, ".cops", "auth.json"),
    }
}

// GetAccessToken returns current access token, reloading from file if stale.
func (s *Service) GetAccessToken() (string, error) {
    // Implementation outline:
    // 1. Lock for reading.
    // 2. If cache is older than 30 seconds, reload from file.
    // 3. If file doesn't exist, return error (not authenticated).
    // 4. Parse auth.json.
    // 5. Check if token is expired (compare ExpiresAt with now).
    // 6. If expired, log warning and return error.
    // 7. Return access token.
}

// IsAuthenticated checks if valid auth state exists.
func (s *Service) IsAuthenticated() bool {
    // Implementation outline:
    // 1. Try to get access token.
    // 2. Return true if successful, false otherwise.
}
```

### Daemon API Client

**Description**: API client initialization remains simple. Auth is handled per-request by services.

```go
// daemon/internal/platform/setup/copsapi.go

import (
    "net/http"

    "github.com/imroc/req/v3"
)

// APIClient is an HTTP client configured for the COps API server.
type APIClient struct {
    *req.Client
}

// InitAPIClient creates a new HTTP client for the COps API server.
// Note: Auth token is NOT injected here - it's added per-request by services.
func InitAPIClient(cfg *Config) *APIClient {
    client := req.C().
        SetBaseURL(cfg.API.URL).
        SetTimeout(cfg.API.Timeout)

    return &APIClient{Client: client}
}

// StandardHTTPClient returns an http.Client that can be used with libraries
// expecting the standard http.Client interface.
func (c *APIClient) StandardHTTPClient() *http.Client {
    return c.Client.GetClient()
}

// WithAuth returns a cloned client with auth header set.
func (c *APIClient) WithAuth(accessToken string) *req.Client {
    return c.Client.Clone().SetCommonBearerAuthToken(accessToken)
}
```

### Service Usage in Daemon

When daemon services need authenticated requests:

```go
// In daemon service that needs auth:
func (s *LogWatcherService) SendRecords(ctx context.Context, records []Record) error {
    // 1. Get access token - authentication is REQUIRED
    token, err := s.authSvc.GetAccessToken()
    if err != nil {
        // Return error - do NOT continue without authentication
        return fmt.Errorf("authentication required: %w", err)
    }

    // 2. Use authenticated client
    authClient := s.apiClient.WithAuth(token)
    return s.sendWithClient(ctx, authClient, records)
}
```

**Important**: Authentication is mandatory for daemon operation. If the user is not logged in via CLI (`cops auth login`), the daemon will fail to send records. This is intentional - unauthenticated requests are not allowed.

---

## Step 14: Update Project Registration to Require Organization

**Files to Read**:
- `api/internal/service/project/project_service.go`: Existing project service
- `api/internal/service/project/outbound/repository/project_repo_port.go`: Repository interface

### Update Project Repository Interface

**Description**: Add organization_id to project operations.

```go
// api/internal/service/project/outbound/repository/project_repo_port.go

// Update FindOrCreateParams:
type FindOrCreateParams struct {
    ConfiguredURL  string
    ActualURL      string
    ExistingID     string
    Name           string
    IsGitProject   bool
    OrganizationID string // Add organization ID
}

// Update FindOrCreateResult:
type FindOrCreateResult struct {
    ProjectID      string
    IsNew          bool
    Name           string
    IsGitProject   bool
    OrganizationID string // Add organization ID
}
```

### Update Project Service

**Description**: Include organization ID in project operations.

```go
// api/internal/service/project/project_service.go

// Update RegisterProjectParams:
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    Name                string
    IsGitProject        bool
    OrganizationID      string // Add organization ID
}

// Update RegisterProject to pass organization ID:
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
    // Implementation outline:
    // 1. Add organization ID to find/create params.
    // 2. Call repository.
    // 3. Return result.
}
```

---

## Step 15: Generate Protobuf Code

**Description**: Generate Go code from new protobuf definitions.

```bash
cd idl/protobuf && buf generate
```

This generates:
- `shared/gen/grpcstub/auth/v1/auth.pb.go`
- `shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go`

---

## Implementation Order

1. **Step 1-2**: Domain models and MongoSchema (shared module)
2. **Step 3**: Protobuf definitions and code generation (Step 15)
3. **Step 4-5**: JWT utility and API configuration
4. **Step 6**: Auth middleware (API module)
5. **Step 7-9**: Auth service, repositories, OAuth adapter, and handlers (API module)
6. **Step 10**: API module registration
7. **Step 14**: Project updates
8. **Step 11-12**: CLI auth commands and token integration
9. **Step 13**: Daemon auth integration

---

## Testing Strategy

### Unit Tests

1. **JWT Utility**: Token generation, validation, expiry handling
2. **Domain Models**: Enum constants, ID type conversion
3. **MongoSchema**: FromDomain/ToDomain conversions
4. **Services**: Business logic with mocked repositories

### Integration Tests

1. **Auth Flow**: Full OAuth code exchange with mock Google API
2. **User Repository**: CRUD operations with embedded accounts
3. **Project Registration**: Organization ID inclusion

### E2E Tests

1. **CLI Login Flow**: Device code display and polling
2. **Daemon Authentication**: Token reading and header injection

---

## Critical Implementation Notes

1. **Token Security**: Never log tokens. Use secure file permissions (0600) for auth.json.

2. **Error Messages**: Authentication errors should not expose internal details. Use generic messages like "Authentication failed".

3. **Token Refresh**: CLI should proactively refresh tokens before expiry (5-minute buffer). Daemon requires valid authentication - if token is expired or missing, operations will fail with an error.

4. **JWT Claims**: Use only `jwt.RegisteredClaims` with UserID in the `Subject` field. Do not include organization ID in tokens since users can belong to multiple organizations.

5. **Embedded Accounts**: Accounts are embedded within User documents, not stored separately. Use `$elemMatch` for querying by provider account.

6. **Database Indexes**: See `TODO.md` for index requirements. Indexes are managed via migration scripts, NOT in Go codebase.

7. **Device Flow Polling**: Use the interval returned by Google. Handle `slow_down` error by increasing interval. Respect `expires_in` timeout.

8. **Organization Service**: Organization service implementation is out of scope. Only domain models and MongoSchema are defined for future use.

9. **Struct Arrays**: When structs are used as array elements, ALWAYS use pointer types (e.g., `[]*Account` not `[]Account`).

10. **Context Keys**: Use camelCase for context key names (e.g., `userId` not `user_id`).

11. **Field Constants**: Each struct type gets its own set of field constants with that type as prefix. Field constant names and values should NOT contain dots. Example: `UserAccountsField = "accounts"` for User struct, `AccountProviderField = "provider"` for Account struct.

12. **OAuth Config**: OAuth configuration (client ID, secret, scopes) is defined in `api/internal/platform/setup/config/config.go` and injected into adapters via constructors.

13. **HTTP Client**: Use `imroc/req/v3` package for HTTP clients. Auth tokens should be added per-request by services using `WithAuth()`, NOT injected into client initialization.

14. **API Client Initialization**: Do NOT inject auth service into API client setup. Keep initialization simple. Services that need authentication should get tokens from auth service and pass them when making requests.

15. **Daemon Authentication**: Authentication is MANDATORY for daemon operation. If the user is not logged in via CLI (`cops auth login`), the daemon will fail to send records with an error. Do NOT fall back to unauthenticated requests.
