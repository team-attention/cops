---
sidebar_position: 4
---

# Daemon Architecture

This page describes the architecture of the C-Ops daemon service.

## Overview

The daemon is a background service that runs on developer machines. It watches Claude Code log files and sends session data to the API server in real-time.

## Technology Stack

| Technology | Purpose |
|------------|---------|
| Go | Primary language |
| fsnotify | File system watching |
| fx | Dependency injection and lifecycle |
| ConnectRPC | gRPC client |

## Architecture Pattern

The daemon follows a service-based architecture with three main components:

```
┌──────────────────────────────────────────────────────────────────┐
│                          Daemon                                   │
│                                                                   │
│  ┌────────────────┐                                              │
│  │ Config Watcher │──────────────────────────────────┐           │
│  │                │                                   │           │
│  │ Watches:       │                                   ▼           │
│  │ ~/.cops/       │      ┌──────────────────────────────────┐    │
│  │ config.json    │      │         Log Watcher               │    │
│  └────────────────┘      │                                   │    │
│                          │ Watches:                          │    │
│                          │ ~/.claude/projects/.../sessions/  │    │
│                          │ <project>/.claude/...             │    │
│                          └──────────────┬───────────────────┘    │
│                                         │                        │
│                                         ▼                        │
│                          ┌──────────────────────────────────┐    │
│                          │         Log Parser                │    │
│                          │                                   │    │
│                          │ Parses JSONL records              │    │
│                          └──────────────┬───────────────────┘    │
│                                         │                        │
│                                         ▼                        │
│                          ┌──────────────────────────────────┐    │
│                          │       Sync Service                │    │
│                          │                                   │    │
│                          │ Sends to API (gRPC/ConnectRPC)    │    │
│                          └──────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

## Services

### Config Watcher

Monitors changes to the global configuration file.

**Watched File:**
```
~/.cops/config.json
```

**Responsibilities:**
- Detect when projects are added or removed
- Trigger Log Watcher updates
- Handle config file errors gracefully

**Flow:**
```
Config file changes
        │
        ▼
fsnotify detects change
        │
        ▼
Parse updated config
        │
        ▼
Compare with current state
        │
        ▼
Add/remove watch paths for Log Watcher
```

### Log Watcher

Monitors Claude Code log files for new session data.

**Watched Paths:**
```
~/.claude/projects/<project-hash>/sessions/*.jsonl
<project-path>/.claude/sessions/*.jsonl  (for Git worktrees)
```

**Responsibilities:**
- Watch multiple directories simultaneously
- Detect new log files
- Track file positions for incremental reads
- Handle file rotations

**Flow:**
```
New JSONL content written
        │
        ▼
fsnotify detects write event
        │
        ▼
Read new content from last position
        │
        ▼
Send to Log Parser
        │
        ▼
Update file position tracker
```

### Log Parser

Parses JSONL records from Claude Code logs.

**Record Types:**
- `init` - Session initialization
- `user_message` - User input
- `assistant_message` - Claude response
- `tool_use` - Tool invocation
- `tool_result` - Tool execution result

**Parsing Flow:**
```
Raw JSONL line
        │
        ▼
JSON decode
        │
        ▼
Identify record type
        │
        ▼
Extract relevant fields
        │
        ▼
Create domain model
        │
        ▼
Send to Sync Service
```

### Sync Service

Sends processed records to the API server.

**Responsibilities:**
- Batch records for efficiency
- Handle network failures with retry
- Manage authentication tokens
- Track sync state

**Batching Strategy:**
```
Records arrive
        │
        ▼
Add to batch buffer
        │
        ├──▶ Buffer full? ──▶ Send immediately
        │
        └──▶ Timer expired? ──▶ Send buffered records
```

## File Watching Strategy

### Watch Path Resolution

For each registered project:

1. **Global Claude directory:**
   ```
   ~/.claude/projects/<sha256(project-path)>/sessions/
   ```

2. **Project-local directory (if exists):**
   ```
   <project-path>/.claude/sessions/
   ```

3. **Git worktrees (if Git project):**
   ```
   <worktree-path>/.claude/sessions/
   ```

## System Service Installation

### macOS (launchd)

The daemon is installed as a LaunchAgent:

```xml
<!-- ~/Library/LaunchAgents/com.cops.daemon.plist -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "...">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.cops.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/user/.cops/bin/cops-daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

### Linux (systemd)

The daemon is installed as a user service:

```ini
# ~/.config/systemd/user/cops-daemon.service
[Unit]
Description=C-Ops Daemon
After=network.target

[Service]
ExecStart=%h/.cops/bin/cops-daemon
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

## Error Handling

- **Network failures**: Exponential backoff retry
- **File not found**: Remove from watch list, log warning
- **Permission denied**: Log error, skip file
