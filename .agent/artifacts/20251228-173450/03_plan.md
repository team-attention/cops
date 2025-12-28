# Implementation Plan

## Overview

Move JSONL parsing logic from Daemon to API Server. Daemon will send raw JSONL text lines via gRPC, and API will handle parsing using `sonic.Unmarshal` into `shareddomain.SessionRecord` before saving to MongoDB.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| JSON Parsing | sonic | `/bytedance/sonic` | Already used in project, high performance |
| gRPC Communication | connect-go | `/connectrpc/connect-go` | Already configured with buf |

No new packages required - all dependencies are already present.

## Architecture Decisions

### Decision 1: Protobuf Schema Simplification
**Choice**: Replace `repeated SessionRecord records` with `repeated string jsonl` in `LogBatch` message. Remove all unused protobuf messages (`SessionRecord`, `Message`, `Usage`, `ContentBlock`, etc.).
**Rationale**: Raw JSONL strings are simpler to transmit, and the complex protobuf messages are no longer needed since parsing moves to API.

### Decision 2: Buffer Type Change in Daemon
**Choice**: Change buffer from `map[ID][]SessionRecord` to `map[ID][]string`.
**Rationale**: Daemon no longer parses JSONL, so it buffers raw strings instead of parsed objects.

### Decision 3: Error Handling Strategy
**Choice**: Fire & Forget - API logs parse errors at ERROR level but always returns success to Daemon.
**Rationale**: Per requirements, invalid lines are skipped and valid lines processed. Daemon should not retry based on parse errors.

### Decision 4: Parsing Location
**Choice**: Parse JSONL in API handler layer (`handler.go`), not service layer.
**Rationale**: Parsing is a transport concern (converting wire format to domain). Service layer receives already-parsed `repository.LogBatch`.

## Implementation Steps

### Step 1: Update Protobuf Schema

**Files to Modify**:
- `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto` (modify)

**Changes**:
Remove lines 9-103 (all messages except LogBatch, SendLogsReq, SendLogsRes, and AggregationService).
Replace `LogBatch` message.

**Before** (lines 105-109):
```protobuf
// LogBatch contains multiple session records for batch sending.
message LogBatch {
  repeated SessionRecord records = 1;
  string project_id = 2;
}
```

**After**:
```protobuf
// LogBatch contains raw JSONL lines for batch sending.
message LogBatch {
  repeated string jsonl = 1;
  string project_id = 2;
}
```

**Complete new file content**:
```protobuf
syntax = "proto3";

package aggregation.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1;aggregationv1";

// LogBatch contains raw JSONL lines for batch sending.
message LogBatch {
  repeated string jsonl = 1;
  string project_id = 2;
}

// SendLogsReq is the request for sending logs.
message SendLogsReq {
  LogBatch batch = 1;
}

// SendLogsRes is the response for sending logs.
message SendLogsRes {
  bool success = 1;
  string error_message = 2;
  int32 processed_count = 3;
}

// AggregationService handles log aggregation from daemons.
service AggregationService {
  // SendLogs sends a batch of raw JSONL lines to the API server.
  rpc SendLogs(SendLogsReq) returns (SendLogsRes);
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| N/A - Schema change | - | - | Compilation check |

---

### Step 2: Run buf generate

**Command**:
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Files Regenerated**:
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect/aggregation.connect.go`
- `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts` (if exists)

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Generation succeeds | buf generate | Exit code 0 | Happy path |

---

