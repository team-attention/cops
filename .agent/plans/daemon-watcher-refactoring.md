# Daemon Watcher Services Refactoring Plan

## Overview

Refactor daemon module to follow proper Hexagonal Architecture:
- **Inbound handlers own event loops** (fsnotify watching)
- **Services contain pure business logic only** (no goroutines, no event loops)
- **fx.Group pattern for handler registration**

## Current Problems

1. **Callback-based communication** - `OnConfigChange`, `OnLogEntry` bypass hexagonal ports
2. **Tightly coupled services** - ConfigWatcher + Project are functionally one but split
3. **Over-separated services** - LogWatcher + LogProcessor are unnecessarily split
4. **Event loop in wrong layer** - Service owns fsnotify event loop (should be Inbound)
5. **Manual orchestrator** - `register_watcher.go` manually wires everything

## Target Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         INBOUND                              │
│  ┌─────────────────────┐    ┌─────────────────────────────┐  │
│  │ ConfigWatcher       │    │ Log (2 workers)             │  │
│  │ worker/fsnotify/    │    │ worker/fsnotify/            │  │
│  │ - Start() event loop│    │ - Handle FileEvent          │  │
│  │ - Handle FileEvent  │    │ - Call Service              │  │
│  │ - Call Service      │    │ worker/pubsub/              │  │
│  └──────────┬──────────┘    │ - Subscribe to targets      │  │
│             │               │ - Call Service.UpdateTargets│  │
│             │               └──────────────┬──────────────┘  │
└─────────────┼──────────────────────────────┼─────────────────┘
              │                              │
              ▼                              ▼
┌──────────────────────────────────────────────────────────────┐
│                         SERVICE                              │
│  ┌─────────────────────┐    ┌─────────────────────────────┐  │
│  │ ConfigWatcher       │    │ Log                         │  │
│  │ - ParseConfig()     │    │ - ParseJSONL()              │  │
│  │ - BuildTargets()    │    │ - BufferRecord()            │  │
│  │ - PublishTargets()  │    │ - Flush() → API             │  │
│  │ (no goroutines)     │    │ (no goroutines)             │  │
│  └──────────┬──────────┘    └──────────────┬──────────────┘  │
└─────────────┼──────────────────────────────┼─────────────────┘
              │                              │
              ▼                              ▼
┌──────────────────────────────────────────────────────────────┐
│                        OUTBOUND                              │
│  ┌─────────────────────┐    ┌─────────────────────────────┐  │
│  │ PubSub WriterPort   │    │ APIClientPort               │  │
│  │ (publish targets)   │    │ StateRepositoryPort         │  │
│  └─────────────────────┘    └─────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**fx.Group Registration:**
- Inbound handlers implement `FsnotifyHandler` interface
- `register_fsnotify.go`가 `group:"fsnotify_handlers"`로 수집하여 일괄 시작/종료

---

## Implementation Steps

### Phase 1: Foundation (New Files)

#### 1.1 Create shared FileEvent types
**File:** `daemon/internal/platform/domain/fileevent.go`
```go
type FileOp uint32
const (OpCreate, OpWrite, OpRemove, OpRename, OpChmod)
type FileEvent struct { Path string; Op FileOp }
```

#### 1.2 Create PubSub package (Generic)
**File:** `daemon/internal/platform/pkg/pubsub/port.go`
```go
type WriterPort[T any] interface {
    Publish(msg T) error
}

type ReaderPort[T any] interface {
    Subscribe() <-chan T
    Unsubscribe(ch <-chan T)
}
```

**File:** `daemon/internal/platform/pkg/pubsub/inmemory/pubsub.go`
- Go generics 기반 in-memory pub/sub 구현
- Thread-safe, buffered channels

#### 1.3 Create PubSub setup
**File:** `daemon/internal/platform/setup/pubsub.go`
```go
func InitTargetPubSub(l *slog.Logger) *inmemory.PubSub[[]domain.WatchTarget]
```

