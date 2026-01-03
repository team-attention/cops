# Implementation Plan: CLI Authentication via C-Ops Web Device Flow

## Overview

This plan modifies the existing CLI authentication flow to route device code approval through the C-Ops Web application instead of directly through Google OAuth. The current flow sends users to Google's device verification URL, but the new flow will:

1. Generate device codes locally (stored in MongoDB) instead of requesting them from Google
2. Display a C-Ops Web URL for device approval instead of Google's URL
3. Users authenticate via their existing Google login session on C-Ops Web
4. The web application calls a new API endpoint to approve the device code
5. CLI polls the API and receives JWT tokens once the device is approved

This change enables future flexibility (multiple OAuth providers) and provides a consistent authentication experience through the C-Ops platform.

---

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| Add | Generate cryptographically secure random strings | `crypto/rand` (stdlib) | Required for secure device code ID generation |
| None | No new external packages required | - | The implementation uses existing stdlib and MongoDB driver |

---

## Step 1: Define DeviceCode Domain Model

**Files to Read**:
- `.agent/rules/go/go-platform-domain.md`: Domain model guidelines
- `.agent/rules/go/go-struct.md`: Struct definition rules (pointer types)
- `shared/domain/user.go`: Existing domain model pattern

### `shared/domain/device_code.go`

**Description**: Define DeviceCode domain model for device flow authentication. Device codes are stored in MongoDB with TTL-based expiration.

```go
package domain

import "time"

// DeviceCode represents a device authentication code for CLI login flow.
// Device codes are stored in MongoDB with automatic TTL expiration.
type DeviceCode struct {
	ID        ID        `json:"-" bson:"-"`
	UserCode  string    `json:"userCode" bson:"userCode"`
	UserID    *ID       `json:"userId,omitempty" bson:"userId,omitempty"`
	Approved  bool      `json:"approved" bson:"approved"`
	ExpiresAt time.Time `json:"expiresAt" bson:"expiresAt"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Create new device code | Valid struct | DeviceCode with zero values for optional fields | Struct creation |
| Device code with user | Set UserID | UserID pointer is non-nil | Optional field set |

---

## Step 2: Define DeviceCode MongoSchema

**Files to Read**:
- `.agent/rules/go/go-platform-domain-mongoschema.md`: MongoSchema guidelines
- `shared/domain/mongoschema/user.go`: Existing MongoSchema pattern

### `shared/domain/mongoschema/device_code.go`

**Description**: MongoDB schema for DeviceCode with ID conversion. The collection uses a TTL index on expiresAt for automatic cleanup.

```go
package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DeviceCodeCollectionName = "deviceCodes"
)

// DeviceCode struct field constants
const (
	DeviceCodeIDField        = "_id"
	DeviceCodeUserCodeField  = "userCode"
	DeviceCodeUserIDField    = "userId"
	DeviceCodeApprovedField  = "approved"
	DeviceCodeExpiresAtField = "expiresAt"
	DeviceCodeCreatedAtField = "createdAt"
)

type DeviceCode struct {
	domain.DeviceCode `bson:",inline"`
	ID                bson.ObjectID  `bson:"_id,omitempty"`
	UserID            *bson.ObjectID `bson:"userId,omitempty"`
}

func (s *DeviceCode) FromDomain(d *domain.DeviceCode) {
	// Implementation outline:
	// 1. Return early if d is nil.
	// 2. Copy the embedded DeviceCode struct from domain.
	// 3. Convert domain.ID to bson.ObjectID if d.ID is not empty.
	// 4. If d.UserID is not nil and not empty:
	//    a. Convert *domain.ID to bson.ObjectID.
	//    b. Assign to s.UserID pointer.
}

func (s *DeviceCode) ToDomain() *domain.DeviceCode {
	// Implementation outline:
	// 1. Return nil if s is nil.
	// 2. Set s.DeviceCode.ID from s.ID.Hex().
	// 3. If s.UserID is not nil:
	//    a. Convert to domain.ID and assign to s.DeviceCode.UserID pointer.
	// 4. Return pointer to embedded DeviceCode.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| FromDomain with nil | nil | No-op (return early) | Nil check |
| FromDomain without UserID | DeviceCode without UserID | Schema with nil UserID | Optional field nil |
| FromDomain with UserID | DeviceCode with UserID set | Schema with UserID converted | Optional field set |
| ToDomain with nil | nil | nil | Nil check |
| ToDomain without UserID | Schema without UserID | Domain with nil UserID | Optional field nil |
| ToDomain with UserID | Schema with UserID | Domain with UserID pointer | Optional field set |

---

## Step 3: Update Protobuf Service Definition

**Files to Read**:
- `.agent/rules/idl/protobuf.md`: Protobuf conventions
- `idl/protobuf/auth/v1/auth.proto`: Current auth service definition

### Update `idl/protobuf/auth/v1/auth.proto`

**Description**: Add DeviceCodeApprove RPC and messages to the AuthService. The existing DeviceCode and DevicePoll RPCs remain unchanged - only the API implementation changes.

```protobuf
// ADD after DevicePollRes message:

// DeviceCodeApproveReq contains the device code to approve.
message DeviceCodeApproveReq {
  string user_code = 1;
}

// DeviceCodeApproveRes contains the approval result.
message DeviceCodeApproveRes {
  bool success = 1;
  string message = 2;
}

// UPDATE AuthService - ADD new RPC:
service AuthService {
  // ... existing RPCs ...

  // DeviceCodeApprove approves a device code from the web application.
  // Requires authenticated user (JWT in Authorization header).
  rpc DeviceCodeApprove(DeviceCodeApproveReq) returns (DeviceCodeApproveRes);
}
```

After updating the proto file, regenerate code:
```bash
cd idl/protobuf && buf generate
```

---

## Step 4: Add Web URL Configuration to API

**Files to Read**:
- `api/internal/platform/setup/config/config.go`: Existing config structure

### Update `api/internal/platform/setup/config/config.go`

**Description**: Add configuration for C-Ops Web URL and device code settings.

```go
// ADD to Config struct:
type Config struct {
	// ... existing fields ...
	DeviceCode DeviceCodeConfig
}

// ADD new config struct:
// DeviceCodeConfig holds device code flow configuration.
type DeviceCodeConfig struct {
	WebBaseURL string        `env:"COPS_WEB_BASE_URL,required"`
	Expiration time.Duration `env:"COPS_DEVICE_CODE_EXPIRATION" envDefault:"15m"`
	Interval   int           `env:"COPS_DEVICE_CODE_INTERVAL" envDefault:"5"`
}
```

---

## Step 5: Implement Device Code Repository Port

**Files to Read**:
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines
- `api/internal/service/auth/outbound/repository/user_repo_port.go`: Existing repository port pattern

### `api/internal/service/auth/outbound/repository/device_code_repo_port.go`

**Description**: Device code repository interface for MongoDB operations.

```go
package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// DeviceCodeRepositoryPort defines interface for device code data persistence.
type DeviceCodeRepositoryPort interface {
	// Create creates a new device code.
	Create(ctx context.Context, deviceCode *domain.DeviceCode) (*domain.DeviceCode, error)

	// GetByID retrieves device code by its secure ID (used for CLI polling).
	GetByID(ctx context.Context, id string) (*domain.DeviceCode, error)

	// GetByUserCode retrieves device code by its human-friendly user code.
	GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceCode, error)

	// Approve marks a device code as approved and links it to a user.
	// Returns error if device code is already approved, expired, or not found.
	Approve(ctx context.Context, userCode string, userID domain.ID) error
}
```

---

## Step 6: Implement Device Code MongoDB Repository

**Files to Read**:
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines
- `api/internal/service/auth/outbound/repository/mongodb/user_repo.go`: Existing MongoDB repository pattern

### `api/internal/service/auth/outbound/repository/mongodb/device_code_repo.go`

**Description**: MongoDB implementation of device code repository. Uses TTL index for automatic expiration.

```go
package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoDeviceCodeRepository struct {
	logger *slog.Logger
	coll   *mongo.Collection
}

func NewMongoDeviceCodeRepository(l *slog.Logger, db *mongo.Database) *MongoDeviceCodeRepository {
	return &MongoDeviceCodeRepository{
		logger: l.With(slog.String("name", "auth.repository.mongodb.device_code")),
		coll:   db.Collection(mongoschema.DeviceCodeCollectionName),
	}
}

