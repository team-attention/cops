---
trigger: always_on
globs: **/internal/platform/domain/*.go
paths: **/internal/platform/domain/*.go
---

# Platform Domain Model Guidelines

Domain models are the core business entities of the application. They should be framework-agnostic and focused on business logic.

## Field Management Rules

### DO NOT create unnecessary fields

**Avoid adding fields that:**
- Can be derived from existing fields
- Are metadata unrelated to business logic (e.g., `CreatedAt`, `UpdatedAt`)
- Represent states that can be inferred from other fields
- Are added "just in case" without immediate need

### Only add fields when explicitly needed

Each field should have a clear business purpose. If you cannot explain why a field is needed for the current requirements, do not add it.

## ID Type Convention

**ALWAYS use `domain.ID` type for entity identifiers.**

## Naming Conventions

### Entity Names
- Use singular nouns (e.g., `User`, `Address`, `Project`, not `Users`, `Addresses`, `Projects`)

### ID Fields
- Single ID: `{Entity}ID` (e.g., `ProjectID`, `SourceID`, `UserID`)
- ID arrays: Use plural form (e.g., `ParentIDs`, `ChildIDs`)

### Enums
Use type + constants pattern:

```go
type AddressStatus string

const (
    AddressStatusCreated    AddressStatus = "created"
    AddressStatusProcessing AddressStatus = "processing"
    AddressStatusDone       AddressStatus = "done"
    AddressStatusRejected   AddressStatus = "rejected"
)
```

## JSON Tag Rules

### Use camelCase for all JSON field names

```go
type Source struct {
    ID          ID     `json:"id" bson:"-"`
    ProjectID   ID     `json:"projectId" bson:"-"`    // camelCase
    CurrentStep int    `json:"currentStep" bson:"currentStep"`
    FilePath    string `json:"filePath" bson:"filePath"`
}
```

### Exclude security-sensitive fields

Fields that could pose security risks when exposed in API responses MUST use `json:"-"`:

```go
type User struct {
    ID       ID         `json:"-" bson:"-"`  // User ID hidden from API
    Username string     `json:"username,omitempty" bson:"username,omitempty"`
    Accounts []*Account `json:"accounts" bson:"accounts"`
}
```

**Common fields to exclude:**
- Internal user IDs
- API keys, tokens, secrets
- Internal system identifiers not meant for client consumption

### Use `omitempty` for optional fields

Optional fields (pointer types) should include `omitempty`:

```go
type Address struct {
    ID         ID      `json:"id" bson:"-"`
    Step       *int    `json:"step,omitempty" bson:"step,omitempty"`
    OptOut     *bool   `json:"optOut,omitempty" bson:"optOut,omitempty"`
    UniqueID   *string `json:"uniqueId,omitempty" bson:"uniqueId,omitempty"`
    Normalized *string `json:"normalized,omitempty" bson:"normalized,omitempty"`
}
```

## BSON Tag Rules

### ALWAYS exclude ID fields with `bson:"-"`

**The ID field must ALWAYS be excluded from BSON tags** because it will be handled in the MongoSchema layer as `bson.ObjectID`:

```go
type Address struct {
    ID        ID   `json:"id" bson:"-"`        // ALWAYS use bson:"-"
    ProjectID ID   `json:"projectId" bson:"-"` // ALWAYS use bson:"-"
    SourceID  ID   `json:"sourceId" bson:"-"`  // ALWAYS use bson:"-"
    ParentIDs []ID `json:"parents" bson:"-"`   // ALWAYS use bson:"-"
}
```

**Why?** The MongoSchema layer converts `domain.ID` (string) to `bson.ObjectID` for MongoDB storage.

### Match JSON tags for non-ID fields

Non-ID fields should have matching JSON and BSON tags:

```go
type Address struct {
    ID         ID            `json:"id" bson:"-"`
    Status     AddressStatus `json:"status" bson:"status"`           // Match
    Original   string        `json:"original" bson:"original"`       // Match
    Normalized *string       `json:"normalized,omitempty" bson:"normalized,omitempty"` // Match
}
```

### Nested ID fields also use `bson:"-"`

If a field contains ID values (even within nested structures), exclude it from BSON:

```go
type Address struct {
    ID        ID   `json:"id" bson:"-"`
    ParentIDs []ID `json:"parents" bson:"-"`  // Array of IDs, still excluded
}
```

## Complete Example
```go
package domain

type AddressStatus string

const (
    AddressStatusCreated    AddressStatus = "created"
    AddressStatusProcessing AddressStatus = "processing"
    AddressStatusDone       AddressStatus = "done"
    AddressStatusRejected   AddressStatus = "rejected"
)

type Address struct {
    ID         ID            `json:"id" bson:"-"`
    ProjectID  ID            `json:"projectId" bson:"-"`
    SourceID   ID            `json:"sourceId" bson:"-"`
    ParentIDs  []ID          `json:"parents" bson:"-"`
    Step       *int          `json:"step,omitempty" bson:"step,omitempty"`
    OptOut     *bool         `json:"optOut,omitempty" bson:"optOut,omitempty"`
    Status     AddressStatus `json:"status" bson:"status"`
    UniqueID   *string       `json:"uniqueId,omitempty" bson:"uniqueId,omitempty"`
    Original   string        `json:"original" bson:"original"`
    Normalized *string       `json:"normalized,omitempty" bson:"normalized,omitempty"`
}
```

## Anti-Patterns to Avoid

### ❌ Using raw strings for entity IDs
```go
// WRONG
type Address struct {
    ID string `json:"id" bson:"-"`
}
```

### ❌ Including ID fields in BSON tags
```go
// WRONG
type Address struct {
    ID ID `json:"id" bson:"_id"`  // Should be bson:"-"
}
```

### ❌ Adding unnecessary metadata fields
```go
// WRONG
type Address struct {
    ID        ID        `json:"id" bson:"-"`
    CreatedAt time.Time `json:"createdAt" bson:"createdAt"`  // Not needed
    UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`  // Not needed
}
```

### ❌ Adding redundant status fields
```go
// WRONG
type Address struct {
    Status      AddressStatus `json:"status" bson:"status"`
    IsProcessed bool          `json:"isProcessed" bson:"isProcessed"`  // Redundant with Status
    IsDone      bool          `json:"isDone" bson:"isDone"`            // Redundant with Status
}
```

## Summary Checklist

When creating a domain model:
- [ ] Use `domain.ID` for all entity identifiers
- [ ] Use singular entity names
- [ ] Only include necessary business fields
- [ ] Use camelCase for JSON field names
- [ ] Exclude security-sensitive fields from JSON with `json:"-"`
- [ ] Use `omitempty` for optional (pointer) fields
- [ ] ALWAYS use `bson:"-"` for all ID fields (including nested ID arrays)
- [ ] Match JSON and BSON tags for non-ID fields
- [ ] Avoid adding CreatedAt, UpdatedAt, or other metadata fields
- [ ] Don't add redundant status or derived fields
