# Implementation Plan: User and Organization Management with Authentication

## Overview

This plan implements User, Account, and Organization domain models with Google OAuth authentication for the C-Ops system. The implementation adds:

1. **Domain Models**: User, Account, Organization, OrganizationMember in the shared module
2. **Authentication**: Google OAuth 2.0 with JWT token generation/validation
3. **Authorization**: Role-based access control (Owner, Admin, Member) for organizations
4. **CLI Authentication**: Device flow OAuth with token storage
5. **Daemon Integration**: Shared authentication using CLI tokens

The implementation follows the existing hexagonal architecture patterns, ConnectRPC for gRPC, MongoDB for persistence, and fx/dig for dependency injection.

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
- `shared/domain/common.go`: Existing ID type definition
- `shared/domain/project.go`: Example domain model pattern

### `shared/domain/user.go`

**Description**: Define User domain model representing authenticated users.

```go
package domain

// User represents an authenticated user in the system.
// A user can have multiple linked OAuth accounts.
type User struct {
    ID              ID     `json:"-" bson:"-"`
    Email           string `json:"email" bson:"email"`
    Name            string `json:"name" bson:"name"`
    ProfileImageURL string `json:"profileImageUrl,omitempty" bson:"profileImageUrl,omitempty"`
}
```

### `shared/domain/account.go`

**Description**: Define Account domain model for OAuth provider accounts.

```go
package domain

// AccountProvider represents supported OAuth providers.
type AccountProvider string

const (
    AccountProviderGoogle AccountProvider = "google"
)

// Account represents an OAuth provider account linked to a user.
// Separating accounts allows future multi-provider support.
type Account struct {
    ID         ID              `json:"-" bson:"-"`
    Provider   AccountProvider `json:"provider" bson:"provider"`
    ProviderID string          `json:"providerId" bson:"providerId"`
    UserID     ID              `json:"-" bson:"-"`
}
```

### `shared/domain/organization.go`

**Description**: Define Organization and OrganizationMember domain models.

```go
package domain

// MemberRole represents the role of a user within an organization.
type MemberRole string

const (
    MemberRoleOwner  MemberRole = "owner"
    MemberRoleAdmin  MemberRole = "admin"
    MemberRoleMember MemberRole = "member"
)

// Organization represents a group that owns projects.
type Organization struct {
    ID      ID     `json:"id" bson:"-"`
    Name    string `json:"name" bson:"name"`
    Slug    string `json:"slug" bson:"slug"`
    OwnerID ID     `json:"-" bson:"-"`
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
| Valid MemberRole constants | `MemberRoleOwner` | `"owner"` | Enum values |
| Valid AccountProvider constants | `AccountProviderGoogle` | `"google"` | Enum values |

---

## Step 2: Define MongoSchema for Domain Models

**Files to Read**:
- `.agent/rules/go/go-platform-domain-mongoschema.md`: MongoSchema guidelines
- `shared/domain/mongoschema/project.go`: Example MongoSchema pattern

### `shared/domain/mongoschema/user.go`

**Description**: MongoDB schema for User with ID conversion.

```go
package mongoschema

import (
    "github.com/team-attention/cops/shared/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    UserCollectionName = "users"
)

const (
    UserIDField              = "_id"
    UserEmailField           = "email"
    UserNameField            = "name"
    UserProfileImageURLField = "profileImageUrl"
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

### `shared/domain/mongoschema/account.go`

**Description**: MongoDB schema for Account with ID conversions.

```go
package mongoschema

import (
    "github.com/team-attention/cops/shared/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    AccountCollectionName = "accounts"
)

const (
    AccountIDField         = "_id"
    AccountProviderField   = "provider"
    AccountProviderIDField = "providerId"
    AccountUserIDField     = "userId"
)

type Account struct {
    domain.Account `bson:",inline"`
    ID             bson.ObjectID `bson:"_id,omitempty"`
    UserID         bson.ObjectID `bson:"userId"`
}

func (s *Account) FromDomain(d *domain.Account) {
    // Implementation outline:
    // 1. Return early if d is nil.
    // 2. Copy the embedded Account struct from domain.
    // 3. Convert d.ID to bson.ObjectID if not empty.
    // 4. Convert d.UserID to bson.ObjectID if not empty.
}

func (s *Account) ToDomain() *domain.Account {
    // Implementation outline:
    // 1. Return nil if s is nil.
    // 2. Set s.Account.ID from s.ID.Hex().
    // 3. Set s.Account.UserID from s.UserID.Hex().
    // 4. Return pointer to embedded Account.
}
```

### `shared/domain/mongoschema/organization.go`

**Description**: MongoDB schema for Organization with ID conversion.

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
    OrganizationIDField      = "_id"
    OrganizationNameField    = "name"
    OrganizationSlugField    = "slug"
    OrganizationOwnerIDField = "ownerId"
)

type Organization struct {
    domain.Organization `bson:",inline"`
    ID                  bson.ObjectID `bson:"_id,omitempty"`
    OwnerID             bson.ObjectID `bson:"ownerId"`
}

func (s *Organization) FromDomain(d *domain.Organization) {
    // Implementation outline:
    // 1. Return early if d is nil.
    // 2. Copy the embedded Organization struct from domain.
    // 3. Convert d.ID to bson.ObjectID if not empty.
    // 4. Convert d.OwnerID to bson.ObjectID if not empty.
}

