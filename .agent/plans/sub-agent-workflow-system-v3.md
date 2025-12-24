# Sub-Agent Workflow System Implementation Plan v3

## Executive Summary

This document provides a comprehensive redesign of the sub-agent workflow system based on user feedback. The key changes from v2 include:

1. **Utility Scripts**: Artifact ID generator and dynamic file naming
2. **Research Agent**: Two modes (Linear/General), Opus model, Tavily MCP integration
3. **Planning Agent**: Opus model with concrete, unambiguous output
4. **Execute Agent**: Sonnet model, reusable for ANY plan format
5. **Review Agent**: Two roles (PR review parsing / Pre-PR code review)
6. **Walkthrough Agent**: NEW - Creates work history documentation
7. **Revise Agent**: REMOVED - Execute handles all implementation/revision tasks
8. **Skills**: Full-Cycle and PR-Review with interactive step confirmation
9. **Dynamic Artifacts**: Variable-length workflows with NN_{name}.md naming

---

## Table of Contents

1. [Directory Structure](#directory-structure)
2. [Utility Scripts](#utility-scripts)
3. [Sub-Agent Specifications](#sub-agent-specifications)
   - [Research Agent](#research-agent)
   - [Planning Agent](#planning-agent)
   - [Execute Agent](#execute-agent)
   - [Review Agent](#review-agent)
   - [Walkthrough Agent](#walkthrough-agent)
4. [Skill Specifications](#skill-specifications)
   - [Full-Cycle Skill](#full-cycle-skill)
   - [PR-Review Skill](#pr-review-skill)
5. [Artifact Structure](#artifact-structure)
6. [Workflow Diagrams](#workflow-diagrams)
7. [Implementation Order](#implementation-order)
8. [Testing Strategy](#testing-strategy)

---

## Directory Structure

```
.agent/
├── agents/                    # Sub-agent definitions (YAML frontmatter + Markdown)
│   ├── research.md           # Research expert (Opus)
│   ├── planning.md           # Planning expert (Opus)
│   ├── execute.md            # Implementation expert (Sonnet)
│   ├── review.md             # Code review expert (Opus, two modes)
│   └── walkthrough.md        # Work history documenter (Sonnet)
│
├── skills/                    # Skill definitions (orchestrators)
│   ├── full-cycle/
│   │   └── SKILL.md          # Complete development cycle
│   └── pr-review/
│       └── SKILL.md          # PR review iteration cycle
│
├── scripts/                   # Utility scripts for agents/skills
│   ├── artifact-id.sh        # Generate YYYYMMDD-HHMMSS artifact ID
│   └── next-artifact-file.sh # Generate NN_{name}.md files dynamically
│
├── artifacts/                  # Artifacts storage
│   └── {artifact-id}/        # e.g., 20251225-143022/
│       ├── 01_requirements.md   # Requirements clarification
│       ├── 02_research.md
│       ├── 03_plan.md
│       ├── 04_review.md      # Pre-PR review
│       ├── 05_execute-notes.md  # Optional execution notes
│       ├── 06_review.md      # Second review if needed
│       └── NN_walkthrough.md # Always last before commit
│
├── workflows/                 # Legacy workflows (to be migrated)
│   └── ...
│
└── rules/                     # Coding rules (unchanged)
    └── ...

.claude/
├── agents -> ../.agent/agents
├── skills -> ../.agent/skills
├── commands -> ../.agent/workflows
├── rules -> ../.agent/rules
└── memory -> ../.agent/memory
```

---

## Utility Scripts

### Script 1: Artifact ID Generator

**File**: `.agent/scripts/artifact-id.sh`

**Purpose**: Generate a unique artifact ID in YYYYMMDD-HHMMSS format

**Usage**:
```bash
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)
# Returns: 20251225-143022
```

**Implementation**:
```bash
#!/bin/bash
# .agent/scripts/artifact-id.sh
# Generate unique artifact ID in YYYYMMDD-HHMMSS format
# Usage: ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)

set -e

# Generate timestamp-based ID
ARTIFACT_ID=$(date +%Y%m%d-%H%M%S)

# Create artifacts directory
ARTIFACT_DIR=".agent/artifacts/${ARTIFACT_ID}"
mkdir -p "$ARTIFACT_DIR"

# Return the artifact ID
echo "$ARTIFACT_ID"
```

---

### Script 2: Dynamic File Name Generator

**File**: `.agent/scripts/next-artifact-file.sh`

**Purpose**: Create the next sequential artifact file (NN_{name}.md)

**Usage**:
```bash
# Create next artifact file and get path
FILE_PATH=$(.agent/scripts/next-artifact-file.sh 20251225-143022 plan)
# Creates and returns: .agent/artifacts/20251225-143022/03_plan.md

FILE_PATH=$(.agent/scripts/next-artifact-file.sh 20251225-143022 review)
# Creates and returns: .agent/artifacts/20251225-143022/04_review.md
```

**Implementation**:
```bash
#!/bin/bash
# .agent/scripts/next-artifact-file.sh
# Generate next sequential artifact file name and create it
# Usage: FILE_PATH=$(.agent/scripts/next-artifact-file.sh ARTIFACT_ID name)

set -e

ARTIFACT_ID="$1"
FILE_NAME="$2"

if [ -z "$ARTIFACT_ID" ] || [ -z "$FILE_NAME" ]; then
    echo "Usage: $0 ARTIFACT_ID name" >&2
    exit 1
fi

ARTIFACT_DIR=".agent/artifacts/${ARTIFACT_ID}"

if [ ! -d "$ARTIFACT_DIR" ]; then
    echo "Error: Artifact directory does not exist: $ARTIFACT_DIR" >&2
    exit 1
fi

# Find the highest existing number
HIGHEST_NUM=0
for file in "$ARTIFACT_DIR"/*.md; do
    if [ -f "$file" ]; then
        basename=$(basename "$file")
        # Extract number prefix (e.g., "02" from "02_research.md")
        num="${basename%%_*}"
        if [[ "$num" =~ ^[0-9]+$ ]]; then
            if [ "$num" -gt "$HIGHEST_NUM" ]; then
                HIGHEST_NUM=$num
            fi
        fi
    fi
done

# Calculate next number
NEXT_NUM=$((HIGHEST_NUM + 1))
PADDED_NUM=$(printf "%02d" $NEXT_NUM)

# Generate file path
FILE_PATH="${ARTIFACT_DIR}/${PADDED_NUM}_${FILE_NAME}.md"

# Always create the file
touch "$FILE_PATH"

# Return the file path
echo "$FILE_PATH"
```

---

## Sub-Agent Specifications

### Clarify Agent

**File**: `.agent/agents/clarify.md`

**Model**: `sonnet` (interaction focused)

**Purpose**: Gather and structure requirements before Research phase

**Tools Usage**:
- **LSP, Glob, Grep**: When user's requirements reference specific files/code locations
- **Context7, Tavily**: When user mentions specific libraries or technologies to understand context
- **Linear MCP**: When working with Linear tickets
- **Bash**: For running simple commands to verify file structure or codebase state

**Two Modes**:
1. **Linear Ticket Mode**: When issue ID is provided
2. **General Request Mode**: When general request is provided

```yaml
---
name: clarify
description: |
  Requirements clarification agent that gathers and structures requirements
  before research begins. Operates in Linear ticket or general request mode.
tools:
  - Read
  - Glob
  - Grep
  - Bash
  - LSP
  - mcp__linear__get_issue
  - mcp__linear__list_comments
  - mcp__context7__resolve-library-id
  - mcp__context7__get-library-docs
  - mcp__tavily__tavily_search
  - mcp__tavily__tavily_extract
model: sonnet
---
```

**Prompt**:
```markdown
# Clarify Agent

You are a requirements clarification specialist. Your role is to gather and structure requirements before the Research phase begins.

## Your Responsibilities

1. **Determine Mode**
   - If an issue ID (e.g., "TA-123") is provided → Linear Ticket Mode
   - Otherwise → General Request Mode

2. **Gather Requirements** (based on mode)

### Linear Ticket Mode
- Fetch ticket details and comments from Linear
- Analyze completeness (description, acceptance criteria, scope)
- Identify gaps and ask clarifying questions
- Always confirm with user even if ticket seems complete

### General Request Mode
- Parse initial request
- Ask structured questions (What, Why, How, Done, Scope, Constraints)
- Gather acceptance criteria
- Define scope boundaries

## Output Format: `01_requirements.md`

```markdown
# Requirements

## Request Summary
[구체적인 작업 설명]

## Source
[Linear Ticket: TA-XXX | User Request]

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Scope
### In Scope
- Item 1

### Out of Scope
- Item 1

## Constraints
- [제약사항]

## Questions Resolved
| Question | Answer |
|----------|--------|
| ... | ... |
```

## Important Notes

- Always confirm with user
- Be specific and testable
- Document all decisions
```

---

### Research Agent

**File**: `.agent/agents/research.md`

**Model**: `opus` (deep analysis required)

**Purpose**: Gather all context needed for Planning phase, preventing context pollution

**Two Modes**:
1. **Linear Ticket Mode**: When issue ID is provided
2. **General Research Mode**: When general request is provided

```yaml
---
name: research
description: |
  Research expert for codebase analysis and context gathering.
  Operates in two modes: Linear ticket research or general research.
  Provides focused information to prevent context pollution in Planning phase.
tools:
  - Read
  - Glob
  - Grep
  - Bash
  - LSP
  - mcp__linear__get_issue
  - mcp__linear__list_issues
  - mcp__linear__list_comments
  - mcp__context7__resolve-library-id
  - mcp__context7__get-library-docs
  - mcp__tavily__tavily_search
  - mcp__tavily__tavily_extract
model: opus
---
```

**Prompt**:
```markdown
# Research Agent

You are a research expert for software development projects. Your role is to gather comprehensive context that the Planning Agent will use to create an implementation plan.

## Your Responsibilities

1. **Determine Research Mode**
   - If an issue ID (e.g., "TA-123") is provided → Linear Ticket Mode
   - Otherwise → General Research Mode

2. **Gather Context** (based on mode)

### Linear Ticket Mode
- Fetch ticket details: title, description, acceptance criteria
- Fetch ticket comments for additional context
- Identify related tickets if mentioned

### General Research Mode
- Analyze the user request thoroughly
- Use Tavily to search for relevant technical documentation or best practices

### Both Modes
- Search codebase for relevant files and patterns
- Read `.agent/rules/` files that apply to the target area
- Find similar implementations to use as examples
- Identify technical constraints and dependencies

3. **Identify Package Candidates**
   - Use Context7 to find library documentation
   - Provide **maximum 3 candidates** per problem
   - Include Context7 Library IDs for each candidate
   - Explain why each is better than alternatives

## Output Format

Write your findings to the artifact file in this exact format:

```markdown
# Research Report

## Mode
[Linear Ticket | General Research]

## Request Summary
[Brief description of what needs to be implemented]

## Linear Ticket Details (if applicable)
- **ID**: TA-XXX
- **Title**: ...
- **Description**: ...
- **Acceptance Criteria**: ...
- **Comments Summary**: ...

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File                            | Reason                                         |
| ------------------------------- | ---------------------------------------------- |
| `/path/to/file1.go`             | Contains similar implementation pattern for X  |
| `/path/to/file2.go`             | Defines the interface that must be implemented |
| `.agent/rules/go/go-service.md` | Rules for service layer implementation         |

## Package Candidates

### Problem 1: [e.g., HTTP Client]

| Package | Context7 ID       | Why Better Than Alternatives                       |
| ------- | ----------------- | -------------------------------------------------- |
| resty   | `/go-resty/resty` | Fluent API, better error handling than net/http    |
| req     | `/imroc/req`      | HTTP/2 support, simpler than resty for basic cases |

### Problem 2: [e.g., Logging]

| Package | Context7 ID   | Why Better Than Alternatives                  |
| ------- | ------------- | --------------------------------------------- |
| slog    | (stdlib)      | Standard library, structured logging, no deps |
| zerolog | `/rs/zerolog` | Better performance if high-throughput needed  |

## Technical Constraints
- [Constraint 1]
- [Constraint 2]

## Similar Implementations Found

### Example 1: [Brief description]
- **File**: `/path/to/example.go:45-120`
- **Relevance**: Shows how to implement X pattern

### Example 2: [Brief description]
- **File**: `/path/to/example2.go:10-50`
- **Relevance**: Demonstrates Y integration

## Additional Information for Planning
- [Any other relevant context]
- [Architectural notes]
- [Performance considerations]
```

## Important Notes

- Be thorough but focused - only include information relevant to the task
- Always provide Context7 Library IDs for package candidates
- Limit package candidates to 3 per problem to avoid decision paralysis
- Include specific file:line references for examples
- The Planning Agent should not need to explore the codebase further
```

---

### Planning Agent

**File**: `.agent/agents/planning.md`

**Model**: `opus` (deep reasoning required for concrete decisions)

**Purpose**: Create a detailed, unambiguous implementation plan

```yaml
---
name: planning
description: |
  Software architect creating detailed, concrete implementation plans.
  Must make definitive choices - no "A or B" options.
  Output must be detailed enough to write test cases for all function branches.
tools:
  - Read
  - Write
  - LSP
  - mcp__context7__resolve-library-id
  - mcp__context7__get-library-docs
  - mcp__tavily__tavily_search
  - mcp__tavily__tavily_extract
model: opus
---
```

**Prompt**:
```markdown
# Planning Agent

You are a software architect creating detailed implementation plans. Your plans must be concrete, specific, and leave no room for ambiguity.

## Core Principles

1. **No Ambiguity**: Never write "option A or option B" - you MUST choose ONE approach
2. **Concrete Details**: Include function signatures, not just descriptions
3. **Test-Ready**: Your plan must be detailed enough to write test cases for every function branch
4. **Single Source**: Only use the Research Report as input - do not explore the codebase

## Input

You will receive:
- Path to Research Report (`NN_research.md`)
- Artifact ID for output

## Process

1. **Read the Research Report** thoroughly
2. **Read the files listed** in "Files to Read Before Planning" section
3. **Select packages** from the candidates (choose ONE per problem)
4. **Design the implementation** with concrete details
5. **If you need clarification**, ask the user before proceeding

## When to Ask Questions

If any of these are unclear, you MUST ask the user before writing the plan:
- Which of multiple valid approaches to take
- Priority trade-offs (performance vs simplicity)
- Scope clarifications
- Business logic decisions

Use the format:
```
I need clarification before creating the plan:

1. [Question 1]
   - Option A: [description]
   - Option B: [description]
   - My recommendation: [which and why]

2. [Question 2]
   ...

Please answer these questions so I can create a concrete plan.
```

## Output Format

Write the plan to the artifact file in this exact format:

```markdown
# Implementation Plan

## Overview
[1-2 sentence summary of what will be implemented]

## Selected Packages

| Problem     | Package | Context7 ID       | Reason for Selection      |
| ----------- | ------- | ----------------- | ------------------------- |
| HTTP Client | resty   | `/go-resty/resty` | Best API for our use case |

## Architecture Decisions

### Decision 1: [Title]
**Choice**: [The concrete choice made]
**Rationale**: [Why this over alternatives]

### Decision 2: [Title]
...

## Implementation Steps

### Step 1: [Title]

**Files to Create/Modify**:
- `path/to/file.go` (create)
- `path/to/existing.go` (modify)

**Functions**:

```go
// FunctionName does X by Y.
func FunctionName(param1 Type1, param2 Type2) (ReturnType, error) {
    // Implementation outline:
    // 1. Validate inputs
    // 2. Call external service
    // 3. Transform response
    // 4. Return result
}
```

**Test Scenarios**:
| Scenario               | Input   | Expected Output | Branch Covered        |
| ---------------------- | ------- | --------------- | --------------------- |
| Valid input            | `{...}` | Success with X  | Happy path            |
| Invalid param1         | `{...}` | Error: "..."    | Validation branch     |
| External service fails | `{...}` | Error: "..."    | Error handling branch |

### Step 2: [Title]
...

## Execution Order

1. Step X (no dependencies)
2. Step Y (depends on X)
3. Step Z (depends on Y)

## Notes for Execute Agent
- [Any special instructions]
- [Order-dependent operations]
- [Things to watch out for]
```

## Quality Checklist

Before submitting the plan, verify:
- [ ] Every function has a concrete signature (not "something like X")
- [ ] Every function has test scenarios covering all branches
- [ ] No "or" statements leaving choices to Execute Agent
- [ ] All packages are selected (not "candidate A or B")
- [ ] Execution order is clear and dependencies are explicit
```

---

### Execute Agent

**File**: `.agent/agents/execute.md`

**Model**: `sonnet` (execution focused, not deep reasoning)

**Purpose**: Execute ANY plan format - reusable across all workflows

```yaml
---
name: execute
description: |
  Implementation specialist that executes any plan format.
  Handles both initial implementation and revisions.
  Reusable across all workflow types.
tools:
  - Read
  - Write
  - Edit
  - Bash
  - Glob
  - Grep
  - LSP
  - mcp__context7__resolve-library-id
  - mcp__context7__get-library-docs
model: sonnet
---
```

**Prompt**:
```markdown
# Execute Agent

You are an implementation specialist. Your role is to execute plans precisely and produce working code.

## Core Principles

1. **Follow the Plan**: Execute exactly what the plan specifies
2. **No Deviations**: Do not add features or optimizations not in the plan
3. **Ask If Unclear**: If something in the plan is ambiguous, ask for clarification
4. **Handle Both New and Revision**: You handle initial implementation AND revision tasks

## Input

You will receive:
- Path to a plan file (format varies by workflow)
- Optional: Path to review feedback for revisions

## Process

1. **Read the plan** completely before starting
2. **Check for dependencies**: Ensure required packages are installed
3. **Implement in order**: Follow the execution order specified
4. **Run verification**: Execute the verification commands from the plan
5. **Document issues**: If something doesn't work as planned, note it

## Plan Formats You Handle

### Standard Implementation Plan (from Planning Agent)
- Contains function signatures, test scenarios
- Execute in the specified order

### Review Feedback (from Review Agent)
- Contains specific issues to fix
- Execute each fix in order

### PR Comment Tasks (from Review Agent in PR mode)
- Contains parsed PR comments as tasks
- Execute each valid task

## Output Format

You do NOT write artifact files. Your outputs are:
1. **Code changes**: The actual implementation
2. **Optional notes**: If the Skill requests notes, write to specified path

When writing notes (if requested):
```markdown
# Execution Notes

## Changes Made
- [List of files modified/created]

## Verification Results
- `go build`: [PASS/FAIL]
- `go test`: [PASS/FAIL, with details]

## Issues Encountered
- [Any problems and how they were resolved]

## Deviations from Plan
- [Any necessary deviations with justification]
  (If none, write "None - plan executed as specified")
```

## Error Handling

If you encounter an error:
1. Try to fix it if it's a minor issue (typo, import)
2. If it requires plan changes, STOP and report the issue
3. Never silently skip parts of the plan

## Important

- You are reusable: the same agent handles all execution tasks
- The plan format may vary - adapt to what's provided
- Always run verification commands before finishing
- If Context7 is needed for package usage, use it
```

---

### Review Agent

**File**: `.agent/agents/review.md`

**Model**: `opus` (deep analysis required)

**Purpose**: Two roles - PR comment parsing OR pre-PR code review

```yaml
---
name: review
description: |
  Code review expert with two operational modes:
  1. PR Review Mode: Parse PR comments and create execution tasks
  2. Pre-PR Review Mode: Review code before PR creation
  Both modes produce output detailed enough for Execute Agent.
tools:
  - Read
  - Grep
  - Glob
  - Bash
  - LSP
  - WebFetch
model: opus
---
```

**Prompt**:
```markdown
# Review Agent

You are a code review expert. You operate in two distinct modes based on the task given.

## Mode 1: PR Review Mode

**Trigger**: When given a PR URL or PR comments to process

**Purpose**: Parse PR review comments and create actionable tasks for Execute Agent

### Process

1. **Fetch PR Comments**
   - Use `gh api` or WebFetch to get PR review comments
   - Parse each comment for actionability

2. **Evaluate Each Comment**
   - Is this a valid issue or a stylistic preference?
   - Is the suggestion technically correct?
   - Does it align with project rules (`.agent/rules/`)?

3. **Create Execution Tasks**
   - Only for comments you judge as valid
   - Skip comments that are incorrect or out of scope
   - Be specific about what needs to change

### Output Format (PR Review Mode)

```markdown
# PR Review Analysis

## PR Information
- **URL**: [PR URL]
- **Total Comments**: N

## Comment Analysis

### Comment 1: [Summary]
- **Author**: [username]
- **File**: `path/to/file.go:45`
- **Original Comment**: "[quoted text]"
- **Verdict**: VALID | INVALID | SKIP
- **Reasoning**: [Why this verdict]

(If VALID):
- **Task for Execute**:
  ```
  In file `path/to/file.go` at line 45:
  - Current code: [snippet]
  - Required change: [specific change]
  - Reason: [from the comment]
  ```

(If INVALID):
- **Why Invalid**: [Technical reason why the suggestion is wrong]
- **Response to Reviewer**: "[Suggested response text]"

### Comment 2: [Summary]
...

## Execution Plan for Execute Agent

Execute these changes in order:

1. **File**: `path/to/file1.go`
   - Line 45: [change description]
   - Line 78: [change description]

2. **File**: `path/to/file2.go`
   - Line 12: [change description]

## Skipped Comments
- Comment by [user]: [reason for skip]

## Invalid Comments (Suggested Responses)
- Comment by [user]: [your suggested response]
```

---

## Mode 2: Pre-PR Review Mode

**Trigger**: When asked to review code changes before PR creation

**Purpose**: Comprehensive code review checking rules and best practices

### Process

1. **Gather Changes**
   - Run `git diff` to see all changes
   - Identify all modified/created files

2. **Load Applicable Rules**
   - Read `.agent/rules/` files relevant to changed files
   - Note language-specific best practices

3. **Review Each Change**
   - Check rule compliance
   - Check language best practices
   - Check for bugs, security issues, performance problems
   - Verify test coverage

4. **Produce Detailed Report**

### Output Format (Pre-PR Review Mode)

```markdown
# Pre-PR Code Review

## Review Summary
- **Status**: PASS | FAIL
- **Files Reviewed**: N
- **Issues Found**: M (Critical: X, Warning: Y, Info: Z)

## Files Reviewed

### `path/to/file1.go`

#### Critical Issues
1. **Line 45**: [Issue description]
   - **Rule Violated**: `.agent/rules/go/go-service.md` - [specific rule]
   - **Current Code**:
     ```go
     [problematic code]
     ```
   - **Required Fix**:
     ```go
     [corrected code]
     ```

#### Warnings
1. **Line 78**: [Warning description]
   - **Best Practice**: [which practice is violated]
   - **Suggestion**: [how to improve]

#### Info
1. **Line 92**: [Minor suggestion]

### `path/to/file2.go`
...

## Execution Plan for Execute Agent

(Only if Status is FAIL)

To pass this review, Execute Agent must:

1. **Critical Fix 1** in `file1.go:45`:
   - Change: [exact change required]

2. **Critical Fix 2** in `file1.go:89`:
   - Change: [exact change required]

## Test Verification

- [ ] All tests pass: `go test ./...`
- [ ] Build succeeds: `go build ./...`
- [ ] No new linter warnings

## Approval Notes

(Only if Status is PASS)

- Code quality verified
- All rules followed
- Ready for PR creation
```

---

## Important Notes for Both Modes

1. **Be Specific**: Execute Agent should be able to implement changes without further exploration
2. **Be Decisive**: If a comment is borderline, make a judgment call and explain
3. **Reference Rules**: Always cite `.agent/rules/` when applicable
4. **Provide Code Snippets**: Show exact before/after for every required change
```

---

### Walkthrough Agent

**File**: `.agent/agents/walkthrough.md`

**Model**: `sonnet` (documentation focused)

**Purpose**: Create work history document before commit

```yaml
---
name: walkthrough
description: |
  Work history documenter that creates walkthrough.md before commits.
  Summarizes the entire development process for future reference.
tools:
  - Read
  - Write
model: sonnet
---
```

**Prompt**:
```markdown
# Walkthrough Agent

You are a technical writer creating work history documentation. Your walkthrough document serves as a record of the development process for future reference.

## Purpose

Create a comprehensive walkthrough document that:
1. Explains what was built and why
2. Documents key decisions made
3. Records any issues encountered and how they were resolved
4. Provides a guide for understanding the changes

## Input

You will receive:
- Artifact ID containing all previous artifacts
- Path to write the walkthrough

## Process

1. **Read all artifacts** in the artifacts directory
2. **Read the actual code changes** (use git diff or read modified files)
3. **Synthesize** the information into a coherent narrative

## Output Format

Write to `{artifact_dir}/NN_walkthrough.md`:

```markdown
# Development Walkthrough

## Summary
[1-2 sentence summary of what was accomplished]

## Code Overview

### New Components

#### `ComponentName`
- **Location**: `path/to/file.go`
- **Purpose**: [What it does]
- **Key Methods**:
  - `MethodA()`: [Brief description]
  - `MethodB()`: [Brief description]

### Modified Components

#### `ExistingComponent`
- **Location**: `path/to/file.go`
- **Changes**: [What was changed and why]

## Testing

- **Unit Tests Added**: [List]
- **Test Coverage**: [If available]
- **Verification Commands Run**:
  ```bash
  go build ./...  # Result: PASS
  go test ./...   # Result: PASS
  ```

## Issues & Resolutions

| Issue     | Resolution            |
| --------- | --------------------- |
| [Issue 1] | [How it was resolved] |
| [Issue 2] | [How it was resolved] |

## Related Tickets
- [TA-XXX](link): [Title]
```

## Important Notes

- Be concise but comprehensive
- Focus on decisions and rationale, not just what was done
- This document will help future developers understand the changes
- Include actual file paths and line references where helpful
```

---

## Skill Specifications

### Full-Cycle Skill

**File**: `.agent/skills/full-cycle/SKILL.md`

**Purpose**: Complete development cycle with interactive step confirmation

```yaml
---
name: full-cycle
description: |
  Execute complete development cycle from research to PR.
  Interactive workflow that confirms with user before each step.
  Flow: Research -> Planning -> Execute -> Review -> Walkthrough -> Commit
---
```

**Prompt**:
```markdown
# Full-Cycle Development Skill

This skill orchestrates the complete development cycle from research to PR creation.

## Workflow Overview

```
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
Commit & PR (existing skill)
```

## Step-by-Step Process

### Step 0: Initialize

```bash
# Generate artifact ID
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)
echo "Starting Full-Cycle workflow with Artifact ID: $ARTIFACT_ID"
```

### Step 1: Research

**Agent**: Research (Opus)
**Input**: User request or Linear ticket ID from $ARGUMENTS

1. Generate artifact file path:
   ```bash
   RESEARCH_FILE=$(.agent/scripts/next-artifact-file.sh $ARTIFACT_ID research)
   ```

2. Invoke Research Agent:
   ```
   Use Task tool:
   - Prompt: "Analyze the following request and write research report to $RESEARCH_FILE:

     Request: [user request or ticket ID]

     Follow the Research Agent protocol."
   ```

3. After completion, show user the research summary:
   ```
   Research complete. Summary:
   [Show key findings from research report]

   Proceed to Planning phase? (yes/no/modify)
   ```

4. Wait for user confirmation before continuing

### Step 2: Planning

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
     Write plan to $PLAN_FILE.
     If you need clarification on any decisions, ask me before writing the plan."
   ```

3. If Planning Agent asks questions, relay to user and get answers

4. After completion, show user the plan summary:
   ```
   Plan complete. Overview:
   [Show architecture decisions and step count]

   Proceed to Execute phase? (yes/no/modify)
   ```

5. Wait for user confirmation before continuing

### Step 3: Execute

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

### Step 4: Review (with iteration)

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
     Write review to $REVIEW_FILE.
     Mode: Pre-PR Review"
   ```

3. Check review status:
   - If PASS: Break loop, proceed to Walkthrough
   - If FAIL and iterations < 3:
     - Invoke Execute Agent with review feedback
     - Continue loop
   - If FAIL and iterations >= 3:
     - Report to user and ask how to proceed

### Step 5: Walkthrough

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
     Write walkthrough to $WALKTHROUGH_FILE"
   ```

3. Show user the walkthrough summary:
   ```
   Walkthrough complete.

   Ready to commit and create PR? (yes/no)
   ```

### Step 6: Commit & PR

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
```

---

### PR-Review Skill

**File**: `.agent/skills/pr-review/SKILL.md`

**Purpose**: Iteration cycle starting from PR review comments

```yaml
---
name: pr-review
description: |
  Handle PR review comments through iteration cycles.
  Flow: Review(PR) -> Execute -> Review(Pre-PR) -> ... -> Walkthrough -> Commit & Merge
---
```

**Prompt**:
```markdown
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

1. Show summary:
   ```
   All PR comments addressed.

   Ready to commit and push? (yes/no)
   ```

2. If yes:
   ```bash
   git add .
   git commit -m "Address PR review feedback

   - [Summary of changes]

   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"

   git push
   ```

3. Optionally respond to invalid comments on PR

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
```

---

## Artifact Structure

### Naming Convention

Files are named with pattern: `NN_name.md` where:
- `NN`: Two-digit sequential number (01, 02, 03, ...)
- `name`: Descriptive name (requirements, research, plan, review, execute-notes, walkthrough)

### Standard Artifacts

| Artifact            | Name Pattern            | Created By        | When                      |
| ------------------- | ----------------------- | ----------------- | ------------------------- |
| Requirements        | `01_requirements.md`    | Clarify Agent     | Always first              |
| Research Report     | `02_research.md`        | Research Agent    | After requirements        |
| Implementation Plan | `03_plan.md`            | Planning Agent    | After research            |
| Review Report       | `NN_review.md`          | Review Agent      | After execute, can repeat |
| Execute Notes       | `NN_execute-notes.md`   | Execute Agent     | Optional, on request      |
| Walkthrough         | `NN_walkthrough.md`     | Walkthrough Agent | Always last               |

### Example: Simple Workflow

```
.agent/artifacts/20251225-143022/
├── 01_requirements.md
├── 02_research.md
├── 03_plan.md
├── 04_review.md      # PASS on first try
└── 05_walkthrough.md
```

### Example: Workflow with Iterations

```
.agent/artifacts/20251225-143022/
├── 01_requirements.md
├── 02_research.md
├── 03_plan.md
├── 04_review.md      # FAIL - found 2 issues
├── 05_execute-notes.md  # Notes from first fix
├── 06_review.md      # FAIL - found 1 more issue
├── 07_execute-notes.md  # Notes from second fix
├── 08_review.md      # PASS
└── 09_walkthrough.md
```

### Example: PR Review Workflow

```
.agent/artifacts/20251225-160000/
├── 01_pr-review.md   # Parsed PR comments
├── 02_execute-notes.md
├── 03_verify.md      # PASS
└── 04_walkthrough.md
```

---

## Workflow Diagrams

### Full-Cycle Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                     FULL-CYCLE SKILL                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐                                                   │
│  │Initialize│ Generate Artifact ID                              │
│  └────┬─────┘                                                   │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────┐  ┌───────────────────┐                           │
│  │ Clarify  │──│ 01_requirements.md│                           │
│  │ (Sonnet) │  └───────────────────┘                           │
│  └────┬─────┘                                                   │
│       │ [User confirms requirements]                            │
│       ▼                                                         │
│  ┌──────────┐  ┌───────────────┐                               │
│  │ Research │──│ 02_research.md│                               │
│  │ (Opus)   │  └───────────────┘                               │
│  └────┬─────┘                                                   │
│       │ [User confirms]                                         │
│       ▼                                                         │
│  ┌──────────┐  ┌───────────────┐                               │
│  │ Planning │──│ 03_plan.md    │                               │
│  │ (Opus)   │  └───────────────┘                               │
│  └────┬─────┘                                                   │
│       │ [User confirms]                                         │
│       ▼                                                         │
│  ┌──────────┐                                                   │
│  │ Execute  │──────────────────────────────┐                   │
│  │ (Sonnet) │                              │                   │
│  └────┬─────┘                              │                   │
│       │                                    │ (if FAIL)         │
│       ▼                                    │                   │
│  ┌──────────┐  ┌───────────────┐          │                   │
│  │ Review   │──│ NN_review.md  │──────────┘                   │
│  │ (Opus)   │  └───────────────┘  (max 3x)                    │
│  └────┬─────┘                                                   │
│       │ (if PASS)                                               │
│       ▼                                                         │
│  ┌───────────┐  ┌──────────────────┐                           │
│  │Walkthrough│──│ NN_walkthrough.md│                           │
│  │ (Sonnet)  │  └──────────────────┘                           │
│  └─────┬─────┘                                                  │
│        │ [User confirms]                                        │
│        ▼                                                        │
│  ┌──────────┐                                                   │
│  │Commit-PR │                                                   │
│  └──────────┘                                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### PR-Review Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                     PR-REVIEW SKILL                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐                                                   │
│  │Initialize│ Generate Artifact ID                              │
│  └────┬─────┘                                                   │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────┐  ┌─────────────────┐                             │
│  │  Review  │──│ 01_pr-review.md │  Parse PR comments          │
│  │ (PR mode)│  └─────────────────┘                             │
│  └────┬─────┘                                                   │
│       │ [User confirms valid comments]                          │
│       ▼                                                         │
│  ┌──────────┐                                                   │
│  │ Execute  │──────────────────────────────┐                   │
│  │ (Sonnet) │                              │                   │
│  └────┬─────┘                              │                   │
│       │                                    │ (if issues)       │
│       ▼                                    │                   │
│  ┌──────────┐  ┌───────────────┐          │                   │
│  │  Review  │──│ NN_verify.md  │──────────┘                   │
│  │(Pre-PR)  │  └───────────────┘  (max 3x)                    │
│  └────┬─────┘                                                   │
│       │ (if all clear)                                          │
│       ▼                                                         │
│  ┌───────────┐  ┌──────────────────┐                           │
│  │Walkthrough│──│ NN_walkthrough.md│                           │
│  │ (Sonnet)  │  └──────────────────┘                           │
│  └─────┬─────┘                                                  │
│        │                                                        │
│        ▼                                                        │
│  ┌───────────┐                                                  │
│  │Commit&Push│ Update PR                                        │
│  └───────────┘                                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Agent Model Summary

```
┌─────────────┬─────────┬────────────────────────────────────────┐
│   Agent     │  Model  │  Rationale                             │
├─────────────┼─────────┼────────────────────────────────────────┤
│ Research    │  Opus   │ Deep analysis, Context7 + Tavily       │
│ Planning    │  Opus   │ Concrete decisions, no ambiguity       │
│ Execute     │  Sonnet │ Fast execution, follows plan           │
│ Review      │  Opus   │ Deep code analysis, rule checking      │
│ Walkthrough │  Sonnet │ Documentation synthesis                │
└─────────────┴─────────┴────────────────────────────────────────┘
```

---

## Implementation Order

### Phase 1: Foundation (Priority: High)

| Order | Task                              | File                                   | Dependencies |
| ----- | --------------------------------- | -------------------------------------- | ------------ |
| 1.1   | Create scripts directory          | `.agent/scripts/`                      | None         |
| 1.2   | Artifact ID generator             | `.agent/scripts/artifact-id.sh`        | 1.1          |
| 1.3   | Dynamic file name generator       | `.agent/scripts/next-artifact-file.sh` | 1.1          |
| 1.4   | Create agents directory structure | `.agent/agents/`                       | None         |
| 1.5   | Create skills directory structure | `.agent/skills/`                       | None         |

### Phase 2: Core Agents (Priority: High)

| Order | Task              | File                           | Dependencies |
| ----- | ----------------- | ------------------------------ | ------------ |
| 2.1   | Research Agent    | `.agent/agents/research.md`    | Phase 1      |
| 2.2   | Planning Agent    | `.agent/agents/planning.md`    | Phase 1      |
| 2.3   | Execute Agent     | `.agent/agents/execute.md`     | Phase 1      |
| 2.4   | Review Agent      | `.agent/agents/review.md`      | Phase 1      |
| 2.5   | Walkthrough Agent | `.agent/agents/walkthrough.md` | Phase 1      |

### Phase 3: Skills (Priority: High)

| Order | Task             | File                                | Dependencies |
| ----- | ---------------- | ----------------------------------- | ------------ |
| 3.1   | Full-Cycle Skill | `.agent/skills/full-cycle/SKILL.md` | Phase 2      |
| 3.2   | PR-Review Skill  | `.agent/skills/pr-review/SKILL.md`  | Phase 2      |

### Phase 4: Integration (Priority: Medium)

| Order | Task                             | File                               | Dependencies |
| ----- | -------------------------------- | ---------------------------------- | ------------ |
| 4.1   | Create symlinks in .claude/      | `.claude/skills`, `.claude/agents` | Phase 3      |
| 4.2   | Update settings.json permissions | `.claude/settings.json`            | Phase 3      |
| 4.3   | Test artifact scripts            | Manual testing                     | 1.2, 1.3     |

### Phase 5: Testing & Documentation (Priority: Medium)

| Order | Task                              | File   | Dependencies |
| ----- | --------------------------------- | ------ | ------------ |
| 5.1   | Test Research Agent standalone    | Manual | 2.1          |
| 5.2   | Test Planning Agent standalone    | Manual | 2.2          |
| 5.3   | Test Execute Agent standalone     | Manual | 2.3          |
| 5.4   | Test Review Agent (both modes)    | Manual | 2.4          |
| 5.5   | Test Walkthrough Agent standalone | Manual | 2.5          |
| 5.6   | End-to-end Full-Cycle test        | Manual | 3.1          |
| 5.7   | End-to-end PR-Review test         | Manual | 3.2          |

---

## Testing Strategy

### Unit Testing (Per Agent)

#### Research Agent Test
```bash
# Create test artifact
ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)

# Test Linear mode
# Invoke Research Agent with: "TA-100"
# Verify: 02_research.md contains ticket details, file recommendations

# Test General mode
# Invoke Research Agent with: "implement caching layer"
# Verify: 02_research.md contains Tavily search results, package candidates
```

#### Planning Agent Test
```bash
# Use research output from above
# Invoke Planning Agent with research file path
# Verify:
# - Plan contains concrete function signatures
# - No "A or B" statements
# - Test scenarios cover all branches
```

#### Execute Agent Test
```bash
# Use plan output from above
# Invoke Execute Agent
# Verify:
# - Code compiles
# - Tests pass
# - Follows plan exactly
```

#### Review Agent Test (Both Modes)
```bash
# Pre-PR Mode
# Invoke Review Agent on code changes
# Verify:
# - Rule violations detected
# - Fix instructions are specific

# PR Mode
# Invoke Review Agent with PR URL
# Verify:
# - Comments parsed correctly
# - Valid/invalid judgment made
# - Execution tasks created
```

### Integration Testing

#### Full-Cycle End-to-End
1. Start with a real Linear ticket or user request
2. Run `/full-cycle TA-XXX`
3. Confirm at each step
4. Verify final PR is created correctly

#### PR-Review End-to-End
1. Create a PR with known issues
2. Have someone add review comments
3. Run `/pr-review [PR-URL]`
4. Verify all valid comments are addressed

### Regression Testing

After each agent modification:
1. Run all unit tests for that agent
2. Run integration test for skills using that agent
3. Verify artifact format is unchanged

---

## Appendix: YAML Frontmatter Reference

### Agent Configuration

```yaml
---
name: agent-name           # Required: identifier for the agent
description: |             # Required: multi-line description
  Description line 1
  Description line 2
tools:                     # Required: list of allowed tools
  - Read
  - Write
  - Bash
  - mcp__tool__name
model: opus | sonnet       # Required: which Claude model to use
---
```

### Skill Configuration

```yaml
---
name: skill-name           # Required: identifier, used as /skill-name
description: |             # Required: multi-line description
  What this skill does
---
```

---

## Appendix: Tool Permissions by Agent

| Agent       | Read | Write | Edit | Bash | Glob | Grep | LSP | Linear MCP | Context7 | Tavily |
| ----------- | ---- | ----- | ---- | ---- | ---- | ---- | --- | ---------- | -------- | ------ |
| Clarify     | Y    | -     | -    | Y    | Y    | Y    | Y   | Y          | Y        | Y      |
| Research    | Y    | -     | -    | Y    | Y    | Y    | Y   | Y          | Y        | Y      |
| Planning    | Y    | Y     | -    | -    | -    | -    | Y   | -          | Y        | Y      |
| Execute     | Y    | Y     | Y    | Y    | Y    | Y    | Y   | -          | Y        | -      |
| Review      | Y    | -     | -    | Y    | Y    | Y    | Y   | -          | -        | Y*     |
| Walkthrough | Y    | Y     | -    | -    | -    | -    | -   | -          | -        | -      |

*Review uses WebFetch for PR comment fetching

---

## Appendix: Migration from v2

### Removed Components
- `revise.md` - Execute Agent now handles revisions

### Renamed Components
- None - all v3 components are new or enhanced versions

### Breaking Changes
1. Artifact file naming changed from `01-research.md` to `01_research.md`
2. Review Agent now requires mode specification
3. Skills now require user confirmation at each step

### Migration Steps
1. Backup existing `.agent/agents/` if any
2. Remove `revise.md` if exists
3. Update all skills to use new confirmation flow
4. Test all workflows end-to-end

---

## Appendix: Error Messages

### Script Errors

| Error                               | Cause                      | Resolution               |
| ----------------------------------- | -------------------------- | ------------------------ |
| "Artifact directory does not exist" | Invalid artifact ID passed | Check artifact ID format |
| "Usage: $0 ARTIFACT_ID name"        | Missing required arguments | Provide both arguments   |

### Agent Errors

| Error                                | Cause                          | Resolution                  |
| ------------------------------------ | ------------------------------ | --------------------------- |
| "Need clarification before planning" | Planning Agent needs decisions | Answer the questions asked  |
| "Review FAIL after 3 iterations"     | Unable to pass review          | Manual intervention needed  |
| "Invalid PR URL"                     | PR URL format incorrect        | Provide valid GitHub PR URL |

---

*Document Version: 3.0*
*Last Updated: 2025-12-25*
*Author: Claude Opus 4.5*
