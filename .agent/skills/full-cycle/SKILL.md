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
Clarify (Sonnet)
    ↓ [AskUserQuestion: Proceed to Research?]
Research (Opus)
    ↓ [AskUserQuestion: Proceed to Planning?]
Planning (Opus)
    ↓ [AskUserQuestion: Proceed to Execute?]
Execute (Sonnet)
    ↓ [No user confirmation - automatic]
Review Loop (Opus, Pre-PR mode) ← iteration = 1
    ↓
    ├─ If PASS → Break loop
    │
    └─ If FAIL (iteration < 3)
           ↓
       Display issues
           ↓ [Automatic - no user confirmation]
       Execute (fix issues)
           ↓
       iteration++
           ↓
       Review again (same loop)
           ↓
       If FAIL (iteration >= 3)
           ↓ [AskUserQuestion: Continue anyway/Manual fix/Stop?]
           └─ Based on user choice

    ↓ [Only if PASS or user chose "Continue anyway"]
Walkthrough (Sonnet)
    ↓ [AskUserQuestion: Ready to commit and PR?]
Commit & PR
```

**Key Changes from Previous Version**:
1. **AskUserQuestion tool used at each approval point** (not just text prompt)
2. **Review Loop is automatic** - Execute runs automatically on FAIL without user confirmation
3. **Iteration tracking** - Each review iteration creates a new artifact file
4. **User only asked after 3 failed iterations** - Not after each iteration

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

3. After completion, show user the requirements summary and ask for confirmation:
   ```
   Requirements clarified. Summary:
   [Show request summary, acceptance criteria, scope]
   ```

4. **IMPORTANT**: Use AskUserQuestion tool to get user approval:
   ```
   Use AskUserQuestion tool:
   - Question: "Proceed to Research phase?"
   - Header: "Next Step"
   - Options:
     - label: "Yes, proceed", description: "Continue to Research phase"
     - label: "No, stop here", description: "Stop the workflow"
     - label: "Modify requirements", description: "Make changes to requirements first"
   - multiSelect: false
   ```

5. Handle user response:
   - "Yes, proceed" → Continue to Step 2
   - "No, stop here" → Exit workflow
   - "Modify requirements" or "Other" → Ask user for changes, update requirements, re-ask

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
   ```

4. **IMPORTANT**: Use AskUserQuestion tool to get user approval:
   ```
   Use AskUserQuestion tool:
   - Question: "Proceed to Planning phase?"
   - Header: "Next Step"
   - Options:
     - label: "Yes, proceed", description: "Continue to Planning phase"
     - label: "No, stop here", description: "Stop the workflow"
     - label: "Modify research", description: "Need to re-research some areas"
   - multiSelect: false
   ```

5. Handle user response:
   - "Yes, proceed" → Continue to Step 3
   - "No, stop here" → Exit workflow
   - "Modify research" or "Other" → Ask user for changes, update research, re-ask

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
   ```

5. **IMPORTANT**: Use AskUserQuestion tool to get user approval:
   ```
   Use AskUserQuestion tool:
   - Question: "Proceed to Execute phase?"
   - Header: "Next Step"
   - Options:
     - label: "Yes, proceed", description: "Continue to Execute phase"
     - label: "No, stop here", description: "Stop the workflow"
     - label: "Modify plan", description: "Need to adjust the implementation plan"
   - multiSelect: false
   ```

6. Handle user response:
   - "Yes, proceed" → Continue to Step 4
   - "No, stop here" → Exit workflow
   - "Modify plan" or "Other" → Ask user for changes, update plan, re-ask

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

### Step 5: Review Loop (with automatic iteration)

**Agent**: Review (Opus, Pre-PR mode)
**Max Iterations**: 3
**Important**: This step loops automatically until PASS or max iterations reached

**Loop Structure**:
```
iteration = 1
while iteration <= 3:
  1. Run Review Agent
  2. Read review status from artifact
  3. If PASS: break loop → proceed to Walkthrough
  4. If FAIL:
     - Show issues to user
     - Automatically invoke Execute Agent to fix issues
     - iteration++
     - Continue loop
  5. If iteration > 3 and still FAIL:
     - Ask user how to proceed
