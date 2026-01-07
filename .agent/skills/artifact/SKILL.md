---
name: artifact
description: Manages artifact directories and files for organizing work outputs during development sessions.
---

# Artifact Skill

## Description

This skill provides commands to initialize artifact directories and create sequential artifact files. Artifacts are stored with timestamped directory names and sequentially numbered files.

**Default prefix:** `.agent/artifacts`

## Commands

### `init`

Initialize a new artifact directory with a unique timestamp-based name.

**Skill Invocation:**
```
skill: artifact
args: init [PREFIX]
```

**Script Equivalent:**
```bash
scripts/init.sh [PREFIX]
```

**Parameters:**
- `PREFIX` - (Optional) Base directory for artifacts. Defaults to `.agent/artifacts`

**Returns:** Full Artifact Directory Path (e.g., `.agent/artifacts/20260105-120000`)

**Side Effects:**
- Creates `{PREFIX}/{ARTIFACT_ID}/` directory
- Prints full Artifact Directory Path to stdout

**Examples:**
```bash
# Use default prefix
ARTIFACT_DIR_PATH=$(scripts/init.sh)
# -> .agent/artifacts/20260105-120000

# Use custom prefix
ARTIFACT_DIR_PATH=$(scripts/init.sh docs/artifacts)
# -> docs/artifacts/20260105-120000
```

### `create`

Create sequentially numbered artifact files in an existing artifact directory.

**Skill Invocation:**
```
skill: artifact
args: create ARTIFACT_DIR_PATH name [name2 name3 ...]
```

**Script Equivalent:**
```bash
scripts/create.sh ARTIFACT_DIR_PATH name [name2 name3 ...]
```

**Parameters:**
- `ARTIFACT_DIR_PATH` - Full Artifact Directory Path (from `init` command)
- `name` - Base name(s) for the file(s) (without number prefix or extension)

**Returns:** Full path(s) to created file(s), one per line

**Side Effects:**
- Creates file(s) at `{ARTIFACT_DIR_PATH}/{NN}_{name}.md`
- All files in a single call share the same number prefix
- Number prefix auto-increments on each call (01, 02, 03, ...)
- Files are created empty (touched)
- Duplicate names in same call result in error

**Examples:**
```bash
# Create files in artifact directory
ARTIFACT_DIR_PATH=.agent/artifacts/20260105-120000
scripts/create.sh $ARTIFACT_DIR_PATH clarify plan
# -> .agent/artifacts/20260105-120000/01_clarify.md
# -> .agent/artifacts/20260105-120000/01_plan.md
```
