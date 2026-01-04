#!/bin/bash
set -e

# Default variables
ARTIFACT_ID="${ARTIFACT_ID:-$1}"
OUTPUT_PATH="${OUTPUT_PATH:-$2}"

if [ -z "$ARTIFACT_ID" ] || [ -z "$OUTPUT_PATH" ]; then
  echo "Error: ARTIFACT_ID and OUTPUT_PATH must be provided."
  exit 1
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROMPT_PATH="$SCRIPT_DIR/../PROMPT.md"
TEMP_PROMPT=$(mktemp)

# Construct full prompt
cat "$PROMPT_PATH" > "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "---" >> "$TEMP_PROMPT"
echo "# Execution Context" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## Inputs" >> "$TEMP_PROMPT"
echo "ARTIFACT_ID: $ARTIFACT_ID" >> "$TEMP_PROMPT"
echo "OUTPUT_PATH: $OUTPUT_PATH" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## Action" >> "$TEMP_PROMPT"
echo "Please generate the walkthrough document at the specified OUTPUT_PATH based on the artifacts and git status." >> "$TEMP_PROMPT"

# Run Gemini
# --approval-mode auto_edit: Automatically accepts edits (validation should be non-destructive)
gemini run "$(cat $TEMP_PROMPT)" --approval-mode auto_edit

# Cleanup
rm "$TEMP_PROMPT"
