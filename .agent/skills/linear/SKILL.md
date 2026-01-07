---
name: linear
description: |
  Manage Linear issues and documents.
  - create-issue: Create issue with cached team/project defaults, sub-issues, blocking relations
  - create-document: Create document and attach to an issue
  - update-document: Update document title or content
---

# Linear Skill

Linear integration with intelligent caching of team and project settings.

## Commands

| Command           | Description                                                               | Docs                                                       |
| ----------------- | ------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `create-issue`    | Creates a Linear issue with optional sub-issue and blocking relationships | [commands/create-issue.md](commands/create-issue.md)       |
| `create-document` | Creates a Linear document attached to an issue using GraphQL API          | [commands/create-document.md](commands/create-document.md) |
| `update-document` | Updates an existing Linear document using GraphQL API                     | [commands/update-document.md](commands/update-document.md) |

---

## Cache File Format

**Location:** `.agent/memory/linear`

```yaml
team: Cops
project: C-Ops Platform
```

- Simple yaml format
- Keys: `team`, `project`
- Values: Human-readable names (Linear MCP accepts names directly)

## Error Handling

### Invalid Cached Values

If cached team or project no longer exists:
1. Detect error from API response
2. Clear invalid cache entry
3. Prompt user to select new value
4. Offer to save new selection

### Invalid Issue IDs

If provided issue IDs do not exist:
1. Report which IDs are invalid
2. Do not create the issue/document
3. Ask user to correct and retry

### API Errors

For any Linear API errors:
1. Display the error message
2. Suggest possible fixes
3. Do not update cache on error

## Environment Variables

- `LINEAR_API_KEY` - Required for `create-document` and `update-document` commands (GraphQL API authentication)
