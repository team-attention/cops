# Implementation Plan: Fix MongoDB Aggregation Field Naming Errors

## Overview

The `dashboard_repo.go` file contains MongoDB aggregation pipelines that fail with error `(Location40235) The field name 'message.usage.outputTokens' cannot contain '.'`. This occurs because dotted field paths (e.g., `mongoschema.MessageUsageInputTokensPath` which evaluates to `"message.usage.inputTokens"`) are being used as **field names** in `$group` and `$addFields` stages, but MongoDB only allows dots when **referencing** field values with the `$` prefix.

The fix requires:
1. Replacing dotted field names with simple camelCase names in `$group` and `$addFields` stages
2. Updating corresponding `$project` stages to reference the new field names
3. Updating `mongoutil.Get` calls to use the new simple field names

**No external packages need to be added or removed.**

## Package Changes

None required.

## Implementation Steps

### Step 1: Define Local Field Name Constants

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`: Code style rules
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Existing mongoschema constants (DO NOT MODIFY)

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Add local constants at the top of the file (after imports) for the aggregation output field names. These constants will be used in `$group`/`$addFields` stages as output field names and in `mongoutil.Get` calls for result extraction.

```go
// Aggregation output field name constants.
// These are used as field names in $group/$addFields stages (must not contain dots)
// and for extracting results via mongoutil.Get.
const (
    // aggInputTokensField is the output field name for input token sums in aggregations.
    aggInputTokensField = "inputTokens"
    // aggOutputTokensField is the output field name for output token sums in aggregations.
    aggOutputTokensField = "outputTokens"
    // aggCacheReadTokensField is the output field name for cache read token sums in aggregations.
    aggCacheReadTokensField = "cacheReadTokens"
    // aggCacheCreationTokensField is the output field name for cache creation token sums in aggregations.
    aggCacheCreationTokensField = "cacheCreationTokens"
)
```

**Test Scenarios**: N/A (constants only)

---

### Step 2: Fix `GetOverviewStats` Method (Lines 37-124)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
The `usagePipeline` in `GetOverviewStats` already uses correct simple field names (`"totalInputTokens"`, `"totalOutputTokens"`, `"totalCacheReadTokens"`). No changes needed for this method.

**Current Code Analysis (Lines 43-50)**:
```go
usagePipeline := bson.A{
    bson.M{"$group": bson.M{
        "_id":                  nil,
        "totalInputTokens":     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
        "totalOutputTokens":    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
        "totalCacheReadTokens": bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
    }},
}
```

This is **CORRECT** - field names are `"totalInputTokens"` (no dots), source references use `"$" + mongoschema.MessageUsageInputTokensPath` (valid dot notation with `$` prefix).

**Action**: No changes required.

---

### Step 3: Fix `ListProjects` Method (Lines 126-250)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Fix the `$lookup` sub-pipeline's `$group` and `$project` stages, and the outer `$project` stage. Replace dotted field names with the new constants.

**Changes Required**:

**Location 1: `$group` stage inside `$lookup` pipeline (Lines 142-149)**

Replace:
```go
bson.M{"$group": bson.M{
    "_id":          nil,
    "sessionIds":   bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
    "lastActivity": bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

With:
```go
bson.M{"$group": bson.M{
    "_id":                   nil,
    "sessionIds":            bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
    "lastActivity":          bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    aggInputTokensField:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    aggOutputTokensField:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    aggCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

**Location 2: `$project` stage inside `$lookup` pipeline (Lines 150-157)**

Replace:
```go
bson.M{"$project": bson.M{
    "_id":          0,
    "sessionCount": bson.M{"$size": "$sessionIds"},
    "lastActivity": 1,
    mongoschema.MessageUsageInputTokensPath:     1,
    mongoschema.MessageUsageOutputTokensPath:    1,
    mongoschema.MessageUsageCacheReadTokensPath: 1,
}},
```

With:
```go
bson.M{"$project": bson.M{
    "_id":                   0,
    "sessionCount":          bson.M{"$size": "$sessionIds"},
    "lastActivity":          1,
    aggInputTokensField:     1,
    aggOutputTokensField:    1,
    aggCacheReadTokensField: 1,
}},
```

**Location 3: Outer `$project` stage (Lines 169-179)**

Replace:
```go
bson.M{"$project": bson.M{
    mongoschema.ProjectIDField:                    1,
    mongoschema.ProjectNameField:                  1,
    mongoschema.ProjectPathField:                  1,
    mongoschema.ProjectGitBranchField:             1,
    "sessionCount":                                bson.M{"$ifNull": bson.A{"$stats.sessionCount", 0}},
    "lastActivity":                                bson.M{"$ifNull": bson.A{"$stats.lastActivity", nil}},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageInputTokensPath, 0}},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageOutputTokensPath, 0}},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageCacheReadTokensPath, 0}},
}},
```

With:
```go
bson.M{"$project": bson.M{
    mongoschema.ProjectIDField:        1,
    mongoschema.ProjectNameField:      1,
    mongoschema.ProjectPathField:      1,
    mongoschema.ProjectGitBranchField: 1,
    "sessionCount":                    bson.M{"$ifNull": bson.A{"$stats.sessionCount", 0}},
    "lastActivity":                    bson.M{"$ifNull": bson.A{"$stats.lastActivity", nil}},
    aggInputTokensField:               bson.M{"$ifNull": bson.A{"$stats." + aggInputTokensField, 0}},
    aggOutputTokensField:              bson.M{"$ifNull": bson.A{"$stats." + aggOutputTokensField, 0}},
    aggCacheReadTokensField:           bson.M{"$ifNull": bson.A{"$stats." + aggCacheReadTokensField, 0}},
}},
```

**Location 4: Result extraction `mongoutil.Get` calls (Lines 239-242)**

Replace:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
    TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
},
```

With:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, aggInputTokensField),
    TotalOutputTokens:    mongoutil.Get[int64](doc, aggOutputTokensField),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, aggCacheReadTokensField),
},
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Projects with sessions | Project with session records | Returns projects with session counts and token usage | Happy path |
| Projects without sessions | Project with no session records | Returns project with zero counts | Empty sessions branch |
| Empty database | No projects | Returns empty paginated result | Empty result |
| Pagination | Multiple projects, page 2 | Returns correct page of results | Pagination branch |

---

### Step 4: Fix `ListSessions` Method (Lines 294-391)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Fix the `$group` stage and result extraction code.

**Changes Required**:

**Location 1: `$group` stage (Lines 309-319)**

Replace:
```go
pipeline = append(pipeline, bson.M{"$group": bson.M{
    "_id":                                   "$" + mongoschema.RecordSessionIDField,
    "messageCount":                          bson.M{"$sum": 1},
    "startedAt":                             bson.M{"$min": "$" + mongoschema.RecordTimestampField},
    "endedAt":                               bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
    mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}})
```

With:
```go
pipeline = append(pipeline, bson.M{"$group": bson.M{
    "_id":                            "$" + mongoschema.RecordSessionIDField,
    "messageCount":                   bson.M{"$sum": 1},
    "startedAt":                      bson.M{"$min": "$" + mongoschema.RecordTimestampField},
    "endedAt":                        bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
    mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
    aggInputTokensField:              bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    aggOutputTokensField:             bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    aggCacheReadTokensField:          bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}})
```

Note: `mongoschema.RecordProjectIDField` and `mongoschema.RecordGitBranchField` are valid because they are simple field names (`"projectId"`, `"gitBranch"`) without dots.

**Location 2: Result extraction `mongoutil.Get` calls (Lines 380-384)**

