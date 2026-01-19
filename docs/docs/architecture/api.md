---
sidebar_position: 3
---

# API Server Architecture

This page describes the architecture of the C-Ops API server.

## Overview

The API server is the central backend that receives session data from daemons and serves data to the dashboard. It provides both gRPC (ConnectRPC) and REST endpoints.

## Technology Stack

| Technology | Purpose |
|------------|---------|
| Go | Primary language |
| Fiber | HTTP framework |
| ConnectRPC | gRPC framework |
| MongoDB | Database |
| fx | Dependency injection and lifecycle |
| JWT | Authentication tokens |

## Architecture Pattern

The API follows **Hexagonal Architecture** (Ports and Adapters):

```
                    ┌─────────────────────────────────────┐
                    │            Inbound Adapters          │
                    │  ┌───────────────┐ ┌─────────────┐  │
                    │  │    gRPC       │ │    HTTP     │  │
                    │  │  (ConnectRPC) │ │   (Fiber)   │  │
                    │  └───────┬───────┘ └──────┬──────┘  │
                    └──────────┼────────────────┼─────────┘
                               │                │
                               ▼                ▼
┌──────────────────────────────────────────────────────────────────┐
│                         Service Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │    Auth      │  │  Collector   │  │      Dashboard       │   │
│  │   Service    │  │   Service    │  │       Service        │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
                               │
                               ▼
                    ┌─────────────────────────────────────┐
                    │           Outbound Adapters          │
                    │  ┌───────────────┐ ┌─────────────┐  │
                    │  │   MongoDB     │ │  External   │  │
                    │  │  Repository   │ │  Services   │  │
                    │  └───────────────┘ └─────────────┘  │
                    └─────────────────────────────────────┘
```

## Services

| Service | Purpose |
|---------|---------|
| **Auth** | Google OAuth Device Code flow authentication |
| **Collector** | Receives and stores session data from daemons |
| **Dashboard** | Serves data to the web dashboard |

## Authentication Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    Client   │     │  API Server │     │   Google    │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ InitiateDeviceAuth│                   │
       │──────────────────▶│                   │
       │                   │  Get Device Code  │
       │                   │──────────────────▶│
       │                   │◀──────────────────│
       │  device_code,     │                   │
       │  user_code, url   │                   │
       │◀──────────────────│                   │
       │                   │                   │
       │   User visits URL and authenticates   │
       │ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ▶│
       │                   │                   │
       │  PollDeviceAuth   │                   │
       │──────────────────▶│                   │
       │                   │  Exchange code    │
       │                   │──────────────────▶│
       │                   │◀──────────────────│
       │                   │  (tokens)         │
       │  JWT tokens       │                   │
       │◀──────────────────│                   │
       │                   │                   │
```