func (s *Organization) ToDomain() *domain.Organization {
    // Implementation outline:
    // 1. Return nil if s is nil.
    // 2. Set s.Organization.ID from s.ID.Hex().
    // 3. Set s.Organization.OwnerID from s.OwnerID.Hex().
    // 4. Return pointer to embedded Organization.
}
```

### `shared/domain/mongoschema/organization_member.go`

**Description**: MongoDB schema for OrganizationMember relationship.

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

**Description**: Authentication service protobuf definition.

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

// UserInfo contains basic user information.
message UserInfo {
  string id = 1;
  string email = 2;
  string name = 3;
  string profile_image_url = 4;
}

// OrganizationInfo contains basic organization information.
message OrganizationInfo {
  string id = 1;
  string name = 2;
  string slug = 3;
}

// GoogleAuthReq contains Google OAuth authorization code.
message GoogleAuthReq {
  // authorization_code is the code received from Google OAuth callback
  string authorization_code = 1;
  // redirect_uri must match the URI used to obtain the code
  string redirect_uri = 2;
}

// GoogleAuthRes contains authentication result.
message GoogleAuthRes {
  TokenPair tokens = 1;
  UserInfo user = 2;
  OrganizationInfo organization = 3;
  bool is_new_user = 4;
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

// DevicePollRes contains poll result.
message DevicePollRes {
  bool pending = 1;
  TokenPair tokens = 2;
  UserInfo user = 3;
  OrganizationInfo organization = 4;
}

// RefreshTokenReq contains refresh token for token renewal.
message RefreshTokenReq {
  string refresh_token = 1;
}

// RefreshTokenRes contains new token pair.
message RefreshTokenRes {
  TokenPair tokens = 1;
}

// GetCurrentUserReq is empty request for getting current user.
message GetCurrentUserReq {}

// GetCurrentUserRes contains current authenticated user information.
message GetCurrentUserRes {
  UserInfo user = 1;
  OrganizationInfo organization = 2;
}

// AuthService handles authentication operations.
service AuthService {
  // GoogleAuth exchanges Google OAuth code for JWT tokens.
  rpc GoogleAuth(GoogleAuthReq) returns (GoogleAuthRes);

  // DeviceCode initiates device flow for CLI authentication.
  rpc DeviceCode(DeviceCodeReq) returns (DeviceCodeRes);

  // DevicePoll polls for device authentication completion.
  rpc DevicePoll(DevicePollReq) returns (DevicePollRes);

  // RefreshToken exchanges refresh token for new token pair.
  rpc RefreshToken(RefreshTokenReq) returns (RefreshTokenRes);

  // GetCurrentUser returns current authenticated user info.
  rpc GetCurrentUser(GetCurrentUserReq) returns (GetCurrentUserRes);
}
```

### `idl/protobuf/organization/v1/organization.proto`

**Description**: Organization management service protobuf definition.

```protobuf
syntax = "proto3";

package organization.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1;organizationv1";

// MemberRole represents organization member roles.
enum MemberRole {
  MEMBER_ROLE_UNSPECIFIED = 0;
  MEMBER_ROLE_OWNER = 1;
  MEMBER_ROLE_ADMIN = 2;
  MEMBER_ROLE_MEMBER = 3;
}

// Organization contains organization information.
message Organization {
  string id = 1;
  string name = 2;
  string slug = 3;
  string owner_id = 4;
}

// OrganizationMember contains member information.
message OrganizationMember {
  string id = 1;
  string user_id = 2;
  string email = 3;
  string name = 4;
  MemberRole role = 5;
}

// CreateOrganizationReq contains new organization details.
message CreateOrganizationReq {
  string name = 1;
  string slug = 2;
}

// CreateOrganizationRes contains created organization.
message CreateOrganizationRes {
  Organization organization = 1;
}

// ListOrganizationsReq is empty for listing user's organizations.
message ListOrganizationsReq {}

// ListOrganizationsRes contains user's organizations.
message ListOrganizationsRes {
  repeated Organization organizations = 1;
}

// GetOrganizationReq contains organization identifier.
message GetOrganizationReq {
  string organization_id = 1;
}

// GetOrganizationRes contains organization details.
message GetOrganizationRes {
  Organization organization = 1;
  repeated OrganizationMember members = 2;
}

// AddMemberReq contains new member details.
message AddMemberReq {
  string organization_id = 1;
  string email = 2;
  MemberRole role = 3;
}

// AddMemberRes contains added member.
message AddMemberRes {
  OrganizationMember member = 1;
}

// SelectOrganizationReq sets current organization context.
message SelectOrganizationReq {
  string organization_id = 1;
}

// SelectOrganizationRes confirms organization selection.
message SelectOrganizationRes {
  Organization organization = 1;
}

// OrganizationService handles organization management.
service OrganizationService {
  // CreateOrganization creates a new organization.
  rpc CreateOrganization(CreateOrganizationReq) returns (CreateOrganizationRes);

  // ListOrganizations returns organizations the user belongs to.
  rpc ListOrganizations(ListOrganizationsReq) returns (ListOrganizationsRes);

  // GetOrganization returns organization details with members.
  rpc GetOrganization(GetOrganizationReq) returns (GetOrganizationRes);

  // AddMember adds a user to an organization.
  rpc AddMember(AddMemberReq) returns (AddMemberRes);

  // SelectOrganization sets the current organization context.
  rpc SelectOrganization(SelectOrganizationReq) returns (SelectOrganizationRes);
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

### `api/internal/platform/util/jwtutil/jwtutil.go`

**Description**: JWT token generation and validation utilities.

```go
package jwtutil

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Claims represents custom JWT claims for C-Ops.
type Claims struct {
    jwt.RegisteredClaims
    UserID         string `json:"userId"`
    Email          string `json:"email"`
    OrganizationID string `json:"orgId"`
    Role           string `json:"role"`
}

