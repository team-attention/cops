---
trigger: always_on
globs: **/*.go
---

# Go Struct Definition Rules

## Pointer vs Value Type Rules

### When to Use Pointer Types

Use pointer types (`*T`) when:
1. **Optional primitive fields** - can be absent/null in JSON
   ```go
   Step     *int    `json:"step,omitempty"`
   UniqueID *string `json:"uniqueId,omitempty"`
   ```

2. **Optional struct fields** - can be absent/null in JSON
   ```go
   Usage *Usage `json:"usage,omitempty"`
   ```

3. **Discriminated union members** - only one of multiple fields is set
   ```go
   Text   *string `json:"text,omitempty"`   // Set when IsText=true
   Blocks []Block `json:"blocks,omitempty"` // Set when IsText=false
   ```

### When to Use Value Types

Use value types when:
1. **Required fields** - always present in JSON
   ```go
   Role   string `json:"role"`
   Status Status `json:"status"`
   ```

2. **Slice/map fields** - nil is equivalent to empty
   ```go
   Tags  []string          `json:"tags"`
   Attrs map[string]string `json:"attrs"`
   ```

### Summary

| Scenario | Type | JSON Tag |
|----------|------|----------|
| Required primitive | `string`, `int`, `bool` | `json:"field"` |
| Optional primitive | `*string`, `*int`, `*bool` | `json:"field,omitempty"` |
| Required struct | `T` | `json:"field"` |
| Optional struct | `*T` | `json:"field,omitempty"` |
| Discriminated union member | `*T` | `json:"field,omitempty"` |
| Slice/map | `[]T`, `map[K]V` | `json:"field"` or `json:"field,omitempty"` |