#### 1.4 Create FsnotifyHandler interface & register
**File:** `daemon/cmd/internal/container/register_fsnotify.go`
```go
// FsnotifyHandler is implemented by Inbound handlers that process fsnotify events
type FsnotifyHandler interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

type fsnotifyParams struct {
    fx.In
    Handlers []FsnotifyHandler `group:"fsnotify_handlers"`
}

func registerFsnotify(p fsnotifyParams) {
    // fx.Group으로 수집, 순서대로 Start, 역순으로 Stop
}
```

---

### Phase 2: ConfigWatcher Refactoring

#### 2.1 ConfigWatcher Service (Pure Business Logic)
**File:** `daemon/internal/service/configwatcher/configwatcher_service.go`
```go
type Service struct {
    logger    *slog.Logger
    pubsub    pubsub.WriterPort[[]domain.WatchTarget]
    configPath string
}

// No Start/Stop, no goroutines - pure business logic only
func (s *Service) HandleConfigChange(path string) error {
    cfg, err := s.loadConfig(path)
    if err != nil { return err }

    targets := s.buildWatchTargets(cfg)
    return s.pubsub.Publish(targets)
}

func (s *Service) loadConfig(path string) (*domain.GlobalConfig, error) { ... }
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget { ... }
```

#### 2.2 ConfigWatcher Inbound Handler (Owns Event Loop)
**File:** `daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go`
```go
package fsnotify

type ConfigWatcherFsnotifyHandler struct {
    logger     *slog.Logger
    svc        *configwatcher.Service
    watcher    *fsnotify.Watcher  // 직접 fsnotify 사용
    configPath string
    ctx        context.Context
    cancel     context.CancelFunc
}

func NewConfigWatcherFsnotifyHandler(
    l *slog.Logger,
    svc *configwatcher.Service,
    cfg *setup.Config,
) (*ConfigWatcherFsnotifyHandler, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil { return nil, err }

    return &ConfigWatcherFsnotifyHandler{
        logger:     l.With(slog.String("name", "configwatcher.worker.fsnotify")),
        svc:        svc,
        watcher:    watcher,
        configPath: cfg.Cops.GlobalConfigPath,
    }, nil
}

// Implements FsnotifyHandler
func (h *ConfigWatcherFsnotifyHandler) Name() string { return "configwatcher" }

func (h *ConfigWatcherFsnotifyHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)

    // Initial config load
    if err := h.svc.HandleConfigChange(h.configPath); err != nil {
        h.logger.Warn("initial config load failed", slog.Any("error", err))
    }

    // Watch config file
    if err := h.watcher.Add(h.configPath); err != nil {
        return err
    }

    go h.loop()
    return nil
}

func (h *ConfigWatcherFsnotifyHandler) Stop(ctx context.Context) error {
    h.cancel()
    return h.watcher.Close()
}

func (h *ConfigWatcherFsnotifyHandler) loop() {
    for {
        select {
        case <-h.ctx.Done():
            return
        case event := <-h.watcher.Events:
            if event.Op&fsnotify.Write != 0 || event.Op&fsnotify.Create != 0 {
                if err := h.svc.HandleConfigChange(h.configPath); err != nil {
                    h.logger.Error("config change handling failed", slog.Any("error", err))
                }
            }
        case err := <-h.watcher.Errors:
            h.logger.Error("watcher error", slog.Any("error", err))
        }
    }
}
```

#### 2.3 ConfigWatcher Outbound (PubSub only)
**Directory:** `daemon/internal/service/configwatcher/outbound/`
```
outbound/
└── pubsub/
    ├── target_writer_port.go           # type alias or wrapper
    └── inmemory/
        └── target_writer.go            # wraps platform/pkg/pubsub
```

Note: fsnotify는 Inbound handler에서 직접 사용하므로 Outbound 아님

---

### Phase 3: Log Service (New - Merged LogWatcher + LogProcessor)

#### 3.0 Log Watcher Setup (Shared fsnotify.Watcher)
**File:** `daemon/internal/platform/setup/log_watcher.go`
```go
func InitLogWatcher(l *slog.Logger) (*fsnotify.Watcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    l.Info("log watcher initialized")
    return watcher, nil
}
```
- Inbound (fsnotify)와 Outbound (FileWatchPort)가 같은 Watcher 공유