func (r *MongoDeviceCodeRepository) Create(ctx context.Context, deviceCode *domain.DeviceCode) (*domain.DeviceCode, error) {
	// Implementation outline:
	// 1. Create mongoschema.DeviceCode from domain.DeviceCode using FromDomain.
	// 2. Insert into deviceCodes collection.
	// 3. Get inserted ID from result.
	// 4. Set deviceCode.ID from inserted ObjectID.
	// 5. Return deviceCode.
}

func (r *MongoDeviceCodeRepository) GetByID(ctx context.Context, id string) (*domain.DeviceCode, error) {
	// Implementation outline:
	// 1. Convert id string to bson.ObjectID.
	// 2. Build filter: { _id: objectID }
	// 3. Find document.
	// 4. If not found, return nil, nil (not error).
	// 5. Convert to domain using ToDomain().
	// 6. Return device code.
}

func (r *MongoDeviceCodeRepository) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceCode, error) {
	// Implementation outline:
	// 1. Build filter using field constant:
	//    { userCode: userCode }
	// 2. Find document.
	// 3. If not found, return nil, nil (not error).
	// 4. Convert to domain using ToDomain().
	// 5. Return device code.
}

func (r *MongoDeviceCodeRepository) Approve(ctx context.Context, userCode string, userID domain.ID) error {
	// Implementation outline:
	// 1. Convert userID to bson.ObjectID.
	// 2. Build filter:
	//    {
	//      userCode: userCode,
	//      approved: false,
	//      expiresAt: { $gt: time.Now() }
	//    }
	// 3. Build update:
	//    {
	//      $set: { approved: true, userId: userIDObjectID }
	//    }
	// 4. Execute UpdateOne with filter and update.
	// 5. If matched count is 0:
	//    a. Check if device code exists at all.
	//    b. If not found, return "device code not found" error.
	//    c. If found but already approved, return "device code already approved" error.
	//    d. If found but expired, return "device code expired" error.
	// 6. Return nil on success.
}

