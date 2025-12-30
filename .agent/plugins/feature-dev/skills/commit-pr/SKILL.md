---
name: commit-pr
description: |
  Create git commit and optionally push & create PR.
  Standalone mode: generates commit message from git diff.
  With walkthrough: uses walkthrough artifact for rich commit details.
---

# Commit & PR Skill

Handles git commit, push, and PR creation with smart commit message generation.

## Modes

### Standalone Mode
- No walkthrough artifact available
- Generates commit message from `git diff`
- Asks user for PR creation

### Walkthrough Mode
- Walkthrough artifact exists
- Uses artifact for rich commit message and PR body
- Automatically creates PR

## Arguments

- `$ARGUMENTS` (optional): Artifact ID for walkthrough mode (e.g., "20251226-123456")
- `--push-only`: Only commit and push, skip PR creation
- `--only-related`: Only commit files related to this workflow (excludes unrelated changes)
- `--files <pattern>`: Only add files matching pattern (e.g., "api/**" or "*.go")

## Workflow

### Step 1: Detect Mode

Check if artifact ID provided:
```bash
if [ -n "$ARGUMENTS" ] && [ -f ".agent/artifacts/$ARGUMENTS/XX_walkthrough.md" ]; then
  MODE="walkthrough"
else
  MODE="standalone"
fi
```

### Step 2: Check Git Status

```bash
# Run git status to see changes
git status

# Run git diff to see changes
git diff
```

If no changes, exit with message: "No changes to commit".

### Step 3: Generate Commit Message

**Standalone Mode:**
- Summarize `git diff` output in 1-2 sentences
- Use format:
  ```
  [type]: [brief summary]

  - [key change 1]
  - [key change 2]

  🤖 Generated with [Claude Code](https://claude.com/claude-code)

  Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
  ```

**Walkthrough Mode:**
- Read walkthrough artifact at `.agent/artifacts/$ARGUMENTS/*_walkthrough.md`
- Extract title and summary
- Use format:
  ```
  [type]: [title from walkthrough]

  [summary from walkthrough]

  🤖 Generated with [Claude Code](https://claude.com/claude-code)

  Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
  ```

### Step 4: Commit

**Determine files to commit:**

1. **If `--files <pattern>` provided:**
   - Add only files matching the pattern: `git add <pattern>`

2. **If `--only-related` provided (Walkthrough mode):**
   - Read the walkthrough artifact to identify modified files
   - Compare with `git status` output
   - Add only files mentioned in the walkthrough
   - Example:
     ```bash
     # Extract file paths from walkthrough's "Modified Files" section
     # Add only those files that appear in git status
     git add <file1> <file2> <file3>
     ```
   - If walkthrough doesn't list specific files, analyze git diff and ask user which files belong to this workflow

3. **Default (no flags):**
   - Add all changes: `git add .`

**Create commit:**
```bash
git commit -m "$(cat <<'EOF'
[generated commit message here]
EOF
)"
```

### Step 5: Push

```bash
git push -u origin HEAD
```

### Step 6: Create PR (Conditional)

**If `--push-only` flag:** Skip PR creation

**Else if Walkthrough Mode:**
- Extract PR body from walkthrough artifact
- Create PR:
  ```bash
  gh pr create \
    --title "[Title from walkthrough]" \
    --body "$(cat .agent/artifacts/$ARGUMENTS/*_walkthrough.md)"
  ```

**Else (Standalone Mode):**
- Ask user:
  ```
  Use AskUserQuestion tool:
  - Question: "Create pull request?"
  - Header: "PR Creation"
  - Options:
    - label: "Yes", description: "Create PR now"
    - label: "No", description: "Skip PR creation"
  - multiSelect: false
  ```
- If Yes:
  ```bash
  gh pr create --fill  # Use commit message as PR title/body
  ```

## Example Usage

```bash
# Standalone (no walkthrough) - commits all changes
/commit-pr

# With walkthrough artifact - commits all changes
/commit-pr 20251226-123456

# Only commit files related to this workflow (excludes unrelated changes)
/commit-pr 20251226-123456 --only-related

# Commit specific file pattern
/commit-pr --files "api/**"

# Push only (no PR) - commits all changes and pushes
/commit-pr --push-only

# Combine flags - only related files, no PR creation
/commit-pr 20251226-123456 --only-related --push-only
```