// Config holds JWT configuration.
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
func GenerateTokenPair(cfg *Config, userID, email, orgID, role string) (*TokenPair, error) {
    // Implementation outline:
    // 1. Create access token claims with short expiry (AccessTokenDuration).
    // 2. Set registered claims: Subject=userID, Issuer, IssuedAt, ExpiresAt.
    // 3. Set custom claims: UserID, Email, OrganizationID, Role.
    // 4. Sign access token with HS256 using SecretKey.
    // 5. Create refresh token claims with longer expiry (RefreshTokenDuration).
    // 6. Set registered claims for refresh token.
    // 7. Sign refresh token with HS256 using SecretKey.
    // 8. Return TokenPair with both tokens and access token expiry.
}

// ValidateAccessToken parses and validates an access token.
func ValidateAccessToken(cfg *Config, tokenString string) (*Claims, error) {
    // Implementation outline:
    // 1. Create parser with HS256 validation method only.
    // 2. Parse token string with claims into Claims struct.
    // 3. Validate signing method is HMAC.
    // 4. Return claims if token is valid.
    // 5. Return appropriate error for expired, invalid, or malformed tokens.
}

// ValidateRefreshToken parses and validates a refresh token.
func ValidateRefreshToken(cfg *Config, tokenString string) (*jwt.RegisteredClaims, error) {
    // Implementation outline:
    // 1. Create parser with HS256 validation method only.
    // 2. Parse token string with registered claims.
    // 3. Validate signing method is HMAC.
    // 4. Return registered claims (Subject contains UserID) if valid.
    // 5. Return appropriate error for expired, invalid, or malformed tokens.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Generate valid token pair | Valid userID, email, orgID, role | TokenPair with valid tokens | Happy path |
| Validate valid access token | Valid token string | Claims struct | Happy path |
| Validate expired access token | Expired token | Error: token expired | Error handling |
| Validate malformed token | Invalid string | Error: malformed token | Validation branch |
| Validate wrong signing method | RS256 signed token | Error: invalid signing method | Security branch |

---

## Step 5: Implement Auth Middleware in API Module

**Files to Read**:
- `.agent/rules/go/go-inbound-http-fiber.md`: Fiber middleware patterns
- `api/internal/platform/setup/server/fiber.go`: Existing Fiber setup

### `api/internal/platform/middleware/auth.go`

**Description**: Authentication middleware for extracting and validating JWT from requests.

```go
package middleware

import (
    "context"
    "log/slog"
    "strings"

    "github.com/gofiber/fiber/v2"

    "github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
    UserContextKey         contextKey = "user"
    OrganizationContextKey contextKey = "organization"
)

// UserContext contains authenticated user information from JWT.
type UserContext struct {
    UserID         string
    Email          string
    OrganizationID string
    Role           string
}

// AuthMiddleware creates a Fiber middleware for JWT authentication.
func AuthMiddleware(l *slog.Logger, jwtCfg *jwtutil.Config) fiber.Handler {
    // Implementation outline:
    // 1. Return fiber.Handler function.
    // 2. Extract Authorization header from request.
    // 3. If header is missing or doesn't start with "Bearer ", return 401.
    // 4. Extract token string after "Bearer " prefix.
    // 5. Call jwtutil.ValidateAccessToken with token.
    // 6. If validation fails, log error and return 401.
    // 7. Create UserContext from validated claims.
    // 8. Store UserContext in Fiber locals for handler access.
    // 9. Call c.Next() to continue request processing.
}

// OptionalAuthMiddleware extracts user context if token is present, but doesn't require it.
func OptionalAuthMiddleware(l *slog.Logger, jwtCfg *jwtutil.Config) fiber.Handler {
    // Implementation outline:
    // 1. Return fiber.Handler function.
    // 2. Extract Authorization header from request.
    // 3. If header is missing, call c.Next() without setting context.
    // 4. If header present, validate token.
    // 5. If valid, store UserContext in Fiber locals.
    // 6. If invalid, log warning but continue without context.
    // 7. Call c.Next().
}

// GetUserContext extracts UserContext from Fiber context.
func GetUserContext(c *fiber.Ctx) *UserContext {
    // Implementation outline:
    // 1. Get value from Fiber locals using UserContextKey.
    // 2. Type assert to *UserContext.
    // 3. Return nil if not found or wrong type.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid Bearer token | `Authorization: Bearer valid_token` | UserContext set, next called | Happy path |
| Missing header | No Authorization header | 401 Unauthorized | Missing auth |
| Invalid token format | `Authorization: Basic token` | 401 Unauthorized | Format validation |
| Expired token | Expired Bearer token | 401 Unauthorized | Token validation |
| Optional with valid token | Valid Bearer token | UserContext set | Optional happy path |
| Optional without token | No header | Next called, no context | Optional skip |

---

## Step 6: Implement Auth Service in API Module

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service implementation guidelines
- `api/internal/service/project/project_service.go`: Example service

### `api/internal/service/auth/auth_service.go`

**Description**: Core authentication service handling OAuth and token operations.

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

// GoogleAuthResult contains the result of Google OAuth authentication.
type GoogleAuthResult struct {
    Tokens       *jwtutil.TokenPair
    User         *domain.User
    Organization *domain.Organization
    IsNewUser    bool
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
    Pending      bool
    Tokens       *jwtutil.TokenPair
    User         *domain.User
    Organization *domain.Organization
}

// Service implements authentication business logic.
type Service struct {
    logger     *slog.Logger
    jwtCfg     *jwtutil.Config
    oauthPort  oauth.GoogleOAuthPort
    userRepo   repository.UserRepositoryPort
    orgRepo    repository.OrganizationRepositoryPort
}

// NewService creates a new auth service.
func NewService(
    l *slog.Logger,
    jwtCfg *jwtutil.Config,
    oauthPort oauth.GoogleOAuthPort,
    userRepo repository.UserRepositoryPort,
    orgRepo repository.OrganizationRepositoryPort,
) *Service {
    return &Service{
        logger:    l.With(slog.String("name", "auth.service")),
        jwtCfg:    jwtCfg,
        oauthPort: oauthPort,
        userRepo:  userRepo,
        orgRepo:   orgRepo,
    }
}

// GoogleAuth handles Google OAuth code exchange and user creation/lookup.
func (s *Service) GoogleAuth(ctx context.Context, params GoogleAuthParams) (*GoogleAuthResult, error) {
    // Implementation outline:
    // 1. Exchange authorization code for Google tokens via oauthPort.
    // 2. Fetch user info from Google using access token.
    // 3. Look up existing account by provider="google" and providerID=googleUserID.
    // 4. If account exists:
    //    a. Fetch user by account.UserID.
    //    b. Fetch user's default organization.
    //    c. Get user's role in organization.
    //    d. Generate JWT token pair.
    //    e. Return result with IsNewUser=false.
    // 5. If account doesn't exist:
    //    a. Create new User with Google profile info.
    //    b. Create new Account linking to User.
    //    c. Create default Organization named "{User.Name}'s Organization".
    //    d. Create OrganizationMember with role=owner.
    //    e. Generate JWT token pair.
    //    f. Return result with IsNewUser=true.
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
    // 2. If still pending, return DevicePollResult{Pending: true}.
    // 3. If complete:
    //    a. Fetch user info from Google.
    //    b. Follow same user lookup/creation logic as GoogleAuth.
    //    c. Generate JWT token pair.
    //    d. Return DevicePollResult with tokens, user, organization.
}

// RefreshToken exchanges refresh token for new token pair.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*jwtutil.TokenPair, error) {
    // Implementation outline:
    // 1. Validate refresh token using jwtutil.ValidateRefreshToken.
    // 2. Extract userID from claims.Subject.
    // 3. Fetch user by ID to ensure still valid.
    // 4. Fetch user's current organization membership.
    // 5. Generate new token pair with current user/org info.
    // 6. Return new TokenPair.
}

// GetCurrentUser returns current authenticated user info.
func (s *Service) GetCurrentUser(ctx context.Context, userID, orgID string) (*GoogleAuthResult, error) {
    // Implementation outline:
    // 1. Fetch user by userID.
    // 2. Fetch organization by orgID.
    // 3. Return result with user and organization (no tokens).
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| New user Google auth | Valid auth code, new Google account | New user, org created, IsNewUser=true | New user path |
| Existing user Google auth | Valid auth code, existing account | Existing user returned, IsNewUser=false | Existing user path |
| Invalid auth code | Invalid authorization code | Error: invalid code | OAuth error |
| Device code initiation | Empty request | Device code result | Happy path |
| Device poll pending | Device code not yet authorized | Pending=true | Pending state |
| Device poll complete | Authorized device code | Tokens and user | Complete state |
| Refresh valid token | Valid refresh token | New token pair | Happy path |
| Refresh expired token | Expired refresh token | Error: token expired | Expired token |

---

## Step 7: Implement Auth Repository Ports and Adapters

**Files to Read**:
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines
- `api/internal/service/project/outbound/repository/project_repo_port.go`: Example port
- `api/internal/service/project/outbound/repository/mongodb/project_repo.go`: Example adapter

### `api/internal/service/auth/outbound/repository/user_repo_port.go`

**Description**: User repository interface.

```go
package repository

import (
    "context"

    "github.com/team-attention/cops/shared/domain"
)

// UserRepositoryPort defines interface for user data persistence.
type UserRepositoryPort interface {
    // Create creates a new user.
    Create(ctx context.Context, user *domain.User) (*domain.User, error)

    // GetByID retrieves user by ID.
    GetByID(ctx context.Context, userID string) (*domain.User, error)

    // GetByEmail retrieves user by email.
    GetByEmail(ctx context.Context, email string) (*domain.User, error)

    // FindByAccountProvider finds user by OAuth provider account.
    FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error)

    // CreateAccount creates an OAuth account linked to a user.
    CreateAccount(ctx context.Context, account *domain.Account) (*domain.Account, error)
}
```

### `api/internal/service/auth/outbound/repository/organization_repo_port.go`

**Description**: Organization repository interface.

```go
package repository

