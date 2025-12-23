# COps Daemon Implementation Plan

## Overview

macOS에서 백그라운드로 동작하는 Daemon 구현. CLI 명령으로 실행하거나 launchd에 등록하여 자동 시작 가능.

## Core Features

1. `~/.cops/config.json` (GlobalConfig) 파일 Watch → 동적 설정 변경 감지
2. Claude Code 로그 디렉토리 Watch (`~/.claude/projects/{encoded-path}/`)
3. 로그 변경 감지 → 파싱 → API 서버 전송 (ConnectRPC)
4. DEBUG 모드: 즉시 전송 / 일반: 주기적 flush

---

## Claude Code Log Structure (조사 결과)

### Log Location
```
~/.claude/projects/{encoded-path}/
```

**Path Encoding Rule**: `/` → `-`
- `/Users/jayce/team-attention/a-team` → `-Users-jayce-team-attention-a-team`

### File Types
1. **Session logs**: `{uuid}.jsonl` - 메인 대화 세션
2. **Agent logs**: `agent-{short-id}.jsonl` - 서브 에이전트 태스크

### JSONL Format (각 라인이 JSON 객체)
```json
{
  "type": "user",                    // user | assistant | file-history-snapshot
  "sessionId": "uuid",               // 세션 ID
  "agentId": "a09903e",              // 에이전트 ID (agent 로그만)
  "cwd": "/Users/jayce/project",     // 현재 작업 디렉토리
  "gitBranch": "main",               // Git 브랜치
  "version": "2.0.70",               // Claude Code 버전
  "timestamp": "2025-12-16T10:33:06.117Z",
  "uuid": "message-uuid",            // 메시지 고유 ID
  "parentUuid": "parent-uuid",       // 부모 메시지 (스레딩)
  "message": {
    "role": "user",
    "content": "..."
  }
}
```

### Watch Strategy
1. **GlobalConfig (`~/.cops/config.json`)**: 프로젝트 경로 목록 (CLI에서 add할 때 IsGitProject 등 미리 판단)
2. **각 프로젝트마다**:
   - Git Project인 경우: `.git/worktrees`에서 모든 worktree 경로 수집
   - 각 경로를 encoded 형태로 변환
   - `~/.claude/projects/{encoded-path}/` 디렉토리 Watch
   - `.jsonl` 파일 변경 감지 → 새 라인 파싱

---

## 1. Directory Structure

```
daemon/
├── go.mod
├── Makefile
├── README.md
├── doc/
│   └── PLAN.md
├── cmd/
│   ├── daemon/
│   │   └── main.go                    # Entry point (Cobra)
│   └── internal/container/
│       ├── application.go             # fx.New() setup
│       ├── module_platform.go         # Config, Logger
│       ├── module_watcher.go          # Watcher services (config, log)
│       ├── module_processor.go        # LogProcessor, APIClient
│       └── register_watcher.go        # Watcher lifecycle & event wiring
└── internal/
    ├── platform/
    │   ├── setup/
    │   │   ├── config/
    │   │   │   └── config.go          # Daemon config (env-based)
    │   │   └── logger/
    │   │       └── logger.go          # slog logger
    │   ├── domain/
    │   │   ├── global_config.go       # ~/.cops/config.json schema (GlobalConfig)
    │   │   └── watch.go               # WatchTarget, FilePosition, LogBatch
    │   └── util/
    │       ├── errutil/
    │       │   └── errutil.go
    │       ├── gitutil/
    │       │   └── gitutil.go         # Git worktree detection
    │       └── pathutil/
    │           └── pathutil.go        # Path encoding for Claude
    └── service/
        ├── configwatcher/             # ~/.cops/config.json watcher
        │   ├── configwatcher_service.go
        │   └── outbound/
        │       └── filesystem/
        │           └── adapter.go     # fsnotify adapter
        ├── project/                   # Project & watch target management
        │   └── project_service.go     # Resolves projects → watch targets
        ├── logwatcher/                # Claude log directory watcher
        │   ├── logwatcher_service.go
        │   └── outbound/
        │       └── filesystem/
        │           └── adapter.go     # fsnotify adapter
        └── logprocessor/              # Log buffering & API sending
            ├── logprocessor_service.go
            └── outbound/
                └── api/
                    └── connectrpc/
                        └── adapter.go # ConnectRPC client
```

