---
name: full-cycle
description: |
  Execute complete development cycle from requirements clarification to PR.
  Interactive workflow using AskUserQuestion tool for approval at key checkpoints.
  Automatic Review→Execute loop (max 3 iterations) for fixing issues.
  Flow: Clarify → Research → Planning → Execute → Review Loop → Walkthrough → Commit
---

# Full-Cycle Development Skill

This skill orchestrates the complete development cycle from requirements clarification to PR creation.

## Workflow Overview

```
Clarify → Research → Planning → Execute → Review (auto-loop) → Walkthrough → Commit & PR
   ↓         ↓          ↓                        ↓                  ↓             ↓
 [User]    [User]    [User]                   [User]             [User]      [Skill]
```

**User Approval Points**: After Clarify, Research, Planning, Review (PASS), Walkthrough
**Automatic Steps**: Execute, Review (FAIL → re-execute, max 3 times)

## Arguments

- `$ARGUMENTS`: User request or Linear ticket ID (e.g., "implement user auth" or "TA-123")

## General Step Pattern

Most steps follow this pattern:

1. **Generate artifact path**:
   ```bash
   FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID {artifact-name})
   ```

2. **Invoke agent** using Task tool:
   ```
   Use Task tool:
   - subagent_type: {agent-type}
   - model: {model}
   - prompt: "...

     **IMPORTANT**: You MUST write the {output} to this file: $FILE

     Follow the {Agent} Agent protocol and output format."
   ```

3. **Show summary** to user from artifact

4. **Ask for approval** (if required):
   ```
   Use AskUserQuestion tool:
   - Question: "Proceed to {NextStep}?"
   - Header: "Next Step"
   - Options:
     - label: "Yes, proceed", description: "Continue to {NextStep}"
     - label: "No, stop here", description: "Stop the workflow"
     - label: "Modify {current}", description: "Make changes first"
   - multiSelect: false
   ```

5. **Handle response**: Yes → next step, No → exit, Modify → wait & re-ask

## Steps Configuration

| # | Step | Agent | Model | Artifact | Approval | Notes |
|---|------|-------|-------|----------|----------|-------|
| 0 | Initialize | - | - | `$ARTIFACT_ID` | No | Generate artifact ID |
| 1 | Clarify | clarify | sonnet | requirements | Yes | Gather requirements |
| 2 | Research | research | opus | research | Yes | Analyze codebase |
| 3 | Planning | planning | opus | plan | Yes | Design implementation |
| 4 | Execute | execute | sonnet | (uses plan) | No | Implement code |
| 5 | Review Loop | review | opus | review / review_iteration{N} | Conditional* | Check quality |
| 6 | Walkthrough | walkthrough | sonnet | walkthrough | Yes | Document changes |
| 7 | Commit & PR | - | - | - | No** | Use commit-pr skill |

*Step 5 approval:
- **PASS**: Ask user (yes/manual fix/stop)
- **FAIL (iteration < 3)**: Automatic re-execute
- **FAIL (iteration >= 3)**: Ask user (continue/manual fix/stop)

**Step 7: Automatically runs after Walkthrough approval

## Special Step Details

### Step 0: Initialize

```bash
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)
echo "Starting Full-Cycle workflow with Artifact ID: $ARTIFACT_ID"
```

### Step 5: Review Loop

**Special logic** - loops until PASS or 3 failures:

```
iteration = 1
while iteration <= 3:
  1. Generate review artifact:
     - First: `.agent/artifacts/$ARTIFACT_ID/review.md`
     - Later: `.agent/artifacts/$ARTIFACT_ID/review_iteration{N}.md`

  2. Invoke Review Agent (Opus, Pre-PR mode)
     - Prompt: "Check git diff against plan at $PLAN_FILE"
     - Must include clear status: PASS or FAIL

  3. Read review status

  4. Handle result:
     a. PASS:
        - Show review summary
        - Ask user: "Proceed to Walkthrough?" (yes/manual fix/stop)
        - If yes → break loop
        - If manual fix → wait for user, then re-run review
        - If stop → exit workflow

     b. FAIL and iteration < 3:
        - Display issues
        - Automatically invoke Execute Agent to fix issues
        - iteration++
        - Continue loop

     c. FAIL and iteration >= 3:
        - Display remaining issues
        - Ask user: "Continue anyway/Manual fix/Stop?"
        - Handle based on choice
```

### Step 7: Commit & PR

Use the `commit-pr` skill:

```
Use Skill tool:
- skill: commit-pr
- args: $ARTIFACT_ID  # Pass artifact ID for walkthrough mode
```

The commit-pr skill will:
1. Read walkthrough artifact
2. Generate commit message
3. Create commit and push
4. Create PR with walkthrough as body

## Example Usage

```bash
# With user request
/full-cycle "fix CLI to API connection issue"

# With Linear ticket
/full-cycle TA-123

# With specific focus
/full-cycle "TA-123 - focus on error handling"
```

## Error Handling

- If any agent fails, report error and ask user how to proceed
- Save all artifacts even on failure for debugging
- Provide option to resume from last successful step
