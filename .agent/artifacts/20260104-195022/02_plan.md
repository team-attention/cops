# Implementation Plan: Auto-create Personal Organization on User Signup

## Overview

When a new user signs up via Google OAuth, the system must automatically create a Personal Organization for that user within the same database transaction. This ensures atomicity: if organization creation fails, user creation is also rolled back. The implementation follows hexagonal architecture with a Transaction Manager abstraction in the platform layer.

**Key Components:**
1. Transaction Manager (Port + MongoDB Adapter) in platform layer
2. Organization Repository `Create` method (new)
3. Slug generation utility function
4. Modified `auth.Service.GoogleAuth()` method with transaction logic
5. Updated fx dependency injection

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | Unicode normalization for slug generation | `golang.org/x/text` | Required for transliterating accented characters (e.g., "Jose" to "jose") |

---

## Step 1: Create Transaction Manager Port (Interface)

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform package guidelines
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-port-adapter-pattern.md`: Port/Adapter pattern fundamentals
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter naming conventions

### `/Users/jayce/team-attention/cops/api/internal/platform/outbound/txmanager/txmanager_port.go`

**Description**:
Create a database-agnostic transaction interface that allows the service layer to use transactions without depending on MongoDB-specific types.

```go
package txmanager

import "context"

// TransactionFunc is a function that executes within a transaction.
// The ctx parameter contains the transaction context and must be passed to all repository operations.
// If the function returns an error, the transaction is rolled back.
// If the function returns nil, the transaction is committed.
type TransactionFunc func(ctx context.Context) (interface{}, error)