---

## 1.1 Service Communication Architecture

**핵심 문제**: 독립적인 서비스들이 어떻게 상태를 공유하고 이벤트를 전달하는가?

**해결책**: **Event-driven + Orchestrator 패턴** (register_watcher.go)

```
┌─────────────────────────────────────────────────────────────────┐
│                    register_watcher.go (Orchestrator)           │
│  - 모든 서비스 주입받음                                            │
│  - 이벤트 핸들러 연결                                              │
│  - fx.Lifecycle로 시작/종료 관리                                   │
└─────────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  ConfigWatcher  │  │  ProjectService │  │   LogWatcher    │
│                 │  │                 │  │                 │
│ OnConfigChange()│─▶│ UpdateProjects()│─▶│ UpdateTargets() │
│                 │  │ GetWatchTargets()│  │                 │
└─────────────────┘  └─────────────────┘  │ OnLogEntry()    │
                                          └────────┬────────┘
                                                   │
                                                   ▼
                                          ┌─────────────────┐
                                          │  LogProcessor   │
                                          │                 │
                                          │ AddEntry()      │
                                          │ Flush()         │
                                          └─────────────────┘

이벤트 흐름:
1. ConfigWatcher.OnConfigChange(cfg) 발생
2. Orchestrator가 ProjectService.UpdateProjects(cfg) 호출
3. Orchestrator가 ProjectService.GetWatchTargets() → LogWatcher.UpdateTargets(targets) 호출
4. LogWatcher.OnLogEntry(entry) 발생
5. Orchestrator가 LogProcessor.AddEntry(entry) 호출
```

---

## 2. Config Schema Design

### 2.1 Daemon Config (Environment Variables)

**File**: `internal/platform/setup/config/config.go`

```go
type Config struct {
    App     AppConfig
    Logging LoggingConfig
    API     APIConfig
    Cops    CopsConfig
}

type AppConfig struct {
    Name    string `env:"COPS_APP_NAME" envDefault:"cops-daemon"`
    Version string `env:"COPS_APP_VERSION" envDefault:"0.0.1"`
    Env     string `env:"COPS_APP_ENV" envDefault:"development"`
    Debug   bool   `env:"COPS_DEBUG" envDefault:"false"`
}

type LoggingConfig struct {
    Level  string `env:"COPS_LOG_LEVEL" envDefault:"info"`
    Format string `env:"COPS_LOG_FORMAT" envDefault:"text"`
}

type APIConfig struct {
    URL     string `env:"COPS_API_URL" envDefault:"http://localhost:8080"`
    Timeout int    `env:"COPS_API_TIMEOUT" envDefault:"30"`
}

type CopsConfig struct {
    GlobalConfigPath string `env:"COPS_GLOBAL_CONFIG_PATH" envDefault:"~/.cops/config.json"`
    FlushInterval    int    `env:"COPS_FLUSH_INTERVAL" envDefault:"60"` // seconds
}
```

### 2.2 Global Config (~/.cops/config.json)

**File**: `internal/platform/domain/global_config.go`

CLI에서 `cops add <path>` 할 때 IsGitProject 등을 미리 판단해서 저장.

```go
// GlobalConfig represents ~/.cops/config.json
type GlobalConfig struct {
    Projects []ProjectConfig `json:"projects"`
}

type ProjectConfig struct {
    Path         string `json:"path"`                   // Project root directory
    Name         string `json:"name,omitempty"`         // Display name (optional)
    IsGitProject bool   `json:"isGitProject"`           // CLI에서 add할 때 판단
    Active       bool   `json:"active"`                 // Whether to watch this project
}
```

