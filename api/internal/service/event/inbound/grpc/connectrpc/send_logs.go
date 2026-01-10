package connectrpc

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/bytedance/sonic"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	eventv1 "github.com/team-attention/cops/shared/gen/grpcstub/event/v1"
)

// SendLogs receives a batch of JSONL log lines and saves them to events collection.
func (h *EventGRPCHandler) SendLogs(
	ctx context.Context,
	req *connect.Request[eventv1.SendLogsReq],
) (*connect.Response[eventv1.SendLogsRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	pbBatch := req.Msg.GetBatch()
	if pbBatch == nil {
		return connect.NewResponse(&eventv1.SendLogsRes{
			Success:      false,
			ErrorMessage: "batch is required",
		}), nil
	}

	batch, parseErrors := h.parseJSONLLines(pbBatch.GetJsonl(), pbBatch.GetProjectId(), pbBatch.GetOrganizationId())

	// Log parse errors at ERROR level (Fire & Forget)
	if len(parseErrors) > 0 {
		h.logger.Error("failed to parse some JSONL lines",
			slog.String("projectId", pbBatch.GetProjectId()),
			slog.String("organizationId", pbBatch.GetOrganizationId()),
			slog.Int("failedCount", len(parseErrors)),
			slog.Int("totalCount", len(pbBatch.GetJsonl())),
			slog.String("sampleError", parseErrors[0].Error()),
		)
	}

	result, err := h.svc.CollectLogs(ctx, userID, batch)
	if err != nil {
		return nil, err
	}

	res := &eventv1.SendLogsRes{
		Success:        result.Success,
		ErrorMessage:   result.ErrorMessage,
		ProcessedCount: result.ProcessedCount,
	}

	return connect.NewResponse(res), nil
}

// parseJSONLLines parses raw JSONL lines into Record domain objects.
// Returns the parsed batch and any parse errors encountered.
func (h *EventGRPCHandler) parseJSONLLines(lines []string, projectID, organizationID string) (*repository.LogBatch, []error) {
	var records []*shareddomain.Record
	var parseErrors []error

	for _, line := range lines {
		if line == "" {
			continue
		}

		var record shareddomain.Record
		if err := sonic.Unmarshal([]byte(line), &record); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("parse error: %s (line: %.100s...)", err.Error(), line))
			continue
		}

		records = append(records, &record)
	}

	return &repository.LogBatch{
		Records:        records,
		ProjectID:      projectID,
		OrganizationID: organizationID,
	}, parseErrors
}