#### 3.1 Log Service
**File:** `daemon/internal/service/log/log_service.go`
```go
type Service struct {
    logger      *slog.Logger
    fileWatcher FileWatchPort   // Outbound: fsnotify Add/Remove
    apiClient   APIClientPort   // Outbound: API 전송
    watchedDirs map[string]bool
    buffer      []shareddomain.SessionRecord
    mu          sync.Mutex
}

// Called by Inbound (pubsub) when targets change
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    newDirs := make(map[string]bool)
    for _, t := range targets {
        newDirs[t.ClaudeDir] = true
    }

    // Remove old watches via Outbound port
    for dir := range s.watchedDirs {
        if !newDirs[dir] {
            s.fileWatcher.Remove(dir)
            delete(s.watchedDirs, dir)
        }
    }

    // Add new watches via Outbound port
    for dir := range newDirs {
        if !s.watchedDirs[dir] {
            if err := s.fileWatcher.Add(dir); err != nil {
                s.logger.Debug("failed to add watch", slog.String("dir", dir))
                continue
            }
            s.watchedDirs[dir] = true
        }
    }
    return nil
}

// Called by Inbound (fsnotify) when file changes
func (s *Service) HandleFileChange(path string, fromOffset int64) ([]shareddomain.SessionRecord, int64, error) {
    records, newOffset, err := s.parseJSONLFrom(path, fromOffset)
    if err != nil { return nil, fromOffset, err }
    return records, newOffset, nil
}

func (s *Service) AddRecords(records []shareddomain.SessionRecord) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.buffer = append(s.buffer, records...)
}

func (s *Service) Flush(ctx context.Context) error {
    s.mu.Lock()
    batch := s.buffer
    s.buffer = nil
    s.mu.Unlock()

    if len(batch) == 0 { return nil }
    return s.apiClient.SendLogs(ctx, domain.LogBatch{Records: batch})
}
```

#### 3.2 Log Inbound Handler 1: Fsnotify (File Events)
**File:** `daemon/internal/service/log/inbound/worker/fsnotify/handler.go`
```go
package fsnotify

type LogFsnotifyHandler struct {
    logger        *slog.Logger
    svc           *log.Service
    stateRepo     repository.StateRepositoryPort
    watcher       *fsnotify.Watcher  // shared with Outbound FileWatchPort
    filePositions map[string]int64
    flushTicker   *time.Ticker
    ctx           context.Context
    cancel        context.CancelFunc
}

func NewLogFsnotifyHandler(
    l *slog.Logger,
    svc *log.Service,
    stateRepo repository.StateRepositoryPort,
    watcher *fsnotify.Watcher,  // injected from setup
) *LogFsnotifyHandler { ... }

// Implements FsnotifyHandler
func (h *LogFsnotifyHandler) Name() string { return "log-fsnotify" }

func (h *LogFsnotifyHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    positions, _ := h.stateRepo.LoadFilePositions(ctx)
    h.filePositions = positions
    h.flushTicker = time.NewTicker(5 * time.Second)
    go h.loop()
    return nil
}

func (h *LogFsnotifyHandler) loop() {
    for {
        select {
        case <-h.ctx.Done():
            return
        case event := <-h.watcher.Events:
            h.handleFileEvent(event)
        case <-h.flushTicker.C:
            h.svc.Flush(h.ctx)
        case err := <-h.watcher.Errors:
            h.logger.Error("watcher error", slog.Any("error", err))
        }
    }
}

func (h *LogFsnotifyHandler) handleFileEvent(event fsnotify.Event) {
    if event.Op&fsnotify.Write != 0 {
        offset := h.filePositions[event.Name]
        records, newOffset, err := h.svc.HandleFileChange(event.Name, offset)
        if err == nil {
            h.svc.AddRecords(records)
            h.filePositions[event.Name] = newOffset
            h.stateRepo.SaveFilePosition(h.ctx, &domain.FilePosition{
                Path: event.Name, Offset: newOffset,
            })
        }
    }
}
```

