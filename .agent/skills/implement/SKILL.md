---
name: implement
description: |
  Implementation skill that executes plans exactly as specified.
  Receives a plan document and implements code strictly according to the plan.
  Never adds features beyond the plan or makes arbitrary decisions.
model: claude-opus-4-5
permissionMode: acceptEdits
---

# Implement Skill

Executes implementation plans exactly as specified. This skill takes a plan document (from a file or Linear) and implements code strictly according to the plan without adding features or making arbitrary decisions.

## Parameters

### Plan Source (OneOf, Required)

Provide one of the following to specify where the plan comes from:

- `PLAN_PATH` - Path to a plan document file (e.g., `.agent/artifacts/20260105/02_plan.md`)
- `DOCUMENT_ID` - Linear Document ID or slug to fetch the plan from Linear

### Optional

- `AUTO_ACCEPT` - If set to `true`, skip user confirmation before starting implementation. Defaults to `false`.

## Usage Examples

```bash
# From local file
skill: implement
args: PLAN_PATH=.agent/artifacts/20260105/02_plan.md

# From Linear document
skill: implement
args: DOCUMENT_ID=abc123-def456

# With auto-accept (skip confirmation)
skill: implement
args: PLAN_PATH=.agent/artifacts/20260105/02_plan.md AUTO_ACCEPT=true
```

## Process

### 1. Read Plan Document

- If `PLAN_PATH` is provided -> Read the file directly
- If `DOCUMENT_ID` is provided -> Read [Linear Task Document](./docs/task/linear-task.md)

Thoroughly understand:
- What needs to be implemented
- Implementation steps and their order
- Function signatures and algorithms specified
- Success criteria

### 2. Confirm with User

Present a summary of what will be implemented:
- Number of files to create/modify
- Key changes to be made
- Dependencies to install (if any)

Ask: "Proceed with implementation?"

> If `AUTO_ACCEPT` is `true`, skip this step and proceed directly.

### 3. Install Dependencies

If the plan specifies external dependencies to install:
- Install each dependency using the appropriate package manager command
- Do NOT manually edit dependency files (go.mod, package.json, etc.)
- Verify installation success before proceeding

### 4. Read Prerequisite Files

If the plan lists files to read before implementation:
- Read all specified rule files
- Read all specified reference implementation files
- Understand the patterns and conventions before coding

### 5. Implement According to Plan

For each implementation step in the plan:
- Create or modify files exactly as specified
- Implement functions with the exact signatures provided
- Follow the algorithm comments/outline in the plan
- Do NOT add extra features, helpers, or optimizations not in the plan
- Do NOT create constants or variables not specified in the plan

**If instructions are unclear**: You MUST use `AskUserQuestion` to ask for clarification. Do not guess or make assumptions.

### 6. Verify Success Criteria

If the plan defines success criteria:
- Run specified build commands
- Run specified test commands
- Verify all criteria are met
- If any criterion fails, continue working to fix it

## Output Format

Report the implementation result:

**On Success**:
```
Implementation completed successfully.

Files created/modified:
- path/to/file1.go
- path/to/file2.go

Verification:
- go build ./... ✓
- go test ./... ✓
```

**On Problems**:
```
Implementation encountered issues:

Problem: [Description of the problem]
Location: [File and context where it occurred]
Cause: [Why following the plan caused this issue]
Suggestion: [Recommended fix or plan adjustment needed]
```

## Constraints

1. **No Feature Creep**: Implement ONLY what the plan specifies
2. **No Arbitrary Decisions**: If something is ambiguous, ask - do not decide
3. **No Over-Engineering**: Do not add error handling, validation, or optimizations beyond the plan
4. **No Extra Comments**: Do not add docstrings or comments not specified in the plan
5. **Strict Adherence**: The plan is your source of truth

## Quality Checklist

Before reporting completion:
- [ ] All files listed in plan are created/modified
- [ ] All function signatures match the plan exactly
- [ ] No extra features or code added beyond the plan
- [ ] All success criteria (if any) are verified
- [ ] Build and tests pass (if specified)
