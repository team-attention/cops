---
sidebar_position: 1
slug: /
---

# Introduction to C-Ops

C-Ops (Claude Code Ops) is a distributed system for tracking and visualizing Claude Code sessions across your team.

## What is C-Ops?

C-Ops is a tool that automatically collects and analyzes sessions from Claude Code, an AI coding assistant. It allows you to view conversations with Claude Code, token usage, and work patterns all in one dashboard.

## What Problems Does It Solve?

When using AI coding assistants across a team, several challenges arise:

- **Usage tracking**: Not knowing how many tokens the entire team is consuming
- **Scattered work history**: Each developer's Claude Code conversation history exists only locally
- **Lack of insights**: Difficulty understanding what tasks the AI assistant is primarily used for

C-Ops solves these problems.

## Key Features

- **Session recording**: Conversations with Claude Code are automatically collected
- **Token usage analysis**: View token consumption by project and time period
- **Dashboard**: Visualize session history and statistics in a web interface

## System Components

C-Ops consists of four components:

| Component | Role |
|-----------|------|
| **CLI** | Command-line tool for project registration and management |
| **Daemon** | Background service that monitors and collects Claude Code logs |
| **API Server** | Stores collected data and serves it to the dashboard |
| **Dashboard** | Web interface for visualizing sessions and statistics |

## Data Flow

```
Claude Code → JSONL logs → Daemon (watch) → API (collect) → Dashboard
                              ↑
                          CLI (register projects)
```

## Next Steps

- Install C-Ops with the [Installation Guide](./getting-started/installation)
- Register your first project with [Quick Start](./getting-started/quick-start)
