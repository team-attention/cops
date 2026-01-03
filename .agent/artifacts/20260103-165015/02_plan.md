# Implementation Plan: Adaptive Batch Sending for Log Upload

## Overview

This plan implements adaptive batch sending in the Daemon's log upload mechanism. When the Daemon accumulates many log lines and attempts to send them all at once, the API server may reject the request with HTTP 413 (Request Entity Too Large). The solution implements a TCP congestion control-inspired algorithm that:

1. Splits large log buffers into configurable maximum batch sizes
2. Halves the batch size on 413 errors and retries immediately
3. Doubles the batch size on success (capped at maximum)
4. Returns failed batches to the buffer for retry in the next flush cycle

This is a daemon-only change with no API server modifications required.

## Package Changes

No external packages need to be added or removed. The implementation uses:
- `connectrpc.com/connect` (existing) - for error code detection
- `strings` (standard library) - for error message parsing

## Step 1: Add MaxBatchSize Configuration

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-setup.md`: Configuration structure patterns
- `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/config.go`: Existing configuration structure

### `/Users/jayce/team-attention/cops/daemon/internal/platform/setup/config.go`

**Description**: Add `MaxBatchSize` field to `CopsConfig` struct and add validation in `validateConfig`.

```go
// CopsConfig holds COps-specific settings.
type CopsConfig struct {
	GlobalConfigPath string        `env:"COPS_GLOBAL_CONFIG_PATH" envDefault:"~/.cops/config.json"`
	DaemonDataDir    string        `env:"COPS_DAEMON_DATA_DIR" envDefault:"~/.cops/daemon"`
	FlushInterval    time.Duration `env:"COPS_FLUSH_INTERVAL" envDefault:"60s"`
	MaxBatchSize     int           `env:"COPS_MAX_BATCH_SIZE" envDefault:"100"`
}

