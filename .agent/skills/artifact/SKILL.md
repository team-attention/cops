---
name: artifact
description: Manages artifact directories and files for organizing work outputs during development sessions.
---

# Artifact Skill

## Description

This skill provides commands to initialize artifact directories and create sequential artifact files. Artifacts are stored in `.agent/artifacts/{ARTIFACT_ID}/` with timestamped directory names and sequentially numbered files.

## Commands

### `init`

Initialize a new artifact directory with a unique timestamp-based ID.

**Usage:**
```bash
scripts/artifact-id.sh
```

**Returns:** Artifact ID in `YYYYMMDD-HHMMSS` format

**Side Effects:**
- Creates `.agent/artifacts/{ARTIFACT_ID}/` directory
- Prints artifact ID to stdout

### `create`

Create a new sequentially numbered artifact file in an existing artifact directory.

**Usage:**
```bash
scripts/next-artifact-file.sh ARTIFACT_ID name
```


**Parameters:**
- `ARTIFACT_ID` - The artifact directory ID (from `init` command)
- `name` - Base name for the file (without number prefix or extension)

**Returns:** Full path to created file

**Side Effects:**
- Creates file at `.agent/artifacts/{ARTIFACT_ID}/{NN}_{name}.md`
- Number prefix is auto-incremented (01, 02, 03, ...)
- File is created empty (touched)
