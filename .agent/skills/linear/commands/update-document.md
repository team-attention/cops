# `update-document` Command

Updates an existing Linear document. Uses GraphQL API directly.

## Parameters

### Required

- `DOCUMENT_ID` - Document UUID or slugId to update

### Optional

- `TITLE` - New document title
- `CONTENT` - New document content in Markdown format (use with short content)
- `CONTENT_FILE` - Path to file containing new document content (use with long content)

> At least one of `TITLE`, `CONTENT`, or `CONTENT_FILE` must be provided.

## Usage Examples

```bash
# Update document title only
skill: linear
args: update-document DOCUMENT_ID=abc-123 TITLE="Updated Title"

# Update document content
skill: linear
args: update-document DOCUMENT_ID=abc-123 CONTENT="### New Content\n..."

# Update document content from file
skill: linear
args: update-document DOCUMENT_ID=abc-123 CONTENT_FILE=/tmp/updated-plan.md

# Update multiple fields
skill: linear
args: update-document DOCUMENT_ID=abc-123 TITLE="New Title" CONTENT_FILE=/tmp/plan.md
```

## Process

### Step 1: Validate Parameters

1. Check that `DOCUMENT_ID` is provided
2. Check that at least one of `TITLE`, `CONTENT`, or `CONTENT_FILE` is provided
3. If `CONTENT_FILE` is provided, read content from file

### Step 2: Update Document

Run the script:

```bash
scripts/update_document.sh "$DOCUMENT_ID" "$TITLE" "$CONTENT"
```

### Step 3: Report Result

Display updated document information:

```
Document updated successfully.

- Document: {title}
- URL: {document URL}
```

## Environment Variables

- `LINEAR_API_KEY` - Required for GraphQL API authentication
