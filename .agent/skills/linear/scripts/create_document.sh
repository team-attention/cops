#!/bin/bash
# Creates a Linear document using GraphQL API
# Usage: create_document.sh TITLE CONTENT ISSUE_ID
#
# Required:
#   TITLE    - Document title
#   CONTENT  - Document content (markdown)
#   ISSUE_ID - Issue UUID or identifier to attach document to
#
# Environment:
#   LINEAR_API_KEY - Required. Linear API key for authentication.
#
# Output:
#   On success: JSON with document id and url
#   On failure: Error message to stderr, exit 1

set -euo pipefail

TITLE="${1:-}"
CONTENT="${2:-}"
ISSUE_ID="${3:-}"

if [[ -z "$TITLE" ]]; then
  echo "Error: TITLE is required" >&2
  exit 1
fi

if [[ -z "$CONTENT" ]]; then
  echo "Error: CONTENT is required" >&2
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

ESCAPED_TITLE=$(echo -n "$TITLE" | escape_json)
ESCAPED_CONTENT=$(echo -n "$CONTENT" | escape_json)

# Build input object
if [[ -n "$ISSUE_ID" ]]; then
  INPUT="{\"title\": $ESCAPED_TITLE, \"content\": $ESCAPED_CONTENT, \"issueId\": \"$ISSUE_ID\"}"
else
  echo "Error: Either ISSUE_ID or PROJECT must be provided" >&2
  exit 1
fi

# GraphQL mutation
QUERY='mutation DocumentCreate($input: DocumentCreateInput!) { documentCreate(input: $input) { success document { id url slugId title } } }'

# Make API request
RESPONSE=$(curl -s -X POST \
  -H "Authorization: $LINEAR_API_KEY" \
  -H "Content-Type: application/json" \
  --data "{\"query\": $(echo -n "$QUERY" | escape_json), \"variables\": {\"input\": $INPUT}}" \
  https://api.linear.app/graphql)

# Check for errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
  echo "Error: $(echo "$RESPONSE" | jq -r '.errors[0].message')" >&2
  exit 1
fi

# Check success
SUCCESS=$(echo "$RESPONSE" | jq -r '.data.documentCreate.success')
if [[ "$SUCCESS" != "true" ]]; then
  echo "Error: Document creation failed" >&2
  echo "$RESPONSE" >&2
  exit 1
fi

# Output document info
echo "$RESPONSE" | jq '.data.documentCreate.document'
