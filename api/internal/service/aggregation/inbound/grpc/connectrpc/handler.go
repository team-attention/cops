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

// parseJSONLLines parses raw JSONL lines into Record domain objects.
// Returns the parsed batch and any parse errors encountered.
func (h *AggregationGRPCHandler) parseJSONLLines(lines []string, projectID string) (*repository.LogBatch, []error) {
	var records []shareddomain.Record
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

		records = append(records, record)
	}

	return &repository.LogBatch{
		Records:   records,
		ProjectID: projectID,
	}, parseErrors
}

// Compile-time interface verification.
var _ aggregationv1connect.AggregationServiceHandler = (*AggregationGRPCHandler)(nil)