import (
    "context"

    "github.com/team-attention/cops/shared/domain"
)

// OrganizationRepositoryPort defines interface for organization data persistence.
type OrganizationRepositoryPort interface {
    // Create creates a new organization.
    Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

    // GetByID retrieves organization by ID.
    GetByID(ctx context.Context, orgID string) (*domain.Organization, error)

    // GetBySlug retrieves organization by slug.
    GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)

    // ListByUserID returns all organizations a user belongs to.
    ListByUserID(ctx context.Context, userID string) ([]*domain.Organization, error)

    // CreateMember adds a user to an organization.
    CreateMember(ctx context.Context, member *domain.OrganizationMember) (*domain.OrganizationMember, error)

    // GetMember retrieves membership for a user in an organization.
    GetMember(ctx context.Context, orgID, userID string) (*domain.OrganizationMember, error)

    // ListMembers returns all members of an organization.
    ListMembers(ctx context.Context, orgID string) ([]*domain.OrganizationMember, error)

    // GetDefaultOrganization returns the first organization for a user.
    GetDefaultOrganization(ctx context.Context, userID string) (*domain.Organization, error)
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
    // ExchangeCode exchanges authorization code for tokens.
    ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)

    // GetUserInfo fetches user profile using access token.
    GetUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error)

    // InitiateDeviceFlow starts device code flow.
    InitiateDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error)

    // PollDeviceCode polls for device code authorization.
    PollDeviceCode(ctx context.Context, deviceCode string) (*TokenResponse, error)
}
```

### `api/internal/service/auth/outbound/repository/mongodb/user_repo.go`

**Description**: MongoDB implementation of user repository.

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
    logger       *slog.Logger
    usersColl    *mongo.Collection
    accountsColl *mongo.Collection
}

func NewMongoUserRepository(l *slog.Logger, db *mongo.Database) *MongoUserRepository {
    return &MongoUserRepository{
        logger:       l.With(slog.String("name", "auth.repository.mongodb.user")),
        usersColl:    db.Collection(mongoschema.UserCollectionName),
        accountsColl: db.Collection(mongoschema.AccountCollectionName),
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
    // 3. Convert to domain.User.
    // 4. Return user or NotFoundError.
}

func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
    // Implementation outline:
    // 1. Find document by email field.
    // 2. Convert to domain.User.
    // 3. Return user or NotFoundError.
}

func (r *MongoUserRepository) FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error) {
    // Implementation outline:
    // 1. Find account by provider and providerID.
    // 2. If not found, return nil (not error).
    // 3. If found, fetch user by account.UserID.
    // 4. Return user.
}

func (r *MongoUserRepository) CreateAccount(ctx context.Context, account *domain.Account) (*domain.Account, error) {
    // Implementation outline:
    // 1. Create mongoschema.Account from domain.Account.
    // 2. Insert into accounts collection.
    // 3. Set account.ID from inserted ObjectID.
    // 4. Return account.
}

var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil)
```