**Example ~/.cops/config.json**:
```json
{
  "projects": [
    {
      "path": "/Users/jayce/projects/my-app",
      "name": "My App",
      "isGitProject": true,
      "active": true
    },
    {
      "path": "/Users/jayce/projects/simple-script",
      "isGitProject": false,
      "active": true
    }
  ]
}
```

---

## 3. Domain Models

### 3.1 Shared 모듈에서 재사용 (import from `shared/domain`)

```go
import "github.com/team-attention/cops/shared/domain"

// 재사용 가능한 타입들:
// - domain.SessionRecord     → Claude Code JSONL 로그 엔트리
// - domain.SessionType       → user, assistant, file-history-snapshot 등
// - domain.Message           → 메시지 (role, content)
// - domain.MessageContent    → polymorphic content (string or []ContentBlock)
// - domain.ContentBlock      → text, tool_use, tool_result
// - domain.Project           → 프로젝트 (ID, Name, Path, GitProject, ClaudeDir)
// - domain.ProjectWithWorktrees → 프로젝트 + Worktrees
// - domain.Usage, domain.ToolUseResult
// - domain.ID
```

### 3.2 Daemon 전용 모델 (신규 생성)

**File**: `internal/platform/domain/watch.go`

```go
// WatchTarget represents a directory to watch for Claude Code logs
type WatchTarget struct {
    ProjectPath string          // Original project path from GlobalConfig
    ClaudeDir   string          // ~/.claude/projects/{encoded-path}
    Type        WatchTargetType
}

type WatchTargetType string
const (
    WatchTargetRoot         WatchTargetType = "root"         // 메인 프로젝트 디렉토리
    WatchTargetWorktree     WatchTargetType = "worktree"     // Git worktree 디렉토리
    WatchTargetSubdirectory WatchTargetType = "subdirectory" // 상위 디렉토리 watch 시 하위에서 Claude 실행된 경우
)

// FilePosition tracks read position for incremental file reading
type FilePosition struct {
    Path      string
    Offset    int64     // Last read byte offset
    UpdatedAt time.Time
}

// LogBatch for API transmission
type LogBatch struct {
    Records   []domain.SessionRecord  // shared 모듈 재사용
    DaemonID  string
    CreatedAt time.Time
}
```

---

## 4. Service Layer Design

### 4.1 ConfigWatcher Service

**Purpose**: Watch `~/.cops/config.json` and notify changes via callback

```go
// Port interface
type FileWatchPort interface {
    Watch(ctx context.Context, path string) error
    Events() <-chan FileEvent
    Errors() <-chan error
    Close() error
}

// Service
type Service struct {
    logger   *slog.Logger
    port     FileWatchPort
    path     string
    onChange func(GlobalConfig)
}

func NewService(l *slog.Logger, port FileWatchPort, cfg *config.Config) *Service
func (s *Service) Start(ctx context.Context) error
func (s *Service) Stop() error
func (s *Service) OnConfigChange(fn func(GlobalConfig))
func (s *Service) LoadConfig() (*GlobalConfig, error)  // 초기 로드용
```

### 4.2 Project Service

**Purpose**: GlobalConfig → WatchTarget[] 변환. Git project면 worktree 정보 수집.

```go
// Service (Port 없음 - gitutil 직접 사용)
type Service struct {
    logger       *slog.Logger
    watchTargets []WatchTarget
}

func NewService(l *slog.Logger) *Service
func (s *Service) UpdateProjects(cfg GlobalConfig) error
func (s *Service) GetWatchTargets() []WatchTarget

// Internal: Git project인 경우 .git/worktrees에서 worktree 경로들 수집
func (s *Service) resolveWorktrees(projectPath string) ([]string, error)
// Internal: 경로를 Claude 프로젝트 디렉토리로 변환
func (s *Service) toClaudeDir(path string) string
```

