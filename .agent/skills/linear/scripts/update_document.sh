#!/bin/bash
# Updates a Linear document using GraphQL API
# Usage: update_document.sh DOCUMENT_ID [TITLE] [CONTENT]
#
# Required:
#   DOCUMENT_ID - Document UUID or slugId to update
#
# Optional:
#   TITLE   - New document title
#   CONTENT - New document content (markdown)
#
# At least one of TITLE or CONTENT must be provided.
#
# Environment:
#   LINEAR_API_KEY - Required. Linear API key for authentication.
#
# Output:
#   On success: JSON with document id and url
#   On failure: Error message to stderr, exit 1

set -euo pipefail

DOCUMENT_ID="${1:-}"
TITLE="${2:-}"
CONTENT="${3:-}"

if [[ -z "$DOCUMENT_ID" ]]; then
  echo "Error: DOCUMENT_ID is required" >&2
  exit 1
fi

if [[ -z "$TITLE" && -z "$CONTENT" ]]; then
  echo "Error: At least one of TITLE or CONTENT must be provided" >&2
  exit 1
fi

if [[ -z "${LINEAR_API_KEY:-}" ]]; then
  echo "Error: LINEAR_API_KEY environment variable is required" >&2
  exit 1
fi

# Escape content for JSON
escape_json() {
  python3 -c "import json,sys; print(json.dumps(sys.stdin.read()))"
}

# Build input object
INPUT_FIELDS=''

if [[ -n "$TITLE" ]]; then
  ESCAPED_TITLE=$(echo -n "$TITLE" | escape_json)
  INPUT_FIELDS="\"title\": $ESCAPED_TITLE"
fi

if [[ -n "$CONTENT" ]]; then
  ESCAPED_CONTENT=$(echo -n "$CONTENT" | escape_json)
  if [[ -n "$INPUT_FIELDS" ]]; then
    INPUT_FIELDS="$INPUT_FIELDS, \"content\": $ESCAPED_CONTENT"
  else
    INPUT_FIELDS="\"content\": $ESCAPED_CONTENT"
  fi
fi

INPUT="{$INPUT_FIELDS}"

# GraphQL mutation
QUERY='mutation DocumentUpdate($id: String!, $input: DocumentUpdateInput!) { documentUpdate(id: $id, input: $input) { success document { id url slugId title } } }'

# Make API request
RESPONSE=$(curl -s -X POST \
  -H "Authorization: $LINEAR_API_KEY" \
  -H "Content-Type: application/json" \
  --data "{\"query\": $(echo -n "$QUERY" | escape_json), \"variables\": {\"id\": \"$DOCUMENT_ID\", \"input\": $INPUT}}" \
  https://api.linear.app/graphql)

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
  echo "Error: $(echo "$RESPONSE" | jq -r '.errors[0].message')" >&2
  exit 1
fi

# Check success
SUCCESS=$(echo "$RESPONSE" | jq -r '.data.documentUpdate.success')
if [[ "$SUCCESS" != "true" ]]; then
  echo "Error: Document update failed" >&2
  echo "$RESPONSE" >&2
  exit 1
fi

# Output document info
echo "$RESPONSE" | jq '.data.documentUpdate.document'
