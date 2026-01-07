---
name: plan
description: |
  Creates detailed execution plans from requirements documents.
  Researches packages, explores codebase, makes architectural decisions, and produces
  concrete implementation specifications for the execution agent.
model: claude-opus-4-5
permissionMode: acceptEdits
---

# Plan Skill

Creates detailed execution plans from requirements. This skill takes clarified requirements (from a file or Linear issue) and produces concrete implementation plans that the Implementation Agent can execute.

## Parameters

### Task Source (OneOf, Required)

Provide one of the following to specify where requirements come from:

- `TASK_PATH` - Path to a task document (e.g., `.agent/artifacts/20260105/01_task.md`)
- `ISSUE_ID` - Linear Issue ID (e.g., `ABC-123`)

### Output Destination (Optional)

- `ARTIFACT_DIR_PATH` - Artifact directory path (e.g., `.agent/artifacts/20260105-120000`)

If not provided and `ISSUE_ID` is provided, the plan will be saved as a Document attached to the Issue in Linear.

> **Note**: If `TASK_PATH` is used without `ARTIFACT_DIR_PATH`, notify the user that an output destination is required and ask them to provide `ARTIFACT_DIR_PATH`.

### Optional

- `AUTO_ACCEPT` - If set to `true`, skip user review at the end. Defaults to `false`.

## Usage Examples

```bash
# Task file -> Artifact output
skill: plan
args: TASK_PATH=.agent/artifacts/20260105/01_task.md ARTIFACT_DIR_PATH=.agent/artifacts/20260105-120000

# Linear issue -> Artifact output
skill: plan
args: ISSUE_ID=TA-123 ARTIFACT_DIR_PATH=.agent/artifacts/20260105-120000

# Linear issue -> Linear Document (attached to issue)
skill: plan
args: ISSUE_ID=TA-123

# Linear issue -> Linear Document (with auto-accept)
skill: plan
args: ISSUE_ID=TA-123 AUTO_ACCEPT=true
```

## Process

### 1. Read Requirements

- If `TASK_PATH` is provided -> Read the file directly
- If `ISSUE_ID` is provided -> Read [Linear Task Document](./docs/task/linear-task.md)

Thoroughly understand:
- What needs to be implemented
- Acceptance criteria
- Scope boundaries
- Constraints

### 2. Research Packages

If external libraries are needed rather than building from scratch:

1. Use Context7 MCP to investigate candidate packages
2. Evaluate based on:
   - Maturity and active community
   - Compatibility with existing packages
3. If multiple candidates are viable, use `AskUserQuestion` to get user selection
4. **MUST select exactly one package** - do not leave as "A or B"

### 3. Explore Codebase

1. Read rules in `.claude/rules/` directory to understand project conventions
2. Explore existing code patterns related to the requirements
3. Identify files that need to be modified or created

### 4. Make Architectural Decisions

1. Determine algorithms and architecture for implementation
2. **MUST select exactly one approach** - do not write "A or B" in the plan
3. If multiple approaches are viable, use `AskUserQuestion` to get user selection

### 5. Create Implementation Plan

Based on investigations above, write a concrete execution plan following the [Output Format](#output-format).

### 6. Write to Temporary Files

1. Use the `mktemp` skill to create temporary file(s):
   ```
   skill: mktemp
   args: plan
   ```
2. Write the execution plan to the temporary file
3. Present to user for review
4. Revise based on feedback until approved

> If `AUTO_ACCEPT` is `true`, skip user review and proceed to the next step.

### 7. Create Final Output

Once review is approved, create the final output:

- If `ARTIFACT_DIR_PATH` is provided -> Read [Artifact Output](./docs/output/artifact-output.md) and follow its instructions
- Else if `ISSUE_ID` is provided -> Read [Linear Output](./docs/output/linear-output.md) and follow its instructions (attaches plan as Document to the Issue)

## Output Format

Each plan document must include YAML frontmatter followed by the content sections.

### YAML Frontmatter

```yaml
---
title: Plan title based on requirements
issueId: Issue-ID  # Optional
---
```

- `title`: Plan title (used for issue title and dependency references)
- `issueId` (Optional): Linear issue ID this plan is associated with. Required only when `ISSUE_ID` parameter was provided.

### Overview

Brief description of the implementation goal (what problem is being solved).

### Package Changes (Optional)

If external packages need to be added or removed:

| Action | Problem               | Package                   | Reason    |
| :----- | :-------------------- | :------------------------ | :-------- |
| Add    | [Problem Description] | `github.com/some/package` | [Reason ] |
| Remove | [Problem Description] | `github.com/old/package`  | [Reason ] |

### Implementation Steps

For each step:

#### Step N: Implement [Feature Name] Logic

**Files to Read**:
- `.agent/rules/[relevant_rule].md`: [Reason for reading]
- `path/to/related/file.go`: [Reason for reading]

##### `path/to/target_file.go`

**Description**:
Brief approach for this file (e.g., "Add validation logic and DB update function").

```go
const (
    // ConstantName description
    ConstantName = "value"
)

type StructName struct {
    // FieldName description
    FieldName string
}

// FunctionName performs [Specific Action] based on [Logic].
func FunctionName(ctx context.Context, input string) (ResultType, error) {
    // Implementation outline:
    // 1. Validate the input parameter.
    // 2. Retrieve data from external source.
    // 3. Iterate through the retrieved items:
    //    a. If item meets Condition A:
    //       - Perform Logic A (e.g. update state).
    //    b. Else if item meets Condition B:
    //       - Perform Logic B (e.g. skip or log).
    // 4. If critical failure occurred during iteration:
    //    - Return specific error.
    // 5. Update repository with final results.
    // 6. Return success.
}
```

**Test Scenarios**:

| Scenario               | Input | Expected Output | Branch Covered        |
| :--------------------- | :---- | :-------------- | :-------------------- |
| Valid input            | `...` | Success with X  | Happy path            |
| Invalid param1         | `...` | Error: "..."    | Validation branch     |
| External service fails | `...` | Error: "..."    | Error handling branch |

## Quality Checklist

Before submitting the plan, verify:

- [ ] Every function has a concrete signature (not "something like X")
- [ ] Detailed algorithm explanation is included as comments in the body of every function (no actual implementation code)
- [ ] Every function has test scenarios covering all branches
- [ ] No "or" statements leaving choices to Implementation Agent
- [ ] All packages are selected (not "candidate A or B")
- [ ] Execution order is clear and dependencies are explicit

## Notice

### Strict Decision Making

The execution plan must not contain vague content such as "Do A or B" or "needs investigation". All necessary details must be investigated and decided before creating the plan. If uncertain:

1. Research using available tools (Context7, codebase exploration)
2. Ask the user using `AskUserQuestion` if multiple valid approaches exist
3. Document the decision in the plan
