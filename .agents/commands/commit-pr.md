# Commit and Create Pull Request

Commit all changes and create a pull request for the current branch.

## Instructions

1. **Identify Linear Ticket**:
   - Extract ticket ID from branch name (e.g., `feature/ta-76-...` → `TA-76`)
   - Fetch ticket details from Linear (title, description)

2. **Analyze Changes**:
   - Run `git status` to see all changes
   - Run `git diff` to understand what changed
   - Check recent commit messages for style reference

3. **Create Commit**:
   - Stage all relevant changes (`git add`)
   - Generate commit message following conventional commits format:
     - Format: `feat|fix|docs|refactor|test|chore: description (TICKET-ID)`
     - Include summary of changes in commit body
   - Commit with co-author signature

4. **Push and Create PR**:
   - Push branch to remote with `-u` flag if needed
   - Create PR using `gh pr create` with:
     - Title: `[TICKET-ID] Ticket Title`
     - Body: Summary from ticket description + changes made
     - Link to Linear ticket

5. **Update Linear Ticket**:
   - Add PR URL as a link to the ticket
   - Optionally update status to "In Review"

6. **Report Result**:
   - Commit hash and message
   - PR URL
   - Linear ticket update confirmation

## Arguments

- `$ARGUMENTS`: Optional commit message override or additional context

## Example Usage

```bash
# Auto-generate commit message and PR
/commit-pr

# With custom commit message
/commit-pr "fix authentication bug"

# With additional context
/commit-pr "ready for review, needs testing on staging"
```

## Commit Message Format

```
feat: short description (TA-XX)

- Detailed change 1
- Detailed change 2

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

## PR Body Format

```markdown
## Summary
[Brief description of changes]

## Linear Ticket
[TA-XX](linear-ticket-url): Ticket Title

## Changes
- Change 1
- Change 2

## Test Plan
- [ ] Build passes
- [ ] Manual testing completed

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```