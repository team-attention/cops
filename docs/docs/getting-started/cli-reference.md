---
sidebar_position: 3
---

# CLI Reference

This page documents all available commands for the C-Ops CLI (`cops`).

## Authentication Commands

### cops auth login

Authenticate with the C-Ops server using Google OAuth.

```bash
cops auth login
```

**Behavior:**
1. Initiates the OAuth Device Code flow
2. Opens your browser for Google authentication
3. Saves credentials to `~/.cops/auth.json` upon success
4. Subsequent commands use the saved credentials

---

## Project Management Commands

### cops add

Register a project for session tracking.

#### Usage

```bash
cops add [directory] [flags]
```

#### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Project path to register | `.` (current directory) |

#### Options

| Option | Description |
|--------|-------------|
| `--no-git` | Treat as non-Git project |

#### TUI Interface

When you run the command, an interactive TUI appears:

1. **Git repository detection**
   - Searches for a Git repository in the specified directory and parent directories
   - If found, uses the repository root as the project path
   - Use `--no-git` to skip Git detection

2. **Project name setup**
   - Enter the project name to display in the dashboard
   - Defaults to the directory name

3. **Historical log sync**
   - Choose whether to upload existing Claude Code conversation history
   - Selecting "Yes" syncs all existing sessions

#### Examples

```bash
# Register current directory as a project (TUI appears)
cops add .

# Register a specific path as a project
cops add /path/to/project

# Register without treating as a Git project
cops add . --no-git
```

#### How It Works

1. Resolves the absolute path of the specified directory
2. Adds project information to `~/.cops/config.json`
3. Daemon detects the config change and starts watching the `.claude` directory
4. If historical sync is selected, existing JSONL log files are sent to the server

---

### cops list

Display the list of registered projects.

#### Usage

```bash
cops list
```

#### Output Format

Displays projects in a table format:

```
ID         Name        Path                      Git   Worktrees
a1b2c3d4   my-project  /Users/dev/my-project     Yes   -
e5f6g7h8   api-server  /Users/dev/api-server     Yes   feature, hotfix
```

#### Output Columns

| Column | Description |
|--------|-------------|
| ID | Project unique identifier (first 8 characters + "..." abbreviated) |
| Name | Project name |
| Path | Project root path |
| Git | Git repository status ("Yes" or "No") |
| Worktrees | Detected Git worktree list (abbreviated if more than 2) |

#### Git Worktree Display

For Git projects, worktree information is shown:

- 1 worktree: `"1 worktree"`
- 2 worktrees: `"feature, hotfix"`
- 3+ worktrees: `"feature, hotfix, ..."`

#### How It Works

1. Reads the `~/.cops/config.json` file
2. Formats the registered project list as a table
3. Detects and displays Git worktree information for each project

---

### cops remove

Unregister a project from session tracking.

#### Usage

```bash
cops remove <project-id>
```

#### Arguments

| Argument | Description |
|----------|-------------|
| `project-id` | Project ID to remove (can use abbreviated ID) |

#### Example

```bash
# Remove a project by its ID
cops remove a1b2c3d4
```

---

## Daemon Management Commands

### cops install

Install the C-Ops daemon as a system service.

#### Usage

```bash
cops install
```

#### Behavior

Registers the daemon as a system service:
- **macOS**: Uses `launchctl` (LaunchAgent)
- **Linux**: Uses `systemd` (user service)

---

### cops uninstall

Remove the C-Ops daemon system service.

#### Usage

```bash
cops uninstall
```

#### Behavior

Stops and removes the daemon system service.

---

## Utility Commands

### cops version

Display the CLI version.

#### Usage

```bash
cops version
```

---

### cops upgrade

Upgrade C-Ops to the latest version.

#### Usage

```bash
cops upgrade
```

#### Behavior

1. Checks for new versions on GitHub Releases
2. Downloads and installs the latest version if available
3. Restarts the daemon service if running
