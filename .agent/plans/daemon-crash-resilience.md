# Daemon Crash Resilience Plan

## Problem
Daemon이 중간에 종료되면:
- 파일 읽기 위치(`filePositions`) 손실 → 재시작 시 기존 파일 건너뜀
- 버퍼 데이터(`buffer`) 손실 → flush 전 크래시 시 데이터 유실

## Solution
SQLite 기반 상태 저장소로 파일 위치를 persist하고, 크래시 복구 시 마지막 읽은 위치부터 재처리.

### Design Decisions
| 항목 | 결정 | 이유 |
|------|------|------|
| 저장소 | `mattn/go-sqlite3` | 8.9k stars, CGO 필요하지만 goreleaser로 빌드 가능 |
| ORM | Raw SQL (`database/sql`) | 테이블 1개라 단순, 필요시 entgo 마이그레이션 가능 |
| 버퍼 WAL | 구현 안함 | Checkpoint만으로 충분, 복잡도 최소화 |
| 중복 처리 | API에서 idempotent 처리 | Daemon 단순하게 유지 |
| 복구 전략 | 마지막 위치부터 끝까지 재전송 | 중복은 API가 처리 |
| 네이밍 | 규칙 따르기 | `state_repo.go` 형식 사용 |

### Data Flow (After)
```
JSONL Files → LogWatcher → LogProcessor → API
                  ↓
              SQLite DB (~/.cops/daemon/state.db)
              └── file_positions (path, offset, updated_at)
```

---

## Implementation

### Phase 1: Infrastructure

#### 1.1 Add SQLite dependency
```bash
cd daemon && go get github.com/mattn/go-sqlite3
```

#### 1.2 Create state repository port
**File: `daemon/internal/service/logwatcher/outbound/repository/state_repo_port.go`**
```go
package repository

// StateRepositoryPort defines persistence operations for daemon state
type StateRepositoryPort interface {
    LoadFilePositions(ctx context.Context) (map[string]*domain.FilePosition, error)
    SaveFilePosition(ctx context.Context, pos *domain.FilePosition) error
    DeleteFilePosition(ctx context.Context, path string) error
    Close() error
}
```

#### 1.3 Create SQLite adapter
**File: `daemon/internal/service/logwatcher/outbound/repository/sqlite/state_repo.go`**
```go
package sqlite

// SQLiteStateRepository implements StateRepositoryPort
type SQLiteStateRepository struct {
    db     *sql.DB
    logger *slog.Logger
}

// NewSQLiteStateRepository creates a new SQLite state repository.
// cfg *config.Config is injected for DaemonDataDir configuration.
func NewSQLiteStateRepository(l *slog.Logger, cfg *config.Config) (*SQLiteStateRepository, error) {
    dataDir := cfg.CopsConfig.DaemonDataDir  // Use config for path
    // Ensure directory exists
    // Open DB with WAL mode
    // Create table if not exists
}
```

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS file_positions (
    path TEXT PRIMARY KEY,
    offset INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);
```

### Phase 2: Integrate with LogWatcher

**Modify: `daemon/internal/service/logwatcher/logwatcher_service.go`**

1. **Constructor**: Inject `StateRepositoryPort` dependency
2. **Start()**: Load positions from SQLite before watching
3. **readNewLines()**: After reading, save position to SQLite
4. **scanExistingFiles()**: Use saved position if exists, otherwise start from current end

```go
// Before (memory only)
s.filePositions[path] = &domain.FilePosition{Offset: info.Size()}

// After (with persistence)
if saved, ok := s.savedPositions[path]; ok {
    s.filePositions[path] = saved  // Resume from saved position
} else {
    s.filePositions[path] = &domain.FilePosition{Offset: info.Size()}
}
```

### Phase 3: Config & DI

**Modify: `daemon/internal/platform/setup/config/config.go`**
```go
type CopsConfig struct {
    // existing...
    DaemonDataDir string `env:"COPS_DAEMON_DATA_DIR" envDefault:"~/.cops/daemon"`
}
```

**Modify: `daemon/cmd/internal/container/module_watcher.go`**
- Create SQLiteStateRepository
- Inject into LogWatcher constructor

### Phase 4: Update Rules (go-outbound.md)

**Modify: `.agent/rules/go/go-outbound.md`**

Add new section after "Constructor Names":

```markdown
## Constructor Dependency Injection

When an adapter requires configuration values:
- **DO**: Inject `cfg *config.Config` and access needed fields
- **DON'T**: Inject individual primitive values (e.g., `dataDir string`)

Example:
```go
// CORRECT: Inject config, access what you need
func NewSQLiteStateRepository(l *slog.Logger, cfg *config.Config) (*SQLiteStateRepository, error) {
    dataDir := cfg.CopsConfig.DaemonDataDir
    // ...
}

// WRONG: Don't inject individual primitive values
func NewSQLiteStateRepository(l *slog.Logger, dataDir string) (*SQLiteStateRepository, error) {
    // ...
}
```

This ensures:
1. Consistent DI pattern across all adapters
2. Easy to add new config fields without changing constructor signatures
3. Clear dependency on configuration module
```

---

## Files to Create
| File | Description |
|------|-------------|
| `daemon/internal/service/logwatcher/outbound/repository/state_repo_port.go` | StateRepositoryPort interface |
| `daemon/internal/service/logwatcher/outbound/repository/sqlite/state_repo.go` | SQLite implementation |

## Files to Modify
| File | Changes |
|------|---------|
| `daemon/go.mod` | Add `github.com/mattn/go-sqlite3` |
| `daemon/internal/platform/setup/config/config.go` | Add `DaemonDataDir` |
| `daemon/internal/service/logwatcher/logwatcher_service.go` | Use StateRepositoryPort |
| `daemon/cmd/internal/container/module_watcher.go` | Wire SQLiteStateRepository |
| `.agent/rules/go/go-outbound.md` | Add DI pattern rule |

---

## Recovery Flow

```
Daemon Start:
1. Open SQLite (~/.cops/daemon/state.db)
2. Load file_positions table → savedPositions map
3. For each watched file:
   - If in savedPositions: resume from saved offset
   - Else: start from file end (new file)
4. On each read: update position in SQLite
5. On file delete: remove from SQLite
```

## Edge Cases

| Case | Handling |
|------|----------|
| DB corruption | Log error, create new DB, start fresh |
| File shrunk (rotation) | Reset offset to 0 if current offset > file size |
| File deleted | Remove from SQLite |
| Duplicate records | API handles idempotently (has UUID) |

---

## Out of Scope
- WAL for buffered records (버퍼는 memory만 사용, 복구 시 재전송으로 해결)
- Retry mechanism improvements (현재 방식 유지)
- Dead letter queue