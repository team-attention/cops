---
name: commit-pr
description: |
  Commit changes with artifact-based message and create PR (or push to main)
---

# Commit-PR Skill

Commits staged changes with a message generated from artifact files, then handles push/PR based on current branch.

## Input

- `ARTIFACT_ID` - Artifact directory identifier for the current work session

## Process

### Run Gemini Agent

Execute the commit-pr logic using the Gemini agent.

```bash
# Execute the commit-pr script
./.agent/plugins/feature-dev-v2/skills/commit-pr/scripts/run.sh
```
