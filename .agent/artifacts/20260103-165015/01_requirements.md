# Requirements: Adaptive Batch Sending for Log Upload

## Request Summary

The Daemon currently sends all buffered logs to the API server in a single batch per project. When logs accumulate for a long period (e.g., daemon offline, infrequent flushes), the batch becomes too large and exceeds HTTP/gRPC payload limits, resulting in `413 Request Entity Too Large` errors. The solution is to implement adaptive batch sending with dynamic size adjustment based on server responses, similar to TCP congestion control algorithms.

## Acceptance Criteria

- [ ] Daemon splits large log buffers into multiple batches with configurable maximum size
- [ ] When 413 error occurs, batch size is reduced by 50% and request is retried
- [ ] On successful send, batch size is doubled for next batch (capped at configured maximum)
- [ ] Adaptive batching state persists across batches within a single flush operation
- [ ] Failed batches are returned to buffer for retry in next flush cycle
- [ ] Configuration supports setting maximum batch size via environment variable
- [ ] Logging includes batch count, current batch size, and success/failure information
- [ ] All buffered logs are eventually sent even when initial batches fail
- [ ] No changes required to API server (daemon-side solution only)

## Scope

### In Scope

**Daemon-side Implementation:**
- Split `Flush()` method to send logs in multiple batches
- Implement adaptive batch sizing algorithm with 413 error handling
- Add retry logic with exponential backoff for batch size reduction
- Add configuration for maximum batch size (environment variable)
- Add logging for batch operations (count, size, success/failure)
- Update buffer management to handle partial failures

**Configuration:**
- New environment variable: `COPS_MAX_BATCH_SIZE` (default: 100)
- Integrated into existing `daemon/internal/platform/setup/config.go`

**Testing Considerations:**
- Unit tests for adaptive batching logic
- Unit tests for buffer management with partial failures
- Integration test simulating 413 errors

### Out of Scope

**API Server Changes:**
- No changes to `api/internal/service/aggregation/` (passive recipient)
- No proto definition changes required
- No server-side validation or batch size limits

**Additional Features:**
- Maximum total buffer size limits (memory management)
- Persistent storage of adaptive state across daemon restarts
- Advanced metrics/monitoring beyond basic logging
- Batch compression or optimization
- Concurrent batch sending (sequential only)

## Problem Statement

### Current Behavior

**Code Location:** `daemon/internal/service/logwatcher/log_service.go` (lines 161-213)

The `Flush()` method:
1. Takes all buffered lines for a project
2. Creates a single `LogBatch` with all lines
3. Calls `apiClient.SendLogs()` once
4. On failure, puts all lines back in buffer

**Failure Scenario:**
```
daemon accumulates 5000 log lines over several hours
→ Flush() sends 5000 lines in one batch
→ API server rejects with 413 Request Entity Too Large
→ All 5000 lines returned to buffer
→ Next flush: same 413 error (infinite retry loop)
```

**Error Evidence:**
```
api-1  | time=2026-01-03T07:46:34.015Z level=ERROR msg="Request error"
         path=/aggregation.v1.AggregationService/SendLogs method=POST
         status=413 error="Request Entity Too Large"
```

### Root Cause

The daemon lacks:
1. **Batch size limits** - no maximum lines per request
2. **Adaptive retry** - no strategy for handling payload size errors
3. **Progressive sending** - all-or-nothing approach fails for large buffers

## Proposed Solution: Adaptive Batch Sending

Implement an adaptive batching algorithm inspired by TCP congestion control:

### Algorithm Overview

```
currentBatchSize = maxBatchSize (configured, default: 100)

for each batch in buffer:
    attempt = SendBatch(currentBatchSize lines)

    if attempt == 413 (Too Large):
        currentBatchSize = currentBatchSize / 2
        retry same batch with new size

    else if attempt == Success:
        currentBatchSize = min(currentBatchSize * 2, maxBatchSize)
        move to next batch

    else if attempt == Other Error:
        put batch back in buffer
        stop flushing (retry in next flush cycle)
```

### Detailed Behavior

**Phase 1: Slow Start (after 413 error)**
- Start with configured max (e.g., 100 lines)
- On 413: halve size (100 → 50 → 25 → 12 → 6...)
- Minimum: 1 line (always attempt to send at least 1)

**Phase 2: Congestion Avoidance (after success)**
- On success: double size (25 → 50 → 100)
- Cap at configured maximum (100)

**Example Execution:**
```
Buffer: 500 lines total
MaxBatchSize: 100

Batch 1: 100 lines → 413 error
  Retry: 50 lines → 413 error
  Retry: 25 lines → SUCCESS ✓

Batch 2: 50 lines (2x previous) → SUCCESS ✓

Batch 3: 100 lines (2x previous, capped) → SUCCESS ✓

Batch 4: 100 lines → SUCCESS ✓

Batch 5: 100 lines → SUCCESS ✓

Batch 6: 25 lines (remaining) → SUCCESS ✓
```

### State Management

**Per-Flush State:**
- `currentBatchSize`: Adaptive size (resets to max at start of each `Flush()`)
- `remainingLines`: Lines not yet sent for current project
- `retriedLines`: Lines being retried after 413

**Cross-Flush State:**
- Failed batches return to `bufferByProject` for next flush cycle
- No persistent state across daemon restarts (starts fresh)

