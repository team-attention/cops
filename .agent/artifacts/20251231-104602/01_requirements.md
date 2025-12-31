# Requirements

## Request Summary

The ListProjects API is failing with MongoDB aggregation errors because field paths containing dots (`message.usage.inputTokens`, `message.usage.outputTokens`, `message.usage.cacheReadInputTokens`) are being used as field **names** in `$group` aggregation stages. MongoDB does not allow field names to contain dots - dots can only be used when **referencing** nested field values (with `$` prefix). The aggregation pipeline needs to be fixed to use valid field names (without dots) for intermediate results while still correctly referencing the nested source fields.

## Acceptance Criteria

- [ ] MongoDB aggregation pipelines in `dashboard_repo.go` successfully execute without "Location40235" errors
- [ ] Field names in `$group` stages use underscore or camelCase format without dots (e.g., `inputTokens`, `outputTokens`, `cacheReadTokens`)
- [ ] Field references for source data correctly use dotted paths with `$` prefix (e.g., `"$message.usage.inputTokens"`)
- [ ] Field references in subsequent `$project` stages correctly reference the renamed fields from `$group`
- [ ] All aggregation functions return correct token usage statistics (no data loss or corruption)
- [ ] ListProjects API successfully returns project data with session counts and token usage
- [ ] ListSessions API successfully returns session data with token usage
- [ ] GetOverviewStats API successfully returns dashboard statistics

## Scope

### In Scope
- Fix field naming in all MongoDB aggregation pipelines in `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- Update `$group` stages to use valid field names without dots
- Update `$project` stages to reference the corrected field names from `$group`
- Update result document extraction code (`mongoutil.Get`) to use the new field names
- Verify all affected methods work correctly:
  - `ListProjects` (lines 126-250)
  - `ListSessions` (lines 294-391)
  - `GetOverviewStats` (lines 37-124)
  - `getRecentProjects` (lines 500-569)
  - `getRecentSessions` (lines 571-626)
  - `getProjectStats` (lines 628-679)

### Out of Scope
- Changing the MongoDB schema or document structure (only aggregation pipeline logic changes)
- Modifying the domain models in `shared/domain/`
- Changing the field path constants in `shared/domain/mongoschema/session_record.go`
- Updating other repositories or services beyond `dashboard_repo.go`
- Adding new features or analytics

## Constraints

- Must maintain backward compatibility with existing MongoDB documents
- Cannot change the actual field paths in stored documents (`message.usage.inputTokens` etc.)
- Field path constants in `mongoschema` package should remain unchanged (they correctly represent the actual document structure)
- Must follow existing code patterns and naming conventions in the repository
- Changes must not break existing API responses or data structures

## Additional Context

**Root Cause:**
MongoDB aggregation error `(Location40235) The field name 'message.usage.outputTokens' cannot contain '.'` occurs because:
1. Constants like `mongoschema.MessageUsageInputTokensPath` evaluate to `"message.usage.inputTokens"` (with dots)
2. These are being used as field **names** in `$group` stages: `mongoschema.MessageUsageInputTokensPath: bson.M{"$sum": ...}`
3. MongoDB requires field names to not contain dots (dots are reserved for nested field access syntax)

**Solution Pattern:**
```go
// WRONG - field name contains dots (current code)
bson.M{"$group": bson.M{
    mongoschema.MessageUsageInputTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    // Evaluates to: "message.usage.inputTokens": {"$sum": "$message.usage.inputTokens"}
    // Error: field name "message.usage.inputTokens" cannot contain '.'
}}

// CORRECT - use simple field name, reference nested source with $
bson.M{"$group": bson.M{
    "inputTokens": bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
    // Evaluates to: "inputTokens": {"$sum": "$message.usage.inputTokens"}
    // Field name is "inputTokens" (valid), source is "$message.usage.inputTokens" (valid)
}}
```

**Affected Files:**
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Related Constants (DO NOT MODIFY):**
- `mongoschema.MessageUsageInputTokensPath` = `"message.usage.inputTokens"`
- `mongoschema.MessageUsageOutputTokensPath` = `"message.usage.outputTokens"`
- `mongoschema.MessageUsageCacheReadTokensPath` = `"message.usage.cacheReadInputTokens"`
- `mongoschema.MessageUsageCacheCreationTokensPath` = `"message.usage.cacheCreationInputTokens"`

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should we modify the mongoschema constants? | No, the constants correctly represent the actual nested field paths in the documents. Only the aggregation pipeline usage needs to change. |
| What field names should we use instead? | Use simple names without dots: `inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheCreationTokens` |
| Do we need to update the database schema? | No, the documents are stored correctly. This is only an aggregation query issue. |
| Will this break existing API responses? | No, the final result extraction already uses `mongoutil.Get` with simple keys. We just need to ensure consistency between pipeline output and extraction code. |
