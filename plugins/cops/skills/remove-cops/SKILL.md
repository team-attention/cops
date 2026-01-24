---
name: remove-cops
description: |
  Use this skill to completely remove C-Ops (Claude Code Ops) from the system.
  Stops the daemon service, removes PATH configuration, and deletes all cops files.

  IMPORTANT: This action is irreversible. All cops data (config, auth, socket) will be deleted.

  Examples:
    /remove-cops
allowed-tools:
  - Bash
---

# Description

Cleanly removes C-Ops from the system by stopping the daemon service, cleaning up shell PATH entries, and deleting the `~/.cops/` directory.

# Process

## 1. Stop Daemon Service

Uninstall the daemon service using the cops CLI:

```bash
~/.cops/bin/cops uninstall
```

If the binary doesn't exist or the command fails, continue with cleanup (the service may already be stopped).

## 2. Remove PATH Entry

Detect the user's shell config file and remove the cops PATH entry:

1. Identify shell config file (same logic as installer):
   - zsh: `~/.zprofile` (if exists) or `~/.zshrc`
   - bash: `~/.bash_profile` (if exists) or `~/.bashrc`
   - other: `~/.profile`

2. Remove the cops PATH lines from the config file:
   - The comment line: `# Added by cops installer`
   - The export line: `export PATH="$HOME/.cops/bin:$PATH"`
   - Any blank line immediately preceding the comment (added by installer)

## 3. Remove Installation Directory

Delete the entire cops directory:

```bash
rm -rf ~/.cops
```

This removes binaries, config, auth credentials, and socket files.

## 4. Report Result

Display the removal result to the user:
- Daemon service status (stopped or was not running)
- Shell config file modified
- Remind user to restart terminal or source shell config

# Output

SUCCESS: (no output fields)

ERROR: Error message string (e.g., "Failed to remove PATH entry from shell config")
