---
name: install-cops
description: |
  Use this skill to install or upgrade C-Ops (Claude Code Ops) CLI tool.
  Downloads the latest release from GitHub and sets up cops and cops-daemon binaries.

  IMPORTANT: This skill requires internet access and only supports macOS/Linux (amd64/arm64).

  Args:
    VERSION=<version> (Optional) - Specific version to install (e.g., v0.1.0)

  Examples:
    /install-cops
    /install-cops VERSION=v0.1.0
allowed-tools:
  - Bash
---

# Description

Installs C-Ops CLI tool by running the official install script from the GitHub repository.
The script handles OS/architecture detection, binary download, PATH configuration, and daemon registration.

# Parameters

## Optional

- `VERSION` - Specific version tag to install (e.g., `v0.1.0`). If omitted, installs the latest release.

# Process

## 1. Run Install Script

Execute the install script from the repository:

```bash
COPS_VERSION="${VERSION}" bash -c "$(curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh)"
```

If `VERSION` is not provided, omit the `COPS_VERSION` env var (the script fetches the latest release automatically).

## 2. Verify Installation

After the script completes, verify the installation:

```bash
~/.cops/bin/cops --version
```

## 3. Authenticate

Run the device-based auth flow to register the user:

```bash
~/.cops/bin/cops auth login
```

This opens the browser for device confirmation and stores the API key in `~/.cops/auth.json`.

## 4. Report Result

Display the installation result to the user:
- Installed version
- Binary location (`~/.cops/bin/`)
- Auth status
- Remind user to restart terminal or source shell config if this was a fresh install

# Output

SUCCESS: (no output fields)

ERROR: Error message string (e.g., "Install script failed: unsupported OS")
