# Start Development Command

This command orchestrates the complete development cycle from requirements clarification to PR creation.

## Workflow Overview

```
Clarify → Plan → Execute → Review (skill) → Walkthrough → Commit & PR
   ↓       ↓                    ↓              ↓             ↓
 [User]  [User]               [Skill]        [User]       [Skill]
```

**User Approval Points**: After Clarify, Plan, Walkthrough
**Automatic Steps**: Execute
**Skill-Handled**: Review (auto-loops), Commit & PR

## Arguments

- `$ARGUMENTS`: User request or Linear ticket ID (e.g., "implement user auth" or "TA-123")

## Important Constraints

**This command must ONLY orchestrate using SubAgents and Skills:**
- ✅ Use Task tool to invoke SubAgents
- ✅ Use Skill tool to invoke Skills
- ✅ Use AskUserQuestion to get user approval
- ✅ Use Read to read artifact outputs
- ❌ DO NOT directly modify or create files
- ❌ DO NOT use Edit, Write tools

**All file operations must happen through SubAgents or Skills.**

## Steps Configuration

| #   | Step        | Agent/Skill         | Model  | Artifact       | Approval | Notes                      |
| --- | ----------- | ------------------- | ------ | -------------- | -------- | -------------------------- |
| 0   | Initialize  | -                   | -      | `$ARTIFACT_ID` | No       | Generate artifact ID       |
| 1   | Clarify     | clarify             | sonnet | requirements   | Yes      | Gather requirements        |
| 2   | Plan        | planning            | opus   | plan           | Yes      | Design implementation      |
| 3   | Execute     | execute             | sonnet | (uses plan)    | No       | Implement code             |
| 4   | Review      | **review-cycle skill** | -      | -              | No*      | Auto-loop handled by skill |
| 5   | Walkthrough | walkthrough         | sonnet | walkthrough    | Yes      | Document changes           |
| 6   | Commit & PR | **commit-pr skill** | -      | -              | No       | Auto commit and PR         |

*Review skill handles its own approval logic and Execute re-invocations

## Step Details

### Step 0: Initialize

Generate unique artifact ID for this workflow using artifact skill:

```
Use Skill tool:
- skill: artifact
- args: init

This returns the artifact ID (e.g., 20251229-143022) and creates the artifact directory.
Store this ID in $ARTIFACT_ID for use in subsequent steps.
```

### Step 1: Clarify Requirements

**Purpose**: Gather and clarify user requirements

**Process**:
1. Generate artifact path using artifact skill:
   ```
   Use Skill tool:
   - skill: artifact
   - args: create $ARTIFACT_ID clarify

   Store the returned file path in $CLARIFY_FILE
   ```

2. Invoke clarify agent:
   ```
   Use Task tool:
   - subagent_type: clarify
   - prompt: "The user wants: $ARGUMENTS

     **IMPORTANT**: You MUST write the requirements document to: $CLARIFY_FILE
   ```

3. Read artifact and show summary to user

4. **Approval Loop**:
   ```
   Use AskUserQuestion:
   - Question: "Does this requirements document capture what you need?"
   - Header: "Requirements"
   - Options:
     - label: "Yes, proceed to planning"
     - label: "No, needs changes"
   - multiSelect: false
   ```

5. **If "No, needs changes"**:
   - Ask user: "What changes are needed?"
   - Resume clarify agent with:
     ```
     prompt: "The user provided feedback on the requirements:

     [user feedback]

     Please update the requirements document at: $CLARIFY_FILE

     Original requirements are already in that file."
     ```

6. **If "Yes"**: Proceed to Step 2

### Step 2: Plan Implementation

**Purpose**: Create detailed implementation plan

**Process**:
1. Generate artifact path using artifact skill:
   ```
   Use Skill tool:
   - skill: artifact
   - args: create $ARTIFACT_ID plan

   Store the returned file path in $PLAN_FILE
   ```

2. Invoke planning agent:
   ```
   Use Task tool:
   - subagent_type: planning
   - prompt: "Read the requirements from: $CLARIFY_FILE

     Create a detailed implementation plan.

     **IMPORTANT**: You MUST write the implementation plan to: $PLAN_FILE
   ```

3. Read artifact and show summary to user

4. **Approval Loop**:
   ```
   Use AskUserQuestion:
   - Question: "Does this implementation plan look good?"
   - Header: "Plan Review"
   - Options:
     - label: "Yes, start implementation"
     - label: "No, needs changes"
   - multiSelect: false
   ```

5. **If "No, needs changes"**:
   - Ask user: "What changes are needed to the plan?"
   - Resume planning agent with feedback

6. **If "Yes"**: Proceed to Step 3

### Step 3: Execute Implementation

**Purpose**: Implement the code according to the plan

**Process**:
1. Invoke implement agent:
   ```
   Use Task tool:
   - subagent_type: implement
   - prompt: "Read the implementation plan from: $PLAN_FILE

     Implement the code according to this plan.
   ```

2. **When agent completes successfully**: Automatically proceed to Step 4
3. **If agent exits/fails**: Ask user how to proceed

### Step 4: Review Code

**Purpose**: Review implementation and auto-fix issues

**Process**:
1. Invoke review cycle skill:
   ```
   Use Skill tool:
   - skill: review-cycle
   - args: $ARTIFACT_ID
   ```

2. **When skill completes successfully**: Proceed to Step 5
3. **If skill exits/fails**: Ask user how to proceed

### Step 5: Walkthrough Documentation

**Purpose**: Create documentation of changes

**Process**:
1. Generate artifact path using artifact skill:
   ```
   Use Skill tool:
   - skill: artifact
   - args: create $ARTIFACT_ID walkthrough

   Store the returned file path in $WALKTHROUGH_FILE
   ```

2. Invoke walkthrough agent:
   ```
   Use Task tool:
   - subagent_type: walkthrough
   - prompt: "Review all changes made in this workflow:
     - Requirements: $CLARIFY_FILE
     - Plan: $PLAN_FILE

     Create a comprehensive walkthrough document.

     **IMPORTANT**: You MUST write the walkthrough to: $WALKTHROUGH_FILE"
   ```

3. Read artifact and show summary to user

4. **Approval**:
   ```
   Use AskUserQuestion:
   - Question: "Ready to commit and create PR?"
   - Header: "Final Check"
   - Options:
     - label: "Yes, commit and create PR"
     - label: "No, I'll do it manually"
   - multiSelect: false
   ```

5. **If "Yes"**: Proceed to Step 6
6. **If "No"**: Exit workflow with message about manual commit

### Step 6: Commit & Create PR

**Purpose**: Create commit and pull request

**Process**:
1. Invoke commit-pr skill:
   ```
   Use Skill tool:
   - skill: commit-pr
   - args: $ARTIFACT_ID
   ```

2. **Workflow Complete** - Finish this workflow

## Example Usage

```bash
# With user request
/start-dev "implement user authentication"

# With Linear ticket
/start-dev TA-123

# With specific details
/start-dev "TA-123 - add JWT token validation to API endpoints"
```
