# Pre-PR Code Review

## Review Summary
- **Status**: PASS
- **Files Reviewed**: 4
- **Issues Found**: 0 (Critical: 0, Warning: 0, Info: 0)

## Files Reviewed

### `api/go.mod`

#### Changes Applied
- Added `github.com/bytedance/sonic v1.14.2` to direct dependencies
- Added `github.com/samber/lo v1.52.0` to direct dependencies
- Indirect dependencies correctly auto-populated by go get

#### Verification
- Dependencies installed via `go get` command as specified in plan (Step 1)
- Both packages are latest versions as of review date
- Dependencies align with existing usage in `daemon` and `cli` modules

#### Status: PASS

---

### `api/go.sum`

#### Changes Applied
- Checksums added for `bytedance/sonic` and all transitive dependencies
- Checksums added for `samber/lo`
- Standard go.sum entries for new dependencies

#### Status: PASS

---

### `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`

#### Changes Applied

**Import Section (lines 3-15):**
```go
import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)
```

- Import ordering follows Go conventions (stdlib, third-party, local)
- `github.com/bytedance/sonic` correctly added

**Content Serialization (lines 106-117):**
```go
if msg.Content != nil {
    contentBytes, err := sonic.Marshal(msg.Content)
    if err != nil {
        // Log warning but continue - don't fail the batch for one message
        slog.Warn("failed to serialize message content",
            slog.String("messageId", msg.ID),
            slog.Any("error", err),
        )
    } else {
        doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
    }
}
```

#### Verification Against Plan

| Plan Requirement | Implementation | Status |
|------------------|----------------|--------|
| Add `github.com/bytedance/sonic` import | Line 8: `"github.com/bytedance/sonic"` | PASS |
| Use `sonic.Marshal(msg.Content)` | Line 107: `contentBytes, err := sonic.Marshal(msg.Content)` | PASS |
| Check `msg.Content != nil` | Line 106: `if msg.Content != nil {` | PASS |
| Log warning on error | Lines 108-113: `slog.Warn("failed to serialize...")` | PASS |
| Store as `string(contentBytes)` | Line 115: `doc[...] = string(contentBytes)` | PASS |
| Non-fatal error handling | Error logged but batch continues | PASS |

#### Rule Compliance

- `.agent/rules/go/go-logging-conventions.md`: Uses `slog.Warn` with structured fields (slog.String, slog.Any)
- `.agent/rules/go/go-outbound.md`: Error handling follows "Log errors at service layer" pattern
- `.agent/rules/go/go-backend.md`: No parameter count violations

#### Status: PASS

---

### `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

#### Changes Applied

**Import Section (lines 3-20):**
```go
import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bytedance/sonic"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/platform/structure"
	"github.com/team-attention/cops/api/internal/platform/util/mongoutil"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)
```

- Import ordering follows Go conventions (stdlib, third-party, local)
- `github.com/bytedance/sonic` and `github.com/samber/lo` correctly added

**Content Deserialization (lines 462-474):**
```go
// Reconstruct content if available (supports both JSON and legacy plain text)
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    var mc shareddomain.MessageContent
    if err := sonic.Unmarshal([]byte(content), &mc); err != nil {
        // Fallback: treat as legacy plain text (backward compatibility)
        msg.Content = &shareddomain.MessageContent{
            Text:     lo.ToPtr(content),
            IsBlocks: false,
        }
    } else {
        msg.Content = &mc
    }
}
```

#### Verification Against Plan

| Plan Requirement | Implementation | Status |
|------------------|----------------|--------|
| Add `github.com/bytedance/sonic` import | Line 9: `"github.com/bytedance/sonic"` | PASS |
| Add `github.com/samber/lo` import | Line 10: `"github.com/samber/lo"` | PASS |
| Use `sonic.Unmarshal([]byte(content), &mc)` | Line 465: `sonic.Unmarshal([]byte(content), &mc)` | PASS |
| Fallback to legacy plain text on error | Lines 466-470: Creates MessageContent with `lo.ToPtr(content)` | PASS |
| Use `lo.ToPtr(content)` | Line 468: `lo.ToPtr(content)` | PASS |
| Set `IsBlocks: false` for fallback | Line 469: `IsBlocks: false` | PASS |
| Assign `&mc` on success | Line 472: `msg.Content = &mc` | PASS |
| Backward compatibility preserved | Fallback branch handles legacy records | PASS |

#### Rule Compliance

- `.agent/rules/go/go-struct.md`: Correctly uses pointer type for optional `msg.Content`
- `.agent/rules/go/go-outbound.md`: Follows repository adapter patterns
- `.agent/rules/go/go-backend.md`: No parameter count violations

#### Status: PASS

---

## Build Verification

```bash
go build ./api/...
```

**Result**: Build succeeded with no errors

---

## Test Scenarios Coverage

### Write Side (adapter.go)

| Scenario | Covered |
|----------|---------|
| Text content (IsBlocks=false) | Yes - `sonic.Marshal` calls `MarshalJSON()` |
| Block content (IsBlocks=true) | Yes - `sonic.Marshal` calls `MarshalJSON()` |
| Nil content | Yes - outer `if msg.Content != nil` check |
| Serialization error | Yes - warning logged, continues |

### Read Side (dashboard_repo.go)

| Scenario | Covered |
|----------|---------|
| JSON text content | Yes - `sonic.Unmarshal` succeeds |
| JSON block content | Yes - `sonic.Unmarshal` succeeds |
| Legacy plain text | Yes - fallback on unmarshal error |
| Empty string | Yes - outer `content != ""` check |

---

## Approval Notes

- All planned changes implemented correctly
- Dependencies installed via proper `go get` commands
- Import statements follow Go conventions (stdlib, third-party, local grouping)
- `adapter.go` uses `sonic.Marshal` for serialization as specified
- `dashboard_repo.go` uses `sonic.Unmarshal` and `lo.ToPtr` as specified
- Error handling implemented correctly (non-fatal on write, fallback on read)
- Backward compatibility maintained for legacy plain text records
- Code follows all applicable rules from `.agent/rules/`
- Build verification passed
- No unintended changes detected

**Ready for PR creation.**