var _ repository.DeviceCodeRepositoryPort = (*MongoDeviceCodeRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Create valid device code | Valid DeviceCode | DeviceCode with ID set | Happy path |
| GetByID found | Valid ID | DeviceCode | Happy path |
| GetByID not found | Invalid ID | nil, nil | Not found |
| GetByUserCode found | Valid user code | DeviceCode | Happy path |
| GetByUserCode not found | Invalid user code | nil, nil | Not found |
| Approve valid | Valid userCode, userID | nil (success) | Happy path |
| Approve not found | Invalid userCode | Error: not found | Not found branch |
| Approve already approved | Already approved code | Error: already approved | Already approved branch |
| Approve expired | Expired code | Error: expired | Expired branch |

---

## Step 7: Implement Device Code Generation Utility

**Files to Read**:
- `api/internal/service/auth/auth_service.go`: Auth service location

### `api/internal/service/auth/devicecode/devicecode.go`

**Description**: Utility functions for generating secure device codes and human-friendly user codes. This utility is auth service-specific, so it's located within the auth service directory rather than platform.

```go
package devicecode

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// userCodeChars contains characters for human-friendly codes.
// Excludes ambiguous characters: 0, O, I, 1, L
const userCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// GenerateDeviceCodeID generates a cryptographically secure device code ID.
// Returns a 32-character hex string (16 bytes of randomness).
func GenerateDeviceCodeID() (string, error) {
	// Implementation outline:
	// 1. Create byte slice of length 16.
	// 2. Fill with crypto/rand.Read.
	// 3. Return hex encoded string.
}

// GenerateUserCode generates a human-friendly 8-character code with hyphen.
// Format: XXXX-XXXX (e.g., "ABCD-EFGH")
func GenerateUserCode() (string, error) {
	// Implementation outline:
	// 1. Create byte slice of length 8.
	// 2. Fill with crypto/rand.Read.
	// 3. Map each byte to a character from userCodeChars using modulo.
	// 4. Insert hyphen after first 4 characters.
	// 5. Return formatted string (e.g., "ABCD-EFGH").
}

// NormalizeUserCode normalizes user code input by removing hyphens and converting to uppercase.
func NormalizeUserCode(code string) string {
	// Implementation outline:
	// 1. Convert to uppercase.
	// 2. Remove all hyphens.
	// 3. Return normalized string.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Generate device code ID | - | 32-char hex string | Happy path |
| Generate user code | - | 9-char string with hyphen (XXXX-XXXX) | Happy path |
| Normalize lowercase | "abcd-efgh" | "ABCDEFGH" | Uppercase conversion |
| Normalize with hyphen | "ABCD-EFGH" | "ABCDEFGH" | Hyphen removal |
| Normalize without hyphen | "ABCDEFGH" | "ABCDEFGH" | No-op |

---

## Step 8: Update Auth Service

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service implementation guidelines
- `api/internal/service/auth/auth_service.go`: Current auth service implementation

### Update `api/internal/service/auth/auth_service.go`

**Description**: Modify auth service to generate device codes locally and store in MongoDB instead of calling Google OAuth. Add new DeviceCodeApprove method.

#### Update Service struct

```go
// UPDATE Service struct - add new dependencies:
type Service struct {
	logger         *slog.Logger
	cfg            *config.Config
	oauthPort      oauth.GoogleOAuthPort
	userRepo       repository.UserRepositoryPort
	deviceCodeRepo repository.DeviceCodeRepositoryPort
}

// UPDATE NewService - receive *config.Config directly:
func NewService(
	l *slog.Logger,
	cfg *config.Config,
	oauthPort oauth.GoogleOAuthPort,
	userRepo repository.UserRepositoryPort,
	deviceCodeRepo repository.DeviceCodeRepositoryPort,
) *Service {
	return &Service{
		logger:         l.With(slog.String("name", "auth.service")),
		cfg:            cfg,
		oauthPort:      oauthPort,
		userRepo:       userRepo,
		deviceCodeRepo: deviceCodeRepo,
	}
}
```

#### Update DeviceCode method

```go
// REPLACE existing DeviceCode method:
func (s *Service) DeviceCode(ctx context.Context) (*DeviceCodeResult, error) {
	// Implementation outline:
	// 1. Generate human-friendly user code using devicecode.GenerateUserCode().
	// 2. Calculate expiration time (now + s.cfg.DeviceCode.Expiration).
	// 3. Create domain.DeviceCode:
	//    - UserCode: generated user code (store without hyphen for consistency)
	//    - Approved: false
	//    - ExpiresAt: calculated expiration
	//    - CreatedAt: time.Now()
	// 4. Save to MongoDB via deviceCodeRepo.Create().
	// 5. Build verification URL: fmt.Sprintf("%s/auth/device?code=%s", s.cfg.DeviceCode.WebBaseURL, userCode)
	// 6. Return DeviceCodeResult:
	//    - DeviceCode: the MongoDB document ID (for CLI polling)
	//    - UserCode: the human-friendly code with hyphen (for display)
	//    - VerificationURL: the C-Ops Web URL
	//    - ExpiresIn: int(s.cfg.DeviceCode.Expiration.Seconds())
	//    - Interval: s.cfg.DeviceCode.Interval
}
```

#### Update DevicePoll method

```go
// REPLACE existing DevicePoll method:
func (s *Service) DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error) {
	// Implementation outline:
	// 1. Get device code from MongoDB by ID via deviceCodeRepo.GetByID(deviceCode).
	// 2. If not found, return error "device code not found".
	// 3. Check if expired (ExpiresAt < time.Now()):
	//    a. If expired, return error "device code expired".
	// 4. If not approved:
	//    a. Return DevicePollResult{Pending: true}.
	// 5. If approved:
	//    a. Get userID from device code.
	//    b. Fetch user from userRepo.GetByID to verify user still exists.
	//    c. If user not found, return error "user not found".
	//    d. Generate JWT token pair using jwtutil.GenerateTokenPair with s.cfg.JWT config.
	//    e. Return DevicePollResult{Pending: false, Tokens: tokens}.
}
```

#### Add DeviceCodeApprove method

```go
// ADD new DeviceCodeApprove method:

