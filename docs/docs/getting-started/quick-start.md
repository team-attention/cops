---
sidebar_position: 2
---

# Quick Start

This guide shows how to authenticate, register your first project, and view it in the dashboard.

## 1. Authentication

After installation, authenticate with the C-Ops cloud:

```bash
cops auth login
```

This opens your browser for Google OAuth authentication. Once approved, your credentials are saved locally and you're ready to register projects.

:::info Cloud vs Self-hosted
By default, C-Ops connects to the cloud service. If you're running a self-hosted instance, set the `COPS_API_URL` environment variable before authentication. See the [Self-Hosted Guide](../deployment/self-hosted) for details.
:::

## 2. Register Your First Project

Navigate to your project directory and run the `cops add` command:

```bash
# Navigate to your project directory
cd ~/workspace/my-project

# Register the project
cops add .
```

### TUI Interface

When you run the `cops add` command, an interactive TUI (Text User Interface) appears:

1. **Git repository detection**: Automatically checks if the current directory is a Git repository. Even if you run it from a subdirectory, it automatically finds the Git root.

2. **Project name setup**: Enter the project name to display in the dashboard. The default is the directory name.

3. **Historical log sync**: Choose whether to upload existing Claude Code conversation history to the server. Selecting "Yes" makes existing sessions visible in the dashboard.

## 3. Verify Registered Projects

Use the `cops list` command to view your registered projects:

```bash
cops list
```

Example output:

```
ID         Name        Path                      Git   Worktrees
a1b2c3d4   my-project  /Users/dev/my-project     Yes   -
e5f6g7h8   api-server  /Users/dev/api-server     Yes   feature, hotfix
```

| Column | Description |
|--------|-------------|
| ID | Project unique identifier (abbreviated) |
| Name | Project name |
| Path | Project path |
| Git | Whether it's a Git repository |
| Worktrees | Detected Git worktree list |

## 4. View the Dashboard

Open your web browser and navigate to the dashboard URL:

```
https://cops.team-attention.com
```

What you can see in the dashboard:

- **Project timeline**: Claude Code session history for registered projects
- **Session details**: Conversation content and token usage for each session
- **Statistics**: Usage analysis by project and time period

:::tip Git Worktree Auto-detection
C-Ops automatically detects Git worktrees. When you register the main repository, sessions from all worktrees are tracked together. There's no need to register each worktree separately.
:::

## Next Steps

- Check all command options in the [CLI Reference](./cli-reference)
- Learn about detailed configuration in [Configuration](./configuration)
