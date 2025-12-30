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
.agent/plugins/feature-dev/skills/artifact/scripts/artifact-id.sh
```

**Returns:** Artifact ID in `YYYYMMDD-HHMMSS` format

**Side Effects:**
- Creates `.agent/artifacts/{ARTIFACT_ID}/` directory
- Prints artifact ID to stdout

**Example:**
```bash
ARTIFACT_ID=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/artifact-id.sh)
echo "Created artifact directory: .agent/artifacts/${ARTIFACT_ID}"
```

### `create`

Create a new sequentially numbered artifact file in an existing artifact directory.

**Usage:**
```bash
.agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh ARTIFACT_ID name
```

**Parameters:**
- `ARTIFACT_ID` - The artifact directory ID (from `init` command)
- `name` - Base name for the file (without number prefix or extension)

**Returns:** Full path to created file

**Side Effects:**
- Creates file at `.agent/artifacts/{ARTIFACT_ID}/{NN}_{name}.md`
- Number prefix is auto-incremented (01, 02, 03, ...)
- File is created empty (touched)

**Example:**
```bash
# First file: 01_requirements.md
FILE1=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh 20241229-143022 requirements)

# Second file: 02_plan.md
FILE2=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh 20241229-143022 plan)

# Third file: 03_implementation.md
FILE3=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh 20241229-143022 implementation)
```

## Workflow Example

```bash
# Step 1: Initialize artifact directory
ARTIFACT_ID=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/artifact-id.sh)
echo "Artifact ID: ${ARTIFACT_ID}"

# Step 2: Create requirements document
REQ_FILE=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh ${ARTIFACT_ID} requirements)
echo "Created: ${REQ_FILE}"
# Write requirements to ${REQ_FILE}

# Step 3: Create implementation plan
PLAN_FILE=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh ${ARTIFACT_ID} plan)
echo "Created: ${PLAN_FILE}"
# Write plan to ${PLAN_FILE}

# Step 4: Create review document
REVIEW_FILE=$(bash .agent/plugins/feature-dev/skills/artifact/scripts/next-artifact-file.sh ${ARTIFACT_ID} review)
echo "Created: ${REVIEW_FILE}"
# Write review to ${REVIEW_FILE}
```