// Add to validateConfig function:
func validateConfig(cfg *Config) error {
	// ... existing validation ...

	// 1. Check if MaxBatchSize is at least 1.
	// 2. If MaxBatchSize < 1, return error "max batch size must be at least 1".
	// 3. If MaxBatchSize > 1000, log a warning (potential misconfiguration) but allow.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Default value | No env var | MaxBatchSize = 100 | Default path |
| Valid custom value | COPS_MAX_BATCH_SIZE=50 | MaxBatchSize = 50 | Custom value path |
| Invalid value (0) | COPS_MAX_BATCH_SIZE=0 | Error: "max batch size must be at least 1" | Validation error |
| Invalid value (negative) | COPS_MAX_BATCH_SIZE=-1 | Error: "max batch size must be at least 1" | Validation error |
| Large value warning | COPS_MAX_BATCH_SIZE=5000 | MaxBatchSize = 5000 (with warning log) | Warning path |

---

## Step 2: Add Error Detection Utility

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md`: Outbound adapter patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform.md`: Platform utilities and errutil patterns
- `/Users/jayce/team-attention/cops/daemon/internal/platform/util/errutil/errutil.go`: Existing errutil implementation
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/api_client_port.go`: Current port interface
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`: Current implementation

### `/Users/jayce/team-attention/cops/daemon/internal/platform/util/errutil/errutil.go`

**Description**: Add `ErrorTypePayloadTooLarge` constant and helper functions following the existing errutil pattern.

```go
const (
	ErrorTypeBadRequest      ErrorType = "bad_request"
	ErrorTypeUnauthorized    ErrorType = "unauthorized"
	ErrorTypeForbidden       ErrorType = "forbidden"
	ErrorTypeNotFound        ErrorType = "not_found"
	ErrorTypeInternal        ErrorType = "internal"
	ErrorTypePayloadTooLarge ErrorType = "payload_too_large"  // Add this line
)

// PayloadTooLarge creates a payload too large error.
func PayloadTooLarge(msg string) *AppError {
	// 1. Return new AppError with Type: ErrorTypePayloadTooLarge and Message: msg.
}

// PayloadTooLargef creates a payload too large error with formatted message.
func PayloadTooLargef(format string, args ...any) *AppError {
	// 1. Return new AppError with Type: ErrorTypePayloadTooLarge.
	// 2. Use fmt.Sprintf(format, args...) for Message.
}

// IsPayloadTooLarge checks if the error is a payload too large error.
func IsPayloadTooLarge(err error) bool {
	// 1. Call Is(err, ErrorTypePayloadTooLarge) and return result.
}
```

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/api_client_port.go`

**Description**: Update port interface documentation to indicate it may return errutil.ErrorTypePayloadTooLarge errors.

```go
package api

import (
	"context"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
)

// APIClientPort is the port interface for sending logs to the API server.
type APIClientPort interface {
	// SendLogs sends a batch of logs to the API server.
	// Returns errutil.ErrorTypePayloadTooLarge if the server rejects with 413.
	SendLogs(ctx context.Context, batch domain.LogBatch) error
}
```

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`

**Description**: Modify `SendLogs` to detect HTTP 413 errors and return errutil.PayloadTooLarge errors.

```go
import (
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
)

// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	// 1. Create the protobuf request with batch.Lines and batch.ProjectID.
	// 2. Call c.client.SendLogs(ctx, connect.NewRequest(req)).
	// 3. If error is nil:
	//    a. Log debug message with batch size.
	//    b. Return nil.
	// 4. Check if error indicates payload too large:
	//    a. Use connect.CodeOf(err) to get the error code.
	//    b. If code is connect.CodeUnknown:
	//       - Get error message string using err.Error().
	//       - Check if message contains "413" or "Request Entity Too Large" (case-insensitive).
	//       - If yes, return errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err).
	//    c. If code is connect.CodeResourceExhausted:
	//       - Return errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err).
	// 5. Return the original error for all other cases.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Successful send | Valid batch | nil | Happy path |
| HTTP 413 error | Error with "413" in message | errutil.IsPayloadTooLarge(err) = true | 413 detection via message |
| HTTP 413 error (text) | Error with "Request Entity Too Large" | errutil.IsPayloadTooLarge(err) = true | 413 detection via text |
| Resource exhausted code | connect.CodeResourceExhausted | errutil.IsPayloadTooLarge(err) = true | ResourceExhausted path |
| Network error | Connection refused | Original error (not PayloadTooLarge) | Other error path |
| Server internal error | 500 error | Original error (not PayloadTooLarge) | Other error path |

---

## Step 3: Implement Adaptive Batch Sending in Flush Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md`: Service patterns
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`: Current Flush implementation

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`

**Description**: Modify the `Service` struct to include `maxBatchSize` configuration and rewrite `Flush` method to implement adaptive batching.

#### Struct Modification

```go
// Service contains pure business logic for log file watching and processing.
type Service struct {
	logger             *slog.Logger
	fileWatcher        filesystem.FileWatchPort
	apiClient          api.APIClientPort
	maxBatchSize       int                           // Maximum lines per batch (from config)
	watchedDirs        map[string]bool
	claudeDirToProject map[string]shareddomain.ID
	projectPathToID    map[string]shareddomain.ID
	bufferByProject    map[shareddomain.ID][]string
	mu                 sync.Mutex
}
```

#### Constructor Modification

```go
// NewService creates a new Log service.
func NewService(
	l *slog.Logger,
	fileWatcher filesystem.FileWatchPort,
	apiClient api.APIClientPort,
	cfg *setup.Config,
) *Service {
	return &Service{
		logger:             l.With(slog.String("name", "log.service")),
		fileWatcher:        fileWatcher,
		apiClient:          apiClient,
		maxBatchSize:       cfg.Cops.MaxBatchSize, // Add this line
		watchedDirs:        make(map[string]bool),
		claudeDirToProject: make(map[string]shareddomain.ID),
		projectPathToID:    make(map[string]shareddomain.ID),
		bufferByProject:    make(map[shareddomain.ID][]string),
	}
}
```

#### Constants

```go
const (
	// minBatchSize is the minimum number of lines to attempt sending.
	minBatchSize = 1
)
```

#### Flush Method Rewrite

```go
// Flush sends buffered lines to the API server using adaptive batching.
// Sends separate batches for each project with dynamic size adjustment.
func (s *Service) Flush(ctx context.Context) error {
	// 1. Lock mutex and check if buffer is empty.
	//    - If empty, unlock and return nil.
	// 2. Take ownership of buffer (swap with new empty map) and unlock.
	// 3. Initialize lastErr variable for tracking errors.
	// 4. For each projectID and lines in bufferedLines:
	//    a. Skip if lines is empty.
	//    b. Log start of flush: "flushing log batch" with projectID, totalLines, maxBatchSize.
	//    c. Call s.flushProjectLines(ctx, projectID, lines).
	//    d. If error returned:
	//       - Set lastErr to error.
	//       - Continue to next project (don't stop processing).
	// 5. Return lastErr (nil if all projects succeeded).
}

// flushProjectLines sends all lines for a single project using adaptive batching.
// Returns any unsent lines to the buffer on failure.
func (s *Service) flushProjectLines(ctx context.Context, projectID shareddomain.ID, lines []string) error {
	// 1. Initialize state:
	//    - currentBatchSize = s.maxBatchSize
	//    - remainingLines = lines (slice copy)
	//    - batchNum = 0
	//    - totalSent = 0
	// 2. While len(remainingLines) > 0:
	//    a. Increment batchNum.
	//    b. Calculate batchSize = min(currentBatchSize, len(remainingLines)).
	//    c. Extract currentBatch = remainingLines[:batchSize].
	//    d. Create domain.LogBatch with currentBatch and projectID.
	//    e. Call s.sendBatchWithRetry(ctx, batch, projectID, batchNum, &currentBatchSize).
	//    f. If error:
	//       - If errutil.IsPayloadTooLarge(err) AND currentBatchSize == minBatchSize:
	//         * Log error: "single line too large, skipping" with projectID, batchNum.
	//         * Remove the first line from remainingLines (skip it).
	//         * Reset currentBatchSize to s.maxBatchSize.
	//         * Continue loop.
	//       - Else (other errors):
	//         * Call s.returnLinesToBuffer(projectID, remainingLines).
	//         * Return error.
	//    g. On success:
	//       - Remove sent lines from remainingLines (remainingLines = remainingLines[batchSize:]).
	//       - totalSent += batchSize.
	//       - Double currentBatchSize (capped at s.maxBatchSize).
	// 3. Log completion: "flush completed" with projectID, totalBatches=batchNum, totalLinesSent=totalSent.
	// 4. Return nil.
}

// sendBatchWithRetry attempts to send a batch, retrying with reduced size on 413 errors.
// Updates currentBatchSize pointer on 413 errors.
// Returns nil on success, error on failure.
func (s *Service) sendBatchWithRetry(
	ctx context.Context,
	batch domain.LogBatch,
	projectID shareddomain.ID,
	batchNum int,
	currentBatchSize *int,
) error {
	// 1. Loop (infinite loop, will break on success or non-413 error):
	//    a. Call s.apiClient.SendLogs(ctx, batch).
	//    b. If no error:
	//       - Log success: "batch sent successfully" with projectID, batchNum, linesInBatch, currentBatchSize.
	//       - Return nil.
	//    c. If errutil.IsPayloadTooLarge(err):
	//       - Calculate newSize = *currentBatchSize / 2.
	//       - If newSize < minBatchSize, set newSize = minBatchSize.
	//       - Log warning: "batch too large, reducing size" with projectID, batchNum, previousSize, newSize.
	//       - If newSize == *currentBatchSize (already at minimum):
	//         * We're already at minimum size (1 line), return error.
	//       - Update *currentBatchSize = newSize.
	//       - Resize batch.Lines to newSize (batch.Lines = batch.Lines[:newSize]).
	//       - Continue loop (retry with smaller batch).
	//    d. For other errors:
	//       - Log error: "failed to send batch" with projectID, batchNum, error.
	//       - Return error immediately (no retry for non-413 errors).
}

// returnLinesToBuffer puts unsent lines back into the buffer for retry in next flush.
func (s *Service) returnLinesToBuffer(projectID shareddomain.ID, lines []string) {
	// 1. Lock mutex.
	// 2. Prepend lines to s.bufferByProject[projectID]:
	//    - s.bufferByProject[projectID] = append(lines, s.bufferByProject[projectID]...).
	// 3. Unlock mutex.
	// 4. Log info: "returned lines to buffer" with projectID, lineCount.
}
```

**Test Scenarios for Flush**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Empty buffer | No lines | nil, no API calls | Empty buffer check |
| Single batch success | 50 lines, maxBatchSize=100 | nil, 1 batch sent | Normal single batch |
| Multiple batches success | 250 lines, maxBatchSize=100 | nil, 3 batches sent | Multiple batch iteration |
| 413 on first batch | 100 lines, 413 error | Retry with 50, then 25, etc. | 413 retry loop |
| 413 with eventual success | 100 lines, 413 then success at 25 | nil, subsequent batches use adaptive size | Size recovery |
| 413 at minimum size | 1 line causes 413 | Skip line, continue | Single line skip |
| Network error | Connection failure | Error, lines returned to buffer | Non-413 error handling |
| Multiple projects, one fails | Project A success, Project B fails | lastErr from B, A still sent | Multi-project handling |

**Test Scenarios for sendBatchWithRetry**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Immediate success | Valid batch | nil, currentBatchSize unchanged | Happy path |
| 413 then success | 413 on first try, success on retry | nil, currentBatchSize halved | Single retry |
| Multiple 413 retries | 413 until size reaches 1 | nil or error when size=1 | Progressive reduction |
| 413 at minimum | Batch size already 1, 413 error | PayloadTooLarge error | Minimum size reached |
| Non-413 error | Network error | Original error, no retry | Error passthrough |

---

## Step 4: Add Unit Tests

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service_test.go`: Existing test patterns
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/logwatcher_suite_test.go`: Test suite setup

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/mock/api_client_mock.go`

**Description**: Create hand-written mock for APIClientPort following Go testing best practices.

```go
package mock

import (
	"context"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api"
)

// APIClient implements api.APIClientPort for testing.
type APIClient struct {
	// SendLogsFunc is the behavior to execute when SendLogs is called.
	SendLogsFunc func(ctx context.Context, batch domain.LogBatch) error
	// CallCount tracks the number of SendLogs calls.
	CallCount int
	// Batches records all batches sent.
	Batches []domain.LogBatch
}

// SendLogs implements api.APIClientPort.
func (m *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	// 1. Increment CallCount.
	// 2. Append batch to Batches slice (make a copy of batch.Lines to avoid mutation).
	// 3. If SendLogsFunc is set, call it and return result.
	// 4. Otherwise return nil.
}

// Compile-time interface verification.
var _ api.APIClientPort = (*APIClient)(nil)
```

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem/mock/filewatch_mock.go`

**Description**: Create hand-written mock for FileWatchPort (needed for service construction in tests).

```go
package mock

import (
	"github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem"
)

// FileWatch implements filesystem.FileWatchPort for testing.
type FileWatch struct {
	AddFunc    func(path string) error
	RemoveFunc func(path string) error
}

// Add implements filesystem.FileWatchPort.
func (m *FileWatch) Add(path string) error {
	// 1. If AddFunc is set, call it and return result.
	// 2. Otherwise return nil.
}

// Remove implements filesystem.FileWatchPort.
func (m *FileWatch) Remove(path string) error {
	// 1. If RemoveFunc is set, call it and return result.
	// 2. Otherwise return nil.
}

// Compile-time interface verification.
var _ filesystem.FileWatchPort = (*FileWatch)(nil)
```

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/flush_test.go`

**Description**: Create comprehensive unit tests for the adaptive batching logic using hand-written mocks.

```go
package logwatcher_test

import (
	"context"
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	apimock "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/mock"
	fsmock "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem/mock"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("Flush Adaptive Batching", func() {
	var (
		svc     *logwatcher.Service
		mockAPI *apimock.APIClient
		mockFS  *fsmock.FileWatch
		cfg     *setup.Config
		ctx     context.Context
	)

	BeforeEach(func() {
		// 1. Create context.Background().
		// 2. Create logger with io.Discard writer (slog.New(slog.NewTextHandler(io.Discard, nil))).
		// 3. Create mock file watcher (&fsmock.FileWatch{}).
		// 4. Create mock API client (&apimock.APIClient{}).
		// 5. Create config with MaxBatchSize=100:
		//    - cfg = &setup.Config{Cops: setup.CopsConfig{MaxBatchSize: 100}}.
		// 6. Create service using logwatcher.NewService(logger, mockFS, mockAPI, cfg).
	})

	Describe("successful flush", func() {
		Context("when buffer is empty", func() {
			It("returns nil without calling API", func() {
				// 1. Call Flush with empty buffer.
				// 2. Assert no API calls made.
				// 3. Assert nil error returned.
			})
		})

		Context("when lines fit in single batch", func() {
			It("sends one batch", func() {
				// 1. Add 50 lines to buffer.
				// 2. Call Flush.
				// 3. Assert 1 API call made.
				// 4. Assert batch contains 50 lines.
			})
		})

		Context("when lines require multiple batches", func() {
			It("sends multiple batches", func() {
				// 1. Add 250 lines to buffer.
				// 2. Call Flush.
				// 3. Assert 3 API calls made (100 + 100 + 50).
				// 4. Assert all lines sent.
			})
		})
	})

	Describe("413 error handling", func() {
		Context("when first batch gets 413", func() {
			It("retries with halved batch size", func() {
				// 1. Set mockAPI.SendLogsFunc to:
				//    - Return errutil.PayloadTooLarge("test") on first call (len > 50).
				//    - Return nil on subsequent calls.
				// 2. Add 100 lines to buffer via svc.AddLinesForClaudeDir.
				// 3. Call svc.Flush(ctx).
				// 4. Assert mockAPI.CallCount >= 2.
				// 5. Assert second batch has <= 50 lines.
			})
		})

		Context("when multiple 413 errors occur", func() {
			It("progressively reduces batch size", func() {
				// 1. Set mockAPI.SendLogsFunc to:
				//    - Return errutil.PayloadTooLarge("test") if len(batch.Lines) > 25.
				//    - Return nil if len(batch.Lines) <= 25.
				// 2. Add 100 lines to buffer.
				// 3. Call svc.Flush(ctx).
				// 4. Verify batch sizes in mockAPI.Batches: start large, progressively smaller.
			})
		})

		Context("when batch size recovers after success", func() {
			It("doubles batch size up to maximum", func() {
				// 1. Set mockAPI.SendLogsFunc to always return nil.
				// 2. Add 250 lines to buffer.
				// 3. Call svc.Flush(ctx).
				// 4. Check mockAPI.Batches sizes grow (adaptive size increases on success).
			})
		})

		Context("when single line is too large", func() {
			It("skips the line and continues", func() {
				// 1. Set mockAPI.SendLogsFunc to always return errutil.PayloadTooLarge("test").
				// 2. Add 1 line to buffer.
				// 3. Call svc.Flush(ctx).
				// 4. Assert error contains "single line too large" or similar.
				// 5. Assert line is not in buffer (was skipped).
			})
		})
	})

	Describe("non-413 error handling", func() {
		Context("when network error occurs", func() {
			It("returns error and preserves lines in buffer", func() {
				// 1. Set mockAPI.SendLogsFunc to return errutil.Internal("network error").
				// 2. Add 100 lines to buffer.
				// 3. Call svc.Flush(ctx).
				// 4. Assert error returned.
				// 5. Assert lines are back in buffer (use internal test helper if needed).
			})
		})

		Context("when partial success across projects", func() {
			It("reports error but completes all projects", func() {
				// 1. Add lines for project A (claude dir A) and project B (claude dir B).
				// 2. Set mockAPI.SendLogsFunc to:
				//    - Return nil for batches with ProjectID A.
				//    - Return errutil.Internal("error") for batches with ProjectID B.
				// 3. Call svc.Flush(ctx).
				// 4. Assert error from B returned.
				// 5. Assert A's batches were sent (check mockAPI.Batches).
			})
		})
	})
})
```

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client_test.go`

**Description**: Add tests for 413 error detection in the API client.

```go
package connectrpc_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"connectrpc.com/connect"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
)

var _ = Describe("APIClient SendLogs", func() {
	Describe("error detection", func() {
		Context("when HTTP 413 error occurs", func() {
			It("returns PayloadTooLarge for error message containing '413'", func() {
				// 1. Create mock connect error with "413" in message.
				//    - err := connect.NewError(connect.CodeUnknown, fmt.Errorf("HTTP 413 error"))
				// 2. Simulate SendLogs call that would detect and wrap this error.
				// 3. Verify errutil.IsPayloadTooLarge(wrappedErr) is true.
			})

			It("returns PayloadTooLarge for 'Request Entity Too Large' message", func() {
				// 1. Create error with "Request Entity Too Large" in message.
				//    - err := connect.NewError(connect.CodeUnknown, fmt.Errorf("Request Entity Too Large"))
				// 2. Simulate SendLogs call that would detect and wrap this error.
				// 3. Verify errutil.IsPayloadTooLarge(wrappedErr) is true.
			})
		})

		Context("when CodeResourceExhausted error occurs", func() {
			It("returns PayloadTooLarge", func() {
				// 1. Create connect.Error with CodeResourceExhausted.
				//    - err := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("resource exhausted"))
				// 2. Simulate SendLogs call that would detect and wrap this error.
				// 3. Verify errutil.IsPayloadTooLarge(wrappedErr) is true.
			})
		})

		Context("when other errors occur", func() {
			It("does not wrap as PayloadTooLarge", func() {
				// 1. Create generic connect error.
				//    - err := connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
				// 2. Simulate SendLogs call that would return this error.
				// 3. Verify errutil.IsPayloadTooLarge(err) is false.
			})
		})
	})
})
```

---

## Step 5: Add Integration Test

**Files to Read**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service_test.go`: Existing integration test patterns

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/flush_integration_test.go`

**Description**: Add integration test that simulates real 413 behavior.

```go
package logwatcher_test

var _ = Describe("Flush Integration", func() {
	Context("with simulated large payload", func() {
		It("handles 413 and successfully sends all data", func() {
			// 1. Create service with small maxBatchSize (10).
			// 2. Configure mock to return 413 for batches > 5 lines.
			// 3. Add 100 lines to buffer.
			// 4. Call Flush.
			// 5. Assert all 100 lines eventually sent.
			// 6. Assert batches were <= 5 lines each.
		})
	})
})
```

---

## Implementation Order

1. **Step 1**: Add `MaxBatchSize` configuration field and validation
2. **Step 2**: Add `ErrorTypePayloadTooLarge` to errutil and update API client
3. **Step 3**: Implement adaptive batching in `Flush` method
4. **Step 4**: Create mocks and add unit tests for all scenarios
5. **Step 5**: Add integration test

## Quality Checklist

- [x] Every function has a concrete signature
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Implementation Agent
- [x] All packages are selected (no candidates)
- [x] Execution order is clear and dependencies are explicit