// TransactionManagerPort defines the interface for managing database transactions.
// This abstraction allows the service layer to use transactions without depending on
// database-specific types (e.g., mongo.Client, mongo.SessionContext).
type TransactionManagerPort interface {
	// WithTransaction executes a function within a transaction.
	// It automatically handles session creation, commit, rollback, and cleanup.
	//
	// Parameters:
	//   - ctx: Parent context for timeout/cancellation
	//   - fn: Function to execute within transaction
	//
	// Returns:
	//   - result: Value returned by fn if transaction commits successfully
	//   - error: Error from fn (triggers rollback) or transaction infrastructure error
	//
	// Behavior:
	//   - If fn returns (result, nil): Transaction commits, returns (result, nil)
	//   - If fn returns (nil, error): Transaction rolls back, returns (nil, error)
	//   - Automatically retries on transient errors (network issues, etc.)
	// Implementation outline:
	// 1. Create a database session with appropriate transaction options.
	// 2. Execute the provided function within the transaction context.
	// 3. If function returns error, rollback transaction and return error.
	// 4. If function returns success, commit transaction and return result.
	// 5. Handle transient errors with automatic retry.
	WithTransaction(ctx context.Context, fn TransactionFunc) (interface{}, error)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - Interface only | N/A | N/A | N/A |

---

## Step 2: Create MongoDB Transaction Manager Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter implementation patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logger binding conventions
- `/Users/jayce/team-attention/cops/api/internal/platform/setup/mongodb/mongodb.go`: MongoDB setup pattern (if exists)

### `/Users/jayce/team-attention/cops/api/internal/platform/outbound/txmanager/mongodb/txmanager.go`

**Description**:
Implement the TransactionManagerPort using MongoDB Go Driver v2. Uses `session.WithTransaction()` for automatic commit/rollback handling.

```go
package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
)

// MongoTransactionManager implements TransactionManagerPort using MongoDB.
type MongoTransactionManager struct {
	client *mongo.Client
	logger *slog.Logger
}

// NewMongoTransactionManager creates a new MongoDB transaction manager.
func NewMongoTransactionManager(l *slog.Logger, client *mongo.Client) *MongoTransactionManager {
	// Implementation outline:
	// 1. Store the MongoDB client reference.
	// 2. Bind logger with component name "platform.txmanager.mongodb".
	// 3. Return the initialized transaction manager.
	return &MongoTransactionManager{
		client: client,
		logger: l.With(slog.String("name", "platform.txmanager.mongodb")),
	}
}

// WithTransaction executes a function within a MongoDB transaction.
func (m *MongoTransactionManager) WithTransaction(ctx context.Context, fn txmanager.TransactionFunc) (interface{}, error) {
	// Implementation outline:
	// 1. Configure transaction options with writeconcern.Majority() for durability.
	txnOpts := options.Transaction().SetWriteConcern(writeconcern.Majority())
	// 2. Create session options with the transaction options as defaults.
	sessOpts := options.Session().SetDefaultTransactionOptions(txnOpts)
	// 3. Start a new MongoDB session using the client.
	session, err := m.client.StartSession(sessOpts)
	// 4. If session creation fails, log error and return wrapped error.
	if err != nil {
		m.logger.Error("failed to start MongoDB session",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to start transaction session: %w", err)
	}
	// 5. Defer session.EndSession() to ensure cleanup.
	defer session.EndSession(context.Background())
	// 6. Call session.WithTransaction() with the provided callback function.
	//    a. The callback receives a context that contains the session.
	//    b. Pass this context to the user-provided function fn.
	//    c. MongoDB's WithTransaction handles commit on success, rollback on error.
	result, err := session.WithTransaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		return fn(sessCtx)
	})
	// 7. If transaction fails, log error with details and return the error.
	if err != nil {
		m.logger.Error("transaction failed",
			slog.Any("error", err),
		)
		return nil, err
	}
	// 8. Return the result from the successful transaction.
	return result, nil
}

// Interface verification
var _ txmanager.TransactionManagerPort = (*MongoTransactionManager)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Successful transaction | fn returns (result, nil) | (result, nil) | Happy path - commit |
| Failed transaction | fn returns (nil, error) | (nil, error) | Rollback on error |
| Session creation fails | Invalid client | (nil, error) with "failed to start transaction session" | Session error handling |

---

## Step 3: Add `Create` Method to Organization Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`: Existing organization repository interface
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Repository port conventions

### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/organization_repo_port.go`

**Description**:
Add the `Create` method to the existing `OrganizationRepositoryPort` interface.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserOrganization represents a user's membership in an organization.
// Contains organization data plus the user's specific role.
type UserOrganization struct {
	Organization *domain.Organization
	Role         domain.MemberRole
}

// OrganizationWithMemberCount represents an organization with its member count.
// Used to determine if cascade deletion is needed.
type OrganizationWithMemberCount struct {
	Organization *domain.Organization
	MemberCount  int
}

// OrganizationRepositoryPort defines interface for organization queries.
type OrganizationRepositoryPort interface {
	// Create creates a new organization.
	// Participates in transaction if ctx contains mongo.SessionContext.
	// Returns created organization with generated ID.
	// Implementation outline:
	// 1. Convert domain.Organization to mongoschema.Organization.
	// 2. Insert document into organizations collection.
	// 3. Extract inserted ID and set on schema.
	// 4. Convert schema back to domain and return.
	Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

	// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
	// Queries organizations collection filtering by embedded members.userId.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizations(ctx context.Context, userID string) ([]*UserOrganization, error)

	// GetUserOrganizationsWithMemberCount retrieves all organizations a user belongs to with member counts.
	// Used to determine which organizations need cascade deletion (sole member) vs membership removal.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizationsWithMemberCount(ctx context.Context, userID string) ([]*OrganizationWithMemberCount, error)

	// RemoveUserFromOrganization removes a user from an organization's members array.
	// Returns nil if successful or if user was not a member.
	// Returns error if database error occurs.
	RemoveUserFromOrganization(ctx context.Context, organizationID, userID string) error

	// DeleteOrganization permanently deletes an organization by ID.
	// Returns nil if successful or if organization did not exist.
	// Returns error if database error occurs.
	DeleteOrganization(ctx context.Context, organizationID string) error
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - Interface only | N/A | N/A | N/A |

---

## Step 4: Implement `Create` Method in MongoDB Organization Repository

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`: Existing MongoDB organization repository
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/organization.go`: Organization MongoDB schema
- `/Users/jayce/team-attention/cops/api/internal/service/auth/outbound/repository/mongodb/user_repo.go`: Reference for Create pattern

### `/Users/jayce/team-attention/cops/api/internal/service/user/outbound/repository/mongodb/organization_repo.go`

**Description**:
Add the `Create` method to `MongoOrganizationRepository`. Follow the same pattern as `MongoUserRepository.Create`.

```go
// Create creates a new organization in MongoDB.
func (r *MongoOrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	// Implementation outline:
	// 1. Create mongoschema.Organization variable.
	var schema mongoschema.Organization
	// 2. Call schema.FromDomain(org) to convert domain to schema.
	schema.FromDomain(org)
	// 3. Execute InsertOne on orgColl with ctx and schema.
	//    - Note: If ctx is mongo.SessionContext, operation participates in transaction.
	result, err := r.orgColl.InsertOne(ctx, schema)
	// 4. If InsertOne fails:
	//    a. Log error with org.Slug and error details.
	//    b. Return nil, error.
	if err != nil {
		r.logger.Error("failed to create organization",
			slog.String("slug", org.Slug),
			slog.Any("error", err),
		)
		return nil, err
	}
	// 5. Extract InsertedID from result and cast to bson.ObjectID.
	insertedID, ok := result.InsertedID.(bson.ObjectID)
	// 6. If cast fails, return nil, error.
	if !ok {
		return nil, fmt.Errorf("failed to get inserted organization ID")
	}
	// 7. Set schema.ID to the inserted ObjectID.
	schema.ID = insertedID
	// 8. Return schema.ToDomain(), nil.
	return schema.ToDomain(), nil
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Successful creation | Valid organization | Created organization with ID | Happy path |
| Duplicate slug | Organization with existing slug | Error (duplicate key) | Duplicate key error |
| Invalid member UserID | Organization with invalid UserID format | Error | FromDomain conversion error |

---

## Step 5: Create Slug Generation Utility

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform util package guidelines
- `/Users/jayce/team-attention/cops/api/internal/platform/util/errutil/errutil.go`: Existing utility pattern reference

### `/Users/jayce/team-attention/cops/api/internal/platform/util/slugutil/slugutil.go`

**Description**:
Create a utility package for generating URL-safe slugs with random suffix.

```go
package slugutil

import (
	"crypto/rand"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	// maxSlugBaseLength is the maximum length of the base slug before adding suffix.
	maxSlugBaseLength = 50

	// suffixLength is the length of the random alphanumeric suffix.
	suffixLength = 4

	// suffixChars contains characters used for random suffix generation.
	suffixChars = "abcdefghijklmnopqrstuvwxyz0123456789"
)

// nonAlphanumericRegex matches any character that is not alphanumeric or hyphen.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// multipleHyphenRegex matches multiple consecutive hyphens.
var multipleHyphenRegex = regexp.MustCompile(`-+`)

// GenerateSlug generates a URL-safe slug from the given name with a random suffix.
// The slug format is: {slugified-name}-{random-suffix}
// Example: "Jayce Kim" -> "jayce-kim-a3f9"
func GenerateSlug(name string) (string, error) {
	// Implementation outline:
	// 1. Trim whitespace from name.
	name = strings.TrimSpace(name)
	// 2. If name is empty, use "user" as base.
	if name == "" {
		name = "user"
	}
	// 3. Normalize unicode characters to ASCII using transform package:
	//    a. Apply NFD normalization to decompose accented characters.
	//    b. Remove combining marks (accents) using runes.Remove.
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, err := transform.String(t, name)
	if err != nil {
		// If normalization fails, continue with original name
		normalized = name
	}
	// 4. Convert to lowercase.
	base := strings.ToLower(normalized)
	// 5. Replace spaces with hyphens.
	base = strings.ReplaceAll(base, " ", "-")
	// 6. Remove all non-alphanumeric characters except hyphens using regex.
	base = nonAlphanumericRegex.ReplaceAllString(base, "")
	// 7. Collapse multiple consecutive hyphens into single hyphen.
	base = multipleHyphenRegex.ReplaceAllString(base, "-")
	// 8. Trim leading and trailing hyphens.
	base = strings.Trim(base, "-")
	// 9. If base is empty after processing, use "user".
	if base == "" {
		base = "user"
	}
	// 10. Truncate base slug to maxSlugBaseLength if too long.
	if len(base) > maxSlugBaseLength {
		base = base[:maxSlugBaseLength]
		// Remove trailing hyphen if truncation created one
		base = strings.TrimSuffix(base, "-")
	}
	// 11. Generate random suffix using generateRandomSuffix().
	suffix, err := generateRandomSuffix()
	// 12. If suffix generation fails, return empty string and error.
	if err != nil {
		return "", err
	}
	// 13. Return base slug + "-" + suffix, nil.
	return base + "-" + suffix, nil
}

// generateRandomSuffix generates a random alphanumeric suffix of suffixLength characters.
func generateRandomSuffix() (string, error) {
	// Implementation outline:
	// 1. Create byte slice of suffixLength.
	bytes := make([]byte, suffixLength)
	// 2. Read random bytes from crypto/rand.
	if _, err := rand.Read(bytes); err != nil {
		// 3. If rand.Read fails, return empty string and error.
		return "", err
	}
	// 4. Map each random byte to a character in suffixChars.
	for i, b := range bytes {
		bytes[i] = suffixChars[int(b)%len(suffixChars)]
	}
	// 5. Return the resulting string, nil.
	return string(bytes), nil
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Normal name | "Jayce Kim" | "jayce-kim-XXXX" (4-char suffix) | Happy path |
| Empty name | "" | "user-XXXX" | Empty name fallback |
| Whitespace only | "   " | "user-XXXX" | Whitespace fallback |
| Unicode characters | "Jose Garcia" | "jose-garcia-XXXX" | Unicode normalization |
| Special characters | "John's Team!" | "johns-team-XXXX" | Special char removal |
| Very long name | 100-char name | Truncated to 50 chars + "-XXXX" | Truncation |
| Multiple spaces | "John   Doe" | "john-doe-XXXX" | Multiple hyphen collapse |
| Leading/trailing spaces | "  John Doe  " | "john-doe-XXXX" | Trim handling |

---

## Step 6: Modify Auth Service to Create Personal Organization

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go`: Current auth service implementation
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service package guidelines
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions
- `/Users/jayce/team-attention/cops/shared/domain/organization.go`: Organization domain model

### `/Users/jayce/team-attention/cops/api/internal/service/auth/auth_service.go`

**Description**:
Update the Service struct to include new dependencies and modify the `GoogleAuth` method to create a Personal Organization within a transaction.

#### Updated Service Struct and Constructor

```go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/platform/util/slugutil"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/oauth"
	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// Service implements authentication business logic.
type Service struct {
	logger         *slog.Logger
	cfg            *config.Config
	oauthPort      oauth.GoogleOAuthPort
	userRepo       repository.UserRepositoryPort
	deviceCodeRepo repository.DeviceCodeRepositoryPort
	orgRepo        userrepo.OrganizationRepositoryPort
	txManager      txmanager.TransactionManagerPort
}

// NewService creates a new auth service.
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	oauthPort oauth.GoogleOAuthPort,
	userRepo repository.UserRepositoryPort,
	deviceCodeRepo repository.DeviceCodeRepositoryPort,
	orgRepo userrepo.OrganizationRepositoryPort,
	txManager txmanager.TransactionManagerPort,
) *Service {
	// Implementation outline:
	// 1. Bind logger with component name "auth.service".
	// 2. Store all injected dependencies in struct.
	// 3. Return initialized Service.
	return &Service{
		logger:         l.With(slog.String("name", "auth.service")),
		cfg:            cfg,
		oauthPort:      oauthPort,
		userRepo:       userRepo,
		deviceCodeRepo: deviceCodeRepo,
		orgRepo:        orgRepo,
		txManager:      txManager,
	}
}
```

#### Updated GoogleAuth Method

```go
// signupResult contains the result of user signup transaction.
type signupResult struct {
	User         *domain.User
	Organization *domain.Organization
}

// GoogleAuth handles Google OAuth code exchange and user creation/lookup.
func (s *Service) GoogleAuth(ctx context.Context, params GoogleAuthParams) (*jwtutil.TokenPair, error) {
	// Implementation outline:
	// 1. Exchange authorization code for tokens using oauthPort.ExchangeCode().
	tokenResp, err := s.oauthPort.ExchangeCode(ctx, params.AuthorizationCode, params.RedirectURI)
	// 2. If exchange fails, log error and return wrapped error.
	if err != nil {
		s.logger.Error("failed to exchange authorization code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	// 3. Get user info from Google using oauthPort.GetUserInfo().
	userInfo, err := s.oauthPort.GetUserInfo(ctx, tokenResp.AccessToken)
	// 4. If getting user info fails, log error and return wrapped error.
	if err != nil {
		s.logger.Error("failed to get user info",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	// 5. Look up existing user by account provider using userRepo.FindByAccountProvider().
	user, err := s.userRepo.FindByAccountProvider(ctx, domain.AccountProviderGoogle, userInfo.ID)
	// 6. If lookup fails (not "not found"), log error and return wrapped error.
	if err != nil {
		s.logger.Error("failed to find user by account provider",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// 7. If user exists (existing user login):
	if user != nil {
		// a. Generate JWT token pair using jwtutil.GenerateTokenPair().
		jwtCfg := &jwtutil.Config{
			SecretKey:            s.cfg.JWT.SecretKey,
			AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
			RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
			Issuer:               s.cfg.JWT.Issuer,
		}
		tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(user.ID))
		// b. If token generation fails, log error and return wrapped error.
		if err != nil {
			s.logger.Error("failed to generate tokens for existing user",
				slog.String("userID", string(user.ID)),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to generate tokens: %w", err)
		}
		// c. Log successful login with userID and email.
		s.logger.Info("user logged in",
			slog.String("userID", string(user.ID)),
			slog.String("email", user.Email),
		)
		// d. Return tokens.
		return tokens, nil
	}

	// 8. If user does not exist (new user signup):
	// a. Create newUser domain object with email, name, profileImageURL, accounts.
	newUser := &domain.User{
		Email:           userInfo.Email,
		Name:            userInfo.Name,
		ProfileImageURL: userInfo.Picture,
		Accounts: []*domain.Account{
			{
				Provider:   domain.AccountProviderGoogle,
				ProviderID: userInfo.ID,
			},
		},
	}

	// b. Execute signup within transaction using s.txManager.WithTransaction():
	result, err := s.txManager.WithTransaction(ctx, func(txCtx context.Context) (interface{}, error) {
		// i. Create user using s.userRepo.Create(txCtx, newUser).
		createdUser, err := s.userRepo.Create(txCtx, newUser)
		// ii. If user creation fails, log error and return nil, error (triggers rollback).
		if err != nil {
			s.logger.Error("failed to create user in transaction",
				slog.String("email", newUser.Email),
				slog.Any("error", err),
			)
			return nil, err
		}
		// iii. Generate organization slug using slugutil.GenerateSlug(createdUser.Name).
		orgSlug, err := slugutil.GenerateSlug(createdUser.Name)
		// iv. If slug generation fails, log error and return nil, error (triggers rollback).
		if err != nil {
			s.logger.Error("failed to generate organization slug",
				slog.String("userID", string(createdUser.ID)),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("failed to generate organization slug: %w", err)
		}
		// v. Create newOrg domain object:
		//    - Name: fmt.Sprintf("%s's Organization", createdUser.Name)
		//    - Slug: generated slug
		//    - Members: single member with createdUser.ID and role MemberRoleAdmin
		newOrg := &domain.Organization{
			Name: fmt.Sprintf("%s's Organization", createdUser.Name),
			Slug: orgSlug,
			Members: []*domain.OrganizationMember{
				{
					UserID: createdUser.ID,
					Role:   domain.MemberRoleAdmin,
				},
			},
		}
		// vi. Create organization using s.orgRepo.Create(txCtx, newOrg).
		createdOrg, err := s.orgRepo.Create(txCtx, newOrg)
		// vii. If org creation fails, log error and return nil, error (triggers rollback).
		if err != nil {
			s.logger.Error("failed to create personal organization in transaction",
				slog.String("userID", string(createdUser.ID)),
				slog.String("orgSlug", orgSlug),
				slog.Any("error", err),
			)
			return nil, err
		}
		// viii. Return signupResult{User: createdUser, Organization: createdOrg}, nil.
		return &signupResult{
			User:         createdUser,
			Organization: createdOrg,
		}, nil
	})
	// c. If transaction fails, log error and return wrapped error.
	if err != nil {
		s.logger.Error("user signup transaction failed",
			slog.String("email", userInfo.Email),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to create user account: %w", err)
	}
	// d. Type assert result to *signupResult.
	signup := result.(*signupResult)
	// e. Log successful signup with userID, email, organizationID, organizationSlug.
	s.logger.Info("new user created with personal organization",
		slog.String("userID", string(signup.User.ID)),
		slog.String("email", signup.User.Email),
		slog.String("organizationID", string(signup.Organization.ID)),
		slog.String("organizationSlug", signup.Organization.Slug),
	)

	// 9. Generate JWT token pair for the new user.
	jwtCfg := &jwtutil.Config{
		SecretKey:            s.cfg.JWT.SecretKey,
		AccessTokenDuration:  s.cfg.JWT.AccessTokenDuration,
		RefreshTokenDuration: s.cfg.JWT.RefreshTokenDuration,
		Issuer:               s.cfg.JWT.Issuer,
	}
	tokens, err := jwtutil.GenerateTokenPair(jwtCfg, string(signup.User.ID))
	// 10. If token generation fails, log error and return wrapped error.
	if err != nil {
		s.logger.Error("failed to generate tokens for new user",
			slog.String("userID", string(signup.User.ID)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	// 11. Return tokens.
	return tokens, nil
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Existing user login | Valid auth code, user exists | Token pair | Existing user branch |
| New user signup success | Valid auth code, user not exists | Token pair, user+org created | New user happy path |
| OAuth exchange fails | Invalid auth code | Error | Exchange code error |
| User info fetch fails | Valid code, Google API error | Error | GetUserInfo error |
| User creation fails in transaction | Valid code, DB error | Error, transaction rolled back | User creation error |
| Org creation fails in transaction | Valid code, DB error | Error, both rolled back | Org creation error, rollback |
| Slug generation fails | Valid code, crypto error | Error, both rolled back | Slug generation error |
| Token generation fails | Valid code, JWT error | Error | Token generation error |

---

## Step 7: Update fx Dependency Injection

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_platform.go`: Platform module
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`: Auth module
- `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_user.go`: User module (for org repo reference)
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-container.md`: Container guidelines

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_platform.go`

**Description**:
Add Transaction Manager provider to the platform module.

```go
package container

import (
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"

	"github.com/team-attention/cops/api/internal/platform/outbound/txmanager"
	mongotx "github.com/team-attention/cops/api/internal/platform/outbound/txmanager/mongodb"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/setup/logger"
	"github.com/team-attention/cops/api/internal/platform/setup/mongodb"
	"github.com/team-attention/cops/api/internal/platform/setup/server"
)

func newPlatformModule() fx.Option {
	return fx.Module("platform",
		// Configuration (root - no dependencies)
		fx.Provide(config.InitConfig),

		// Logger (depends on config)
		fx.Provide(logger.InitLogger),

		// MongoDB (depends on config, logger)
		fx.Provide(mongodb.InitMongoDB),

		// Transaction Manager (depends on logger, mongodb)
		// Implementation outline:
		// 1. Use fx.Provide with fx.Annotate.
		// 2. Create provider function that takes *slog.Logger and *mongo.Database.
		// 3. Extract *mongo.Client from database using db.Client().
		// 4. Call mongotx.NewMongoTransactionManager(l, client).
		// 5. Use fx.As to cast to txmanager.TransactionManagerPort interface.
		fx.Provide(
			fx.Annotate(
				func(l *slog.Logger, db *mongo.Database) *mongotx.MongoTransactionManager {
					return mongotx.NewMongoTransactionManager(l, db.Client())
				},
				fx.As(new(txmanager.TransactionManagerPort)),
			),
		),

		// Fiber app (depends on config, logger)
		fx.Provide(server.InitFiber),
	)
}
```

### `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_auth.go`

**Description**:
No changes needed to module_auth.go. The auth.NewService constructor signature is updated (Step 6), and fx will automatically inject the new dependencies:
- `userrepo.OrganizationRepositoryPort` is already provided by `module_user.go`
- `txmanager.TransactionManagerPort` is now provided by `module_platform.go`

fx automatically resolves dependencies by type, so adding new parameters to `auth.NewService` will work without modifying the module registration.

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - DI configuration | N/A | N/A | N/A |

---

## Step 8: Install Required Package for Unicode Normalization

**Description**:
The slug generation utility requires the `golang.org/x/text` package for Unicode normalization (handling accented characters like "Jose" from "Jose").

**Command to Run**:
```bash
cd /Users/jayce/team-attention/cops/api && go get golang.org/x/text
```

---

## Implementation Order and Dependencies

```
Step 8: Install golang.org/x/text package (no dependencies)
    |
Step 1: Transaction Manager Port (Interface)
    |
    v
Step 2: MongoDB Transaction Manager Adapter
    |
Step 3: Organization Repository Port (add Create method)
    |
    v
Step 4: MongoDB Organization Repository (implement Create)
    |
Step 5: Slug Generation Utility
    |
    v
Step 6: Auth Service Modifications (depends on Steps 2, 4, 5)
    |
    v
Step 7: Update module_platform.go (depends on Step 2)
```

**Execution Order**:
1. Step 8 (install package) - can be done first
2. Step 1 (txmanager port)
3. Step 2 (txmanager mongodb adapter)
4. Step 3 (org repo port - add Create method)
5. Step 4 (org repo mongodb - implement Create)
6. Step 5 (slugutil)
7. Step 6 (auth service)
8. Step 7 (module_platform.go)

---

## Testing Considerations

### Unit Tests Required

1. **`slugutil_test.go`**: Test all slug generation scenarios
2. **`auth_service_test.go`**: Mock transaction manager, repositories to test GoogleAuth flow

### Integration Tests Required

1. **Transaction rollback test**: Verify that if organization creation fails, user is also rolled back
2. **Successful signup test**: Verify both user and organization are created with correct data

### Manual Testing Checklist

- [ ] New user signup creates user and organization in same transaction
- [ ] Organization name follows pattern: "{User's Name}'s Organization"
- [ ] Organization slug is URL-safe with random suffix
- [ ] User is added as admin member of the organization
- [ ] Existing user login flow is unchanged
- [ ] Transaction rollback works when organization creation fails

---

## MongoDB Replica Set Requirement

**IMPORTANT**: MongoDB transactions require a replica set deployment.

For local development, ensure `docker-compose.yml` configures MongoDB as a replica set:

```yaml
mongodb:
  image: mongo:8.0
  command: ["--replSet", "rs0", "--bind_ip_all"]
  ports:
    - "27017:27017"
  healthcheck:
    test: echo "try { rs.status() } catch (err) { rs.initiate({_id:'rs0',members:[{_id:0,host:'localhost:27017'}]}) }" | mongosh --quiet
    interval: 5s
    timeout: 5s
    retries: 3
```

Connection string must include `?replicaSet=rs0` parameter.
