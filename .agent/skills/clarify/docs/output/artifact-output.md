# Artifact Output Document

This document defines how to save clarified requirements to an artifact directory.

## Input

- `ARTIFACT_DIR_PATH` - Full Artifact Directory Path (e.g., `.agent/artifacts/20240101-120000`)
- Temporary files from the previous step (e.g., `.agent/tmp/task1.xxxxxxxx`, `.agent/tmp/task2.yyyyyyyy`)

## Process

1. Extract file names from temporary file paths (remove random suffix)
   - `.agent/tmp/task1.xxxxxxxx` → `task1`
   - `.agent/tmp/task2.yyyyyyyy` → `task2`

2. Use `artifact` skill's `create` command to create artifact files:
   ```
   skill: artifact
   args: create {ARTIFACT_DIR_PATH} task1 task2
   ```
   This creates sequentially numbered files like:
   - `{ARTIFACT_DIR_PATH}/01_task1.md`
   - `{ARTIFACT_DIR_PATH}/01_task2.md`

3. Copy content from each temporary file to the corresponding artifact file

### Example

```
ARTIFACT_DIR_PATH: .agent/artifacts/20240101-120000

Temp files:
  .agent/tmp/task1.xxxxxxxx (contains requirements for task 1)
  .agent/tmp/task2.yyyyyyyy (contains requirements for task 2)

After create command:
  .agent/artifacts/20240101-120000/01_task1.md ← content from .agent/tmp/task1.xxxxxxxx
  .agent/artifacts/20240101-120000/01_task2.md ← content from .agent/tmp/task2.yyyyyyyy
```

## Output

List of created artifact file paths.