**Watch Target 생성 로직**:
```
for each project in GlobalConfig.Projects:
    if not project.Active:
        continue

    // 메인 프로젝트 디렉토리
    targets.append(WatchTarget{
        ProjectPath: project.Path,
        ClaudeDir:   toClaudeDir(project.Path),
        Type:        WatchTargetRoot,
    })

    // Git project인 경우 worktree들도 추가
    if project.IsGitProject:
        worktrees := resolveWorktrees(project.Path)
        for each wt in worktrees:
            targets.append(WatchTarget{
                ProjectPath: wt,
                ClaudeDir:   toClaudeDir(wt),
                Type:        WatchTargetWorktree,
            })
```

### 4.3 LogWatcher Service

**Purpose**: Watch Claude Code log directories for changes, read new log entries incrementally

```go
// Port interface (ConfigWatcher와 동일한 인터페이스 재사용 가능)
type FileWatchPort interface {
    Add(path string) error
    Remove(path string) error
    Events() <-chan FileEvent
    Errors() <-chan error
    Close() error
}

type FileEvent struct {
    Path string
    Op   FileOp  // Create, Write, Remove, etc.
}

// Service
type Service struct {
    logger        *slog.Logger
    port          FileWatchPort
    targets       []WatchTarget
    filePositions map[string]*FilePosition  // Track read positions per file
    onLogEntry    func(domain.SessionRecord)
    mu            sync.RWMutex
}

func NewService(l *slog.Logger, port FileWatchPort) *Service
func (s *Service) Start(ctx context.Context) error
func (s *Service) Stop() error
func (s *Service) UpdateTargets(targets []WatchTarget) error  // 동적으로 watch 대상 변경
func (s *Service) OnLogEntry(fn func(domain.SessionRecord))

// Internal methods
func (s *Service) handleFileEvent(event FileEvent)
func (s *Service) readNewLines(path string) ([]domain.SessionRecord, error)
func (s *Service) parseJSONLLine(line string) (*domain.SessionRecord, error)
```

**증분 읽기 로직**:
```
handleFileEvent(event):
    if event.Op != Write && event.Op != Create:
        return
    if !strings.HasSuffix(event.Path, ".jsonl"):
        return

    pos := s.filePositions[event.Path]
    if pos == nil:
        pos = &FilePosition{Path: event.Path, Offset: 0}

    // 마지막 위치부터 새로운 라인만 읽기
    newLines := readFromOffset(event.Path, pos.Offset)
    for each line in newLines:
        record := parseJSONLLine(line)
        s.onLogEntry(record)

    pos.Offset = currentFileSize
    pos.UpdatedAt = time.Now()
```

### 4.4 LogProcessor Service

**Purpose**: Buffer log entries and send to API server in batches

```go
// Port interface
type APIClientPort interface {
    SendLogs(ctx context.Context, batch LogBatch) error
}

// Service
type Service struct {
    logger        *slog.Logger
    apiPort       APIClientPort
    buffer        []domain.SessionRecord
    flushInterval time.Duration
    debug         bool
    daemonID      string
    mu            sync.Mutex
}

func NewService(l *slog.Logger, port APIClientPort, cfg *config.Config) *Service
func (s *Service) Start(ctx context.Context) error  // flush ticker 시작
func (s *Service) Stop(ctx context.Context) error   // 남은 버퍼 flush
func (s *Service) AddEntry(entry domain.SessionRecord)
func (s *Service) Flush(ctx context.Context) error

// Behavior:
// - DEBUG mode (cfg.App.Debug == true): AddEntry마다 즉시 Flush
// - Normal mode: flushInterval (기본 60초)마다 Flush
// - Stop() 시: 남은 버퍼 모두 Flush
```

---

## 5. fx Module Organization

