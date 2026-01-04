---
name: walkthrough
description: |
  Generates a comprehensive walkthrough document summarizing the development session.
---

# Walkthrough Skill

Generates `walkthrough.md` by analyzing artifacts and code changes using the Gemini agent.

## Input

- `ARTIFACT_ID`: Identifier for the directory containing all previous artifacts.
- `OUTPUT_PATH`: Path where the walkthrough document should be written.

## Process

### Run Gemini Agent

Execute the walkthrough generation script.

```bash
# Execute the walkthrough script
./.agent/plugins/feature-dev-v2/skills/walkthrough/scripts/run.sh
```