// DeviceCodeApproveParams contains parameters for device code approval.
type DeviceCodeApproveParams struct {
	UserCode string
	UserID   domain.ID
}

// DeviceCodeApprove approves a device code and links it to the authenticated user.
func (s *Service) DeviceCodeApprove(ctx context.Context, params DeviceCodeApproveParams) error {
	// Implementation outline:
	// 1. Normalize user code using devicecode.NormalizeUserCode().
	// 2. Verify user exists via userRepo.GetByID(params.UserID).
	// 3. If user not found, return error "user not found".
	// 4. Call deviceCodeRepo.Approve(normalizedCode, params.UserID).
	// 5. If error, return the error (already contains appropriate message).
	// 6. Log successful approval.
	// 7. Return nil.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| DeviceCode success | - | DeviceCodeResult with C-Ops Web URL | Happy path |
| DevicePoll pending | Valid device code, not approved | Pending: true | Pending state |
| DevicePoll approved | Valid device code, approved | Tokens returned | Approved state |
| DevicePoll not found | Invalid device code | Error: not found | Not found |
| DevicePoll expired | Expired device code | Error: expired | Expired |
| DeviceCodeApprove success | Valid user code, valid user | nil | Happy path |
| DeviceCodeApprove invalid user | Valid user code, invalid user | Error: user not found | User validation |
| DeviceCodeApprove not found | Invalid user code | Error: not found | Not found |
| DeviceCodeApprove already approved | Already approved code | Error: already approved | Already approved |
| DeviceCodeApprove expired | Expired code | Error: expired | Expired |

---

## Step 9: Update Auth ConnectRPC Handler

**Files to Read**:
- `.agent/rules/go/go-inbound-grpc-connectrpc.md`: ConnectRPC handler guidelines
- `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`: Current handler implementation

### Update `api/internal/service/auth/inbound/grpc/connectrpc/handler.go`

**Description**: Add DeviceCodeApprove handler method that extracts user ID from JWT and calls the service.

```go
// ADD new import for jwtutil:
import (
	// ... existing imports ...
	"strings"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/shared/domain"
)

// UPDATE AuthGRPCHandler struct:
type AuthGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
	cfg    *config.Config
}

// UPDATE NewAuthGRPCHandler - receive *config.Config directly:
func NewAuthGRPCHandler(l *slog.Logger, svc *auth.Service, cfg *config.Config) *AuthGRPCHandler {
	return &AuthGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "auth.grpc.connectrpc")),
		cfg:    cfg,
	}
}

// ADD DeviceCodeApprove method:
func (h *AuthGRPCHandler) DeviceCodeApprove(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
	// Implementation outline:
	// 1. Extract Authorization header from req.Header().
	// 2. Validate format (must start with "Bearer ").
	// 3. Extract token string.
	// 4. Build jwtutil.Config from h.cfg.JWT.
	// 5. Validate JWT using jwtutil.ValidateAccessToken with the config.
	// 6. If validation fails, return connect.NewError(connect.CodeUnauthenticated, ...).
	// 7. Build params:
	//    params := auth.DeviceCodeApproveParams{
	//        UserCode: req.Msg.UserCode,
	//        UserID:   domain.ID(userID),
	//    }
	// 8. Call h.svc.DeviceCodeApprove(ctx, params).
	// 9. If error:
	//    a. Log the error.
	//    b. Determine appropriate connect error code:
	//       - Contains "not found" -> CodeNotFound
	//       - Contains "expired" -> CodeDeadlineExceeded
	//       - Contains "already approved" -> CodeAlreadyExists
	//       - Default -> CodeInternal
	//    c. Return error.
	// 10. Return success response:
	//    res := &authv1.DeviceCodeApproveRes{
	//        Success: true,
	//        Message: "Device approved successfully",
	//    }
}
```

---

## Step 10: Update Auth Module Registration

**Files to Read**:
- `api/cmd/internal/container/module_auth.go`: Current module registration

### Update `api/cmd/internal/container/module_auth.go`

**Description**: Register new device code repository and update service/handler constructors.

```go
// UPDATE the module to include new dependencies:

func newAuthModule() fx.Option {
	return fx.Module("auth",
		// OAuth adapter
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

		// ADD: Device code repository
		fx.Provide(
			fx.Annotate(
				mongodb.NewMongoDeviceCodeRepository,
				fx.As(new(repository.DeviceCodeRepositoryPort)),
			),
		),

		// Service - receives *config.Config directly
		fx.Provide(auth.NewService),

		// ConnectRPC handler - receives *config.Config directly
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

---

## Step 11: Create Web Auth Feature Directory Structure

**Files to Read**:
- `.agent/rules/react/react-web-src.md`: Web source directory rules
- `web/src/feature/dashboard/hook/use-get-overview.ts`: Example hook pattern

### Create directory structure

Create the following directory structure for the auth feature:

```
web/src/feature/auth/
├── component/
│   └── device-approval.tsx
├── hook/
│   └── use-approve-device.ts
└── type/
    └── device-code.ts
```

---

## Step 12: Implement Web Auth Types

**Files to Read**:
- `.agent/rules/react/react-web.md`: TypeScript type rules

### `web/src/feature/auth/type/device-code.ts`

**Description**: Type definitions for device code approval flow.

```typescript
// DeviceApprovalState represents the current state of device approval flow.
interface DeviceApprovalPending {
  status: 'pending';
}

interface DeviceApprovalSuccess {
  status: 'success';
  message: string;
}

interface DeviceApprovalError {
  status: 'error';
  errorCode: DeviceApprovalErrorCode;
  message: string;
}

type DeviceApprovalErrorCode =
  | 'NOT_FOUND'
  | 'EXPIRED'
  | 'ALREADY_APPROVED'
  | 'UNAUTHORIZED'
  | 'UNKNOWN';

export type DeviceApprovalState =
  | DeviceApprovalPending
  | DeviceApprovalSuccess
  | DeviceApprovalError;
```

---

## Step 13: Implement Web Auth Hook

**Files to Read**:
- `web/src/feature/project/hook/use-get-project.ts`: Example hook pattern
- `web/src/shared/service/connect-transport.ts`: Transport configuration

### `web/src/feature/auth/hook/use-approve-device.ts`

**Description**: Hook for calling the DeviceCodeApprove API endpoint.

```typescript
import { useMutation } from '@connectrpc/connect-query';
import { deviceCodeApprove } from '@/gen/grpcstub/auth/v1/auth-AuthService_connectquery';
import { transport } from '@/shared/service/connect-transport';

export const useApproveDevice = () => {
  return useMutation(deviceCodeApprove, { transport });
};
```

Note: The transport needs to be updated to include authentication headers. See Step 15.

---

## Step 14: Implement Device Approval Component

**Files to Read**:
- `.agent/rules/react/react-web.md`: Component rules
- `web/src/route/dashboard.tsx`: Example page component pattern

### `web/src/feature/auth/component/device-approval.tsx`

**Description**: Component for displaying device approval UI with approve button.

```typescript
import { useState } from 'react';
import { CheckCircle, XCircle, Loader2, Terminal, Shield } from 'lucide-react';
import { Button } from '@/gen/shadcn/ui/button';
import { Card } from '@/gen/shadcn/ui/card';
import { useApproveDevice } from '../hook/use-approve-device';
import type { DeviceApprovalState } from '../type/device-code';

interface DeviceApprovalProps {
  userCode: string;
}

export const DeviceApproval = ({ userCode }: DeviceApprovalProps) => {
  // Component implementation outline:
  // 1. State for approval status: DeviceApprovalState (initial: 'pending').
  // 2. useApproveDevice mutation hook.
  // 3. handleApprove function:
  //    a. Call mutation with { userCode }.
  //    b. On success: Set state to 'success'.
  //    c. On error: Parse error and set appropriate error state.
  // 4. Render based on state:
  //    a. 'pending': Show user code display + Approve button.
  //    b. 'success': Show success message with checkmark icon.
  //    c. 'error': Show error message with appropriate icon.

  // UI Elements:
  // - Card container with dark theme styling (zinc-900 background).
  // - Terminal icon for CLI context.
  // - User code displayed prominently in monospace font.
  // - Approve button with loading state during mutation.
  // - Success state shows "Device approved! You can return to your terminal."
  // - Error states show specific messages based on error code.
};
```

---

## Step 15: Update Web Transport for Authentication

**Files to Read**:
- `web/src/shared/service/connect-transport.ts`: Current transport
- `.agent/rules/react/react-web-src.md`: Service layer guidelines

### Update `web/src/shared/service/connect-transport.ts`

**Description**: Update transport to include authentication token from localStorage.

```typescript
import { createConnectTransport } from '@connectrpc/connect-web';

// createAuthTransport creates a transport that includes the access token in requests.
const createAuthTransport = () => {
  return createConnectTransport({
    baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:8080',
    interceptors: [
      (next) => async (req) => {
        // Implementation outline:
        // 1. Get access token from localStorage (key: 'cops_access_token').
        // 2. If token exists:
        //    a. Set Authorization header: `Bearer ${token}`.
        // 3. Call next(req).
        // 4. Return response.
      },
    ],
  });
};

export const transport = createAuthTransport();
```

---

## Step 16: Create Device Approval Route

**Files to Read**:
- `web/src/route/dashboard.tsx`: Example route pattern
- `web/src/route/__root.tsx`: Root layout

### `web/src/route/auth/device.tsx`

**Description**: Route component for device approval page at `/auth/device`.

```typescript
import { createFileRoute, useNavigate, useSearch } from '@tanstack/react-router';
import { useEffect } from 'react';
import { Shield, AlertCircle } from 'lucide-react';
import { DeviceApproval } from '@/feature/auth/component/device-approval';

// Route search params type
interface DeviceSearchParams {
  code?: string;
}

export const Route = createFileRoute('/auth/device')({
  component: DeviceApprovalPage,
  validateSearch: (search: Record<string, unknown>): DeviceSearchParams => {
    return {
      code: typeof search.code === 'string' ? search.code : undefined,
    };
  },
});

function DeviceApprovalPage() {
  // Implementation outline:
  // 1. Get 'code' from search params using useSearch().
  // 2. Check if user is authenticated:
  //    a. Check for access token in localStorage.
  //    b. If not authenticated:
  //       - Store current URL (with code) in sessionStorage.
  //       - Redirect to /auth/login?returnUrl=/auth/device?code=XXX
  // 3. If no code parameter:
  //    a. Show error: "No device code provided".
  // 4. If authenticated and code exists:
  //    a. Render DeviceApproval component with userCode prop.

  // UI Layout:
  // - Centered card layout.
  // - Shield icon header.
  // - "Approve CLI Access" title.
  // - DeviceApproval component or error state.
}
```

---

## Step 17: Update CLI Login Display (Minimal Change)

**Files to Read**:
- `cli/internal/service/auth/inbound/cli/cobra/login.go`: Current login command

### Update `cli/internal/service/auth/inbound/cli/cobra/login.go`

**Description**: Update the display message to remove the user code entry instruction since users no longer need to enter the code manually - it's included in the URL.

```go
// CURRENT implementation shows:
// fmt.Println("To sign in, open this URL in your browser:")
// fmt.Printf("  %s\n\n", result.VerificationURL)
// fmt.Println("Then enter this code:")
// fmt.Printf("  %s\n\n", result.UserCode)
// fmt.Println("Waiting for authentication...")

// UPDATE to:
func (h *AuthCLIHandler) NewLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in to C-Ops",  // UPDATE: Remove "with Google OAuth"
		RunE: func(cmd *cobra.Command, args []string) error {
			// ... existing context setup ...

			result, err := h.svc.InitiateLogin(ctx)
			if err != nil {
				return fmt.Errorf("failed to initiate login: %w", err)
			}

			// UPDATE display message:
			fmt.Println("To sign in, open this URL in your browser:")
			fmt.Printf("\n  %s\n\n", result.VerificationURL)
			fmt.Printf("Device code: %s\n\n", result.UserCode)
			fmt.Println("Waiting for authentication...")

			// ... rest of polling loop unchanged ...
		},
	}
}
```

Note: The CLI polling logic remains unchanged - it still polls the same DevicePoll endpoint.

---

## Step 18: Document MongoDB TTL Index

**Files to Read**:
- Project documentation for migration patterns

### Document MongoDB indexes for deviceCodes collection

**Description**: Document the MongoDB TTL index that needs to be created for the deviceCodes collection. These indexes should be added to the project documentation (e.g., `doc/mongodb-indexes.md` or `TODO.md`).

```javascript
// MongoDB indexes for deviceCodes collection
// Note: Collection name is camelCase "deviceCodes"

