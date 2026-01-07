# Linear Task Document

This document defines how to gather requirements from a Linear issue.

## Input

- `ISSUE_ID` - Linear Issue ID (e.g., `ABC-123`, `TA-456`)

## Process

1. Use `mcp__linear__get_issue` to fetch ticket details
2. Use `mcp__linear__list_comments` to get additional context
3. Extract the following information:
   - Title
   - Description
   - Acceptance criteria (if defined)
   - Labels
   - Assignee
   - Any attached documents or links

## Output

Requirements gathered from the Linear issue, ready for the planning process.
