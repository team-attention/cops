---
trigger: glob
globs: **/internal/service/*/inbound/worker/queue/*.go
paths: **/internal/service/*/inbound/worker/queue/*.go
---

# Queue Worker Inbound Handler Guidelines

This document defines guidelines specific to Queue implementation of worker handlers.

**For common patterns, see:**
- [go-inbound.md](./go-inbound.md) - Handler structure, module registration
- [go-logging-conventions.md](./go-logging-conventions.md) - Logger binding

## Worker Interface

```go
type Worker interface {
    Start(ctx context.Context)
}
```

## Handler Implementation

```go
package queue

import (
    "context"
    "log/slog"
    "time"

    "github.com/.../internal/platform/domain"
    "github.com/.../internal/platform/pkg/queue"
    "{domain}"
)

type {Domain}WorkerHandler struct {
    svc    *{domain}.Service
    queue  queue.Queue[domain.Job]
    logger *slog.Logger
}

func New{Domain}WorkerHandler(l *slog.Logger, svc *{domain}.Service, q queue.Queue[domain.Job]) *{Domain}WorkerHandler {
    return &{Domain}WorkerHandler{
        svc:    svc,
        queue:  q,
        logger: l.With(slog.String("name", "{domain}.worker.queue")),
    }
}

// Start implements Worker interface
func (w *{Domain}WorkerHandler) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            w.logger.Info("stopping worker")
            return
        case <-ticker.C:
            w.processNext(ctx)
        }
    }
}

func (w *{Domain}WorkerHandler) processNext(ctx context.Context) {
    // 1. Dequeue message
    msg, err := w.queue.Dequeue(ctx, string(domain.JobType{Domain}))
    if err != nil || msg == nil {
        return
    }

    job := msg.Data
    w.logger.Info("processing job", slog.String("jobId", string(job.ID)))

    // 2. Start heartbeat
    heartbeatCtx, cancel := context.WithCancel(ctx)
    defer cancel()
    go w.heartbeatLoop(heartbeatCtx, msg.ID)

    // 3. Process job
    if err := w.svc.ProcessJob(ctx, job); err != nil {
        w.logger.Error("failed to process job", slog.String("jobId", string(job.ID)), slog.Any("error", err))
        w.queue.Fail(ctx, msg.ID, err.Error())
        return
    }

    // 4. Mark complete
    w.queue.Complete(ctx, msg.ID)
    w.logger.Info("job completed", slog.String("jobId", string(job.ID)))
}

func (w *{Domain}WorkerHandler) heartbeatLoop(ctx context.Context, msgID string) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.queue.Heartbeat(ctx, msgID)
        }
    }
}
```

## Key Requirements

- **Package name**: `queue`
- **Logger name**: `"{domain}.worker.queue"`
- **Start method**: Must handle context cancellation for graceful shutdown
- **Heartbeat**: Keep job alive during long processing