## Implementation Details

### Code Changes

**File:** `daemon/internal/service/logwatcher/log_service.go`

**Modify:** `Flush(ctx context.Context) error` method

**New Logic:**
```go
func (s *Service) Flush(ctx context.Context) error {
    // 1. Take ownership of buffer (existing)
    // 2. For each project's lines:
    //    a. Initialize currentBatchSize = maxBatchSize
    //    b. While lines remain:
    //       - Take min(currentBatchSize, len(remainingLines)) lines
    //       - Send batch
    //       - If 413: halve currentBatchSize, retry same batch
    //       - If success: double currentBatchSize (capped), move to next batch
    //       - If other error: return remaining lines to buffer, continue to next project
    // 3. Log results per project
}
```

**File:** `daemon/internal/platform/setup/config.go`

**Add to `CopsConfig`:**
```go
type CopsConfig struct {
    // ... existing fields ...
    MaxBatchSize int `env:"COPS_MAX_BATCH_SIZE" envDefault:"100"`
}
```

**Validation:**
- `MaxBatchSize` must be >= 1
- Warning if > 1000 (likely misconfiguration)

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `COPS_MAX_BATCH_SIZE` | `100` | Maximum number of JSONL lines per batch sent to API server |

**Usage:**
```bash
export COPS_MAX_BATCH_SIZE=50
./daemon
```

### Logging Requirements

**Log Events:**

1. **Start of flush per project:**
   ```
   level=INFO msg="flushing log batch"
     projectID=<uuid> totalLines=500 maxBatchSize=100
   ```

2. **Successful batch send:**
   ```
   level=INFO msg="batch sent successfully"
     projectID=<uuid> batchNum=1 linesInBatch=100 currentBatchSize=100
   ```

3. **413 error and retry:**
   ```
   level=WARN msg="batch too large, reducing size"
     projectID=<uuid> batchNum=1 previousSize=100 newSize=50
   ```

4. **Other errors:**
   ```
   level=ERROR msg="failed to send batch"
     projectID=<uuid> batchNum=1 error=<error>
   ```

5. **Completion summary:**
   ```
   level=INFO msg="flush completed"
     projectID=<uuid> totalBatches=5 totalLinesSent=500
   ```

## Error Handling

### 413 Too Large
- **Action:** Reduce batch size by 50%, retry immediately
- **Minimum:** 1 line (if even 1 line fails with 413, log error and skip)
- **Recovery:** Adaptive increase on subsequent successes

### Network/Connection Errors
- **Action:** Return unsent lines to buffer, stop flushing this project
- **Recovery:** Next flush cycle (controlled by `COPS_FLUSH_INTERVAL`)

### Other HTTP Errors (4xx, 5xx)
- **Action:** Same as network errors
- **Logging:** ERROR level with full error details

### Partial Failure
- **Scenario:** Project A sends successfully, Project B fails
- **Action:** Continue processing all projects, report individual errors

## Constraints

**Technical:**
- Must maintain existing gRPC/ConnectRPC interface (no proto changes)
- Must preserve existing buffer-on-failure semantics
- Adaptive state does not persist across daemon restarts

**Performance:**
- Maximum 10 retries per batch (prevent infinite 413 loops)
- Sequential batching only (no parallel sends)
- Flush completes within `COPS_API_TIMEOUT` * (totalLines / maxBatchSize)

**Backward Compatibility:**
- No API server changes required
- Existing daemon behavior preserved (just adds batching)

## Additional Context

### Related Code Locations

**Daemon:**
- `daemon/internal/service/logwatcher/log_service.go` - Main flush logic
- `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` - API client
- `daemon/internal/platform/setup/config.go` - Configuration

**API Server:**
- `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` - Receives batches (no changes)

**Proto:**
- `idl/protobuf/aggregation/v1/aggregation.proto` - `SendLogsReq/Res` definitions (no changes)

### Design Decisions

**Why adaptive vs fixed batching?**
- Fixed batching (e.g., always 100) fails if server limits change
- Adaptive approach self-adjusts to server capacity
- Mimics proven TCP congestion control strategies

**Why daemon-side only?**
- API server is passive receiver (no validation logic needed)
- Simpler deployment (no API changes)
- Daemon has full context of buffer size

**Why not persistent state?**
- Adds complexity (state storage, recovery)
- Fresh start after restart is acceptable
- Algorithm converges quickly (2-3 batches typically)

### Dependencies

**No new dependencies required:**
- Uses existing `apiClient.SendLogs()` interface
- Uses existing `setup.Config` pattern
- Uses existing error handling patterns

## Questions Resolved

| Question | Answer |
|----------|--------|
| What should be the maximum batch size? | Default: 100 lines, configurable via `COPS_MAX_BATCH_SIZE` environment variable |
| How to handle 413 errors? | Adaptive batching: halve size on 413, double on success (similar to TCP congestion control) |
| Should API validate batch size? | No, daemon-side solution only. API remains passive receiver |
| What about logging? | Basic logging: batch count, size, success/failure per batch |
| Need backward compatibility? | No concerns (not a live service, can modify freely) |
| Minimum batch size? | 1 line (always attempt to send at least 1 line) |
| Maximum retries per batch? | 10 retries (prevent infinite loops if server always rejects) |
| Should state persist across restarts? | No, algorithm converges quickly so fresh start is acceptable |