Replace:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
    TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
},
```

With:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, aggInputTokensField),
    TotalOutputTokens:    mongoutil.Get[int64](doc, aggOutputTokensField),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, aggCacheReadTokensField),
},
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Sessions for project | Valid projectID with sessions | Returns sessions with token usage | Happy path with filter |
| All sessions | No projectID filter | Returns all sessions | No filter branch |
| Session with usage | Session with assistant records | Returns correct token sums | Token aggregation |
| Empty result | Project with no sessions | Returns empty paginated result | Empty result |
| Custom sort | SortBy="endedAt", SortDesc=false | Returns sessions sorted ascending | Sort branch |

---

### Step 5: Fix `getRecentProjects` Method (Lines 500-569)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Fix the `$addFields` stage and result extraction code.

**Changes Required**:

**Location 1: `$addFields` stage (Lines 512-520)**

Replace:
```go
bson.M{"$addFields": bson.M{
    "lastActivity": bson.M{"$max": "$sessions." + mongoschema.RecordTimestampField},
    "sessionCount": bson.M{"$size": bson.M{
        "$setUnion": bson.A{"$sessions." + mongoschema.RecordSessionIDField, bson.A{}},
    }},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$sessions." + mongoschema.MessageUsageInputTokensPath},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$sessions." + mongoschema.MessageUsageOutputTokensPath},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$sessions." + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

With:
```go
bson.M{"$addFields": bson.M{
    "lastActivity": bson.M{"$max": "$sessions." + mongoschema.RecordTimestampField},
    "sessionCount": bson.M{"$size": bson.M{
        "$setUnion": bson.A{"$sessions." + mongoschema.RecordSessionIDField, bson.A{}},
    }},
    aggInputTokensField:     bson.M{"$sum": "$sessions." + mongoschema.MessageUsageInputTokensPath},
    aggOutputTokensField:    bson.M{"$sum": "$sessions." + mongoschema.MessageUsageOutputTokensPath},
    aggCacheReadTokensField: bson.M{"$sum": "$sessions." + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

**Location 2: Result extraction `mongoutil.Get` calls (Lines 557-561)**

Replace:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
    TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
},
```

With:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, aggInputTokensField),
    TotalOutputTokens:    mongoutil.Get[int64](doc, aggOutputTokensField),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, aggCacheReadTokensField),
},
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Recent projects | Projects with recent activity | Returns top N projects by lastActivity | Happy path |
| Projects without sessions | Projects with no session records | Returns projects with zero usage | Empty sessions |
| Limit parameter | limit=3 | Returns exactly 3 projects | Limit branch |

---

### Step 6: Fix `getRecentSessions` Method (Lines 571-626)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Fix the `$group` stage and result extraction code.

**Changes Required**:

**Location 1: `$group` stage (Lines 573-584)**