// TTL index for automatic expiration (documents deleted after expiresAt time)
db.deviceCodes.createIndex(
  { "expiresAt": 1 },
  { expireAfterSeconds: 0 }
);

// Unique index on userCode for quick lookup
db.deviceCodes.createIndex(
  { "userCode": 1 },
  { unique: true }
);
```

**Important**: Add this documentation to `doc/mongodb-indexes.md` or `TODO.md`. The indexes will be created manually or via a separate database setup process - they are not part of the application code.

---

## Implementation Order

1. **Step 1-2**: DeviceCode domain model and MongoSchema (shared module)
2. **Step 3**: Protobuf definitions and regenerate code (`buf generate`)
3. **Step 4**: API configuration updates
4. **Step 5-6**: Device code repository port and MongoDB implementation
5. **Step 7**: Device code generation utility
6. **Step 8**: Auth service updates (DeviceCode, DevicePoll, DeviceCodeApprove)
7. **Step 9-10**: ConnectRPC handler and module registration updates
8. **Step 11-16**: Web application (auth feature, route, transport)
9. **Step 17**: CLI display update
10. **Step 18**: MongoDB index documentation

---

## Testing Strategy

### Unit Tests

1. **Device Code Generation**: Test random code generation produces valid format
2. **MongoSchema Conversions**: Test FromDomain/ToDomain with various states (nil UserID, set UserID)
3. **Service Logic**: Test DeviceCode, DevicePoll, DeviceCodeApprove with mocked repositories

### Integration Tests

1. **Device Code Repository**: CRUD operations with real MongoDB
2. **Auth Service Flow**: Full device code lifecycle (create -> poll -> approve -> poll with tokens)
3. **ConnectRPC Handler**: JWT extraction and error mapping

### E2E Tests

1. **CLI Login Flow**: Verify C-Ops Web URL is displayed (not Google URL)
2. **Web Approval Flow**: Visit URL -> Approve -> CLI receives tokens
3. **Error Scenarios**: Expired codes, already approved codes, invalid codes

### Manual Testing Checklist

- [ ] Run `cops auth login` and verify C-Ops Web URL is displayed
- [ ] Visit URL while logged out -> Should redirect to login -> Then back to approval page
- [ ] Visit URL while logged in -> Should show approval page immediately
- [ ] Approve device -> Should show success message
- [ ] Verify CLI receives tokens and stores in `~/.cops/auth.json`
- [ ] Try to approve same code twice -> Should show error
- [ ] Wait 15 minutes -> Code should be expired and rejected

---

## Critical Implementation Notes

1. **Remove Google Device Flow Calls**: The `oauthPort.InitiateDeviceFlow` and `oauthPort.PollDeviceCode` methods are no longer used for CLI authentication. They can remain for potential future use but are not called by the new flow.

2. **TTL Index is Critical**: The MongoDB TTL index on `expiresAt` is essential for automatic cleanup. Without it, expired device codes will accumulate.

3. **User Code Format**: The hyphenated format (XXXX-XXXX) is for display only. Store and compare without hyphen for simplicity.

4. **JWT in ConnectRPC**: The DeviceCodeApprove endpoint requires authentication. Since ConnectRPC handlers don't use Fiber middleware, JWT validation must be done directly in the handler.

5. **Web Auth State**: The web application stores authentication state in localStorage. The transport interceptor reads from there.

6. **Single-Use Codes**: Once a device code is approved, it cannot be approved again. The repository enforces this.

7. **Error Mapping**: The service returns descriptive errors ("not found", "expired", "already approved") which the handler maps to appropriate gRPC error codes.

8. **No Organization Scoping**: JWT tokens contain only user ID. Users can access all organizations they belong to.

9. **CLI Minimal Changes**: The CLI code changes are minimal - only the display message changes. The polling mechanism remains identical.

10. **Web Redirect Flow**: If user is not logged in, they are redirected to login with a returnUrl parameter that brings them back to the device approval page with the code preserved.
