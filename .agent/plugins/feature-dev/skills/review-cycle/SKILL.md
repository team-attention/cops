---
name: review-cycle
description: |
  Automated code review cycle with two phases:
  Phase A - Automatic review/fix loop until code passes
  Phase B - User feedback loop until user approves
---

# Review-Cycle Skill

Orchestrates automated code review cycles with two distinct phases: automated validation and user feedback integration.

## Input

- `$ARTIFACT_ID` - Artifact directory identifier for storing review documents

## Phase A: Automated Review Cycle

Execute an automated review loop that continues until the code passes review.

### Step A.0: Create Review Artifact File

Initialize the iteration counter if not already set:
```bash
ITERATION_A=${ITERATION_A:-1}
```

Create an artifact file for the review document:
```bash
if [ $ITERATION_A -eq 1 ]; then
  REVIEW_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID review)
else
  REVIEW_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID "review_iteration${ITERATION_A}")
fi
```

### Step A.1: Code Review Agent

Invoke the Code Review Agent to analyze the current codebase:

```
Use Task tool:
- subagent_type: code-review
- prompt: "Review the current codebase changes.

  **IMPORTANT**: You MUST write the review document to: $REVIEW_FILE"
```

Wait for the Code Review Agent to complete.

### Step A.2: Check Pass/Fail Status

Read the review document to determine the status:

```bash
REVIEW_STATUS=$(grep "^\*\*Status\*\*:" "$REVIEW_FILE" | cut -d: -f2 | xargs)
```

**Decision Point:**
- If `$REVIEW_STATUS` equals "Pass" → **Go to Phase B**
- If `$REVIEW_STATUS` equals "Changes Required" → **Continue to A.3**

### Step A.3: Plan Agent (Fail Path)

Create a plan artifact file:
```bash
if [ $ITERATION_A -eq 1 ]; then
  PLAN_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID plan)
else
  PLAN_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID "plan_iteration${ITERATION_A}")
fi
```

Invoke the Plan Agent:
```
Use Task tool:
- subagent_type: planning
- prompt: "Read the review document from: $REVIEW_FILE

  Create a detailed implementation plan to address the identified issues.

  **IMPORTANT**: You MUST write the implementation plan to: $PLAN_FILE"
```

### Step A.4: Implement Agent

Invoke the Implement Agent:
```
Use Task tool:
- subagent_type: implement
- prompt: "Read the implementation plan from: $PLAN_FILE

  Implement the code changes according to this plan."
```

### Step A.5: Loop Back

Increment the iteration counter and return to Step A.0:
```bash
ITERATION_A=$((ITERATION_A + 1))
# Go back to Step A.0
```

---

## Phase B: User Review Loop

Once Phase A completes with a passing review, enter the user feedback loop.

### Step B.0: Summarize and Request User Decision

Read all artifact files from the current artifact directory to prepare a summary.

**Find the most recent iteration's documents:**
```bash
ARTIFACT_DIR=".agent/artifacts/$ARTIFACT_ID"
LATEST_REVIEW=$(ls -1 "$ARTIFACT_DIR"/*review*.md | tail -1)
LATEST_PLAN=$(ls -1 "$ARTIFACT_DIR"/*plan*.md | tail -1)
```

**Generate summary:**
- Read and summarize the latest review document
- Read and summarize the latest plan document (if exists)
- Highlight key changes made during Phase A iterations

**Present to user:**

Output the summary to the user, then ask:

```
Use AskUserQuestion tool:
- questions:
  - question: "The code has passed automated review. Would you like to end the review cycle, or do you have additional feedback?"
    header: "Next Step"
    options:
      - label: "End Review Cycle"
        description: "Complete the review process with current state"
      - label: "Provide Feedback"
        description: "Give feedback for additional changes"
    multiSelect: false
```

**Decision Point:**
- If user selects "End Review Cycle" → **Go to Output**
- If user provides feedback → **Continue to B.1**

### Step B.1: Code Review Agent with User Feedback

Initialize the iteration counter if not already set:
```bash
ITERATION_B=${ITERATION_B:-1}
```

Create a new review artifact file:
```bash
REVIEW_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID "user_review_iteration${ITERATION_B}")
```

Invoke the Code Review Agent with user feedback included:
```
Use Task tool:
- subagent_type: code-review
- prompt: "Review the current codebase with the following user feedback:

  ---
  USER FEEDBACK:
  {Insert user feedback from B.0}
  ---

  Treat the user feedback as required changes that must be addressed.

  **IMPORTANT**: You MUST write the review document to: $REVIEW_FILE"
```

**Note:** Since user feedback is treated as required changes, the review status will always be "Changes Required".

### Step B.2: Plan Agent

Create a plan artifact file:
```bash
PLAN_FILE=$(Skill tool: artifact, args: create $ARTIFACT_ID "user_plan_iteration${ITERATION_B}")
```

Invoke the Plan Agent:
```
Use Task tool:
- subagent_type: planning
- prompt: "Read the review document from: $REVIEW_FILE

  Create a detailed implementation plan to address the user feedback.

  **IMPORTANT**: You MUST write the implementation plan to: $PLAN_FILE"
```

### Step B.3: Implement Agent

Invoke the Implement Agent:
```
Use Task tool:
- subagent_type: implement
- prompt: "Read the implementation plan from: $PLAN_FILE

  Implement the code changes according to this plan."
```

### Step B.4: Loop Back to B.0

Increment the iteration counter:
```bash
ITERATION_B=$((ITERATION_B + 1))
```

Return to Step B.0 to summarize the **Phase B** iteration's results (not Phase A):
- Summarize documents from the current B iteration
- Present updated changes to the user
- Ask again for user decision

---

## Output

When the review cycle completes (user selects "End Review Cycle" in Phase B.0):

**Wrapup Summary:**

```
Review cycle completed successfully.

## Summary
- Phase A iterations: $ITERATION_A
- Phase B iterations: $ITERATION_B
- Final status: Pass
- Artifact directory: .agent/artifacts/$ARTIFACT_ID

## Artifacts Created
{List all review and plan files created}

## Final State
{Brief description of the final codebase state}
```

Exit the skill.
