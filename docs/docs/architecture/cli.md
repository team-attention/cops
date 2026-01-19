---
sidebar_position: 2
---

# CLI Architecture

This page describes the architecture of the C-Ops CLI tool.

## Overview

The CLI is a command-line tool for managing projects and authentication. It provides an interactive TUI (Text User Interface) for user-friendly project registration.

## Technology Stack

| Technology | Purpose |
|------------|---------|
| Go | Primary language |
| Cobra | CLI framework |
| Dig | Dependency injection |
| BubbleTea | TUI framework |

## Architecture Pattern

The CLI follows **Hexagonal Architecture** with these layers:

### Service Layer

Core business logic for each domain:

- **Auth Service**: OAuth authentication flow management
- **Tracking Service**: Project registration and listing
- **Daemon Service**: System service installation/uninstallation
- **Upgrade Service**: Version checking and upgrades

### Inbound Layer (CLI Handlers)

Cobra command handlers that translate CLI input to service calls.

## Command Structure

```
cops
├── auth
│   └── login         # Google OAuth authentication
├── add [directory]   # Register project (TUI)
├── list              # List registered projects
├── remove <id>       # Unregister project
├── install           # Install daemon service
├── uninstall         # Uninstall daemon service
├── upgrade           # Upgrade to latest version
└── version           # Show version
```

## TUI Components

The CLI uses BubbleTea for interactive TUI experiences:

### Add Command TUI Flow

```
┌─────────────────────────────────────────────┐
│  Step 1: Git Detection                       │
│  ────────────────────                        │
│  Detected Git repository at:                 │
│  /Users/dev/my-project                       │
│                                              │
│  [Continue]                                  │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│  Step 2: Project Name                        │
│  ────────────────────                        │
│  Enter project name:                         │
│  ┌─────────────────────────────────────┐    │
│  │ my-project                          │    │
│  └─────────────────────────────────────┘    │
│                                              │
│  [Continue]                                  │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│  Step 3: Historical Sync                     │
│  ────────────────────                        │
│  Sync existing Claude Code sessions?         │
│                                              │
│  ○ Yes - Upload existing sessions            │
│  ● No  - Start fresh                         │
│                                              │
│  [Continue]                                  │
└─────────────────────────────────────────────┘
```

## Configuration Flow

```
Environment Variables (COPS_*)
            │
            ▼
   ┌────────────────┐
   │  Config Loader │
   │                │
   │  - Reads env   │
   │  - Sets defaults│
   └───────┬────────┘
           │
           ▼
   ┌────────────────┐
   │    Services    │
   │                │
   │  - Auth        │
   │  - Tracking    │
   │  - Daemon      │
   └───────┬────────┘
           │
           ▼
   ┌────────────────┐
   │  CLI Handlers  │
   │                │
   │  - Commands    │
   │  - TUI views   │
   └────────────────┘
```

## Authentication Flow

The CLI implements OAuth Device Code flow for authentication:

```
1. User runs `cops auth login`
           │
           ▼
2. CLI requests device code from API
           │
           ▼
3. CLI displays URL and code to user
   "Visit: https://accounts.google.com/..."
   "Enter code: ABCD-1234"
           │
           ▼
4. User authenticates in browser
           │
           ▼
5. CLI polls API for token
           │
           ▼
6. On success, stores JWT in ~/.cops/auth.json
```

