# Implementation Plan: Rules Management Domain Model

## Overview

This plan implements a minimal domain model for managing team-shared coding rules in the C-Ops API server. The model supports a hierarchical directory-like structure for organizing Markdown-based rules (e.g., `go/struct`, `go/testing/unit`). The domain model follows the YAGNI principle, including only essential fields required for:
- Unique identification (MongoDB ObjectID)
- Hierarchical organization (path and parent references)
- Basic metadata (name, timestamps)
- Content storage (Markdown text)

The implementation follows existing C-Ops patterns observed in `shared/domain/` and `shared/domain/mongoschema/`.

## Package Changes

None required. The implementation uses only the existing `go.mongodb.org/mongo-driver/v2/bson` package already in use by the project.

## Implementation Steps

### Step 1: Create Rule Domain Model

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`: Pointer vs value type rules for struct fields
- `/Users/jayce/team-attention/cops/shared/domain/project.go`: Reference for domain model patterns (ID type usage, struct embedding, timestamp fields)
- `/Users/jayce/team-attention/cops/shared/domain/common.go`: ID type definition

#### `/Users/jayce/team-attention/cops/shared/domain/rule.go`

**Description**:
Create the Rule domain entity representing a single coding rule/guideline document. All fields are required (value types) since they are mandatory for a valid Rule entity.

```go
package domain

import "time"