### `api/internal/service/auth/outbound/repository/mongodb/organization_repo.go`

**Description**: MongoDB implementation of organization repository.

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

type MongoOrganizationRepository struct {
    logger      *slog.Logger
    orgsColl    *mongo.Collection
    membersColl *mongo.Collection
}

func NewMongoOrganizationRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationRepository {
    return &MongoOrganizationRepository{
        logger:      l.With(slog.String("name", "auth.repository.mongodb.organization")),
        orgsColl:    db.Collection(mongoschema.OrganizationCollectionName),
        membersColl: db.Collection(mongoschema.OrganizationMemberCollectionName),
    }
}

func (r *MongoOrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Create mongoschema.Organization from domain.
    // 2. Insert into organizations collection.
    // 3. Set org.ID from inserted ObjectID.
    // 4. Return organization.
}

func (r *MongoOrganizationRepository) GetByID(ctx context.Context, orgID string) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Convert orgID to ObjectID.
    // 2. Find document by _id.
    // 3. Convert to domain.Organization.
    // 4. Return org or NotFoundError.
}

func (r *MongoOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Find document by slug field.
    // 2. Convert to domain.Organization.
    // 3. Return org or NotFoundError.
}

func (r *MongoOrganizationRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Organization, error) {
    // Implementation outline:
    // 1. Convert userID to ObjectID.
    // 2. Find all memberships for userID.
    // 3. Collect organization IDs from memberships.
    // 4. Find all organizations by IDs.
    // 5. Convert to domain.Organization slice.
    // 6. Return organizations.
}

func (r *MongoOrganizationRepository) CreateMember(ctx context.Context, member *domain.OrganizationMember) (*domain.OrganizationMember, error) {
    // Implementation outline:
    // 1. Create mongoschema.OrganizationMember from domain.
    // 2. Insert into organization_members collection.
    // 3. Set member.ID from inserted ObjectID.
    // 4. Return member.
}

func (r *MongoOrganizationRepository) GetMember(ctx context.Context, orgID, userID string) (*domain.OrganizationMember, error) {
    // Implementation outline:
    // 1. Convert orgID and userID to ObjectIDs.
    // 2. Find document by organizationId and userId.
    // 3. Convert to domain.OrganizationMember.
    // 4. Return member or NotFoundError.
}

func (r *MongoOrganizationRepository) ListMembers(ctx context.Context, orgID string) ([]*domain.OrganizationMember, error) {
    // Implementation outline:
    // 1. Convert orgID to ObjectID.
    // 2. Find all documents by organizationId.
    // 3. Convert each to domain.OrganizationMember.
    // 4. Return members.
}

func (r *MongoOrganizationRepository) GetDefaultOrganization(ctx context.Context, userID string) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Convert userID to ObjectID.
    // 2. Find first membership for userID (sorted by _id for consistency).
    // 3. Fetch organization by membership.OrganizationID.
    // 4. Return organization or nil if no membership.
}

var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
```

### `api/internal/service/auth/outbound/oauth/google/google_oauth.go`

**Description**: Google OAuth implementation.

```go
package google

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"

    oauthport "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
)

// Config holds Google OAuth configuration.
type Config struct {
    ClientID     string
    ClientSecret string
    Scopes       []string
}

type GoogleOAuthAdapter struct {
    logger     *slog.Logger
    config     *oauth2.Config
    httpClient *http.Client
}

func NewGoogleOAuthAdapter(l *slog.Logger, cfg *Config) *GoogleOAuthAdapter {
    return &GoogleOAuthAdapter{
        logger: l.With(slog.String("name", "auth.oauth.google")),
        config: &oauth2.Config{
            ClientID:     cfg.ClientID,
            ClientSecret: cfg.ClientSecret,
            Scopes:       cfg.Scopes,
            Endpoint:     google.Endpoint,
        },
        httpClient: http.DefaultClient,
    }
}

func (a *GoogleOAuthAdapter) ExchangeCode(ctx context.Context, code, redirectURI string) (*oauthport.TokenResponse, error) {
    // Implementation outline:
    // 1. Set redirect URI on config.
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
    // 1. POST to https://oauth2.googleapis.com/device/code with:
    //    - client_id
    //    - scope (space-separated)
    // 2. Parse JSON response.
    // 3. Return DeviceCodeResponse with device_code, user_code, verification_url.
}

