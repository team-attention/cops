# Linear Task Document

This document defines how to retrieve a plan document from Linear.

## Input

- `DOCUMENT_ID` - Linear Document ID or slug (e.g., `abc123-def456`, `my-document-slug`)

## Process

1. Use `mcp__linear__get_document` to fetch the document by ID or slug
2. Extract the document content (markdown format)
3. Parse the plan structure:
   - YAML frontmatter (title, issueId if present)
   - Overview section
   - Package Changes section (if present)
   - Implementation Steps section

## Output

Plan content ready for implementation, containing:
- Clear implementation steps
- File paths to create/modify
- Function signatures and algorithms
- Success criteria (if defined)
