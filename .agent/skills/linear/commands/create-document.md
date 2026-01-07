# `create-document` Command

Creates a Linear document attached to an issue. Uses GraphQL API directly since the MCP tool doesn't support `issueId` parameter.

## Parameters

### Required

- `TITLE` - Document title
- `ISSUE_ID` - Issue identifier (e.g., `TA-123`) to attach document to

### Optional

- `CONTENT` - Document content in Markdown format (use with short content)
- `CONTENT_FILE` - Path to file containing document content (use with long content)

> One of `CONTENT` or `CONTENT_FILE` must be provided.

## Usage Examples

```bash
# Create document with inline content
skill: linear
args: create-document TITLE="Implementation Plan" CONTENT="### Overview\n..." ISSUE_ID=TA-123

# Create document from file
skill: linear
args: create-document TITLE="[Plan] API Design" CONTENT_FILE=/tmp/plan.md ISSUE_ID=TA-456
```

## Process

### Step 1: Prepare Content

If `CONTENT_FILE` is provided:
1. Read content from file
2. Use as document content

Otherwise, use `CONTENT` parameter directly.

### Step 2: Create Document

Run the script:

```bash
scripts/create_document.sh "$TITLE" "$CONTENT" "$ISSUE_ID"
```

### Step 3: Report Result

Display created document information:

```
Document created successfully.

- Document: {title}
- Attached to: {ISSUE_ID}
- URL: {document URL}
```

## Environment Variables

- `LINEAR_API_KEY` - Required for GraphQL API authentication
