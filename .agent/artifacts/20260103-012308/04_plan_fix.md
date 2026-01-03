# Implementation Plan: Fix DeviceCode Domain Model Rule Violations

## Overview

This plan addresses two rule violations identified in the code review of `shared/domain/device_code.go`:

1. **Remove `CreatedAt` field**: The field is metadata unrelated to business logic. MongoDB provides TTL-based expiration using `ExpiresAt`, making `CreatedAt` unnecessary.
2. **Fix `UserID` BSON tag**: Change from `bson:"userId,omitempty"` to `bson:"-"` to follow the rule that ID fields must be excluded from BSON serialization (handled in MongoSchema layer).

The changes are minimal and focused solely on fixing these violations without introducing new features or refactoring unrelated code.

## Package Changes

No external packages need to be added or removed.

## Step 1: Remove CreatedAt Field from Domain Model

**Files to Read**:
- `.agent/rules/go/go-platform-domain.md`: Contains the rule about avoiding metadata fields
- `shared/domain/device_code.go`: Current implementation with violations
- `shared/domain/project.go`: Example showing `RegisteredAt` is used for business logic (project registration timestamp), not generic metadata

#### `shared/domain/device_code.go`

**Description**:
Remove the `CreatedAt` field from the `DeviceCode` struct. This field is metadata that serves no business purpose. MongoDB's TTL mechanism uses `ExpiresAt` for automatic document deletion, making creation timestamps unnecessary.

**Current Implementation**:
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
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`  // REMOVE THIS LINE
}
```

**Changes Required**:
1. Remove line 13: `CreatedAt time.Time `json:"createdAt" bson:"createdAt"``
2. Keep the `time` import as it's still needed for `ExpiresAt`

**Expected Result**:
```go
package domain

import "time"

// DeviceCode represents a device authentication code for CLI login flow.
// Device codes are stored in MongoDB with automatic TTL expiration.
type DeviceCode struct {
	ID        ID        `json:"-" bson:"-"`
	UserCode  string    `json:"userCode" bson:"userCode"`
	UserID    *ID       `json:"userId,omitempty" bson:"-"`  // Note: bson tag will be fixed in Step 2
	Approved  bool      `json:"approved" bson:"approved"`
	ExpiresAt time.Time `json:"expiresAt" bson:"expiresAt"`
}
```

## Step 2: Fix UserID BSON Tag in Domain Model

**Files to Read**:
- `.agent/rules/go/go-platform-domain.md`: Contains the rule "ALWAYS exclude ID fields with `bson:\"-\"`"
- `shared/domain/user.go`: Example showing `ID` field uses `bson:"-"`
- `shared/domain/device_code.go`: Current implementation (already read in Step 1)

#### `shared/domain/device_code.go`

**Description**:
Change the `UserID` field's BSON tag from `bson:"userId,omitempty"` to `bson:"-"`. According to the platform domain rules, ALL ID fields (including nested ID references like `UserID`) must be excluded from BSON serialization because they are handled in the MongoSchema layer where they're converted to `bson.ObjectID`.

**Current Line (after Step 1)**:
```go
UserID    *ID       `json:"userId,omitempty" bson:"userId,omitempty"`
```

**Changes Required**:
1. Change `bson:"userId,omitempty"` to `bson:"-"`
2. Keep `json:"userId,omitempty"` unchanged (JSON serialization is separate from BSON)

**Expected Result**:
```go
UserID    *ID       `json:"userId,omitempty" bson:"-"`
```

**Full File After Step 2**:
```go
package domain

import "time"

// DeviceCode represents a device authentication code for CLI login flow.
// Device codes are stored in MongoDB with automatic TTL expiration.
type DeviceCode struct {
	ID        ID        `json:"-" bson:"-"`
	UserCode  string    `json:"userCode" bson:"userCode"`
	UserID    *ID       `json:"userId,omitempty" bson:"-"`
	Approved  bool      `json:"approved" bson:"approved"`
	ExpiresAt time.Time `json:"expiresAt" bson:"expiresAt"`
}
```

## Step 3: Update MongoSchema Field Constants

**Files to Read**:
- `.agent/rules/go/go-platform-domain-mongoschema.md`: Contains rules for MongoSchema field constants
- `shared/domain/mongoschema/device_code.go`: Current implementation with field constants

#### `shared/domain/mongoschema/device_code.go`