func (a *GoogleOAuthAdapter) PollDeviceCode(ctx context.Context, deviceCode string) (*oauthport.TokenResponse, error) {
    // Implementation outline:
    // 1. POST to https://oauth2.googleapis.com/token with:
    //    - client_id
    //    - client_secret
    //    - device_code
    //    - grant_type=urn:ietf:params:oauth:grant-type:device_code
    // 2. If response contains "authorization_pending" error, return nil (pending).
    // 3. If success, return TokenResponse.
    // 4. If other error, return error.
}

var _ oauthport.GoogleOAuthPort = (*GoogleOAuthAdapter)(nil)
```

---

## Step 8: Implement Auth ConnectRPC Handler

**Files to Read**:
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler guidelines
- `api/internal/service/project/inbound/grpc/connectrpc/handler.go`: Example handler

### `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**: ConnectRPC handler for auth service.

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
    // 3. Convert result to protobuf response.
    // 4. Return connect.NewResponse(res).
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
    // 3. Convert result to protobuf response.
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

func (h *AuthGRPCHandler) GetCurrentUser(
    ctx context.Context,
    req *connect.Request[authv1.GetCurrentUserReq],
) (*connect.Response[authv1.GetCurrentUserRes], error) {
    // Implementation outline:
    // 1. Extract user context from request (set by middleware).
    // 2. Call h.svc.GetCurrentUser(ctx, userID, orgID).
    // 3. Convert result to protobuf response.
    // 4. Return connect.NewResponse(res).
}

var _ authv1connect.AuthServiceHandler = (*AuthGRPCHandler)(nil)
```

---

## Step 9: Implement Organization Service

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service implementation guidelines
- `api/internal/service/auth/auth_service.go`: Auth service for reference

### `api/internal/service/organization/organization_service.go`

**Description**: Organization management service.

```go
package organization

import (
    "context"
    "log/slog"
    "regexp"
    "strings"

    "github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
    "github.com/team-attention/cops/shared/domain"
)

// CreateOrganizationParams contains parameters for creating an organization.
type CreateOrganizationParams struct {
    UserID string
    Name   string
    Slug   string
}

// AddMemberParams contains parameters for adding a member.
type AddMemberParams struct {
    OrganizationID string
    ActorUserID    string
    TargetEmail    string
    Role           domain.MemberRole
}

// Service implements organization management business logic.
type Service struct {
    logger   *slog.Logger
    orgRepo  repository.OrganizationRepositoryPort
    userRepo repository.UserRepositoryPort
}

// NewService creates a new organization service.
func NewService(
    l *slog.Logger,
    orgRepo repository.OrganizationRepositoryPort,
    userRepo repository.UserRepositoryPort,
) *Service {
    return &Service{
        logger:   l.With(slog.String("name", "organization.service")),
        orgRepo:  orgRepo,
        userRepo: userRepo,
    }
}

// CreateOrganization creates a new organization with the user as owner.
func (s *Service) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Validate name is not empty.
    // 2. Generate slug from name if not provided (lowercase, hyphenated).
    // 3. Validate slug format (alphanumeric and hyphens only).
    // 4. Check slug uniqueness via orgRepo.GetBySlug.
    // 5. Create Organization with OwnerID=params.UserID.
    // 6. Call orgRepo.Create.
    // 7. Create OrganizationMember with role=owner for the user.
    // 8. Call orgRepo.CreateMember.
    // 9. Return created organization.
}

// ListOrganizations returns all organizations a user belongs to.
func (s *Service) ListOrganizations(ctx context.Context, userID string) ([]*domain.Organization, error) {
    // Implementation outline:
    // 1. Call orgRepo.ListByUserID(ctx, userID).
    // 2. Return organizations list.
}

// GetOrganization returns organization details with members.
func (s *Service) GetOrganization(ctx context.Context, orgID, userID string) (*domain.Organization, []*domain.OrganizationMember, error) {
    // Implementation outline:
    // 1. Verify user is member of organization via orgRepo.GetMember.
    // 2. If not member, return ForbiddenError.
    // 3. Fetch organization via orgRepo.GetByID.
    // 4. Fetch members via orgRepo.ListMembers.
    // 5. Return organization and members.
}

// AddMember adds a user to an organization.
func (s *Service) AddMember(ctx context.Context, params AddMemberParams) (*domain.OrganizationMember, error) {
    // Implementation outline:
    // 1. Verify actor is admin or owner of organization.
    // 2. If not, return ForbiddenError.
    // 3. Look up target user by email via userRepo.GetByEmail.
    // 4. If user not found, return NotFoundError.
    // 5. Check if target is already member via orgRepo.GetMember.
    // 6. If already member, return BadRequestError.
    // 7. Create OrganizationMember with specified role.
    // 8. Call orgRepo.CreateMember.
    // 9. Return created member.
}

// SelectOrganization validates and returns selected organization.
func (s *Service) SelectOrganization(ctx context.Context, orgID, userID string) (*domain.Organization, error) {
    // Implementation outline:
    // 1. Verify user is member of organization.
    // 2. If not member, return ForbiddenError.
    // 3. Fetch organization by ID.
    // 4. Return organization.
}