### Step 3: Update Daemon Domain (LogBatch)

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` (modify)

**Before** (lines 36-40):
```go
// LogBatch contains multiple session records for API transmission.
type LogBatch struct {
	Records   []shareddomain.SessionRecord // Session records from shared domain
	ProjectID shareddomain.ID              // Project ID (for aggregation API)
}
```

**After**:
```go
// LogBatch contains raw JSONL lines for API transmission.
type LogBatch struct {
	Lines     []string        // Raw JSONL lines (unparsed)
	ProjectID shareddomain.ID // Project ID (for aggregation API)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| N/A - Type change | - | - | Compilation check |

---

### Step 4: Update Daemon Service (log_service.go)

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` (modify)

**Change 1: Import removal** (line 12)
Remove:
```go
	"github.com/bytedance/sonic"
```

**Change 2: Buffer type** (line 29)
**Before**:
```go
	bufferByProject    map[shareddomain.ID][]shareddomain.SessionRecord
```
**After**:
```go
	bufferByProject    map[shareddomain.ID][]string
```

**Change 3: Constructor buffer initialization** (line 46)
**Before**:
```go
		bufferByProject:    make(map[shareddomain.ID][]shareddomain.SessionRecord),
```
**After**:
```go
		bufferByProject:    make(map[shareddomain.ID][]string),
```

**Change 4: HandleFileChange method** (lines 103-150)
**Before**:
```go
// HandleFileChange handles a file change event and returns parsed records.
// Called by Inbound (fsnotify) when a log file is modified.
func (s *Service) HandleFileChange(path string, fromOffset int64) ([]shareddomain.SessionRecord, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fromOffset, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to last known position
	if _, err := file.Seek(fromOffset, io.SeekStart); err != nil {
		return nil, fromOffset, fmt.Errorf("failed to seek file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var records []shareddomain.SessionRecord
	newOffset := fromOffset

	for scanner.Scan() {
		line := scanner.Text()
		newOffset += int64(len(line)) + 1 // +1 for newline

		if line == "" {
			continue
		}

		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			s.logger.Debug("failed to parse JSONL line",
				slog.String("path", path),
				slog.Any("error", err),
			)
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return records, newOffset, fmt.Errorf("error reading file: %w", err)
	}

	return records, newOffset, nil
}
```

**After**:
```go
// HandleFileChange handles a file change event and returns raw JSONL lines.
// Called by Inbound (fsnotify) when a log file is modified.
func (s *Service) HandleFileChange(path string, fromOffset int64) ([]string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fromOffset, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Seek to last known position
	if _, err := file.Seek(fromOffset, io.SeekStart); err != nil {
		return nil, fromOffset, fmt.Errorf("failed to seek file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for potentially large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []string
	newOffset := fromOffset

	for scanner.Scan() {
		line := scanner.Text()
		newOffset += int64(len(line)) + 1 // +1 for newline

		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return lines, newOffset, fmt.Errorf("error reading file: %w", err)
	}

	return lines, newOffset, nil
}
```

**Change 5: AddRecordsForClaudeDir -> AddLinesForClaudeDir** (lines 152-159)
**Before**:
```go
// AddRecordsForClaudeDir adds records to the buffer, associating them with the given ClaudeDir.
func (s *Service) AddRecordsForClaudeDir(claudeDir string, records []shareddomain.SessionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectID := s.claudeDirToProject[claudeDir]
	s.bufferByProject[projectID] = append(s.bufferByProject[projectID], records...)
}
```

**After**:
```go
// AddLinesForClaudeDir adds raw JSONL lines to the buffer, associating them with the given ClaudeDir.
func (s *Service) AddLinesForClaudeDir(claudeDir string, lines []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	projectID := s.claudeDirToProject[claudeDir]
	s.bufferByProject[projectID] = append(s.bufferByProject[projectID], lines...)
}
```

**Change 6: Flush method** (lines 161-215)
**Before**:
```go
// Flush sends buffered records to the API server.
// Sends separate batches for each project.
func (s *Service) Flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.bufferByProject) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Take ownership of buffer
	bufferedRecords := s.bufferByProject
	s.bufferByProject = make(map[shareddomain.ID][]shareddomain.SessionRecord)
	s.mu.Unlock()

	var totalCount int
	var lastErr error

	for projectID, records := range bufferedRecords {
		if len(records) == 0 {
			continue
		}

		batch := domain.LogBatch{
			Records:   records,
			ProjectID: projectID,
		}

		s.logger.Info("flushing log batch",
			slog.Int("count", len(records)),
			slog.String("projectID", projectID.String()),
		)

		if err := s.apiClient.SendLogs(ctx, batch); err != nil {
			// Put records back in buffer on failure
			s.mu.Lock()
			s.bufferByProject[projectID] = append(records, s.bufferByProject[projectID]...)
			s.mu.Unlock()

			lastErr = fmt.Errorf("failed to send logs for project %s: %w", projectID.String(), err)
			s.logger.Error("failed to send logs",
				slog.String("projectID", projectID.String()),
				slog.Any("error", err),
			)
			continue
		}

		totalCount += len(records)
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}
```

**After**:
```go
// Flush sends buffered lines to the API server.
// Sends separate batches for each project.
func (s *Service) Flush(ctx context.Context) error {
	s.mu.Lock()
	if len(s.bufferByProject) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Take ownership of buffer
	bufferedLines := s.bufferByProject
	s.bufferByProject = make(map[shareddomain.ID][]string)
	s.mu.Unlock()

	var totalCount int
	var lastErr error

	for projectID, lines := range bufferedLines {
		if len(lines) == 0 {
			continue
		}

		batch := domain.LogBatch{
			Lines:     lines,
			ProjectID: projectID,
		}

		s.logger.Info("flushing log batch",
			slog.Int("count", len(lines)),
			slog.String("projectID", projectID.String()),
		)

		if err := s.apiClient.SendLogs(ctx, batch); err != nil {
			// Put lines back in buffer on failure
			s.mu.Lock()
			s.bufferByProject[projectID] = append(lines, s.bufferByProject[projectID]...)
			s.mu.Unlock()

			lastErr = fmt.Errorf("failed to send logs for project %s: %w", projectID.String(), err)
			s.logger.Error("failed to send logs",
				slog.String("projectID", projectID.String()),
				slog.Any("error", err),
			)
			continue
		}

		totalCount += len(lines)
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Read new lines from file | JSONL file with 3 lines | Returns 3 raw strings | Happy path |
| Empty lines skipped | File with empty lines | Skips empty, returns non-empty | Empty line filter |
| File open error | Non-existent path | Returns error | Error handling |
| Buffer adds lines | Call AddLinesForClaudeDir | Lines appended to buffer | Buffer accumulation |
| Flush empty buffer | No buffered lines | Returns nil immediately | Empty buffer |
| Flush with lines | Buffered lines exist | Sends to API, clears buffer | Happy path |
| Flush API error | API returns error | Lines returned to buffer | Error recovery |

---

### Step 5: Update Daemon API Client

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` (modify)

**Change 1: Remove unused imports** (lines 7-8)
Remove:
```go
	"google.golang.org/protobuf/types/known/timestamppb"
```
And:
```go
	shareddomain "github.com/team-attention/cops/shared/domain"
```

**Change 2: SendLogs method** (lines 36-59)
**Before**:
```go
// SendLogs sends a batch of logs to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			Records:   convertRecords(batch.Records),
			ProjectId: batch.ProjectID.String(),
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}
```

**After**:
```go
// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			Jsonl:     batch.Lines,
			ProjectId: batch.ProjectID.String(),
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}
```

**Change 3: Remove all conversion functions** (lines 61-137)
Delete the following functions entirely:
- `convertRecords()`
- `convertSessionRecord()`
- `convertSessionType()`
- `convertMessage()`

**Complete new file content**:
```go
package connectrpc

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// APIClient implements APIClientPort using ConnectRPC.
type APIClient struct {
	logger *slog.Logger
	client aggregationv1connect.AggregationServiceClient
}

// NewAPIClient creates a new ConnectRPC API client adapter.
func NewAPIClient(l *slog.Logger, apiClient *setup.APIClient, cfg *setup.Config) *APIClient {
	client := aggregationv1connect.NewAggregationServiceClient(
		apiClient.StandardHTTPClient(),
		cfg.API.URL,
	)

	return &APIClient{
		logger: l.With(slog.String("name", "log.api.connectrpc")),
		client: client,
	}
}

// SendLogs sends a batch of raw JSONL lines to the API server.
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
	req := &aggregationv1.SendLogsReq{
		Batch: &aggregationv1.LogBatch{
			Jsonl:     batch.Lines,
			ProjectId: batch.ProjectID.String(),
		},
	}

	resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
	if err != nil {
		return err
	}

	if !resp.Msg.Success {
		c.logger.Warn("API returned failure")
	}

	c.logger.Debug("logs sent",
		slog.Int("processed", int(resp.Msg.ProcessedCount)),
	)

	return nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Send lines successfully | LogBatch with lines | Returns nil, logs processed count | Happy path |
| API connection error | Network failure | Returns error | Error handling |
| API returns failure | Success=false | Logs warning, returns nil | Partial success |

---

### Step 6: Update Caller of AddRecordsForClaudeDir

**Files to Search and Modify**:
Search for callers:
```bash
grep -rn "AddRecordsForClaudeDir\|HandleFileChange" /Users/jayce/team-attention/cops/daemon/internal/
```

The caller is likely in `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/fsnotify/` or similar.

**Changes Required**:
- Change `AddRecordsForClaudeDir` to `AddLinesForClaudeDir`
- Update variable names from `records` to `lines` if applicable
- The return type of `HandleFileChange` changes from `[]shareddomain.SessionRecord` to `[]string`

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| N/A - Caller update | - | - | Compilation check |

---

### Step 7: Update API Handler (Add JSONL Parsing)

**Files to Modify**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` (modify)

**Change 1: Add sonic and fmt imports** (after line 4)
Add:
```go
	"fmt"

	"github.com/bytedance/sonic"
```

**Change 2: SendLogs method** (lines 36-59)
**Before**:
```go
// SendLogs implements aggregationv1connect.AggregationServiceHandler.
func (h *AggregationGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&aggregationv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch := convertToDomain(pbBatch)
	result := h.svc.CollectLogs(ctx, batch)

	res := &aggregationv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}
```

**After**:
```go
// SendLogs implements aggregationv1connect.AggregationServiceHandler.
func (h *AggregationGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&aggregationv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch, parseErrors := h.parseJSONLLines(pbBatch.GetJsonl(), pbBatch.GetProjectId())

	// Log parse errors at ERROR level (Fire & Forget)
	if len(parseErrors) > 0 {
		h.logger.Error("failed to parse some JSONL lines",
			slog.String("projectId", pbBatch.GetProjectId()),
			slog.Int("failedCount", len(parseErrors)),
			slog.Int("totalCount", len(pbBatch.GetJsonl())),
			slog.String("sampleError", parseErrors[0].Error()),
		)
	}

	result := h.svc.CollectLogs(ctx, batch)

	res := &aggregationv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}
```

**Change 3: Add parseJSONLLines function** (add after SendLogs method)
```go
// parseJSONLLines parses raw JSONL lines into SessionRecord domain objects.
// Returns the parsed batch and any parse errors encountered.
func (h *AggregationGRPCHandler) parseJSONLLines(lines []string, projectID string) (*repository.LogBatch, []error) {
	var records []shareddomain.SessionRecord
	var parseErrors []error

	for _, line := range lines {
		if line == "" {
			continue
		}

		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("parse error: %s (line: %.100s...)", err.Error(), line))
			continue
		}

		records = append(records, record)
	}

	return &repository.LogBatch{
		Records:   records,
		ProjectID: projectID,
	}, parseErrors
}
```

**Change 4: Remove all old conversion functions** (lines 61-136)
Delete the following functions entirely:
- `convertToDomain()`
- `convertSessionType()`
- `convertMessage()`

**Complete new file content**:
```go
package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/bytedance/sonic"

	aggregationservice "github.com/team-attention/cops/api/internal/service/aggregation"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect"
)

// AggregationGRPCHandler handles aggregation service gRPC endpoints.
type AggregationGRPCHandler struct {
	svc    *aggregationservice.Service
	logger *slog.Logger
}

// NewAggregationGRPCHandler creates a new aggregation gRPC handler.
func NewAggregationGRPCHandler(l *slog.Logger, svc *aggregationservice.Service) *AggregationGRPCHandler {
	return &AggregationGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "aggregation.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *AggregationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return aggregationv1connect.NewAggregationServiceHandler(h, opts...)
}

// SendLogs implements aggregationv1connect.AggregationServiceHandler.
func (h *AggregationGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&aggregationv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch, parseErrors := h.parseJSONLLines(pbBatch.GetJsonl(), pbBatch.GetProjectId())

	// Log parse errors at ERROR level (Fire & Forget)
	if len(parseErrors) > 0 {
		h.logger.Error("failed to parse some JSONL lines",
			slog.String("projectId", pbBatch.GetProjectId()),
			slog.Int("failedCount", len(parseErrors)),
			slog.Int("totalCount", len(pbBatch.GetJsonl())),
			slog.String("sampleError", parseErrors[0].Error()),
		)
	}

	result := h.svc.CollectLogs(ctx, batch)

	res := &aggregationv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}

// parseJSONLLines parses raw JSONL lines into SessionRecord domain objects.
// Returns the parsed batch and any parse errors encountered.
func (h *AggregationGRPCHandler) parseJSONLLines(lines []string, projectID string) (*repository.LogBatch, []error) {
	var records []shareddomain.SessionRecord
	var parseErrors []error

	for _, line := range lines {
		if line == "" {
			continue
		}

		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("parse error: %s (line: %.100s...)", err.Error(), line))
			continue
		}

		records = append(records, record)
	}

	return &repository.LogBatch{
		Records:   records,
		ProjectID: projectID,
	}, parseErrors
}

