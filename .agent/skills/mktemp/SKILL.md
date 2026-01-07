---
name: mktemp
description: Creates temporary files in .agent/tmp/ with random suffixes. (project)
---

# mktemp Skill

Creates temporary files in `.agent/tmp/` (project-local temp directory). Background subtasks have write access to this directory.

## Parameters

### Optional

- `PREFIX...` - One or more filename prefixes. Defaults to `tmp` if none provided.

## Usage Examples

```bash
# Default prefix (single file)
skill: mktemp
# -> .agent/tmp/tmp.xxxxxxxx

# Custom prefix (single file)
skill: mktemp
args: cops
# -> .agent/tmp/cops.xxxxxxxx

# Multiple files
skill: mktemp
args: report summary data
# -> .agent/tmp/report.xxxxxxxx
# -> .agent/tmp/summary.yyyyyyyy
# -> .agent/tmp/data.zzzzzzzz
```

## Process

1. Run `scripts/mktemp.sh [prefix1] [prefix2] ...`
2. Return each created file path (one per line)

## Output

```
.agent/tmp/{PREFIX1}.{random}
.agent/tmp/{PREFIX2}.{random}
...
```
