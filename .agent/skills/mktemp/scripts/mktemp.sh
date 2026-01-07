#!/bin/bash
# Create temporary files with optional prefixes
# Usage: mktemp.sh [prefix1] [prefix2] ...

set -e

# Use project-local .agent/tmp/ directory
TMPDIR_LOCAL=".agent/tmp"
mkdir -p "$TMPDIR_LOCAL"

DEFAULT_PREFIX="tmp"

generate_random() {
    # Random suffix using timestamp + PID + shasum
    echo "$(date +%s%N 2>/dev/null || date +%s)$$" | shasum | head -c 8
}

create_temp_file() {
    local prefix="${1:-$DEFAULT_PREFIX}"
    local random_suffix=$(generate_random)
    local filepath="$TMPDIR_LOCAL/${prefix}.${random_suffix}"
    touch "$filepath"
    echo "$filepath"
}

# No arguments: create single file with default prefix
if [ $# -eq 0 ]; then
    create_temp_file "$DEFAULT_PREFIX"
    exit 0
fi

# Create a file for each prefix
for PREFIX in "$@"; do
    create_temp_file "$PREFIX"
done
