# Artifact Output Document

This document defines how to save execution plans to an artifact directory.

## Input

- `ARTIFACT_DIR_PATH` - Full Artifact Directory Path (e.g., `.agent/artifacts/20240101-120000`)
- Temporary files from the previous step (e.g., `.agent/tmp/plan.xxxxxxxx`)

## Process

1. Extract file names from temporary file paths (remove random suffix)
   - `.agent/tmp/plan.xxxxxxxx` -> `plan`

2. Use `artifact` skill's `create` command to create artifact files:
   ```
   skill: artifact
   args: create {ARTIFACT_DIR_PATH} plan
   ```
   This creates sequentially numbered files like:
   - `{ARTIFACT_DIR_PATH}/{NN}_plan.md` (NN is the next sequential number)

3. Copy content from each temporary file to the corresponding artifact file

### Example

```
ARTIFACT_DIR_PATH: .agent/artifacts/20240101-120000

Temp files:
  .agent/tmp/plan.xxxxxxxx (contains execution plan)

After create command:
  .agent/artifacts/20240101-120000/{NN}_plan.md <- content from .agent/tmp/plan.xxxxxxxx
```

## Output

List of created artifact file paths.