**Description**:
Remove the `DeviceCodeCreatedAtField` constant since the `CreatedAt` field no longer exists in the domain model. The MongoSchema layer should not reference fields that don't exist.

**Current Field Constants (lines 13-20)**:
```go
// DeviceCode struct field constants
const (
	DeviceCodeIDField        = "_id"
	DeviceCodeUserCodeField  = "userCode"
	DeviceCodeUserIDField    = "userId"
	DeviceCodeApprovedField  = "approved"
	DeviceCodeExpiresAtField = "expiresAt"
	DeviceCodeCreatedAtField = "createdAt"  // REMOVE THIS LINE
)
```

**Changes Required**:
1. Remove line 19: `DeviceCodeCreatedAtField = "createdAt"`

**Expected Result**:
```go
// DeviceCode struct field constants
const (
	DeviceCodeIDField        = "_id"
	DeviceCodeUserCodeField  = "userCode"
	DeviceCodeUserIDField    = "userId"
	DeviceCodeApprovedField  = "approved"
	DeviceCodeExpiresAtField = "expiresAt"
)
```

**Note**: The `DeviceCode` struct in mongoschema (lines 22-26) does not need changes. It uses `bson:",inline"` to embed the domain model, and overrides only the ID fields. Since we're removing `CreatedAt` from the domain model and fixing the `UserID` BSON tag, the mongoschema struct will automatically inherit these changes through the inline embedding.

## Step 4: Remove CreatedAt Assignment in Service Layer

**Files to Read**:
- `.agent/rules/go/go-service.md`: Service layer patterns
- `api/internal/service/auth/auth_service.go`: Current implementation that assigns `CreatedAt`

#### `api/internal/service/auth/auth_service.go`

**Description**:
Remove the line that assigns `CreatedAt: time.Now()` when creating a new DeviceCode. This field no longer exists in the domain model.

**Current Implementation (lines 172-177)**:
```go
deviceCodeData := &domain.DeviceCode{
	UserCode:  devicecode.NormalizeUserCode(userCode),
	Approved:  false,
	ExpiresAt: expiresAt,
	CreatedAt: time.Now(),  // REMOVE THIS LINE
}
```

**Changes Required**:
1. Remove line 176: `CreatedAt: time.Now(),`

**Expected Result**:
```go
deviceCodeData := &domain.DeviceCode{
	UserCode:  devicecode.NormalizeUserCode(userCode),
	Approved:  false,
	ExpiresAt: expiresAt,
}
```

## Step 5: Verify No Other CreatedAt References

**Files to Read**:
- All files found by grep search for `CreatedAt` related to DeviceCode (already searched)

**Description**:
Based on the grep search performed during planning, verify that no other code references `DeviceCode.CreatedAt`. The search found:
- `api/internal/service/auth/auth_service.go:176` - Assignment (removed in Step 4)
- `shared/domain/device_code.go:13` - Field definition (removed in Step 1)
- `shared/domain/mongoschema/device_code.go:19` - Constant (removed in Step 3)

Other `CreatedAt` references found are unrelated (dashboard converter uses `Project.RegisteredAt`, not `DeviceCode.CreatedAt`).

**Verification Steps**:
1. After completing Steps 1-4, run `go build ./shared/... ./api/...` to ensure no compilation errors
2. If build succeeds, all DeviceCode.CreatedAt references have been properly removed
3. If build fails, identify the file and remove the reference

**Expected Outcome**:
Successful build with no errors.

## Quality Checklist

Before considering this implementation complete:

- [ ] `CreatedAt` field removed from `shared/domain/device_code.go`
- [ ] `UserID` BSON tag changed to `bson:"-"` in `shared/domain/device_code.go`
- [ ] `DeviceCodeCreatedAtField` constant removed from `shared/domain/mongoschema/device_code.go`
- [ ] `CreatedAt: time.Now()` assignment removed from `api/internal/service/auth/auth_service.go`
- [ ] Code builds successfully with `go build ./shared/... ./api/...`
- [ ] No other files reference `DeviceCode.CreatedAt`

## Out of Scope

The following are explicitly excluded from this implementation:

- Refactoring JWT config duplication (noted as code quality suggestion in review)
- Changing error handling patterns (noted as code quality suggestion in review)
- Adding new features or functionality
- Modifying any files beyond the four specified in this plan
- Performance optimizations
- Test file updates (no tests currently exist for DeviceCode)
