# Create Git Worktree from Linear Ticket

Create a git worktree for a Linear ticket.

## Instructions

1. **Get Project Name**:
   - First, read `.claude/memory/linear` file for cached project name
   - If not found, ask user for project name and save it to `.claude/memory/linear`

2. **Find the Ticket**:
   - If `$ARGUMENTS` is provided: Search for ticket by ID or title
   - If `$ARGUMENTS` is empty: Find the highest priority ticket (Urgent > High > Medium > Low) that is in "Backlog" or "Todo" status

3. **Choose Worktree Path**:
   - Default path: `../worktree/{gitBranchName}` (outside current repo to avoid Claude Code loading parent rules)
   - Ask user to confirm or provide custom path using AskUserQuestion
   - Preserve `/` in branch name as nested directories

4. **Create Git Worktree**:
   - Directory: User-chosen path
   - Branch: Use the exact `gitBranchName` from Linear ticket
   - Base: `main` branch

5. **Copy Local Settings and Environment Files**:
   - If `.claude/settings.local.json` exists, copy it to the new worktree's `.claude/` directory
   - Find all `.env` and `.env*` files in the current repository (excluding `.env.example` files)
   - Copy each found environment file to the same relative path in the new worktree, preserving directory structure

6. **Update Linear Ticket**:
   - Change status to "In Progress"

7. **Report Result**:
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
