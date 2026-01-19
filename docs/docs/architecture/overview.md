---
sidebar_position: 1
---

# Architecture Overview

This page provides an overview of the C-Ops system architecture.

## System Components

C-Ops is a distributed system consisting of four main components:

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Machine                              │
│                                                                  │
│  ┌──────────┐     ┌──────────┐     ┌──────────────────────────┐ │
│  │  Claude  │────▶│  JSONL   │◀────│         Daemon           │ │
│  │   Code   │     │   Logs   │     │  (Background Watcher)    │ │
│  └──────────┘     └──────────┘     └───────────┬──────────────┘ │
│                                                 │                │
│  ┌──────────────────────────┐                  │                │
│  │           CLI            │                  │                │
│  │  (Project Registration)  │                  │                │
│  └──────────────────────────┘                  │                │
└────────────────────────────────────────────────┼────────────────┘
                                                  │ gRPC/ConnectRPC
                                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Server                                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                      API Server                           │   │
│  │  ┌─────────────────┐      ┌─────────────────────────┐    │   │
│  │  │    Collector    │      │    Dashboard Service    │    │   │
│  │  │    Service      │      │                         │    │   │
│  │  └────────┬────────┘      └───────────┬─────────────┘    │   │
│  └───────────┼───────────────────────────┼──────────────────┘   │
│              │                           │                       │
│              ▼                           ▼                       │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                       MongoDB                             │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                                  │ gRPC-Web
                                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Browser                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Web Dashboard                          │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Component Overview

### 1. CLI

The command-line interface for managing projects.

- **Purpose**: Project registration and management
- **Technology**: Go, Cobra, Dig (DI), BubbleTea (TUI)
- **Key Commands**: `add`, `list`, `remove`, `auth login`

See [CLI Architecture](./cli) for details.

### 2. Daemon

Background service running on developer machines.

- **Purpose**: Monitor Claude Code logs and send to server
- **Technology**: Go, fsnotify (file watching), fx (lifecycle)
- **Key Features**: Config watching, log parsing, real-time sync

See [Daemon Architecture](./daemon) for details.

### 3. API Server

Central backend for data collection and dashboard queries.

- **Purpose**: Receive and store session data, serve dashboard API
- **Technology**: Go, Fiber (HTTP), ConnectRPC (gRPC), MongoDB
- **Services**: Collector Service, Dashboard Service

See [API Architecture](./api) for details.

### 4. Web Dashboard

Frontend application for visualizing session data.

- **Purpose**: Display sessions, statistics, and insights
- **Technology**: React, Vite, TailwindCSS, TanStack (Query/Router)

## Data Flow

### 1. Project Registration Flow

```
User runs `cops add .`
        │
        ▼
CLI detects Git repository
        │
        ▼
User configures via TUI (name, sync options)
        │
        ▼
CLI writes to ~/.cops/config.json
        │
        ▼
Daemon detects config change
        │
        ▼
Daemon starts watching project's .claude directory
```

### 2. Session Collection Flow

```
Claude Code writes to ~/.claude/projects/.../sessions/*.jsonl
        │
        ▼
Daemon detects file change (fsnotify)
        │
        ▼
Daemon parses JSONL records
        │
        ▼
Daemon sends records to API (gRPC/ConnectRPC)
        │
        ▼
API Collector Service stores in MongoDB
        │
        ▼
Dashboard queries and displays data
```

### 3. Authentication Flow

```
User runs `cops auth login`
        │
        ▼
CLI initiates OAuth Device Code flow
        │
        ▼
User authenticates via browser (Google OAuth)
        │
        ▼
API server validates and issues JWT
        │
        ▼
CLI stores credentials in ~/.cops/auth.json
        │
        ▼
Subsequent requests use stored JWT
```

## Technology Stack

| Component | Technology |
|-----------|------------|
| CLI | Go, Cobra, Dig, BubbleTea |
| Daemon | Go, fsnotify, fx |
| API Server | Go, Fiber, ConnectRPC, MongoDB |
| Dashboard | React, Vite, TailwindCSS, TanStack |
| Authentication | Google OAuth, JWT |
| Protocol | gRPC (ConnectRPC), gRPC-Web |

## Directory Structure

```
cops/
├── cli/          # CLI tool
├── daemon/       # Background daemon
├── api/          # API server
├── web/          # Web dashboard
├── shared/       # Shared Go libraries
│   ├── gen/      # Generated protobuf code
│   └── domain/   # Shared domain models
├── idl/          # Protocol buffer definitions
│   └── protobuf/
└── docs/         # Documentation (Docusaurus)
```