#### 3.3 Log Inbound Handler 2: PubSub (Target Changes)
**File:** `daemon/internal/service/log/inbound/worker/pubsub/handler.go`
```go
package pubsub

type LogPubsubHandler struct {
    logger       *slog.Logger
    svc          *log.Service      // calls Service.UpdateTargets()
    targetReader pubsub.ReaderPort[[]domain.WatchTarget]
    targetCh     <-chan []domain.WatchTarget
    ctx          context.Context
    cancel       context.CancelFunc
}

func NewLogPubsubHandler(
    l *slog.Logger,
    svc *log.Service,
    targetReader pubsub.ReaderPort[[]domain.WatchTarget],
) *LogPubsubHandler { ... }

// Implements FsnotifyHandler (same lifecycle interface)
func (h *LogPubsubHandler) Name() string { return "log-pubsub" }

func (h *LogPubsubHandler) Start(ctx context.Context) error {
    h.ctx, h.cancel = context.WithCancel(ctx)
    h.targetCh = h.targetReader.Subscribe()
    go h.loop()
    return nil
}

func (h *LogPubsubHandler) loop() {
    for {
        select {
        case <-h.ctx.Done():
            return
        case targets := <-h.targetCh:
            // Service.UpdateTargets() → Outbound FileWatchPort.Add()/Remove()
            if err := h.svc.UpdateTargets(targets); err != nil {
                h.logger.Error("failed to update targets", slog.Any("error", err))
            }
        }
    }
}
```

#### 3.4 Log Outbound Ports
**Directory:** `daemon/internal/service/log/outbound/`
```
outbound/
├── filesystem/
│   ├── filewatch_port.go              # FileWatchPort interface (Add/Remove)
│   └── fsnotify/
│       └── filewatch.go               # Wraps *fsnotify.Watcher (shared with Inbound)
├── repository/
│   ├── state_repo_port.go             # StateRepositoryPort interface
│   └── sqlite/
│       └── state_repo.go              # SQLite implementation
└── api/
    ├── api_client_port.go             # APIClientPort interface
    └── connectrpc/
        └── api_client.go              # ConnectRPC implementation
```

**FileWatchPort interface:**
```go
type FileWatchPort interface {
    Add(path string) error
    Remove(path string) error
}
```

**Shared Watcher Flow:**
```
                    ┌─────────────────────────────────────┐
                    │        *fsnotify.Watcher            │
                    │         (from setup/)               │
                    └─────────────────────────────────────┘
                              ▲                │
                              │                │ Events channel
              Add()/Remove()  │                ▼
┌─────────────────────────────┴───┐    ┌─────────────────────────────┐
│  Outbound: FileWatchPort        │    │  Inbound: FsnotifyHandler   │
│  (wraps watcher for Add/Remove) │    │  (reads watcher.Events)     │
└─────────────────────────────────┘    └─────────────────────────────┘
              ▲                                      │
              │                                      │
              │                                      ▼
┌─────────────┴───────────────────────────────────────────────────────┐
│                          Log Service                                 │
│  UpdateTargets() → fileWatcher.Add()/Remove()                       │
│  HandleFileChange() ← called by Inbound                             │
└─────────────────────────────────────────────────────────────────────┘
              ▲
              │
┌─────────────┴───────────────────┐
│  Inbound: PubsubHandler         │
│  targets → svc.UpdateTargets()  │
└─────────────────────────────────┘
```

---

### Phase 4: Container Refactoring

#### 4.1 Update module_platform.go
**File:** `daemon/cmd/internal/container/module_platform.go`
- Add `fx.Provide(setup.InitTargetPubSub)`

#### 4.2 Create module_config.go
**File:** `daemon/cmd/internal/container/module_config.go`
```go
func newConfigModule() fx.Option {
    return fx.Module("config",
        // Outbound: PubSub WriterPort
        fx.Provide(fx.Annotate(
            func(ps *inmemory.PubSub[[]domain.WatchTarget]) pubsub.WriterPort[[]domain.WatchTarget] {
                return ps
            },
            fx.As(new(pubsub.WriterPort[[]domain.WatchTarget])),
        )),

        // Service (pure business logic)
        fx.Provide(configwatcher.NewService),

        // Inbound: FsnotifyHandler with fx.Group
        fx.Provide(fx.Annotate(
            fsnotify.NewConfigWatcherFsnotifyHandler,
            fx.As(new(FsnotifyHandler)),
            fx.ResultTags(`group:"fsnotify_handlers"`),
        )),
    )
}
```