**File**: `cmd/internal/container/application.go`

```go
func Run() {
    app := fx.New(
        fx.StartTimeout(30*time.Second),
        fx.StopTimeout(30*time.Second),

        newPlatformModule(),
        newWatcherModule(),
        newProcessorModule(),

        fx.Invoke(registerWatcher),  // Orchestrator 등록
    )
    app.Run()
}
```

**File**: `cmd/internal/container/module_platform.go`

```go
func newPlatformModule() fx.Option {
    return fx.Module("platform",
        fx.Provide(config.LoadConfig),
        fx.Provide(logger.InitLogger),
    )
}
```

**File**: `cmd/internal/container/module_watcher.go`

```go
func newWatcherModule() fx.Option {
    return fx.Module("watcher",
        // ConfigWatcher
        fx.Provide(
            fx.Annotate(
                filesystemadapter.NewFileWatchAdapter,
                fx.ResultTags(`name:"configFileWatch"`),
            ),
        ),
        fx.Provide(configwatcher.NewService),

        // Project Service
        fx.Provide(project.NewService),

        // LogWatcher
        fx.Provide(
            fx.Annotate(
                filesystemadapter.NewFileWatchAdapter,
                fx.ResultTags(`name:"logFileWatch"`),
            ),
        ),
        fx.Provide(logwatcher.NewService),
    )
}
```

**File**: `cmd/internal/container/module_processor.go`

```go
func newProcessorModule() fx.Option {
    return fx.Module("processor",
        fx.Provide(connectrpcadapter.NewAPIClientAdapter),
        fx.Provide(logprocessor.NewService),
    )
}
```

---

## 6. Watcher Lifecycle (Orchestrator)

**File**: `cmd/internal/container/register_watcher.go`

```go
type watcherParams struct {
    fx.In
    Lifecycle     fx.Lifecycle
    Logger        *slog.Logger
    Config        *config.Config
    ConfigWatcher *configwatcher.Service
    Project       *project.Service
    LogWatcher    *logwatcher.Service
    LogProcessor  *logprocessor.Service
}

func registerWatcher(p watcherParams) {
    // 1. 이벤트 핸들러 연결
    p.ConfigWatcher.OnConfigChange(func(cfg GlobalConfig) {
        p.Logger.Info("config changed, updating watch targets")
        if err := p.Project.UpdateProjects(cfg); err != nil {
            p.Logger.Error("failed to update projects", slog.Any("error", err))
            return
        }
        targets := p.Project.GetWatchTargets()
        if err := p.LogWatcher.UpdateTargets(targets); err != nil {
            p.Logger.Error("failed to update log watch targets", slog.Any("error", err))
        }
    })

    p.LogWatcher.OnLogEntry(func(record domain.SessionRecord) {
        p.LogProcessor.AddEntry(record)
    })

    // 2. Lifecycle 훅 등록
    p.Lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // 초기 설정 로드 및 서비스 시작
            cfg, err := p.ConfigWatcher.LoadConfig()
            if err != nil {
                return fmt.Errorf("failed to load initial config: %w", err)
            }

            if err := p.Project.UpdateProjects(*cfg); err != nil {
                return fmt.Errorf("failed to initialize projects: %w", err)
            }

            targets := p.Project.GetWatchTargets()
            if err := p.LogWatcher.UpdateTargets(targets); err != nil {
                return fmt.Errorf("failed to set initial watch targets: %w", err)
            }

            // 모든 서비스 시작
            if err := p.ConfigWatcher.Start(ctx); err != nil {
                return err
            }
            if err := p.LogWatcher.Start(ctx); err != nil {
                return err
            }
            if err := p.LogProcessor.Start(ctx); err != nil {
                return err
            }

            p.Logger.Info("daemon started", slog.Int("watchTargets", len(targets)))
            return nil
        },
        OnStop: func(ctx context.Context) error {
            p.Logger.Info("daemon stopping")

            // 역순으로 정리
            if err := p.LogProcessor.Stop(ctx); err != nil {
                p.Logger.Error("failed to stop log processor", slog.Any("error", err))
            }
            if err := p.LogWatcher.Stop(); err != nil {
                p.Logger.Error("failed to stop log watcher", slog.Any("error", err))
            }
            if err := p.ConfigWatcher.Stop(); err != nil {
                p.Logger.Error("failed to stop config watcher", slog.Any("error", err))
            }

            p.Logger.Info("daemon stopped")
            return nil
        },
    })
}
```

