#!/bin/bash
set -e

# Default variables
ARTIFACT_ID="${ARTIFACT_ID:-$1}"

if [ -z "$ARTIFACT_ID" ]; then
  echo "Error: ARTIFACT_ID must be provided."
  exit 1
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROMPT_PATH="$SCRIPT_DIR/../PROMPT.md"
TEMP_PROMPT=$(mktemp)

# Construct validation prompts
cat "$PROMPT_PATH" > "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "---" >> "$TEMP_PROMPT"
echo "# Execution Context" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## Artifact ID" >> "$TEMP_PROMPT"
echo "$ARTIFACT_ID" >> "$TEMP_PROMPT"
echo "" >> "$TEMP_PROMPT"
echo "## Instructions" >> "$TEMP_PROMPT"
echo "You are running in a headless mode but have permission to execute git commands. Please proceed with the Analysis, Commit, and Integration steps defined in the prompt." >> "$TEMP_PROMPT"
echo "Remember the Critical Rule: ASK before pushing to main." >> "$TEMP_PROMPT"

# Run Gemini
# Using --approval-mode default to ensure the agent ASKS the user before running critical commands (like push on main) if the prompt logic relies on tool approval.
# HOWEVER, the user request implies we want to delegate.
# If we set yolo, it acts automatically. But we have a logic "Ask user for permission".
# If gemini runs in YOLO mode, it auto-approves tools.
# BUT we want the Agent to explicitly ask a question using the `AskUserQuestion` tool or simply stop and ask for input.
# Since we want it to potentially STOP and wait for confirmation on main, we should perhaps stick to 'default' or 'auto_edit' but allow interaction.
# Let's try `auto_edit` which approves edits/reads but might prompt for shell commands?
# Actually, if we use `gemini run`, it's an agent loop.
# We'll use --approval-mode auto_edit. This allows it to read files and edit (if needed) freely, but might prompt for `run_command`?
# Let's check the user intent: "Delegate to Gemini".
# If we want the agent to ASK the user, it will use the `ask_user` tool (or notify_user equivalent).
# So we run in interactive mode? `gemini run` is interactive by default unless input is closed?
# The `run.sh` will likely be called by another agent or user.
# Let's use `auto_edit` and hope `run_command` prompts if necessary, OR rely on the Agent using `ask_user` tool for the "Permission".

gemini -i "$(cat $TEMP_PROMPT)" --approval-mode auto_edit

# Cleanup
rm "$TEMP_PROMPT"
