package connectrpc

import (
	"context"
	"log/slog"
	"net/http"
	"sort"

	"connectrpc.com/connect"

	featuredservice "github.com/team-attention/cops/api/internal/service/featured"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1/dashboardv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FeaturedBoardGRPCHandler handles featured board gRPC endpoints.
type FeaturedBoardGRPCHandler struct {
	svc    *featuredservice.FeaturedBoardService
	logger *slog.Logger
}

// NewFeaturedBoardGRPCHandler creates a new featured board gRPC handler.
func NewFeaturedBoardGRPCHandler(l *slog.Logger, svc *featuredservice.FeaturedBoardService) *FeaturedBoardGRPCHandler {
	return &FeaturedBoardGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "featured.grpc.connectrpc")),
	}
}

// GetHandler implements the ConnectHandler interface.
func (h *FeaturedBoardGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return dashboardv1connect.NewFeaturedBoardServiceHandler(h, opts...)
}

// GetFeaturedBoard returns aggregated stats for featured members.
// This is a public endpoint - no RBAC check required.
func (h *FeaturedBoardGRPCHandler) GetFeaturedBoard(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetFeaturedBoardReq],
) (*connect.Response[dashboardv1.GetFeaturedBoardRes], error) {
	msg := req.Msg

	// Call service (validation of org_slug is done in the service)
	result, err := h.svc.GetFeaturedBoard(ctx, msg.GetOrgSlug(), msg.GetSinceUnix())
	if err != nil {
		return nil, err
	}

	// Convert members to proto
	protoMembers := make([]*dashboardv1.FeaturedMember, 0, len(result.Members))
	for _, m := range result.Members {
		protoMember := &dashboardv1.FeaturedMember{
			UserId:      m.UserID,
			UserName:    m.UserName,
			ProjectId:   m.ProjectID,
			ProjectName: m.ProjectName,
			PromptCount: m.PromptCount,
			Usage: &dashboardv1.TokenUsageSummary{
				TotalInputTokens:         m.Usage.TotalInputTokens,
				TotalOutputTokens:        m.Usage.TotalOutputTokens,
				TotalCacheCreationTokens: m.Usage.TotalCacheCreationTokens,
				TotalCacheReadTokens:     m.Usage.TotalCacheReadTokens,
			},
		}

		if !m.LastActivityAt.IsZero() {
			protoMember.LastActivityAt = timestamppb.New(m.LastActivityAt)
		}

		// Convert tool calls map to sorted slice
		if len(m.ToolCalls) > 0 {
			toolCallSummaries := make([]*dashboardv1.ToolCallSummary, 0, len(m.ToolCalls))
			for toolName, count := range m.ToolCalls {
				toolCallSummaries = append(toolCallSummaries, &dashboardv1.ToolCallSummary{
					ToolName: toolName,
					Count:    count,
				})
			}
			// Sort by tool name for deterministic ordering
			sort.Slice(toolCallSummaries, func(i, j int) bool {
				return toolCallSummaries[i].ToolName < toolCallSummaries[j].ToolName
			})
			protoMember.ToolCalls = toolCallSummaries
		}

		protoMembers = append(protoMembers, protoMember)
	}

	res := &dashboardv1.GetFeaturedBoardRes{
		Members:          protoMembers,
		OrganizationName: result.OrganizationName,
	}

	return connect.NewResponse(res), nil
}

// Compile-time interface verification.
var _ dashboardv1connect.FeaturedBoardServiceHandler = (*FeaturedBoardGRPCHandler)(nil)
