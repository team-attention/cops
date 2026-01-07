#!/bin/bash
# Create sequentially numbered artifact files
# Usage: FILE_PATH=$(scripts/create.sh ARTIFACT_DIR name [name2 name3 ...])

set -e

ARTIFACT_DIR=""
FILE_NAMES=()

while [ $# -gt 0 ]; do
    if [ -z "$ARTIFACT_DIR" ]; then
        ARTIFACT_DIR="$1"
    else
        FILE_NAMES+=("$1")
    fi
    shift
done

if [ -z "$ARTIFACT_DIR" ] || [ ${#FILE_NAMES[@]} -eq 0 ]; then
    echo "Usage: $0 ARTIFACT_DIR name [name2 ...]" >&2
    exit 1
fi

if [ ! -d "$ARTIFACT_DIR" ]; then
    echo "Error: Directory does not exist: $ARTIFACT_DIR" >&2
    exit 1
fi

# Check for duplicate names in arguments
seen_names=""
for name in "${FILE_NAMES[@]}"; do
    if echo "$seen_names" | grep -qw "$name"; then
        echo "Error: Duplicate name '$name'" >&2
        exit 1
    fi
    seen_names="$seen_names $name"
done

# Find the highest existing number
HIGHEST_NUM=0
for file in "$ARTIFACT_DIR"/*.md; do
    if [ -f "$file" ]; then
        basename_file=$(basename "$file")
        num="${basename_file%%_*}"
        if [[ "$num" =~ ^[0-9]+$ ]]; then
            if [ "$num" -gt "$HIGHEST_NUM" ]; then
                HIGHEST_NUM=$num
            fi
        fi
    fi
done

# Calculate next number (same for all files in this call)
NEXT_NUM=$((HIGHEST_NUM + 1))
PADDED_NUM=$(printf "%02d" $NEXT_NUM)

# Create files for each name argument with the same number
for FILE_NAME in "${FILE_NAMES[@]}"; do
    FILE_PATH="${ARTIFACT_DIR}/${PADDED_NUM}_${FILE_NAME}.md"
    touch "$FILE_PATH"
    echo "$FILE_PATH"
done
