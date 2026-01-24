---
name: add-project
description: |
  Use this skill to register the current project for C-Ops session tracking.
  Launches the interactive `cops add` TUI for project configuration.

  IMPORTANT: This command requires interactive terminal input (TUI).
  Claude cannot run it directly - the user must execute it in their terminal.

  Args:
    DIR=<path> (Optional) - Directory to register (defaults to current directory)
    NO_GIT=true (Optional) - Treat directory as non-git project

  Examples:
    /add-project
    /add-project DIR=/path/to/project
    /add-project NO_GIT=true
allowed-tools:
  - Bash
---

# Description

Registers a project directory for Claude Code session tracking via the `cops add` interactive TUI.
The TUI handles git repository detection, project naming, organization selection, and past log sync configuration.

# Parameters

## Optional

- `DIR` - Directory path to register (defaults to `.` / current working directory)
- `NO_GIT` - If `true`, skips git detection and treats as non-git project

# Process

## 1. Verify Installation

Check that cops is installed and accessible:

```bash
~/.cops/bin/cops --version
```

If this fails, inform the user to run `/install-cops` first.

## 2. Guide User to Run Command

The `cops add` command launches an interactive TUI (bubbletea) that requires terminal input.
This cannot be executed through a non-interactive shell.

Construct the command for the user:

```bash
~/.cops/bin/cops add ${DIR:-.} ${NO_GIT:+--no-git}
```

Inform the user to run this command in their terminal. The TUI will guide them through:
1. Git repository detection (finds git repos in parent directories)
2. Project name configuration
3. Organization selection
4. Past log sync preference

## 3. Verify Registration

After the user confirms they ran the command, verify the project was registered:

```bash
~/.cops/bin/cops list
```

Check that the project directory appears in the output.

## 4. Report Result

Display the registration result:
- Project name and path from `cops list` output
- Confirm session tracking is active

# Output

SUCCESS: (no output fields)

ERROR: Error message string (e.g., "cops is not installed", "Project not found in cops list after registration")
