package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	dashboardservice "github.com/team-attention/cops/api/internal/service/dashboard"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	dashboardv1 "github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/dashboard/v1/dashboardv1connect"
)

// DashboardGRPCHandler handles dashboard gRPC endpoints.
type DashboardGRPCHandler struct {
	svc    *dashboardservice.Service
	logger *slog.Logger
}

// NewDashboardGRPCHandler creates a new dashboard gRPC handler.
func NewDashboardGRPCHandler(l *slog.Logger, svc *dashboardservice.Service) *DashboardGRPCHandler {
	return &DashboardGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "dashboard.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *DashboardGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return dashboardv1connect.NewDashboardServiceHandler(h, opts...)
}

// GetOverview returns dashboard summary statistics.
func (h *DashboardGRPCHandler) GetOverview(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetOverviewReq],
) (*connect.Response[dashboardv1.GetOverviewRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	// Call service
	stats, err := h.svc.GetOverview(ctx, userID, req.Msg.GetOrganizationId())
	if err != nil {
		return nil, err
	}

	// Convert recent projects
	recentProjects := make([]*dashboardv1.ProjectSummary, len(stats.RecentProjects))
	for i, p := range stats.RecentProjects {
		recentProjects[i] = toProtoProjectSummary(p)
	}

	// Convert recent sessions
	recentSessions := make([]*dashboardv1.SessionSummary, len(stats.RecentSessions))
	for i, s := range stats.RecentSessions {
		recentSessions[i] = toProtoSessionSummary(s)
	}

	// Build response
	res := &dashboardv1.GetOverviewRes{
		TotalUsage:     toProtoTokenUsageSummary(stats.TotalUsage),
		ProjectCount:   stats.ProjectCount,
		SessionCount:   stats.SessionCount,
		RecentProjects: recentProjects,
		RecentSessions: recentSessions,
	}

	return connect.NewResponse(res), nil
}

// ListProjects returns a paginated list of projects.
func (h *DashboardGRPCHandler) ListProjects(
	ctx context.Context,
	req *connect.Request[dashboardv1.ListProjectsReq],
) (*connect.Response[dashboardv1.ListProjectsRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	// Parse request
	msg := req.Msg
	params := repository.ListProjectsParams{
		Page:     msg.GetPagination().GetPage(),
		PageSize: msg.GetPagination().GetPageSize(),
		Query: repository.ListProjectsQuery{
			OrganizationID: msg.GetOrganizationId(),
		},
	}

	// Call service
	result, err := h.svc.ListProjects(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	// Convert projects
	projects := make([]*dashboardv1.ProjectSummary, len(result.Items))
	for i, p := range result.Items {
		projects[i] = toProtoProjectSummary(p)
	}

	// Build response
	res := &dashboardv1.ListProjectsRes{
		Projects:   projects,
		Pagination: toProtoPagination(result.CurrentPage, result.PageSize, result.TotalPages, result.TotalCount),
	}

	return connect.NewResponse(res), nil
}

// GetProject returns detailed project information.
func (h *DashboardGRPCHandler) GetProject(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetProjectReq],
) (*connect.Response[dashboardv1.GetProjectRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	// Call service
	project, err := h.svc.GetProject(ctx, userID, req.Msg.GetOrganizationId(), req.Msg.GetProjectId())
	if err != nil {
		return nil, err
	}

	// Build response
	res := &dashboardv1.GetProjectRes{
		Project: toProtoProjectDetail(project),
	}

	return connect.NewResponse(res), nil
}

// ListSessions returns sessions for a project.
func (h *DashboardGRPCHandler) ListSessions(
	ctx context.Context,
	req *connect.Request[dashboardv1.ListSessionsReq],
) (*connect.Response[dashboardv1.ListSessionsRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	// Parse request
	msg := req.Msg
	params := repository.ListSessionsParams{
		Page:     msg.GetPagination().GetPage(),
		PageSize: msg.GetPagination().GetPageSize(),
		Query: repository.ListSessionsQuery{
			OrganizationID: msg.GetOrganizationId(),
			ProjectID:      msg.GetProjectId(),
			SortBy:         msg.GetSortBy(),
			SortDesc:       msg.GetSortDesc(),
		},
	}

	// Call service
	result, err := h.svc.ListSessions(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	// Convert sessions
	sessions := make([]*dashboardv1.SessionSummary, len(result.Items))
	for i, s := range result.Items {
		sessions[i] = toProtoSessionSummary(s)
	}

	// Build response
	res := &dashboardv1.ListSessionsRes{
		Sessions:   sessions,
		Pagination: toProtoPagination(result.CurrentPage, result.PageSize, result.TotalPages, result.TotalCount),
	}

	return connect.NewResponse(res), nil
}

// GetSession returns detailed session information with records.
func (h *DashboardGRPCHandler) GetSession(
	ctx context.Context,
	req *connect.Request[dashboardv1.GetSessionReq],
) (*connect.Response[dashboardv1.GetSessionRes], error) {
	userID := interceptor.UserIDFromContext(ctx)

	// Call service
	session, err := h.svc.GetSession(ctx, userID, req.Msg.GetOrganizationId(), req.Msg.GetSessionId())
	if err != nil {
		return nil, err
	}

	// Build response
	res := &dashboardv1.GetSessionRes{
		Session: toProtoSessionDetail(session),
	}

	return connect.NewResponse(res), nil
}

// Compile-time interface verification.
var _ dashboardv1connect.DashboardServiceHandler = (*DashboardGRPCHandler)(nil)
