---
name: clarify
description: |
  Clarifies user requirements by loading appropriate task and output documents based on input types.
  Supports user requests or Linear issues as task source, and artifact directories or Linear projects as output destination.
model: claude-opus-4-5
permissionMode: acceptEdits
---

# Clarify Skill

Gathers and clarifies requirements from different sources (user request or Project Management System(PMS)) and determines output destination (local artifact or PMS).

## Parameters

### Task Source (OneOf, Required)

Provide one of the following to specify where requirements come from:

- `REQUEST` - User's requirement text (free-form description)
- `ISSUE_ID` - Linear Issue ID (e.g., `ABC-123`)

### Output Destination (OneOf, Required)

Provide one of the following to specify where clarified requirements are saved:

- `ARTIFACT_DIR_PATH` - Artifact directory path (e.g., `.agent/artifacts/20260105-120000`)
- `PROJECT_ID` - Linear Project ID or name

### Optional

- `AUTO_ACCEPT` - If set to `true`, skip user review at the end. Defaults to `false`.

## Usage Examples

```bash
# User request → Artifact output
skill: clarify
args: REQUEST="Add user authentication feature" ARTIFACT_DIR_PATH=.agent/artifacts/20260105-120000

# Linear issue → Artifact output
skill: clarify
args: ISSUE_ID=ABC-123 ARTIFACT_DIR_PATH=.agent/artifacts/20260105-120000

# User request → Linear project output
skill: clarify
args: REQUEST="Add user authentication feature" PROJECT_ID=my-project

# Linear issue → Linear project output (with auto-accept)
skill: clarify
args: ISSUE_ID=ABC-123 PROJECT_ID=my-project AUTO_ACCEPT=true
```

## Process

### 1. Gather Initial Request

- If `ISSUE_ID` is provided → Read [Linear Task Document](./docs/task/linear-task.md)
- If `REQUEST` is provided → Use the request text as initial requirements

### 2. Analyze Request

Check if all necessary information is present:
- Clear task description
- Acceptance criteria (what defines "done")
- Scope boundaries (what's included/excluded)
- Any constraints or dependencies

### 3. Identify Gaps and Ask Questions

If information is missing or unclear, prepare structured questions:

- **What**: What exactly needs to be built/changed?
- **Why**: What problem does this solve?
- **How**: Are there specific implementation preferences?
- **Done**: How will we know it's complete? (Acceptance Criteria)
- **Scope**: What's in scope? What's out of scope?
- **Constraints**: Any technical constraints, deadlines, or dependencies?

**ALWAYS confirm with user**, even if requirements seem complete:
- Show summary of understood requirements
- Ask: "Is this understanding correct?"
- Ask: "Are there any additional requirements or constraints?"

### 4. Break Down into Tasks

Based on the clarified requirements, break down the work into individual tasks. Tasks represent work units that can be executed in parallel.

Consider:
- Dependencies between tasks
- Logical groupings of related work
- Optimal parallelization opportunities

For example, if the requirement involves implementing an API server and its client, you might define tasks as: (1) API interface definition (prerequisite), then (2) client implementation and (3) server implementation (parallelizable).

### 5. Write to Temporary Files

Use the `mktemp` skill to create temporary files and write the requirements following the [Output Format](#output-format).

1. Create temporary file(s) using `mktemp` skill:
   ```
   skill: mktemp
   args: task1 task2
   ```
   This creates files like `.agent/tmp/task1.xxxxxxxx` and `.agent/tmp/task2.yyyyyyyy`

2. Write each task's requirements document to a separate file
3. Present the files to the user for review
4. Revise the file contents based on user feedback until approved

> If `AUTO_ACCEPT` is `True`, skip user review and proceed to the next step.

### 6. Create Final Output

Once the review is approved, create the final output.

- If `ARTIFACT_DIR_PATH` is provided → Read [Artifact Output](./docs/output/artifact-output.md) and follow its instructions
- If `PROJECT_ID` is provided → Read [Linear Output](./docs/output/linear-output.md) and follow its instructions

## Output Format

Each task document must include YAML frontmatter followed by the content sections.

### YAML Frontmatter

```yaml
---
name: Task name
blockedBy:
  - Prerequisite task name
---
```

- `name`: Task identifier (used for issue title and dependency references)
- `blockedBy`: List of task names this task depends on (empty array `[]` if no dependencies)

### Task Summary

Provide a 2-3 sentence summary of what needs to be implemented. This should clearly explain the goal and context.

### Acceptance Criteria

List specific, testable requirements as checklist items. Each criterion should be verifiable and define what "done" means.

### Scope

Clearly define boundaries to prevent scope creep:

#### In Scope
- What will be built in this iteration
- What features will be included

#### Out of Scope
- What will NOT be built in this iteration
- What features are explicitly excluded

### Constraints

Document any technical, business, or timeline constraints that affect the implementation.

### Additional Context

Include any other relevant information:
- Prerequisite tasks that must be completed first
- Links to related documentation or tickets
- Dependencies on other work
- Background information

### Questions Resolved

Record all clarifying questions and user answers in a table format. This documents decisions made during clarification.

## Output Example

```markdown
---
name: User authentication
blockedBy: []
---

# Task Summary

[Provide 2-3 sentence summary explaining what needs to be implemented and the problem it solves]

# Acceptance Criteria

- [ ] [Specific, testable criterion 1]
- [ ] [Specific, testable criterion 2]
- [ ] [Specific, testable criterion 3]

# Scope

## In Scope
- [Feature or component to be built]
- [Another feature to be included]

## Out of Scope
- [Feature explicitly excluded from this iteration]
- [Another out-of-scope item]

# Constraints
- [Technical constraint if applicable]
- [Business constraint if applicable]
- [Timeline constraint if applicable]

# Additional Context
- [Link to related documentation]
- [Dependencies on other work]
- [Any other relevant background]

# Questions Resolved

| Question                      | Answer            |
| ----------------------------- | ----------------- |
| [Question asked to user]      | [User's answer]   |
| [Another clarifying question] | [User's response] |
```

## Output Quality Checklist

Before submitting the requirements document, verify:

- [ ] **Output format followed**: YAML frontmatter (`name`, `blockedBy`) and all content sections are present
- [ ] **User confirmation obtained**: Even if Linear ticket seems complete, show summary and ask for confirmation
- [ ] **Specific requirements**: Avoid vague requirements like "make it better" - drill down to specifics
- [ ] **Testable criteria**: Each acceptance criterion should be verifiable
- [ ] **Clear scope**: Explicitly state what's in and out of scope to prevent scope creep
- [ ] **Documented decisions**: Record all clarifying questions and answers in the "Questions Resolved" section

## Notice

### Document Acceptance Criteria

Work with user to define clear success criteria:
- Format as testable checklist items
- Each criterion should be specific and verifiable
