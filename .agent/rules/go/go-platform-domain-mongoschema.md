---
trigger: always_on
globs: **/internal/platform/domain/mongoschema/*.go
paths: **/internal/platform/domain/mongoschema/*.go
---

# Platform Domain MongoSchema Guidelines

MongoSchema files bridge domain models and MongoDB storage by converting `domain.ID` (string) to `bson.ObjectID` and vice versa.

## Purpose

The `mongoschema` package:
- Embeds domain models for code reuse
- Converts domain IDs (strings) to MongoDB ObjectIDs
- Provides bidirectional conversion methods (FromDomain, ToDomain)
- Defines collection and field name constants to avoid magic strings

## Mandatory File Structure

**Files MUST follow this exact order:**

1. Package declaration
2. Imports
3. Collection name constant
4. Field name constants
5. Struct definition
6. FromDomain method
7. ToDomain method

**Example:**
```go
package mongoschema

import (
    "github.com/rcvbridge/recoverybridge/server/api/internal/platform/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    AddressCollectionName = "addresses"
)

const (
    AddressIDField         = "_id"
    AddressProjectIDField  = "projectId"
    // ... more fields
)

type Address struct {
    domain.Address `bson:",inline"`
    ID             bson.ObjectID `bson:"_id,omitempty"`
    ProjectID      bson.ObjectID `bson:"projectId"`
}

func (s *Address) FromDomain(d *domain.Address) {
    // Implementation
}

func (s *Address) ToDomain() *domain.Address {
    // Implementation
}
```

## Collection Name Constants

### Pattern
```go
const (
    {Entity}CollectionName = "{collection}"
)
```

### Rules
- **Naming**: `{Entity}CollectionName` (e.g., `UserCollectionName`, `AddressCollectionName`)
- **Placement**: First const block in the file
- **Type**: Untyped string constant

## Field Name Constants

### Pattern
```go
const (
    {Entity}{FieldName}Field = "{bsonName}"
)
```

### Rules
- **Naming**: `{Entity}{FieldName}Field` (e.g., `AddressIDField`, `AddressProjectIDField`)
- **Placement**: Second const block in the file
- **All fields**: Include constants for ALL fields in the struct (both ID and domain fields)
- **Ordering**: ID fields first, then domain fields
- **Values**: Match the BSON tag names exactly (single field name, no dots)

### Embedded Struct Types

When a struct contains embedded structs (e.g., `User` contains `[]Account`), each struct type gets its OWN set of field constants with that type as prefix.

**Example: User with embedded Accounts array**

```go
// User struct field constants
const (
    UserIDField              = "_id"
    UserEmailField           = "email"
    UserNameField            = "name"
    UserProfileImageURLField = "profileImageUrl"
    UserAccountsField        = "accounts"  // The array field on User struct
)

// Account struct field constants (separate type, separate constants)
const (
    AccountProviderField   = "provider"    // Account struct's Provider field
    AccountProviderIDField = "providerId"  // Account struct's ProviderID field
)
```

**IMPORTANT Rules:**
- Each struct type has its own prefix (e.g., `User`, `Account`)
- Field constant names and values should NOT contain dots (`.`)
- Values match the exact BSON field name for that struct only
- When querying nested fields in MongoDB, combine constants at query time:
  ```go
  // Query example: find user by account provider
  filter := bson.M{
      mongoschema.UserAccountsField: bson.M{
          "$elemMatch": bson.M{
              mongoschema.AccountProviderField:   provider,
              mongoschema.AccountProviderIDField: providerID,
          },
      },
  }
  ```

## Struct Embedding Pattern

### Rules
- **Embed domain model** with `bson:",inline"` tag
- **Override ID fields** as `bson.ObjectID` type
- **Use proper BSON tags** for each field

### Pattern
```go
type {Entity} struct {
    domain.{Entity} `bson:",inline"`
    ID              bson.ObjectID   `bson:"_id,omitempty"`
    // Additional ID fields as needed
}
```

## FromDomain Method

### Purpose
Converts a domain model to a MongoSchema model for database storage.

### Pattern
```go
func (s *{Entity}) FromDomain(d *domain.{Entity}) {
    if d == nil {
        return
    }

    s.{Entity} = *d

    // Convert ID fields
    if d.ID != "" {
        s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
    }

    // Additional ID field conversions as needed
}
```

### Rules
1. **Nil-safe**: Return early if domain object is nil
2. **Copy embedded struct**: Assign `s.{Entity} = *d` to copy all domain fields
3. **Validate before convert**: Check `d.ID != ""` before conversion
4. **Implicit error ignore**: Use `s.ID, _ = bson.ObjectIDFromHex(...)` to ignore conversion errors
5. **Type casting**: Convert `domain.ID` to string: `string(d.ID)`

## ToDomain Method

### Purpose
Converts a MongoSchema model back to a domain model for business logic.

### Pattern
```go
func (s *{Entity}) ToDomain() *domain.{Entity} {
    if s == nil {
        return nil
    }

    s.{Entity}.ID = domain.ID(s.ID.Hex())
    // Additional ID conversions as needed

    return &s.{Entity}
}
```

### Rules
1. **Nil-safe**: Return nil if schema is nil
2. **Direct modification**: Modify the embedded struct directly (`s.{Entity}.ID = ...`)
3. **Convert to hex**: Use `s.ID.Hex()` to get string representation
4. **Type casting**: Wrap in `domain.ID()`: `domain.ID(s.ID.Hex())`
5. **Return embedded struct**: Return pointer to the embedded struct: `return &s.{Entity}`

## IMPORTANT: No Factory Functions

**DO NOT create factory functions like `To{Entity}Schema`.**

Only include:
- `FromDomain` method
- `ToDomain` method

**❌ WRONG:**
```go
// DO NOT create this
func ToAddressSchema(d *domain.Address) *Address {
    s := &Address{}
    s.FromDomain(d)
    return s
}
```

**✅ CORRECT:**
```go
// Only these two methods
func (s *Address) FromDomain(d *domain.Address) {
    // Implementation
}

func (s *Address) ToDomain() *domain.Address {
    // Implementation
}
```

## Complete Example
```go
package mongoschema

import (
    "example.com/internal/platform/domain"
    "go.mongodb.org/mongo-driver/v2/bson"
)

const (
    AddressCollectionName = "addresses"
)

const (
    AddressIDField         = "_id"
    AddressProjectIDField  = "projectId"
    AddressSourceIDField   = "sourceId"
    AddressParentIDsField  = "parents"
    AddressStepField       = "step"
    AddressOptOutField     = "optOut"
    AddressStatusField     = "status"
    AddressUniqueIDField   = "uniqueId"
    AddressOriginalField   = "original"
    AddressNormalizedField = "normalized"
)

type Address struct {
    domain.Address `bson:",inline"`
    ID             bson.ObjectID   `bson:"_id,omitempty"`
    ProjectID      bson.ObjectID   `bson:"projectId"`
    SourceID       bson.ObjectID   `bson:"sourceId"`
    ParentIDs      []bson.ObjectID `bson:"parents"`
}

func (s *Address) FromDomain(d *domain.Address) {
    if d == nil {
        return
    }

    s.Address = *d

    if d.ID != "" {
        s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
    }

    if d.ProjectID != "" {
        s.ProjectID, _ = bson.ObjectIDFromHex(string(d.ProjectID))
    }

    if d.SourceID != "" {
        s.SourceID, _ = bson.ObjectIDFromHex(string(d.SourceID))
    }

    for _, pid := range d.ParentIDs {
        if string(pid) != "" {
            if oid, err := bson.ObjectIDFromHex(string(pid)); err == nil {
                s.ParentIDs = append(s.ParentIDs, oid)
            }
        }
    }
}

func (s *Address) ToDomain() *domain.Address {
    if s == nil {
        return nil
    }

    s.Address.ID = domain.ID(s.ID.Hex())
    s.Address.ProjectID = domain.ID(s.ProjectID.Hex())
    s.Address.SourceID = domain.ID(s.SourceID.Hex())

    parents := make([]domain.ID, len(s.ParentIDs))
    for i, pid := range s.ParentIDs {
        parents[i] = domain.ID(pid.Hex())
    }
    s.Address.ParentIDs = parents

    return &s.Address
}
```

## Summary Checklist

When creating a MongoSchema file:
- [ ] Follow mandatory file structure order (imports → collection const → field consts → struct → FromDomain → ToDomain)
- [ ] Use `{Entity}CollectionName` naming for collection constant
- [ ] Define field constants for ALL fields using `{Entity}{FieldName}Field` pattern
- [ ] Embed domain model with `bson:",inline"`
- [ ] Override all ID fields as `bson.ObjectID`
- [ ] FromDomain: Copy embedded struct, then convert IDs with implicit error ignore (`_`)
- [ ] FromDomain: Validate empty strings before conversion
- [ ] ToDomain: Directly modify embedded struct
- [ ] ToDomain: Convert ObjectIDs to hex and wrap in `domain.ID()`
- [ ] Handle ID arrays with proper iteration and conversion
- [ ] **DO NOT create factory functions** - only FromDomain and ToDomain methods