// generateSlug creates a URL-safe slug from organization name.
func (s *Service) generateSlug(name string) string {
    // Implementation outline:
    // 1. Convert to lowercase.
    // 2. Replace spaces with hyphens.
    // 3. Remove non-alphanumeric characters except hyphens.
    // 4. Trim leading/trailing hyphens.
    // 5. Return slug.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Create organization | Valid name, userID | Organization created, user is owner | Happy path |
| Create with duplicate slug | Existing slug | Error: slug taken | Slug validation |
| List user's organizations | Valid userID | List of organizations | Happy path |
| Get organization as member | Valid orgID, member userID | Organization and members | Member access |
| Get organization as non-member | Valid orgID, non-member userID | ForbiddenError | Access denied |
| Add member as owner | Owner adds member | Member created | Owner privilege |
| Add member as regular member | Member tries to add | ForbiddenError | Insufficient permission |
| Add already existing member | Email of existing member | BadRequestError | Duplicate check |
| Add non-existent user | Unknown email | NotFoundError | User lookup |

---

## Step 10: Update API Configuration

**Files to Read**:
- `api/internal/platform/setup/config/config.go`: Existing config structure
- `.agent/rules/go/go-platform-setup.md`: Setup guidelines

### Update `api/internal/platform/setup/config/config.go`

**Description**: Add JWT and OAuth configuration.

```go
// Add to Config struct:
type Config struct {
    // ... existing fields ...
    JWT    JWTConfig
    OAuth  OAuthConfig
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
    GoogleClientID     string `env:"GOOGLE_CLIENT_ID,required"`
    GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET,required"`
}
```

---

## Step 11: Register Auth Module in API Container

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
    "github.com/team-attention/cops/api/internal/service/auth/outbound/oauth/google"
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

        // Google OAuth config
        fx.Provide(func(cfg *config.Config) *google.Config {
            return &google.Config{
                ClientID:     cfg.OAuth.GoogleClientID,
                ClientSecret: cfg.OAuth.GoogleClientSecret,
                Scopes:       []string{"email", "profile"},
            }
        }),

        // OAuth adapter
        fx.Provide(
            fx.Annotate(
                google.NewGoogleOAuthAdapter,
                fx.As(new(oauth.GoogleOAuthPort)),
            ),
        ),

        // Repositories
        fx.Provide(
            fx.Annotate(
                mongodb.NewMongoUserRepository,
                fx.As(new(repository.UserRepositoryPort)),
            ),
        ),
        fx.Provide(
            fx.Annotate(
                mongodb.NewMongoOrganizationRepository,
                fx.As(new(repository.OrganizationRepositoryPort)),
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
    newOrganizationModule(),
    // ...
)
```

---

## Step 12: Implement CLI Auth Commands

**Files to Read**:
- `.agent/rules/go/go-dig-container.md`: Dig container guidelines
- `cli/internal/service/tracking/inbound/cli/cobra/handler.go`: Example CLI handler
- `cli/internal/platform/setup/config/config.go`: CLI config structure

### `cli/internal/service/auth/auth_service.go`

**Description**: CLI auth service for managing local authentication state.

```go
package auth

import (
    "context"
    "encoding/json"
    "log/slog"
    "os"
    "path/filepath"

    "github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
)

// AuthState represents the local authentication state.
type AuthState struct {
    User         *UserInfo         `json:"user"`
    Organization *OrganizationInfo `json:"organization"`
    Tokens       *TokenInfo        `json:"tokens"`
}

// UserInfo contains user information.
type UserInfo struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

// OrganizationInfo contains organization information.
type OrganizationInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Slug string `json:"slug"`
}

// TokenInfo contains token data.
type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    string `json:"expiresAt"`
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

// Login initiates the device flow login process.
func (s *Service) Login(ctx context.Context) (*AuthState, error) {
    // Implementation outline:
    // 1. Call apiClient.DeviceCode(ctx) to get device code.
    // 2. Return device code info for display to user.
    // 3. Poll apiClient.DevicePoll(ctx, deviceCode) at interval.
    // 4. On success, create AuthState from response.
    // 5. Save AuthState to auth.json with 0600 permissions.
    // 6. Return AuthState.
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

// GetAccessToken returns a valid access token, refreshing if needed.
func (s *Service) GetAccessToken(ctx context.Context) (string, error) {
    // Implementation outline:
    // 1. Get current auth state.
    // 2. If not logged in, return error.
    // 3. Parse expiresAt time.
    // 4. If token expired or will expire in <5 minutes:
    //    a. Call apiClient.RefreshToken with refresh token.
    //    b. Update auth.json with new tokens.
    // 5. Return access token.
}

// saveAuthState writes auth state to file with secure permissions.
func (s *Service) saveAuthState(state *AuthState) error {
    // Implementation outline:
    // 1. Ensure ~/.cops directory exists.
    // 2. Marshal state to JSON.
    // 3. Write to auth.json with os.WriteFile and 0600 permissions.
}
```

### `cli/internal/service/auth/outbound/api/auth_port.go`

**Description**: API client interface for auth operations.

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
    ExpiresAt    string
    User         *UserInfo
    Organization *OrganizationInfo
}

// UserInfo contains user data.
type UserInfo struct {
    ID    string
    Email string
    Name  string
}

// OrganizationInfo contains organization data.
type OrganizationInfo struct {
    ID   string
    Name string
    Slug string
}

// TokenResult contains new tokens.
type TokenResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    string
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
    // 2. Call client.DeviceCode(ctx, req).
    // 3. Convert response to DeviceCodeResult.
    // 4. Return result.
}

func (c *AuthAPIClient) DevicePoll(ctx context.Context, deviceCode string) (*api.PollResult, error) {
    // Implementation outline:
    // 1. Create DevicePollReq with device code.
    // 2. Call client.DevicePoll(ctx, req).
    // 3. Convert response to PollResult.
    // 4. Return result.
}

func (c *AuthAPIClient) RefreshToken(ctx context.Context, refreshToken string) (*api.TokenResult, error) {
    // Implementation outline:
    // 1. Create RefreshTokenReq with refresh token.
    // 2. Call client.RefreshToken(ctx, req).
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
            // 1. Get context with timeout.
            // 2. Call svc to get device code.
            // 3. Display: "Go to: {verification_url}"
            // 4. Display: "Enter code: {user_code}"
            // 5. Start polling loop with interval.
            // 6. On success, display user info.
            // 7. Display: "Logged in as {email}"
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
            // 1. Call svc.Logout(ctx).
            // 2. Display: "Logged out successfully"
        },
    }
}
```

### `cli/internal/service/auth/inbound/cli/cobra/status.go`

**Description**: Status command implementation.

```go
package cobra