#### 4.3 Create module_log.go
**File:** `daemon/cmd/internal/container/module_log.go`
```go
func newLogModule() fx.Option {
    return fx.Module("log",
        // Shared fsnotify.Watcher from setup (used by both Inbound and Outbound)
        fx.Provide(setup.InitLogWatcher),

        // Outbound: FileWatchPort (wraps shared watcher)
        fx.Provide(fx.Annotate(
            fsnotifyadapter.NewFileWatchAdapter,  // receives *fsnotify.Watcher
            fx.As(new(filesystem.FileWatchPort)),
        )),

        // Outbound: PubSub ReaderPort
        fx.Provide(fx.Annotate(
            func(ps *inmemory.PubSub[[]domain.WatchTarget]) pubsub.ReaderPort[[]domain.WatchTarget] {
                return ps
            },
            fx.As(new(pubsub.ReaderPort[[]domain.WatchTarget])),
        )),

        // Outbound: StateRepositoryPort, APIClientPort
        fx.Provide(fx.Annotate(
            sqlite.NewStateRepository,
            fx.As(new(repository.StateRepositoryPort)),
        )),
        fx.Provide(fx.Annotate(
            connectrpc.NewAPIClient,
            fx.As(new(api.APIClientPort)),
        )),

        // Service
        fx.Provide(log.NewService),

        // Inbound 1: Fsnotify Handler (reads watcher.Events)
        fx.Provide(fx.Annotate(
            fsnotifyhandler.NewLogFsnotifyHandler,  // receives *fsnotify.Watcher
            fx.As(new(FsnotifyHandler)),
            fx.ResultTags(`group:"fsnotify_handlers"`),
        )),

        // Inbound 2: PubSub Handler (target changes → Service.UpdateTargets)
        fx.Provide(fx.Annotate(
            pubsubhandler.NewLogPubsubHandler,
            fx.As(new(FsnotifyHandler)),
            fx.ResultTags(`group:"fsnotify_handlers"`),
        )),
    )
}
```

#### 4.4 Update application.go
**File:** `daemon/cmd/internal/container/application.go`
```go
fx.New(
    newPlatformModule(),
    newConfigModule(),
    newLogModule(),
    fx.Invoke(registerFsnotify),
)
```

---

### Phase 5: Cleanup

#### 5.1 Delete obsolete files/directories
- `daemon/internal/service/project/` (directory)
- `daemon/internal/service/logwatcher/` (directory)
- `daemon/internal/service/logprocessor/` (directory)
- `daemon/cmd/internal/container/register_watcher.go`
- `daemon/cmd/internal/container/module_watcher.go`
- `daemon/cmd/internal/container/module_processor.go`
- `daemon/internal/service/configwatcher/port.go`
- `daemon/internal/service/configwatcher/outbound/filesystem/` (directory)

---

## Critical Files Summary

