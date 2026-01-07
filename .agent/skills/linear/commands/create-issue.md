# `create-issue` Command

Creates a Linear issue with optional sub-issue and blocking relationships.

## Parameters

### Required

- `TITLE` - Issue title

### Optional

- `DESCRIPTION` - Issue description in Markdown format
- `PARENT` - Parent issue ID or identifier (e.g., `TA-123`) to create as sub-issue
- `BLOCKED_BY` - Comma-separated issue IDs that block this issue (e.g., `TA-100,TA-101`)
- `TEAM` - Team name or ID (overrides cache)
- `PROJECT` - Project name or ID (overrides cache)
- `ASSIGNEE` - User to assign (ID, name, email, or "me")
- `LABELS` - Comma-separated label names to apply
- `NO_CACHE` - If `true`, do not use or update cache

## Usage Examples

```bash
# Simple issue with cached defaults
skill: linear
args: create-issue TITLE="Fix login button alignment"

# Issue with description
skill: linear
args: create-issue TITLE="Add user profile page" DESCRIPTION="Create a new profile page with avatar, bio, and settings sections."

# Sub-issue under a parent
skill: linear
args: create-issue TITLE="Implement avatar upload" PARENT=TA-456

# Issue blocked by others
skill: linear
args: create-issue TITLE="Deploy to production" BLOCKED_BY="TA-100,TA-101"

# Override cached team/project
skill: linear
args: create-issue TITLE="New feature" TEAM="Backend" PROJECT="API Improvements"

# Full example with all options
skill: linear
args: create-issue TITLE="Add pagination" DESCRIPTION="Implement cursor-based pagination" PARENT=TA-200 BLOCKED_BY="TA-150" ASSIGNEE=me LABELS="enhancement,backend"
```

## Process

### Step 1: Load Cache

Read cached settings from `.agent/memory/linear`:

```
CACHE_FILE=".agent/memory/linear"
Parse key-value pairs:
  - team: cached team name
  - project: cached project name
```

### Step 2: Resolve Team

**Priority order:**
1. Use `TEAM` parameter if provided
2. Use cached team if available
3. Prompt user to select team

**If prompting required:**

1. Call `mcp__linear__list_teams` to get available teams
2. Use `AskUserQuestion` tool to ask user:
   - "Which team should this issue be created in?"
   - Options: list of team names
3. Ask: "Save this as the default team?"
   - If yes, mark for cache update

### Step 3: Resolve Project

**Priority order:**
1. Use `PROJECT` parameter if provided
2. Use cached project if available
3. Prompt user to select project

**If prompting required:**

1. Call `mcp__linear__list_projects` with team filter to get available projects
2. Use `AskUserQuestion` tool to ask user:
   - "Which project should this issue be added to?"
   - Options: list of project names
3. Ask: "Save this as the default project?"
   - If yes, mark for cache update

### Step 4: Validate Relationships

If `PARENT` is provided:
1. Call `mcp__linear__get_issue` with id=PARENT
2. If not found, report error and exit

If `BLOCKED_BY` is provided:
1. Split comma-separated IDs
2. For each ID, call `mcp__linear__get_issue`
3. If any not found, report which IDs are invalid and exit

### Step 5: Update Cache

If user chose to save defaults (and `NO_CACHE` is not `true`):

Write to `.agent/memory/linear`:
```yaml
team: {resolved_team}
project: {resolved_project}
```

Preserve existing values for fields not being updated.

### Step 6: Create Issue

Call `mcp__linear__create_issue` with:
- `title`: TITLE parameter
- `team`: resolved team
- `project`: resolved project
- `description`: DESCRIPTION (if provided)
- `parentId`: PARENT (if provided)
- `blockedBy`: array from BLOCKED_BY (if provided)
- `assignee`: ASSIGNEE (if provided)
- `labels`: array from LABELS (if provided)

### Step 7: Report Result

Display created issue information:

```
Issue created successfully.

- Issue: {identifier} (e.g., TA-789)
- Title: {title}
- Team: {team name}
- Project: {project name}
- Parent: {parent ID or "None"}
- Blocked By: {list of blocking issue IDs or "None"}
- URL: {issue URL}
```
