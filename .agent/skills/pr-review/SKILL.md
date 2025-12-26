---
name: pr-review
description: |
  Handle PR review comments through iteration cycles.
  Flow: Review(PR) -> Execute -> Review(Pre-PR) -> ... -> Walkthrough -> Commit & Merge
---

# PR-Review Iteration Skill

This skill handles PR review feedback through iterative cycles until all issues are resolved.

## Workflow Overview

```
Review (Opus, PR mode) - Parse PR comments
    ↓
Execute (Sonnet) - Implement valid fixes
    ↓
Review (Opus, Pre-PR mode) - Verify fixes
    ↓
    ├─ If issues remain → Execute → Review (loop)
    └─ If all clear ↓
Walkthrough (Sonnet)
    ↓
Commit & Push (update PR)
```

## Step-by-Step Process

### Step 0: Initialize

```bash
# Generate artifact ID for this review cycle
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)
echo "Starting PR-Review cycle with Artifact ID: $ARTIFACT_ID"
```

### Step 1: Parse PR Comments

**Agent**: Review (Opus, PR mode)
**Input**: PR URL from $ARGUMENTS

1. Generate artifact file path:
   ```bash
   PR_REVIEW_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID pr-review)
   ```

2. Invoke Review Agent in PR mode:
   ```
   Use Task tool:
   - Prompt: "Analyze PR review comments.
     PR URL: [from arguments]
     Mode: PR Review
     Write analysis to $PR_REVIEW_FILE

     For each comment:
     1. Evaluate if it's valid
     2. Create execution tasks for valid comments
     3. Prepare responses for invalid comments"
   ```

3. Show user the analysis:
   ```
   PR Review Analysis complete.

   Valid comments (will be addressed): N
   Invalid comments (with suggested responses): M
   Skipped comments: K

   Proceed with fixes? (yes/no/review-details)
   ```

4. Wait for user confirmation

### Step 2: Execute Fixes

**Agent**: Execute (Sonnet)
**Input**: PR review analysis with execution tasks

1. Invoke Execute Agent:
   ```
   Use Task tool:
   - Prompt: "Execute the fixes from PR review analysis at $PR_REVIEW_FILE.
     Follow the 'Execution Plan for Execute Agent' section.
     Implement each change as specified."
   ```

### Step 3: Verify Fixes

**Agent**: Review (Opus, Pre-PR mode)
**Loop until PASS or max iterations (3)**

1. Generate review file:
   ```bash
   VERIFY_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID verify)
   ```

2. Invoke Review Agent in Pre-PR mode:
   ```
   Use Task tool:
   - Prompt: "Verify the PR comment fixes were implemented correctly.
     Original PR comments: $PR_REVIEW_FILE
     Mode: Pre-PR Review
     Write verification to $VERIFY_FILE"
   ```

3. Check status and iterate if needed

### Step 4: Walkthrough

**Agent**: Walkthrough (Sonnet)

1. Generate walkthrough file:
   ```bash
   WALKTHROUGH_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID walkthrough)
   ```

2. Invoke Walkthrough Agent:
   ```
   Use Task tool:
   - Prompt: "Create walkthrough for PR review cycle.
     Document what comments were addressed and how.
     Write to $WALKTHROUGH_FILE"
   ```

### Step 5: Commit & Push

Use the `commit-pr` skill with `--push-only` flag (PR already exists):

```
Use Skill tool:
- skill: commit-pr
- args: "$ARTIFACT_ID --push-only"
```

The commit-pr skill will:
1. Read walkthrough artifact
2. Generate commit message ("Address PR review feedback" + summary)
3. Create commit and push (no PR creation since it's an update)

Optionally respond to invalid comments on PR if requested by user.

## Arguments

- `$ARGUMENTS`: PR URL (e.g., "https://github.com/org/repo/pull/123")

## Example Usage

```bash
# With PR URL
/pr-review https://github.com/team-attention/cops/pull/42

# With PR number (if in the repo)
/pr-review 42
```

## Responding to Invalid Comments

If there are invalid comments, offer to respond:
```
There are M comments marked as invalid.

Would you like me to post responses explaining why these suggestions were not implemented?
(yes/no/customize)
```

## Error Handling

- If PR URL is invalid, ask for correct URL
- If no comments to address, report and exit
- Save all artifacts for debugging
