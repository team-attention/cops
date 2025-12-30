#!/bin/bash
# .agent/scripts/artifact-id.sh
# Generate unique artifact ID in YYYYMMDD-HHMMSS format
# Usage: ARTIFACT_ID=$(.agent/scripts/artifact-id.sh)

set -e

# Generate timestamp-based ID
ARTIFACT_ID=$(date +%Y%m%d-%H%M%S)

# Create artifacts directory
ARTIFACT_DIR=".agent/artifacts/${ARTIFACT_ID}"
mkdir -p "$ARTIFACT_DIR"

# Return the artifact ID
echo "$ARTIFACT_ID"
