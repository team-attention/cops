# Create Git Worktree from Linear Ticket

Create a git worktree for a Linear ticket.

## Instructions

1. **Get Project Name**:
   - First, read `.claude/memory/linear` file for cached project name
   - If not found, ask user for project name and save it to `.claude/memory/linear`

2. **Find the Ticket**:
   - If `$ARGUMENTS` is provided: Search for ticket by ID or title
   - If `$ARGUMENTS` is empty: Find the highest priority ticket (Urgent > High > Medium > Low) that is in "Backlog" or "Todo" status

3. **Create Git Worktree**:
   - Directory: `.worktree/{gitBranchName}` (preserving `/` as nested directories)
   - Branch: Use the exact `gitBranchName` from Linear ticket
   - Base: `main` branch

4. **Copy Local Settings**:
   - If `.claude/settings.local.json` exists, copy it to the new worktree's `.claude/` directory

5. **Update Linear Ticket**:
   - Change status to "In Progress"

6. **Report Result**:
   - Ticket ID, title, and URL
   - Worktree path created
   - Branch name
   - Confirmation of status update

## Example Usage

```bash
# Use highest priority ticket
/worktree

# Use specific ticket by ID
/worktree TA-76

# Use specific ticket by title keyword
/worktree domain models
```
