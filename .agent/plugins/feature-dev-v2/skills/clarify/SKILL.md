---
name: clarify
description: |
  Runs the Clarify agent using Gemini CLI to gather and structure requirements.
---

# Clarify Skill

Runs the clarification process using the Gemini CLI. It combines a system prompt with user input to guide the agent.

## Input

- `ISSUE_OR_REQUEST`: Linear Issue ID or general request text.
- `OUTPUT_PATH`: Path where the requirements document/artifact should be created.

## Process

```bash
# Execute the clarify script packaged with the skill
# SCRIPT_DIR is provided by the agent runtime or we assume standard path
# If running from root:
./.agent/plugins/feature-dev-v2/skills/clarify/scripts/run.sh
```