Replace:
```go
bson.M{"$group": bson.M{
    "_id":                                   "$" + mongoschema.RecordSessionIDField,
    "messageCount":                          bson.M{"$sum": 1},
    "startedAt":                             bson.M{"$min": "$" + mongoschema.RecordTimestampField},
    "endedAt":                               bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
    mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

With:
```go
bson.M{"$group": bson.M{
    "_id":                            "$" + mongoschema.RecordSessionIDField,
    "messageCount":                   bson.M{"$sum": 1},
    "startedAt":                      bson.M{"$min": "$" + mongoschema.RecordTimestampField},
    "endedAt":                        bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
    mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
    aggInputTokensField:              bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    aggOutputTokensField:             bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    aggCacheReadTokensField:          bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

**Location 2: Result extraction `mongoutil.Get` calls (Lines 615-619)**

Replace:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
    TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
},
```

With:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, aggInputTokensField),
    TotalOutputTokens:    mongoutil.Get[int64](doc, aggOutputTokensField),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, aggCacheReadTokensField),
},
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Recent sessions | Sessions with records | Returns top N sessions by startedAt | Happy path |
| Session with multiple records | Session with 5 records | Returns session with messageCount=5 | Message count aggregation |
| Limit parameter | limit=5 | Returns exactly 5 sessions | Limit branch |

---

### Step 7: Fix `getProjectStats` Method (Lines 628-679)

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation

#### `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Fix the `$group` stage, `$project` stage, and result extraction code.

**Changes Required**:

**Location 1: `$group` stage (Lines 636-643)**

Replace:
```go
bson.M{"$group": bson.M{
    "_id":          nil,
    "sessionCount": bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
    "lastActivity": bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

With:
```go
bson.M{"$group": bson.M{
    "_id":                   nil,
    "sessionCount":          bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
    "lastActivity":          bson.M{"$max": "$" + mongoschema.RecordTimestampField},
    aggInputTokensField:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    aggOutputTokensField:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
    aggCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
}},
```

**Location 2: `$project` stage (Lines 644-650)**

Replace:
```go
bson.M{"$project": bson.M{
    "sessionCount": bson.M{"$size": "$sessionCount"},
    "lastActivity": 1,
    mongoschema.MessageUsageInputTokensPath:     1,
    mongoschema.MessageUsageOutputTokensPath:    1,
    mongoschema.MessageUsageCacheReadTokensPath: 1,
}},
```

With:
```go
bson.M{"$project": bson.M{
    "sessionCount":          bson.M{"$size": "$sessionCount"},
    "lastActivity":          1,
    aggInputTokensField:     1,
    aggOutputTokensField:    1,
    aggCacheReadTokensField: 1,
}},
```

**Location 3: Result extraction `mongoutil.Get` calls (Lines 673-676)**

Replace:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
    TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
},
```

With:
```go
Usage: repository.TokenUsageSummary{
    TotalInputTokens:     mongoutil.Get[int64](doc, aggInputTokensField),
    TotalOutputTokens:    mongoutil.Get[int64](doc, aggOutputTokensField),
    TotalCacheReadTokens: mongoutil.Get[int64](doc, aggCacheReadTokensField),
},
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Project with sessions | Valid projectID with sessions | Returns stats with token usage | Happy path |
| Project without sessions | Valid projectID, no sessions | Returns empty ProjectSummary | Empty cursor branch |
| Invalid projectID | Malformed hex string | Returns error | Error branch |

---

## Summary of All Changes

| Method | Line Range | Changes Required |
| :----- | :--------- | :--------------- |
| (constants) | After imports | Add 4 local constants for aggregation field names |
| `GetOverviewStats` | 37-124 | No changes needed (already correct) |
| `ListProjects` | 126-250 | Fix `$group`, `$project` stages and `mongoutil.Get` calls |
| `ListSessions` | 294-391 | Fix `$group` stage and `mongoutil.Get` calls |
| `getRecentProjects` | 500-569 | Fix `$addFields` stage and `mongoutil.Get` calls |
| `getRecentSessions` | 571-626 | Fix `$group` stage and `mongoutil.Get` calls |
| `getProjectStats` | 628-679 | Fix `$group`, `$project` stages and `mongoutil.Get` calls |

## Field Name Mapping Reference

| Dotted Path (Source Reference) | Simple Name (Output Field) |
| :----------------------------- | :------------------------- |
| `mongoschema.MessageUsageInputTokensPath` (`"message.usage.inputTokens"`) | `aggInputTokensField` (`"inputTokens"`) |
| `mongoschema.MessageUsageOutputTokensPath` (`"message.usage.outputTokens"`) | `aggOutputTokensField` (`"outputTokens"`) |
| `mongoschema.MessageUsageCacheReadTokensPath` (`"message.usage.cacheReadInputTokens"`) | `aggCacheReadTokensField` (`"cacheReadTokens"`) |
| `mongoschema.MessageUsageCacheCreationTokensPath` (`"message.usage.cacheCreationInputTokens"`) | `aggCacheCreationTokensField` (`"cacheCreationTokens"`) |

## Testing Strategy

### Manual API Testing

After implementing the changes, verify the following API endpoints work correctly:

1. **GET /api/v1/dashboard/overview** - Should return overview stats without errors
2. **GET /api/v1/dashboard/projects** - Should return paginated projects with session counts and token usage
3. **GET /api/v1/dashboard/projects/{id}** - Should return project details with aggregated stats
4. **GET /api/v1/dashboard/sessions** - Should return paginated sessions with token usage
5. **GET /api/v1/dashboard/sessions?projectId={id}** - Should return filtered sessions

### Verification Checklist

- [ ] No `Location40235` MongoDB errors in logs
- [ ] Token usage values are non-zero for projects/sessions with actual usage
- [ ] Session counts match expected values
- [ ] Pagination works correctly
- [ ] Sorting works correctly for sessions
- [ ] API responses contain all expected fields

### Build Verification

Run the following commands to ensure the code compiles:

```bash
cd /Users/jayce/team-attention/cops/api && go build ./...
```