import "github.com/spf13/cobra"

// NewStatusCommand creates the status command.
func (h *AuthCLIHandler) NewStatusCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show current authentication status",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation outline:
            // 1. Call svc.GetAuthState().
            // 2. If not logged in, display: "Not logged in"
            // 3. If logged in, display:
            //    - User: {name} ({email})
            //    - Organization: {org name} ({org slug})
        },
    }
}
```

---

## Step 13: Update CLI to Include Auth Token in Requests

**Files to Read**:
- `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`: Example API client

### Update CLI HTTP Client

**Description**: Add auth interceptor to include Bearer token.

```go
// cli/internal/platform/setup/httpclient/httpclient.go

// Add interceptor that reads auth token and adds to requests:
// 1. Create custom http.RoundTripper.
// 2. In RoundTrip, check for auth token from auth service.
// 3. If token exists, add Authorization: Bearer {token} header.
// 4. Call underlying transport.
```

---

## Step 14: Update Daemon for Authentication

**Files to Read**:
- `daemon/internal/platform/setup/copsapi.go`: Existing API client setup
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
    User         *UserInfo         `json:"user"`
    Organization *OrganizationInfo `json:"organization"`
    Tokens       *TokenInfo        `json:"tokens"`
}

type UserInfo struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

type OrganizationInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Slug string `json:"slug"`
}

type TokenInfo struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
    ExpiresAt    string `json:"expiresAt"`
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
    // 5. Check if token is expired.
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

### Update Daemon API Client

**Description**: Add auth token to API requests.

```go
// daemon/internal/platform/setup/copsapi.go

// Update InitAPIClient to accept auth service:
func InitAPIClient(cfg *Config, authSvc *auth.Service) *APIClient {
    // Implementation outline:
    // 1. Create req client with base URL.
    // 2. Add request middleware that:
    //    a. Gets access token from authSvc.
    //    b. If token available, sets Authorization header.
    //    c. If not available, logs warning.
    // 3. Return client.
}
```

---

## Step 15: Update Project Registration to Require Organization

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

**Description**: Validate organization membership before project operations.

```go
// api/internal/service/project/project_service.go

// Add org repository dependency and validation:
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
    // Implementation outline:
    // 1. Validate user is member of organization (from context).
    // 2. If not member, return ForbiddenError.
    // 3. Add organization ID to find/create params.
    // 4. Call repository.
    // 5. Return result.
}
```

---

## Step 16: Create Database Indexes

**Files to Read**:
- `api/internal/platform/setup/mongodb/mongodb.go`: MongoDB setup

### Update MongoDB Setup

**Description**: Create indexes for new collections.

```go
// api/internal/platform/setup/mongodb/mongodb.go

func ensureIndexes(db *mongo.Database) error {
    // Implementation outline:
    // 1. Create unique index on accounts: (provider, providerId).
    // 2. Create index on accounts: userId.
    // 3. Create unique index on organizations: slug.
    // 4. Create unique index on organization_members: (organizationId, userId).
    // 5. Create index on organization_members: userId.
    // 6. Create index on projects: organizationId.
    // 7. Create unique index on users: email.
}
```

---

## Step 17: Generate Protobuf Code

**Description**: Generate Go code from new protobuf definitions.

```bash
cd idl/protobuf && buf generate
```

This generates:
- `shared/gen/grpcstub/auth/v1/auth.pb.go`
- `shared/gen/grpcstub/auth/v1/authv1connect/auth.connect.go`
- `shared/gen/grpcstub/organization/v1/organization.pb.go`
- `shared/gen/grpcstub/organization/v1/organizationv1connect/organization.connect.go`

---

## Implementation Order

1. **Step 1-2**: Domain models and MongoSchema (shared module)
2. **Step 3**: Protobuf definitions and code generation (Step 17)
3. **Step 4-5**: JWT utility and auth middleware (API module)
4. **Step 6-8**: Auth service, repositories, and handlers (API module)
5. **Step 9**: Organization service (API module)
6. **Step 10-11**: API configuration and module registration
7. **Step 15-16**: Project updates and database indexes
8. **Step 12-13**: CLI auth commands and token integration
9. **Step 14**: Daemon auth integration

---

## Testing Strategy

### Unit Tests

1. **JWT Utility**: Token generation, validation, expiry handling
2. **Domain Models**: Enum constants, ID type conversion
3. **MongoSchema**: FromDomain/ToDomain conversions
4. **Services**: Business logic with mocked repositories

### Integration Tests

1. **Auth Flow**: Full OAuth code exchange with mock Google API
2. **Organization Management**: CRUD operations with real MongoDB
3. **Project Registration**: Organization validation

### E2E Tests

1. **CLI Login Flow**: Device code display and polling
2. **Daemon Authentication**: Token refresh on API calls

---

## Critical Implementation Notes

1. **Token Security**: Never log tokens. Use secure file permissions (0600) for auth.json.

2. **Error Messages**: Authentication errors should not expose internal details. Use generic messages like "Authentication failed".

3. **Token Refresh**: CLI should proactively refresh tokens before expiry (5-minute buffer). Daemon should handle 401 responses by stopping until re-authenticated.

4. **Organization Context**: JWT tokens include organization ID and role. Middleware extracts this for authorization checks.

5. **Database Indexes**: Unique indexes on (provider, providerId) for accounts and (organizationId, userId) for members prevent duplicates at database level.

6. **Slug Generation**: Organization slugs must be URL-safe. Use regex validation: `^[a-z0-9-]+$`.

7. **Device Flow Polling**: Use exponential backoff if server returns slow_down error. Respect interval from device code response.

8. **Backward Compatibility**: Existing projects without organization_id should be migrated or handled gracefully during transition period.
