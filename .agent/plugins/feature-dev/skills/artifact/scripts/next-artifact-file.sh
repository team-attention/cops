#!/bin/bash
# .agent/scripts/next-artifact-file.sh
# Generate next sequential artifact file name and create it
# Usage: FILE_PATH=$(.agent/scripts/next-artifact-file.sh ARTIFACT_ID name)

set -e

ARTIFACT_ID="$1"
FILE_NAME="$2"

if [ -z "$ARTIFACT_ID" ] || [ -z "$FILE_NAME" ]; then
    echo "Usage: $0 ARTIFACT_ID name" >&2
    exit 1
fi

ARTIFACT_DIR=".agent/artifacts/${ARTIFACT_ID}"

if [ ! -d "$ARTIFACT_DIR" ]; then
    echo "Error: Artifacts directory does not exist: $ARTIFACT_DIR" >&2
    exit 1
fi

# Find the highest existing number
HIGHEST_NUM=0
for file in "$ARTIFACT_DIR"/*.md; do
    if [ -f "$file" ]; then
        basename=$(basename "$file")
        # Extract number prefix (e.g., "01" from "01_research.md")
        num="${basename%%_*}"
        if [[ "$num" =~ ^[0-9]+$ ]]; then
            if [ "$num" -gt "$HIGHEST_NUM" ]; then
                HIGHEST_NUM=$num
            fi
        fi
    fi
done

# Calculate next number
NEXT_NUM=$((HIGHEST_NUM + 1))
PADDED_NUM=$(printf "%02d" $NEXT_NUM)

# Generate file path
FILE_PATH="${ARTIFACT_DIR}/${PADDED_NUM}_${FILE_NAME}.md"

# Always create the file
touch "$FILE_PATH"

# Return the file path
echo "$FILE_PATH"
