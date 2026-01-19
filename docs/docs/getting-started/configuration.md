---
sidebar_position: 4
---

# Configuration

This page explains C-Ops configuration files and environment variables.

## Global Configuration (~/.cops/config.json)

The global configuration file applies to all projects for the user.

### File Path

```
~/.cops/config.json
```

### Main Fields

```json
{
  "projects": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "my-project",
      "path": "/Users/dev/my-project",
      "isGitProject": true
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `projects` | array | List of registered projects |
| `projects[].id` | string | Project unique identifier (UUID) |
| `projects[].name` | string | Project name |
| `projects[].path` | string | Project root path |
| `projects[].isGitProject` | boolean | Whether it's a Git repository |

:::warning Direct Modification Warning
This file is automatically managed by the `cops add` command. Direct modification may cause unexpected Daemon behavior. Use CLI commands when possible.
:::

## Project Configuration (`<project>/.cops/config.json`)

Project-specific configuration file.

### File Path

```
<project path>/.cops/config.json
```

### Auto-creation

Created automatically when you register a project with the `cops add` command.

### Main Fields

```json
{
  "projectId": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `projectId` | string | Project unique identifier (same as the id in global config) |

:::tip Add to .gitignore
It's recommended to add the project config file to `.gitignore`:
```
.cops/
```
:::

## Authentication File (~/.cops/auth.json)

Stores authentication information required for API server communication.

### File Path

```
~/.cops/auth.json
```

### Auto-creation

Created automatically upon successful server authentication.

### Security Notes

- This file contains sensitive authentication information
- Do not share with other users
- Ensure it's not included in version control systems

## Environment Variables

You can override configurations using environment variables prefixed with `COPS_`.

### Available Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `COPS_API_URL` | Cloud URL* | API server endpoint |
| `COPS_LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `COPS_VERSION` | - | Specify version during installation |

\* Cloud URL: `https://cops-api-392947101616.asia-northeast3.run.app`

### Examples

```bash
# Connect to a self-hosted server
export COPS_API_URL=https://your-api-domain.com
cops auth login

# Enable debug logging
export COPS_LOG_LEVEL=debug
cops add .

# Install a specific version
COPS_VERSION=v1.0.0 curl -fsSL .../install.sh | bash
```

## Configuration File Locations Summary

| File | Purpose | Management |
|------|---------|------------|
| `~/.cops/config.json` | Global config (project list) | `cops add`, `cops remove` |
| `~/.cops/auth.json` | Authentication credentials | `cops auth login` |
| `<project>/.cops/config.json` | Project config | `cops add` |
