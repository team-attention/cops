---
name: commit-pr
description: |
  Commit changes with artifact-based message and create PR (or push to main)
---

# Commit-PR Skill

Commits staged changes with a message generated from artifact files, then handles push/PR based on current branch.

## Input

- `$ARTIFACT_ID` - Artifact directory identifier for the current work session

## Process

### Step 1: Read Artifact Files

List and read artifact files to understand the work done:
```bash
ARTIFACT_DIR=".agent/artifacts/$ARTIFACT_ID"
ls -1 "$ARTIFACT_DIR"
```

Read artifact files in order of priority:
1. `*_walkthrough.md` - Primary source for commit summary
2. other files - Implementation, Planning, Requirements details

Summarize the changes made based on artifact content.

### Step 2: Check Git Status and Branch

```bash
# Check current branch
CURRENT_BRANCH=$(git branch --show-current)

# Check for uncommitted changes
git status --short

# Determine branch type
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
  IS_MAIN_BRANCH=true
else
  IS_MAIN_BRANCH=false
fi
```

### Step 3: Stage and Commit Changes

Stage all changes including artifact files:
```bash
git add -A
```

Generate commit message using conventional commit format:
- Analyze artifact content to determine type (feat, fix, refactor, docs, etc.)
- Optionally add scope in parentheses: `{type}(scope):`
- Create descriptive summary from walkthrough
- Include Artifact-ID in commit body

```bash
git commit -m "$(cat <<'EOF'
{type}(scope): {brief description}

Artifact-ID: $ARTIFACT_ID

{detailed summary from artifacts}

EOF
)"
```

### Step 4: Branch-Based Action

#### If Main Branch ($IS_MAIN_BRANCH = true)

Ask user whether to push:
```
Use AskUserQuestion tool:
- questions:
  - question: "Changes committed. Push to remote?"
    header: "Push"
    options:
      - label: "Push"
        description: "Push commits to remote main branch"
      - label: "Skip"
        description: "Keep commits local for now"
    multiSelect: false
```

**If Push selected:**
```bash
git push
```

#### If Feature Branch ($IS_MAIN_BRANCH = false)

Ask user whether to create PR:
```
Use AskUserQuestion tool:
- questions:
  - question: "Changes committed. How would you like to proceed?"
    header: "Next Step"
    options:
      - label: "Create PR"
        description: "Push and create a Pull Request"
      - label: "Push Only"
        description: "Push to remote without creating PR"
      - label: "Skip"
        description: "Keep commits local for now"
    multiSelect: false
```

**If Create PR selected:**
```bash
# Push with upstream tracking
git push -u origin HEAD

# Create PR with artifact-informed content
gh pr create --title "{type}(scope): {brief description}" --body "$(cat <<'EOF'
Artifact-ID: $ARTIFACT_ID

## Summary
{bullet points from walkthrough}

EOF
)"
```

**If Push Only selected:**
```bash
git push -u origin HEAD
```

## Output

Report completion summary:
```
Commit-PR completed.

## Summary
- Commit: {commit hash}
- Branch: $CURRENT_BRANCH
- Action: {Push / PR Created / Skipped}
- PR URL: {URL if PR created}
- Artifact-ID: $ARTIFACT_ID
```