---

## 7. CLI Integration

Entry point에서 Cobra 명령 추가 가능:

```go
// cmd/daemon/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:   "cops-daemon",
        Short: "COps background daemon",
    }

    startCmd := &cobra.Command{
        Use:   "start",
        Short: "Start the daemon",
        Run: func(cmd *cobra.Command, args []string) {
            container.Run()
        },
    }

    installCmd := &cobra.Command{
        Use:   "install",
        Short: "Install as launchd service",
        Run: func(cmd *cobra.Command, args []string) {
            // Generate and install launchd plist
        },
    }

    uninstallCmd := &cobra.Command{
        Use:   "uninstall",
        Short: "Uninstall launchd service",
        Run: func(cmd *cobra.Command, args []string) {
            // Remove launchd plist
        },
    }

    rootCmd.AddCommand(startCmd, installCmd, uninstallCmd)
    rootCmd.Execute()
}
```

---

## 8. Files to Modify/Delete (from API module)

### Delete
- `internal/service/health/` - Health service not needed
- `internal/platform/setup/server/` - Fiber server not needed
- `cmd/internal/container/module_health.go`
- `cmd/internal/container/register_fiber.go`

### Modify
- `go.mod` - Change module name from `api` to `daemon`, add fsnotify
- `cmd/internal/container/application.go` - Replace modules
- `cmd/internal/container/module_platform.go` - Remove Fiber
- `internal/platform/setup/config/config.go` - New config structure

---

## 9. Dependencies to Add

```bash
go get github.com/fsnotify/fsnotify
go get github.com/spf13/cobra
```

---

## 10. Implementation Order

1. **Phase 1: Protobuf & Cleanup**
   - Create `idl/protobuf/daemon/v1/daemon.proto`
   - Run `buf generate` to generate Go code
   - Delete unused files (health service, Fiber)
   - Update go.mod module name to `daemon`

2. **Phase 2: Platform Setup**
   - Update config structure for daemon
   - Keep logger setup (already good)
   - Add dependencies: fsnotify, cobra

3. **Phase 3: Domain & Utils**
   - Import shared domain models (SessionRecord, Project, Message, etc.)
   - Create daemon-only models (GlobalConfig, WatchTarget, FilePosition, LogBatch)
   - Create pathutil (EncodePathForClaude, GetClaudeProjectDir)
   - Create gitutil (IsGitRepo, GetWorktrees)

4. **Phase 4: ConfigWatcher Service**
   - Implement filesystem adapter (fsnotify for ~/.cops/config.json)
   - Implement ConfigWatcher service

5. **Phase 5: ProjectManager Service**
   - Implement git adapter
   - Implement ProjectManager service (maps project paths to Claude log dirs)

6. **Phase 6: LogWatcher Service**
   - Implement file watcher adapter (fsnotify for .jsonl files)
   - Implement incremental file reader
   - Implement JSONL parser
   - Implement LogWatcher service

7. **Phase 7: LogProcessor Service**
   - Implement ConnectRPC API client adapter
   - Implement LogProcessor service with buffering & flush

8. **Phase 8: Wire Everything**
   - Create fx modules
   - Implement daemon lifecycle registration
   - Create CLI entry point (start command)

9. **Phase 9: launchd Integration**
   - Implement install command (generate plist)
   - Implement uninstall command
   - Add interactive prompt for launchd registration

