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

```bash
git add .

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
# Standalone (no walkthrough)
/commit-pr

# With walkthrough artifact
/commit-pr 20251226-123456

# Push only (no PR)
/commit-pr --push-only
```
