# Requirements: Rules Management Domain Model

## Request Summary

Design a **simple, minimal** domain model for managing team-shared coding rules in the C-Ops API server. Rules are Markdown documents that can be organized in a hierarchical directory-like structure (e.g., `go/struct`, `go/testing`). The system will serve as a central repository for rules shared across multiple projects, stored in MongoDB. While CRUD operations will be needed eventually, the current focus is on designing a minimal domain model with only essential fields, following the YAGNI (You Aren't Gonna Need It) principle. Avoid adding fields that might be needed in the future - add them only when actually needed.

## Acceptance Criteria

- [ ] Domain model is **simple and minimal** - includes only essential fields (YAGNI principle)
- [ ] Domain model supports hierarchical grouping with unlimited nesting levels
- [ ] Rule entity stores: ID, name, content, group reference, path, timestamps only
- [ ] RuleGroup entity stores: ID, name, path, parent reference, timestamps only
- [ ] Domain model is designed for MongoDB storage with appropriate schema considerations
- [ ] Path-based addressing is supported (e.g., `go/struct/pointer-rules`)
- [ ] Domain model can represent the full tree structure efficiently
- [ ] Relationships between Rule and RuleGroup are clearly defined
- [ ] Domain model follows existing C-Ops patterns (similar to Project, Session, Message)
- [ ] No speculative fields added for future features

## Scope

### In Scope

**Domain Model Design:**
- **Minimal** Rule entity with essential fields only (ID, name, content, group_id, path, timestamps)
- **Minimal** RuleGroup entity with essential fields only (ID, name, path, parent_id, timestamps)
- Path-based addressing mechanism for rules and groups
- MongoDB schema design considerations
- Relationship modeling between entities
- Follow YAGNI principle - no speculative fields

**Hierarchical Structure:**
- Multi-level nesting support (e.g., `go/testing/unit`, `typescript/react/hooks`)
- Parent-child relationships in RuleGroup
- Path resolution and navigation logic design

**MongoDB Considerations:**
- Collection structure for Rules and RuleGroups
- Index strategy for efficient querying
- Path-based query optimization
- Document embedding vs referencing decisions

### Out of Scope

- **CRUD API implementation** - Will be implemented later
- **REST/gRPC endpoint definitions** - Future work
- **Authentication/Authorization** - Future work
- **Rule versioning system** - Not required, add only when needed
- **Rule validation logic** - Future enhancement
- **File system synchronization** - Future integration
- **Search/indexing functionality** - Future feature
- **UI/Dashboard integration** - Separate concern
- **Optional metadata fields** (tags, description, author, status, etc.) - Add only when actually needed (YAGNI)

## Constraints

**Technical Constraints:**
- Must use MongoDB as the primary storage backend
- Must follow existing C-Ops domain model patterns (see `shared/domain/`)
- Must be compatible with Go's struct marshaling/unmarshaling
- Must support efficient path-based queries in MongoDB

**Design Constraints:**
- Domain model must remain storage-agnostic at the domain layer (repository pattern)
- Must not introduce circular dependencies in the codebase
- Should follow hexagonal architecture principles used in C-Ops
- **Simplicity First**: Follow YAGNI principle - only essential fields, no speculative features

**Future Compatibility:**
- Domain model should accommodate future CRUD operations without major refactoring
- If additional fields are needed in the future, they can be added incrementally (not now)

## Domain Model Requirements

### Core Entities

#### 1. Rule Entity

**Purpose**: Represents a single coding rule/guideline document

**Fields (Minimal - YAGNI):**
- `id` - Unique identifier (MongoDB ObjectID)
- `name` - Name/title of the rule
- `content` - Markdown content (full text)
- `group_id` - Reference to RuleGroup this belongs to
- `path` - Full hierarchical path (e.g., `go/struct/pointer-rules`)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

**Future Considerations (NOT included now):**
These fields should only be added when actually needed:
- Description/summary
- Tags for categorization
- Author information
- Status (draft, published, archived)

**Relationships:**
- Belongs to exactly one RuleGroup
- Referenced by path and ID

#### 2. RuleGroup Entity

**Purpose**: Represents a logical grouping/directory for organizing rules

**Fields (Minimal - YAGNI):**
- `id` - Unique identifier (MongoDB ObjectID)
- `name` - Segment name (e.g., `struct` in `go/struct`)
- `path` - Full hierarchical path (e.g., `go/struct`)
- `parent_id` - Parent RuleGroup reference (null for root groups)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

**Future Considerations (NOT included now):**
These fields should only be added when actually needed:
- Description
- Order/priority for display
- Additional metadata

**Relationships:**
- Has zero or one parent RuleGroup (null = root level)
- Has zero or more child RuleGroups
- Has zero or more Rules

### Hierarchical Structure Design

#### Path Representation

**Path Format**: Use forward-slash separated segments
- Root groups: `go`, `typescript`, `python`
- Nested groups: `go/testing`, `go/testing/unit`
- Rule paths: `go/struct/pointer-rules`

**Path Storage**: Store full path in both Rule and RuleGroup for efficient querying
- Allows direct path-based lookups without traversing tree
- Enables efficient prefix queries (e.g., all rules under `go/*`)

#### Tree Navigation

**Parent-Child Relationships:**
- RuleGroup stores reference to parent RuleGroup ID
- Enables upward traversal (child → parent → root)
- Root groups have `parent_id = null`

**Children Discovery:**
- Query RuleGroups by `parent_id` to find children
- Query Rules by `group_id` to find rules in a group

### MongoDB Schema Considerations

#### Collections

**`rule_groups` Collection (Minimal Schema):**
```json
{
  "_id": ObjectId,
  "name": "struct",
  "path": "go/struct",
  "parent_id": ObjectId | null,
  "created_at": ISODate,
  "updated_at": ISODate
}
```

**`rules` Collection (Minimal Schema):**
```json
{
  "_id": ObjectId,
  "name": "Pointer vs Value Types",
  "path": "go/struct/pointer-rules",
  "group_id": ObjectId,
  "content": "# Markdown content here...",
  "created_at": ISODate,
  "updated_at": ISODate
}
```

Note: Optional fields like `description`, `tags`, `author`, `status` are NOT included in the initial schema (YAGNI principle).

#### Indexes

**Required Indexes (Minimal - Only What's Needed):**
- `rule_groups.path` (unique) - Fast path-based group lookup
- `rule_groups.parent_id` - Efficient children queries
- `rules.path` (unique) - Fast path-based rule lookup
- `rules.group_id` - Find all rules in a group

**Future Indexes (Add only when needed):**
- Full-text search on `rules.name` or `rules.content` - Only if search is actually implemented
- Tag-based indexing - Only if tags are added to the model
- Other indexes should be added based on actual query patterns, not speculation

#### Design Decisions

**Embedding vs Referencing:**
- Use **referencing** (not embedding) for RuleGroup relationships
  - Supports unlimited nesting depth
  - Avoids document size limits
  - Easier to move rules between groups

**Denormalization:**
- Store full `path` in both collections (denormalized)
  - Trade-off: Storage space for query performance
  - Justification: Paths change rarely, lookups are frequent
  - Update strategy: When renaming a group, update all descendant paths

**Path Uniqueness:**
- Paths must be globally unique within their collection
- Enforced by unique index on `path` field
- Prevents duplicate groups/rules at the same location

### Domain Model Relationships

#### Relationship Diagram

```
RuleGroup (root)
    ├─ path: "go"
    ├─ parent_id: null
    └─ children:
        ├─ RuleGroup
        │   ├─ path: "go/struct"
        │   ├─ parent_id: → "go"
        │   └─ rules:
        │       └─ Rule
        │           ├─ path: "go/struct/pointer-rules"
        │           └─ group_id: → "go/struct"
        └─ RuleGroup
            ├─ path: "go/testing"
            └─ parent_id: → "go"
```

#### Invariants

**Path Consistency:**
- Rule's `group_id` must reference an existing RuleGroup
- Rule's path must start with its group's path
- RuleGroup's path must start with its parent's path (if parent exists)

**Tree Structure:**
- No circular references in parent-child relationships
- Root groups have `parent_id = null`
- All non-root groups must have valid parent reference

**Uniqueness:**
- Paths are unique within their collection
- Names must be unique within the same parent group

### Future CRUD Considerations

The domain model includes fields to support future CRUD operations:

**Create Operations:**
- `created_at` timestamp for audit trail
- Path validation to ensure proper hierarchy
- Parent reference validation

**Read Operations:**
- Path-based retrieval (efficient with indexes)
- Tree traversal (parent/children queries)
- Listing rules within a group

**Update Operations:**
- `updated_at` timestamp for tracking changes
- Path recalculation when moving rules/groups
- Cascading updates for group renames

**Delete Operations:**
- Cascade delete: Deleting a group should handle child groups/rules
- Orphan prevention: Ensure rules aren't left without a group
- Hard delete (no soft delete for now) - add `deleted_at` only if needed later

## Additional Context

### Integration with Existing .agent/rules/

The current C-Ops repository uses `.agent/rules/` for project-specific rules:
- `.agent/rules/common.md`
- `.agent/rules/go/go-struct.md`
- `.agent/rules/workflow.md`

**Future Integration Possibilities:**
- API could serve as centralized rule repository
- Projects could sync rules from API to local `.agent/rules/`
- Daemon could watch API for rule updates
- Rules could be project-specific or global (add scope field only when this feature is implemented)

### Reference to Existing Domain Models

Follow patterns from existing C-Ops domain models:
- `shared/domain/record_assistant.go` - For entity structure
- `shared/domain/record_user.go` - For metadata patterns
- Use JSON tags for MongoDB marshaling
- Use pointer types for optional fields (see `.agent/rules/go/go-struct.md`)

### MongoDB Repository Pattern

Follow hexagonal architecture (see `.agent/rules/project.md`):
- Domain models in `shared/domain/`
- Repository interfaces in `shared/domain/`
- MongoDB implementation in `api/internal/service/*/outbound/repository/mongodb/`

## Questions Resolved

| Question | Answer |
|----------|--------|
| Should the API server provide CRUD operations for Rules? | CRUD will be needed eventually, but current focus is on domain model design only |
| Can groups be nested multiple levels deep? | Yes, multi-level nesting should be supported (e.g., `go/testing/unit`) |
| Where should Rules be stored? | MongoDB |
| Who are the main users and use case? | Central rule repository shared across multiple projects |
| Should Rules support versioning? | Not required in initial model (add only when needed - YAGNI) |
| Should a Rule belong to one or multiple groups? | One group only (simpler model, can be extended later if needed) |
| How deep can group nesting go? | Unlimited depth (no arbitrary limit) |
| What format for Rules content? | Markdown (string field in MongoDB) |
| Should there be validation for Rule content? | Future enhancement, not part of initial domain model |
| Should we include optional fields like tags, description, author? | **NO** - Follow YAGNI principle, add only essential fields now |
| What about metadata for display ordering or priorities? | **NO** - Add only when actually needed |

## Success Metrics

The domain model design will be considered successful when:

1. **Simplicity**: Model includes only essential fields, following YAGNI principle
2. **Completeness**: All entities and relationships are clearly defined
3. **MongoDB-Ready**: Schema can be directly implemented in MongoDB
4. **Path Efficiency**: Path-based queries can be performed without tree traversal
5. **Extensibility**: CRUD operations can be added without refactoring the model
6. **Consistency**: Follows existing C-Ops domain model patterns
7. **Scalability**: Can handle hundreds of groups and thousands of rules efficiently
8. **No Speculation**: No fields added for hypothetical future features
