# Delete Git Worktree

Delete a git worktree and its associated branch.

## Instructions

1. **List Available Worktrees**:
   - Run `git worktree list` to get all worktrees
   - Exclude the main working directory (first entry)

2. **Select Worktree**:
   - If `$ARGUMENTS` provided: Match by branch name or path keyword
   - If `$ARGUMENTS` empty: Use AskUserQuestion to let user select from available worktrees

3. **Delete Worktree and Branch**:
   - `git worktree remove <path>` - Remove worktree
   - `git branch -D <branch>` - Delete the branch

4. **Report Result**:
   - Confirm deletion of worktree path
   - Confirm deletion of branch
   - Show remaining worktrees (if any)

## Example Usage

```bash
# Ask user to select
/delete-worktree

# Delete by branch keyword
/delete-worktree ta-76

# Delete by full branch name
/delete-worktree feature/ta-76-phase-1-initialize-go-module-and-project-structure
```