---

## 11. Protobuf Definition (신규 생성 필요)

**File**: `idl/protobuf/daemon/v1/daemon.proto`

```protobuf
syntax = "proto3";

package daemon.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/daemon/v1;daemonv1";

// SessionRecord represents a single Claude Code JSONL entry
// (matches shared/domain/SessionRecord)
message SessionRecord {
  string uuid = 1;
  string parent_uuid = 2;
  string session_id = 3;
  string type = 4;                  // user | assistant | file-history-snapshot | etc
  string timestamp = 5;             // ISO 8601
  string cwd = 6;
  string git_branch = 7;
  string version = 8;               // Claude Code version
  string user_type = 9;
  bool is_sidechain = 10;
  bool is_meta = 11;
  string slug = 12;
  string request_id = 13;
  Message message = 14;
}

message Message {
  string id = 1;
  string type = 2;
  string role = 3;
  string model = 4;
  string content = 5;               // JSON serialized content
  string stop_reason = 6;
}

// LogBatch contains multiple session records for batch sending
message LogBatch {
  repeated SessionRecord records = 1;
  string daemon_id = 2;             // Daemon instance ID
  string created_at = 3;            // Batch creation timestamp
}

// SendLogsRequest is the request for sending logs
message SendLogsRequest {
  LogBatch batch = 1;
}

// SendLogsResponse is the response for sending logs
message SendLogsResponse {
  bool success = 1;
  string error_message = 2;
  int32 processed_count = 3;
}

// DaemonService handles daemon-to-API communication
service DaemonService {
  // SendLogs sends a batch of session records to the API server
  rpc SendLogs(SendLogsRequest) returns (SendLogsResponse);
}
```

**생성 후**:
```bash
cd idl/protobuf && buf generate
```

→ `shared/gen/grpcstub/daemon/v1/` 에 Go 코드 생성됨

---

## 12. Critical Files Summary

| File | Action | Purpose |
|------|--------|---------|
| `shared/domain/*.go` | **재사용** | SessionRecord, Project, Message 등 |
| `idl/protobuf/daemon/v1/daemon.proto` | Create | API 통신용 protobuf 정의 |
| `daemon/go.mod` | Modify | module name 변경, deps 추가, shared 참조 |
| `daemon/cmd/daemon/main.go` | Create | CLI entry point |
| `daemon/cmd/internal/container/application.go` | Modify | fx setup |
| `daemon/internal/platform/domain/global_config.go` | Create | GlobalConfig (daemon 전용) |
| `daemon/internal/platform/domain/watch.go` | Create | WatchTarget, FilePosition, LogBatch |
| `daemon/internal/platform/util/gitutil/gitutil.go` | Create | Git utilities |
| `daemon/internal/platform/util/pathutil/pathutil.go` | Create | Path encoding utility |
| `daemon/internal/service/configwatcher/` | Create | Config watcher |
| `daemon/internal/service/projectmanager/` | Create | Project management |
| `daemon/internal/service/logwatcher/` | Create | Log watcher |
| `daemon/internal/service/logprocessor/` | Create | Log processor |

---

## 13. Path Encoding Utility

**File**: `daemon/internal/platform/util/pathutil/pathutil.go`

```go
package pathutil

import "strings"

// EncodePathForClaude converts a file path to Claude Code's encoded format
// e.g., "/Users/jayce/project" → "-Users-jayce-project"
func EncodePathForClaude(path string) string {
    return strings.ReplaceAll(path, "/", "-")
}

// GetClaudeProjectDir returns the Claude Code project directory for a given path
// e.g., "/Users/jayce/project" → "~/.claude/projects/-Users-jayce-project"
func GetClaudeProjectDir(projectPath string) string {
    encoded := EncodePathForClaude(projectPath)
    return filepath.Join(os.Getenv("HOME"), ".claude", "projects", encoded)
}
```