// Compile-time interface verification.
var _ aggregationv1connect.AggregationServiceHandler = (*AggregationGRPCHandler)(nil)
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Valid JSONL lines | 3 valid JSON lines | Returns batch with 3 records, no errors | Happy path |
| Empty batch | nil batch | Returns error response | Nil check |
| All lines valid | Valid JSON array | Parses all, no parse errors | All valid |
| Some lines invalid | Mix of valid/invalid | Parses valid, logs errors for invalid | Partial parse |
| All lines invalid | All malformed JSON | Returns empty batch, logs all errors | All invalid |
| Empty lines skipped | Lines with empty strings | Skips empty, parses non-empty | Empty filter |
| Truncated error sample | Very long invalid line | Truncates to 100 chars in error | Error truncation |

---

### Step 8: Verify sonic dependency in API module

**Command**:
```bash
cd /Users/jayce/team-attention/cops/api && grep "sonic" go.mod
```

If not present, add it:
```bash
cd /Users/jayce/team-attention/cops/api && go get github.com/bytedance/sonic
```

---

### Step 9: Update Daemon Tests

**Files to Modify**:
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service_test.go` (keep as-is)

The current test file tests JSONL parsing using `sonic.Unmarshal` directly on files. This test validates the `shareddomain.SessionRecord` parsing logic which is now used by API. The test should continue to pass and validates that the parsing logic works correctly.

**Note**: No changes needed to this file. It tests the shared domain parsing which is still valid.

---

### Step 10: Add API Handler Unit Tests

**Files to Create**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go` (create)

