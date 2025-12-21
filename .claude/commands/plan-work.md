# Plan Work from Current Branch

Extract Linear ticket from current branch and create a work plan using Claude Code's planning mode.

## Instructions

1. **Extract Ticket ID from Branch**:
   - Parse current branch name (e.g., `feature/ta-92-...` → `TA-92`)
   - Branch format: `{type}/{ticket-id}-{description}`

2. **Fetch Linear Ticket**:
   - Get ticket details: title, description, priority, status
   - Get acceptance criteria if available
   - Get sub-issues if any

3. **Analyze Codebase Context**:
   - Read relevant `.claude/rules/` files based on ticket content
   - Understand current project structure
   - Identify architectural patterns to follow

4. **Enter Planning Mode**:
   - Use `EnterPlanMode` tool to start planning
   - Think deeply about the implementation approach (use ultrathink/extended thinking)
   - Write detailed implementation plan
   - Get user approval before proceeding

## Arguments

- `$ARGUMENTS`: Optional additional context or focus area

## Example Usage

```bash
# Plan work from current branch
/plan-work

# Plan with specific focus
/plan-work "focus on domain models first"
```

## Notes

- This command will enter planning mode after gathering context
- User must approve the plan before implementation begins
- Use extended thinking for complex architectural decisions
