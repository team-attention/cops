# Linear Output Document

This document defines how to save execution plans to Linear as a Document attached to the Issue.

## Input

- `ISSUE_ID` - Issue ID from plan input (required for Linear output)
- Temporary file from the previous step containing the execution plan

## Process

### 1. Create Document

Use the `linear` skill's `create-document` command to attach the plan as a Document to the Issue:

```
skill: linear
args: create-document TITLE="[Plan] {title from frontmatter}" CONTENT_FILE={temp_file_path} ISSUE_ID={ISSUE_ID}
```

### 2. Update Issue Status

After creating the document, update the Issue status to "Todo":

Call `mcp__linear__update_issue` with:
- `id`: ISSUE_ID
- `state`: "Todo"

## Example

```
Input:
  ISSUE_ID: TA-123
  Temp file: .agent/tmp/plan.xxxxxxxx

Step 1 - Create Document:
  skill: linear
  args: create-document TITLE="[Plan] API Implementation" CONTENT_FILE=.agent/tmp/plan.xxxxxxxx ISSUE_ID=TA-123

Step 2 - Update Issue Status:
  mcp__linear__update_issue(id="TA-123", state="Todo")

Result:
  Document "[Plan] API Implementation" attached to TA-123
  Issue TA-123 status updated to "Todo"
```

## Output

- Document URL (visible in the Issue's Resources section)
- Issue status updated to "Todo"