// Rule represents a single coding rule/guideline document.
// Rules contain Markdown content and belong to exactly one RuleGroup.
type Rule struct {
	// ID is the unique identifier (MongoDB ObjectID as hex string).
	ID ID `json:"id" bson:"_id,omitempty"`

	// Name is the display name/title of the rule.
	Name string `json:"name" bson:"name"`

	// Content is the full Markdown content of the rule.
	Content string `json:"content" bson:"content"`

	// GroupID references the RuleGroup this rule belongs to.
	GroupID ID `json:"groupId" bson:"groupId"`

	// Path is the full hierarchical path (e.g., "go/struct/pointer-rules").
	// Denormalized for efficient path-based queries.
	Path string `json:"path" bson:"path"`

	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - This is a data struct with no methods | N/A | N/A | N/A |

---

### Step 2: Create RuleGroup Domain Model

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md`: Pointer vs value type rules (ParentID is optional, use pointer)
- `/Users/jayce/team-attention/cops/shared/domain/project.go`: Reference for domain model patterns
- `/Users/jayce/team-attention/cops/shared/domain/common.go`: ID type definition

#### `/Users/jayce/team-attention/cops/shared/domain/rule_group.go`

**Description**:
Create the RuleGroup domain entity representing a logical grouping/directory for organizing rules. ParentID uses pointer type (`*ID`) because root groups have no parent (null value).

```go
package domain

import "time"

// RuleGroup represents a logical grouping/directory for organizing rules.
// Groups can be nested to any depth via ParentID references.
// Root groups have ParentID = nil.
type RuleGroup struct {
	// ID is the unique identifier (MongoDB ObjectID as hex string).
	ID ID `json:"id" bson:"_id,omitempty"`

	// Name is the segment name (e.g., "struct" in "go/struct").
	Name string `json:"name" bson:"name"`

	// Path is the full hierarchical path (e.g., "go/struct").
	// Denormalized for efficient path-based queries.
	Path string `json:"path" bson:"path"`

	// ParentID references the parent RuleGroup.
	// nil for root-level groups.
	ParentID *ID `json:"parentId,omitempty" bson:"parentId,omitempty"`

	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`

	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// IsRoot returns true if this is a root-level group (no parent).
func (g *RuleGroup) IsRoot() bool {
	// Implementation outline:
	// 1. Check if ParentID is nil.
	// 2. Return true if nil, false otherwise.
}
```

**Test Scenarios** for `IsRoot()`:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Root group (no parent) | `RuleGroup{ParentID: nil}` | `true` | ParentID is nil |
| Child group (has parent) | `RuleGroup{ParentID: &parentID}` | `false` | ParentID is not nil |

---

### Step 3: Create MongoDB Schema Constants for Rules

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Reference for collection name and field constant patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Reference for field constant naming conventions

#### `/Users/jayce/team-attention/cops/shared/domain/mongoschema/rule.go`

**Description**:
Define MongoDB collection name and field path constants for the Rules collection. Follow the existing naming pattern: `{Entity}{Field}Field`.

```go
package mongoschema

// Collection name for rules.
const (
	RuleCollectionName = "rules"
)

// Rule document field names.
// Naming pattern: Rule<FieldName>Field
const (
	RuleIDField        = "_id"
	RuleNameField      = "name"
	RuleContentField   = "content"
	RuleGroupIDField   = "groupId"
	RulePathField      = "path"
	RuleCreatedAtField = "createdAt"
	RuleUpdatedAtField = "updatedAt"
)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - Constants only | N/A | N/A | N/A |

---

### Step 4: Create MongoDB Schema Constants for RuleGroups

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`: Reference for collection name and field constant patterns
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Reference for field constant naming conventions

#### `/Users/jayce/team-attention/cops/shared/domain/mongoschema/rule_group.go`

**Description**:
Define MongoDB collection name and field path constants for the RuleGroups collection. Follow the existing naming pattern: `{Entity}{Field}Field`.

```go
package mongoschema

// Collection name for rule groups.
const (
	RuleGroupCollectionName = "ruleGroups"
)

// RuleGroup document field names.
// Naming pattern: RuleGroup<FieldName>Field
const (
	RuleGroupIDField        = "_id"
	RuleGroupNameField      = "name"
	RuleGroupPathField      = "path"
	RuleGroupParentIDField  = "parentId"
	RuleGroupCreatedAtField = "createdAt"
	RuleGroupUpdatedAtField = "updatedAt"
)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| N/A - Constants only | N/A | N/A | N/A |

---

## Summary of Files to Create

| File Path | Description |
| :-------- | :---------- |
| `/Users/jayce/team-attention/cops/shared/domain/rule.go` | Rule domain entity struct |
| `/Users/jayce/team-attention/cops/shared/domain/rule_group.go` | RuleGroup domain entity struct with `IsRoot()` method |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/rule.go` | MongoDB field constants for Rules collection |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/rule_group.go` | MongoDB field constants for RuleGroups collection |

## Design Decisions

### 1. ID Type Usage
Use the existing `domain.ID` type (string alias) for all ID fields to maintain consistency with `Project` and other domain models. The `bson:"_id,omitempty"` tag allows MongoDB to auto-generate ObjectIDs on insert.

### 2. Path Denormalization
Store the full `Path` in both `Rule` and `RuleGroup` entities. This enables:
- Direct path-based lookups without tree traversal
- Efficient prefix queries (e.g., find all rules under `go/*`)
- Trade-off: Extra storage for query performance (paths change rarely)

### 3. Pointer vs Value Types
Following `.agent/rules/go/go-struct.md`:
- **Value types** for required fields: `ID`, `Name`, `Content`, `Path`, `GroupID`, `CreatedAt`, `UpdatedAt`
- **Pointer type** for optional fields: `ParentID *ID` in RuleGroup (null for root groups)

### 4. No Soft Delete
Following YAGNI principle, no `DeletedAt` field is included. Add only when soft-delete functionality is actually needed.

### 5. No Additional Metadata
Following YAGNI principle, optional fields like `Description`, `Tags`, `Author`, `Status`, `Order` are NOT included. These can be added incrementally when needed.

### 6. Separate Files per Entity
Each domain entity gets its own file (`rule.go`, `rule_group.go`) following the pattern observed in `project.go`, `record.go`, etc.

## MongoDB Index Strategy (Documentation Only)

The following indexes should be created when implementing the repository layer (out of scope for domain model):

**Rules Collection**:
- `path` (unique) - Fast path-based rule lookup
- `groupId` - Find all rules in a group

**RuleGroups Collection**:
- `path` (unique) - Fast path-based group lookup
- `parentId` - Efficient children queries

## Validation Invariants (For Future CRUD Implementation)

These invariants should be enforced at the service layer when CRUD is implemented:

1. **Path Consistency**:
   - Rule's path must start with its group's path
   - RuleGroup's path must start with its parent's path (if parent exists)

2. **Uniqueness**:
   - Paths are unique within their collection
   - Names must be unique within the same parent group

3. **Referential Integrity**:
   - Rule's `GroupID` must reference an existing RuleGroup
   - RuleGroup's `ParentID` must reference an existing RuleGroup (if not null)
