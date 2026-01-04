#!/bin/bash
set -e

# Default variables from environment or arguments
ISSUE_OR_REQUEST="${ISSUE_OR_REQUEST:-$1}"
OUTPUT_PATH="${OUTPUT_PATH:-$2}"

if [ -z "$ISSUE_OR_REQUEST" ] || [ -z "$OUTPUT_PATH" ]; then
  echo "Error: ISSUE_OR_REQUEST and OUTPUT_PATH must be provided."
  exit 1
fi

# Define paths
# Assuming this script is running from the project root or we need to find the plugin dir relative to the script
# But typically skills might be run from root. Let's stick to the relative path used in SKILL.md for now, 
# or better, determine the script directory.
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROMPT_PATH="$SCRIPT_DIR/../PROMPT.md"
TEMP_PROMPT=$(mktemp)

# Construct the full prompt
cat "$PROMPT_PATH" > "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "---" >> "$TEMP_PROMPT"
echo "# Execution Context" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## User Input" >> "$TEMP_PROMPT"
echo "$ISSUE_OR_REQUEST" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## Output Location" >> "$TEMP_PROMPT"
echo "You MUST write the final requirements document to this path:" >> "$TEMP_PROMPT"
echo "$OUTPUT_PATH" >> "$TEMP_PROMPT"

# Run Gemini
# --prompt: Sets the initial prompt
# --approval-mode auto_edit: Automatically accepts
gemini  -i "$(cat $TEMP_PROMPT)" --approval-mode auto_edit

# Cleanup
rm "$TEMP_PROMPT"
