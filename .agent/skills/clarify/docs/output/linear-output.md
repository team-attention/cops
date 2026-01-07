# Linear Output Document

This document defines how to save clarified requirements to Linear using the `linear` skill's `create-issue` command.

## Input

- `PROJECT_ID` - Linear Project ID or name
- `ISSUE_ID` - (Optional) Original issue ID from clarify input
- Temporary files from the previous step with dependency metadata

### Temporary File Format

```yaml
---
name: API Interface
blockedBy: []
---
# Task Summary
...
```

```yaml
---
name: Client development
blockedBy:
  - API Interface
---
# Task Summary
...
```

- `name`: Task identifier (used for issue title and dependency references)
- `blockedBy`: List of task names this task depends on

## Process

### Step 1: Parse and Sort

1. Parse temporary files and extract:
   - `name` from YAML frontmatter
   - `blockedBy` array from YAML frontmatter
   - Content (everything after frontmatter) as description
2. Sort tasks by dependency order (prerequisites first)

### Step 2: Create Issues

For each task (in dependency order):

1. Resolve `blockedBy` task names to issue IDs (from previously created issues)
2. Call `linear` skill's `create-issue` command:

```bash
skill: linear
args: create-issue TITLE="{name}" DESCRIPTION="{content}" PROJECT="{PROJECT_ID}" PARENT="{ISSUE_ID}" BLOCKED_BY="{resolved_issue_ids}"
```

**Parameters:**
- `TITLE`: Task name from frontmatter
- `DESCRIPTION`: Task content (Task Summary, Acceptance Criteria, etc.)
- `PROJECT`: PROJECT_ID from clarify input
- `PARENT`: ISSUE_ID from clarify input (if provided, creates sub-issue)
- `BLOCKED_BY`: Comma-separated issue IDs resolved from blockedBy names

3. Collect the created issue ID and map: `task_name → issue_id`

### Step 3: Report Results

List all created issue IDs with their titles.

## Example

```
Input:
  PROJECT_ID: my-project
  ISSUE_ID: ABC-123

Temp files:
  .agent/tmp/api-interface.xxxxxxxx (name: "API Interface", blockedBy: [])
  .agent/tmp/client.yyyyyyyy (name: "Client", blockedBy: ["API Interface"])
  .agent/tmp/server.zzzzzzzz (name: "Server", blockedBy: ["API Interface"])

Execution:
  1. skill: linear
     args: create-issue TITLE="API Interface" PROJECT="my-project" PARENT="ABC-123"
     → Created: ABC-124
     → Map: {"API Interface": "ABC-124"}

  2. skill: linear
     args: create-issue TITLE="Client" PROJECT="my-project" PARENT="ABC-123" BLOCKED_BY="ABC-124"
     → Created: ABC-125

  3. skill: linear
     args: create-issue TITLE="Server" PROJECT="my-project" PARENT="ABC-123" BLOCKED_BY="ABC-124"
     → Created: ABC-126

Result:
  ABC-123 (original)
  ├── ABC-124 (API Interface)
  ├── ABC-125 (Client) ← blockedBy: ABC-124
  └── ABC-126 (Server) ← blockedBy: ABC-124
```

## Output

List of created issue IDs (e.g., `ABC-124`, `ABC-125`, `ABC-126`).
