#!/bin/bash
# Initialize artifact directory and return full path
# Usage: ARTIFACT_DIR=$(scripts/init.sh [PREFIX])

set -e

PREFIX="${1:-.agent/artifacts}"
ARTIFACT_ID=$(date +%Y%m%d-%H%M%S)
ARTIFACT_DIR="${PREFIX}/${ARTIFACT_ID}"

mkdir -p "$ARTIFACT_DIR"

# Return full Artifact Directory Path
echo "$ARTIFACT_DIR"
