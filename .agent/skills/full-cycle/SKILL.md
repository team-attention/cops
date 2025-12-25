---
name: full-cycle
description: |
  Execute complete development cycle from requirements clarification to PR.
  Interactive workflow that confirms with user before each step.
  Flow: Clarify -> Research -> Planning -> Execute -> Review -> Walkthrough -> Commit
---

# Full-Cycle Development Skill

This skill orchestrates the complete development cycle from requirements clarification to PR creation.

## Workflow Overview

```
Clarify (Sonnet)
    ↓ [User confirms requirements]
Research (Opus)
    ↓ [User confirms]
Planning (Opus)
    ↓ [User confirms]
Execute (Sonnet)
    ↓
Review (Opus, Pre-PR mode)
    ↓
    ├─ If FAIL → Execute (fix issues) → Review again
    │            (max 3 iterations)
    └─ If PASS ↓
Walkthrough (Sonnet)
    ↓ [User confirms]
Commit & PR
```

## Step-by-Step Process

### Step 0: Initialize

```bash
# Generate artifact ID
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)
echo "Starting Full-Cycle workflow with Artifact ID: $ARTIFACT_ID"
```

### Step 1: Clarify

**Agent**: Clarify (Sonnet)
**Input**: User request or Linear ticket ID from $ARGUMENTS

1. Generate artifact file path:
   ```bash
   REQUIREMENTS_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID requirements)
   ```

2. Invoke Clarify Agent:
   ```
   Use Task tool:
   - Prompt: "Clarify requirements for the following request.

     Request: [user request or ticket ID from $ARGUMENTS]

     If this is a Linear ticket ID, fetch the ticket details and organize the information.
     If this is a general request, ask structured questions to gather complete requirements.

     **IMPORTANT**: You MUST write the final requirements document to this file:
     $REQUIREMENTS_FILE

     Follow the Clarify Agent protocol and output format."
   ```

3. After completion, show user the requirements summary:
   ```
   Requirements clarified. Summary:
   [Show request summary, acceptance criteria, scope]

   Proceed to Research phase? (yes/no/modify)
   ```

4. Wait for user confirmation before continuing

### Step 2: Research

**Agent**: Research (Opus)
**Input**: Requirements from Step 1

1. Generate artifact file path:
   ```bash
   RESEARCH_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID research)
   ```

2. Invoke Research Agent:
   ```
   Use Task tool:
   - Prompt: "Analyze the requirements at $REQUIREMENTS_FILE and conduct research.

     **IMPORTANT**: You MUST write the research report to this file:
     $RESEARCH_FILE

     Follow the Research Agent protocol and output format."
   ```

3. After completion, show user the research summary:
   ```
   Research complete. Summary:
   [Show key findings from research report]

   Proceed to Planning phase? (yes/no/modify)
   ```

4. Wait for user confirmation before continuing

### Step 3: Planning

**Agent**: Planning (Opus)
**Input**: Research report path

1. Generate artifact file path:
   ```bash
   PLAN_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID plan)
   ```

2. Invoke Planning Agent:
   ```
   Use Task tool:
   - Prompt: "Read research report at $RESEARCH_FILE and create implementation plan.
     If you need clarification on any decisions, ask me before writing the plan.

     **IMPORTANT**: You MUST write the implementation plan to this file:
     $PLAN_FILE

     Follow the Planning Agent protocol and output format."
   ```

3. If Planning Agent asks questions, relay to user and get answers

4. After completion, show user the plan summary:
   ```
   Plan complete. Overview:
   [Show architecture decisions and step count]

   Proceed to Execute phase? (yes/no/modify)
   ```

5. Wait for user confirmation before continuing

### Step 4: Execute

**Agent**: Execute (Sonnet)
**Input**: Plan file path

1. Invoke Execute Agent:
   ```
   Use Task tool:
   - Prompt: "Execute the implementation plan at $PLAN_FILE.
     Follow each step in order.
     Run verification commands when complete."
   ```

2. No user confirmation needed - proceed directly to Review

### Step 5: Review (with iteration)

**Agent**: Review (Opus, Pre-PR mode)
**Max Iterations**: 3

Loop:
1. Generate review file path:
   ```bash
   REVIEW_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID review)
   ```

2. Invoke Review Agent:
   ```
   Use Task tool:
   - Prompt: "Perform Pre-PR code review.
     Check git diff against plan at $PLAN_FILE.
     Mode: Pre-PR Review

     **IMPORTANT**: You MUST write the review report to this file:
     $REVIEW_FILE

     Follow the Review Agent protocol and output format."
   ```

3. Check review status:
   - If PASS: Break loop, proceed to Walkthrough
   - If FAIL and iterations < 3:
     - Invoke Execute Agent with review feedback
     - Continue loop
   - If FAIL and iterations >= 3:
     - Report to user and ask how to proceed

### Step 6: Walkthrough

**Agent**: Walkthrough (Sonnet)
**Input**: All artifacts in artifacts directory

1. Generate walkthrough file path:
   ```bash
   WALKTHROUGH_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID walkthrough)
   ```

2. Invoke Walkthrough Agent:
   ```
   Use Task tool:
   - Prompt: "Create walkthrough document for artifact $ARTIFACT_ID.
     Read all artifacts in .agent/artifacts/$ARTIFACT_ID/

     **IMPORTANT**: You MUST write the walkthrough document to this file:
     $WALKTHROUGH_FILE

     Follow the Walkthrough Agent protocol and output format."
   ```

3. Show user the walkthrough summary:
   ```
   Walkthrough complete.

   Ready to commit and create PR? (yes/no)
   ```

### Step 7: Commit & PR

1. Extract information from artifacts:
   ```bash
   # Get ticket ID if available (from research artifact)
   # Get summary from walkthrough artifact
   ```

2. Create commit:
   ```bash
   git add .

   # Generate commit message from walkthrough
   git commit -m "$(cat <<'EOF'
   [type]: [summary from walkthrough] ([TICKET-ID if available])

   [Key changes from walkthrough]

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
   EOF
   )"
   ```

3. Push and create PR:
   ```bash
   # Push branch
   git push -u origin HEAD

   # Create PR with walkthrough content
   gh pr create \
     --title "[TICKET-ID] [Title from ticket/request]" \
     --body "$(cat $WALKTHROUGH_FILE)"
   ```

4. Update Linear ticket (if applicable):
   ```bash
   # Add PR URL as link to Linear ticket
   # Optionally update status to "In Review"
   ```

## Arguments

- `$ARGUMENTS`: User request or Linear ticket ID (e.g., "implement user auth" or "TA-123")

## Example Usage

```bash
# With user request
/full-cycle "implement HTTP client for external API"

# With Linear ticket
/full-cycle TA-123

# With specific focus
/full-cycle "TA-123 - focus on error handling"
```

## Skip Options

At each confirmation step, user can:
- `yes` - proceed to next step
- `no` - stop the workflow
- `skip [step]` - skip to a specific step
- `modify` - make changes before proceeding

## Error Handling

- If any agent fails, report error and ask user how to proceed
- Save all artifacts even on failure for debugging
- Provide option to resume from last successful step