```go
package connectrpc_test

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

func TestParseJSONLLines_ValidLines(t *testing.T) {
	lines := []string{
		`{"uuid":"1","type":"user","sessionId":"s1","timestamp":"2024-01-01T00:00:00Z"}`,
		`{"uuid":"2","type":"assistant","sessionId":"s1","timestamp":"2024-01-01T00:00:01Z"}`,
	}

	var records []shareddomain.SessionRecord
	var parseErrors []error

	for _, line := range lines {
		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		records = append(records, record)
	}

	assert.Len(t, records, 2)
	assert.Empty(t, parseErrors)
	assert.Equal(t, "1", records[0].UUID)
	assert.Equal(t, shareddomain.SessionTypeUser, records[0].Type)
}

func TestParseJSONLLines_InvalidLines(t *testing.T) {
	lines := []string{
		`{"uuid":"1","type":"user"}`,
		`invalid json`,
		`{"uuid":"2","type":"assistant"}`,
	}

	var records []shareddomain.SessionRecord
	var parseErrors []error

	for _, line := range lines {
		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		records = append(records, record)
	}

	assert.Len(t, records, 2)
	assert.Len(t, parseErrors, 1)
}

func TestParseJSONLLines_EmptyLines(t *testing.T) {
	lines := []string{
		`{"uuid":"1","type":"user"}`,
		``,
		`{"uuid":"2","type":"assistant"}`,
	}

	var records []shareddomain.SessionRecord

	for _, line := range lines {
		if line == "" {
			continue
		}
		var record shareddomain.SessionRecord
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		records = append(records, record)
	}

	assert.Len(t, records, 2)
}

func TestParseJSONLLines_MessageContent(t *testing.T) {
	// Test with text content
	textLine := `{"uuid":"1","type":"user","message":{"role":"user","content":"hello"}}`

	var record shareddomain.SessionRecord
	err := sonic.Unmarshal([]byte(textLine), &record)
	require.NoError(t, err)
	require.NotNil(t, record.Message)
	require.NotNil(t, record.Message.Content)
	assert.False(t, record.Message.Content.IsBlocks)
	assert.NotNil(t, record.Message.Content.Text)
	assert.Equal(t, "hello", *record.Message.Content.Text)
}

func TestParseJSONLLines_ContentBlocks(t *testing.T) {
	// Test with block content
	blockLine := `{"uuid":"1","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"t1","name":"read","input":{}}]}}`

	var record shareddomain.SessionRecord
	err := sonic.Unmarshal([]byte(blockLine), &record)
	require.NoError(t, err)
	require.NotNil(t, record.Message)
	require.NotNil(t, record.Message.Content)
	assert.True(t, record.Message.Content.IsBlocks)
	assert.Len(t, record.Message.Content.Blocks, 2)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Valid JSON lines | Array of valid JSON | All parsed, no errors | Happy path |
| Invalid JSON lines | Mix of valid/invalid | Valid parsed, errors for invalid | Error handling |
| Empty lines | Array with empty strings | Empty lines skipped | Empty filter |
| Text content | Message with string content | Content.IsBlocks=false, Text set | Text content |
| Block content | Message with array content | Content.IsBlocks=true, Blocks set | Block content |

---

## Execution Order

1. **Step 1**: Update protobuf schema (no dependencies)
2. **Step 2**: Run buf generate (depends on Step 1)
3. **Step 3**: Update Daemon domain watch.go (depends on Step 2 for compilation)
4. **Step 4**: Update Daemon log_service.go (depends on Step 3)
5. **Step 5**: Update Daemon api_client.go (depends on Step 2, Step 3)
6. **Step 6**: Update Daemon inbound handler caller (depends on Step 4)
7. **Step 7**: Update API handler.go (depends on Step 2)
8. **Step 8**: Verify sonic dependency in API (depends on Step 7)
9. **Step 9**: Verify existing Daemon tests still pass (depends on Steps 4-6)
10. **Step 10**: Add API handler tests (depends on Step 7)

## Validation Steps

### Build Commands
```bash
# From project root
cd /Users/jayce/team-attention/cops

# Generate protobuf code
cd idl/protobuf && buf generate && cd ../..

# Build all modules
go build ./shared/...
go build ./daemon/...
go build ./api/...
```

### Test Commands
```bash
# Run all tests
go test ./daemon/... -v
go test ./api/... -v
go test ./shared/... -v

# Run specific test files
go test ./daemon/internal/service/logwatcher/... -v
go test ./api/internal/service/aggregation/... -v
```

### Lint Check
```bash
# If golangci-lint is configured
golangci-lint run ./daemon/...
golangci-lint run ./api/...
```

## Notes for Execute Agent

1. **Order is critical**: Protobuf must be regenerated before Go code changes, or compilation will fail.

2. **Find all callers**: Before Step 6, search for all usages of:
   - `AddRecordsForClaudeDir` - rename to `AddLinesForClaudeDir`
   - `HandleFileChange` - update to handle `[]string` return type instead of `[]shareddomain.SessionRecord`

   Use this command:
   ```bash
   grep -rn "AddRecordsForClaudeDir\|HandleFileChange" /Users/jayce/team-attention/cops/daemon/internal/
   ```

3. **Import cleanup**: After removing conversion functions, run `goimports` or your IDE's organize imports to clean up unused imports.

4. **Breaking change**: This is a breaking change. Old Daemon will not work with new API and vice versa. Deploy both together.

5. **No backward compatibility**: Per requirements, we are not maintaining backward compatibility. The old `SessionRecord` protobuf messages are completely removed.

6. **Test the existing integration test**: The existing `log_service_test.go` tests parsing of `shareddomain.SessionRecord` which is still valid - it now tests the parsing logic used by API. Run it to ensure parsing still works.

7. **sonic package**: The API now needs to import `github.com/bytedance/sonic`. Verify it's in `api/go.mod` or run `go get github.com/bytedance/sonic`.

8. **Error message truncation**: The `parseJSONLLines` function truncates line content to 100 characters in error messages using `%.100s` format specifier. This prevents huge log entries for large malformed lines.
