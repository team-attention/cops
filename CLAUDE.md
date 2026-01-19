# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

C-Ops (Claude Code Ops) is a distributed system for tracking and visualizing Claude Code sessions across registered repositories.

## System Components

### CLI (`cli/`)
User-facing command-line tool for project management:
- `cops add [directory]` - Register a project for session tracking
- `cops list` - Display registered projects with Git worktrees

Uses `dig` for DI (stateless command execution).

### API Server (`api/`)
REST and gRPC server providing:
- Health check endpoints (liveness/readiness)
- Dashboard API for session data access
- Collector endpoint for receiving session records

Uses Fiber (HTTP) + ConnectRPC (gRPC), managed by `fx` lifecycle.

### Daemon (`daemon/`)
Background service that:
- Watches `~/.cops/config.json` for project changes
- Monitors Claude Code JSONL logs in registered projects
- Parses session records and sends to API server

Uses `fx` lifecycle with file watcher.

### Shared (`shared/`)
Common module containing:
- `gen/grpcstub/` - Generated protobuf/ConnectRPC code
- `domain/` - Shared domain models (Project, Session, Message)

## Data Flow

```
Claude Code → JSONL logs → Daemon (watch) → API (collect) → Dashboard
                              ↑
                          CLI (register projects)
```

## Build Commands

```bash
# Generate protobuf code
cd idl/protobuf && buf generate

# Build all modules
go build ./cli/... ./api/... ./daemon/... ./shared/...

# Development servers (Docker + hot reload)
cd api && make dev      # Start API server
cd daemon && make dev   # Start daemon
make dev-down           # Stop
```

## Configuration Paths

- Global config: `~/.cops/config.json`
- Project config: `{project}/.cops/config.json`
- Environment prefix: `COPS_`