| Action | File |
|--------|------|
| CREATE | `daemon/internal/platform/domain/fileevent.go` |
| CREATE | `daemon/internal/platform/pkg/pubsub/port.go` |
| CREATE | `daemon/internal/platform/pkg/pubsub/inmemory/pubsub.go` |
| CREATE | `daemon/internal/platform/setup/pubsub.go` |
| CREATE | `daemon/cmd/internal/container/register_fsnotify.go` |
| CREATE | `daemon/cmd/internal/container/module_config.go` |
| CREATE | `daemon/cmd/internal/container/module_log.go` |
| CREATE | `daemon/internal/service/configwatcher/inbound/worker/fsnotify/handler.go` |
| CREATE | `daemon/internal/service/configwatcher/outbound/pubsub/target_writer_port.go` |
| CREATE | `daemon/internal/service/configwatcher/outbound/pubsub/inmemory/target_writer.go` |
| CREATE | `daemon/internal/service/log/log_service.go` |
| CREATE | `daemon/internal/service/log/inbound/worker/fsnotify/handler.go` |
| CREATE | `daemon/internal/service/log/inbound/worker/pubsub/handler.go` |
| CREATE | `daemon/internal/service/log/outbound/repository/state_repo_port.go` |
| CREATE | `daemon/internal/service/log/outbound/repository/sqlite/state_repo.go` |
| CREATE | `daemon/internal/service/log/outbound/api/api_client_port.go` |
| CREATE | `daemon/internal/service/log/outbound/api/connectrpc/api_client.go` |
| CREATE | `daemon/internal/service/log/outbound/pubsub/target_reader_port.go` |
| CREATE | `daemon/internal/service/log/outbound/pubsub/inmemory/target_reader.go` |
| MODIFY | `daemon/internal/service/configwatcher/configwatcher_service.go` |
| MODIFY | `daemon/cmd/internal/container/module_platform.go` |
| MODIFY | `daemon/cmd/internal/container/application.go` |
| DELETE | `daemon/internal/service/project/` (directory) |
| DELETE | `daemon/internal/service/logwatcher/` (directory) |
| DELETE | `daemon/internal/service/logprocessor/` (directory) |
| DELETE | `daemon/cmd/internal/container/register_watcher.go` |
| DELETE | `daemon/cmd/internal/container/module_watcher.go` |
| DELETE | `daemon/cmd/internal/container/module_processor.go` |
| DELETE | `daemon/internal/service/configwatcher/port.go` |
| DELETE | `daemon/internal/service/configwatcher/outbound/filesystem/` (directory) |

---

## Directory Structure After Refactoring

```
daemon/
├── cmd/internal/container/
│   ├── application.go
│   ├── module_platform.go
│   ├── module_config.go              # NEW
│   ├── module_log.go                 # NEW
│   └── register_fsnotify.go          # NEW (FsnotifyHandler interface + fx.Group)
│
└── internal/
    ├── platform/
    │   ├── domain/
    │   │   ├── global_config.go
    │   │   ├── watch.go
    │   │   └── fileevent.go          # NEW
    │   ├── pkg/
    │   │   └── pubsub/               # NEW
    │   │       ├── port.go
    │   │       └── inmemory/
    │   │           └── pubsub.go
    │   └── setup/
    │       ├── config.go
    │       ├── logger.go
    │       ├── sqlite.go
    │       ├── copsapi.go
    │       └── pubsub.go             # NEW
    │
    └── service/
        ├── configwatcher/
        │   ├── configwatcher_service.go   # MODIFIED (pure business logic)
        │   ├── inbound/                   # NEW
        │   │   └── worker/
        │   │       └── fsnotify/
        │   │           └── handler.go     # Owns event loop
        │   └── outbound/
        │       └── pubsub/
        │           ├── target_writer_port.go
        │           └── inmemory/
        │               └── target_writer.go
        │
        └── log/                           # NEW (merged logwatcher + logprocessor)
            ├── log_service.go             # Pure business logic
            ├── inbound/
            │   └── worker/
            │       ├── fsnotify/
            │       │   └── handler.go     # Owns event loop, reads watcher.Events
            │       └── pubsub/
            │           └── handler.go     # Subscribes to target changes
            └── outbound/
                ├── repository/
                │   ├── state_repo_port.go
                │   └── sqlite/
                │       └── state_repo.go
                ├── api/
                │   ├── api_client_port.go
                │   └── connectrpc/
                │       └── api_client.go
                └── pubsub/
                    ├── target_reader_port.go
                    └── inmemory/
                        └── target_reader.go
```

---

## Key Architectural Principles

1. **Inbound owns event loops** - fsnotify watching, pubsub subscription, flush ticker
2. **Service is pure business logic** - no goroutines, no event loops, easily testable
3. **Outbound is for external calls** - API, database, pubsub publish
4. **fsnotify는 Outbound가 아님** - Inbound에서 직접 사용 (들어오는 이벤트이므로)
5. **fx.Group으로 핸들러 일괄 관리** - `group:"fsnotify_handlers"`