```

**Detailed Steps**:

1. **Generate review file path** (with iteration number):
   ```bash
   if [ $iteration -eq 1 ]; then
     REVIEW_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID review)
   else
     REVIEW_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID "review_iteration${iteration}")
   fi
   ```
   - First review: `04_review.md`
   - Second review: `05_review_iteration2.md`
   - Third review: `06_review_iteration3.md`

2. **Invoke Review Agent**:
   ```
   Use Task tool:
   - Prompt: "Perform Pre-PR code review (Iteration ${iteration}/3).
     Check git diff against plan at $PLAN_FILE.
     Mode: Pre-PR Review

     **IMPORTANT**: You MUST write the review report to this file:
     $REVIEW_FILE

     The review MUST include a clear status: PASS or FAIL.
     If FAIL, list all issues that need to be fixed.

     Follow the Review Agent protocol and output format."
   ```

3. **Read and parse review status**:
   - Read the review artifact file
   - Look for "Status: PASS" or "Status: FAIL" in the document
   - Extract list of issues if FAIL

4. **Handle review result**:

   **Case A: PASS**
   ```
   - Display: "✓ Review PASS - All checks passed"
   - Break loop
   - Proceed to Step 6 (Walkthrough)
   ```

   **Case B: FAIL and iteration < 3**
   ```
   - Display issues found:
     "✗ Review FAIL (Iteration ${iteration}/3)
      Issues found:
      [List all issues from review]

      Automatically invoking Execute Agent to fix these issues..."

   - Invoke Execute Agent:
     Use Task tool:
     - Prompt: "Fix all issues identified in the review at $REVIEW_FILE.

       This is iteration ${iteration} of the review-fix cycle.
       Focus on addressing each issue listed in the review document.

       After fixes, run verification commands to ensure everything builds."

   - increment iteration
   - Continue loop (go back to step 1)
   ```

   **Case C: FAIL and iteration >= 3**
   ```
   - Display:
     "✗ Review still FAIL after 3 iterations

      Issues remaining:
      [List all issues from review]"

   - Use AskUserQuestion tool:
     - Question: "Review failed after 3 iterations. How should we proceed?"
     - Header: "Review Failed"
     - Options:
       - label: "Continue anyway", description: "Proceed to Walkthrough despite issues (Recommended)"
       - label: "Manual fix", description: "I'll fix the issues manually, then re-run review"
       - label: "Stop workflow", description: "Stop and investigate"
     - multiSelect: false

   - Handle response:
     - "Continue anyway" → Proceed to Step 6 (Walkthrough)
     - "Manual fix" → Wait for user to fix, then re-run Review Agent
     - "Stop workflow" → Exit
   ```

**Important Notes**:
- **DO NOT proceed to Walkthrough until Review PASS** (unless user chooses "Continue anyway")
- Each iteration creates a new review artifact file for debugging
- Execute Agent runs automatically on FAIL (no user confirmation needed)
- User is only asked when max iterations exhausted

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
   [Brief summary of changes from walkthrough]
   ```

4. **IMPORTANT**: Use AskUserQuestion tool to get user approval:
   ```
   Use AskUserQuestion tool:
   - Question: "Ready to commit and create PR?"
   - Header: "Final Step"
   - Options:
     - label: "Yes, create commit and PR", description: "Commit changes and create pull request"
     - label: "No, stop here", description: "Stop before committing"
     - label: "Modify walkthrough", description: "Update walkthrough documentation first"
   - multiSelect: false
   ```

5. Handle user response:
   - "Yes, create commit and PR" → Continue to Step 7
   - "No, stop here" → Exit workflow
   - "Modify walkthrough" or "Other" → Ask user for changes, update walkthrough, re-ask

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

## User Approval Points

The workflow uses **AskUserQuestion tool** at these approval points:

1. **After Clarify**: Proceed to Research? (yes/no/modify)
2. **After Research**: Proceed to Planning? (yes/no/modify)
3. **After Planning**: Proceed to Execute? (yes/no/modify)
4. **After Review (3 failures)**: Continue anyway/Manual fix/Stop?
5. **After Walkthrough**: Ready to commit and PR? (yes/no/modify)

**User can always select "Other" to provide custom input** at any approval point.

**Automatic Steps (No User Approval)**:
- Execute → Review (always proceeds)
- Review FAIL → Execute (auto-fixes, max 3 iterations)
- Review iteration loop (automatic until PASS or 3 failures)

## Error Handling

- If any agent fails, report error and ask user how to proceed
- Save all artifacts even on failure for debugging
- Provide option to resume from last successful step
