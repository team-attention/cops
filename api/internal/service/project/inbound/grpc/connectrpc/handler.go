package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	projectservice "github.com/team-attention/cops/api/internal/service/project"
	projectv1 "github.com/team-attention/cops/shared/gen/grpcstub/project/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/project/v1/projectv1connect"
)

// ProjectGRPCHandler handles project gRPC endpoints.
type ProjectGRPCHandler struct {
	svc    *projectservice.Service
	logger *slog.Logger
}

// NewProjectGRPCHandler creates a new project gRPC handler.
func NewProjectGRPCHandler(l *slog.Logger, svc *projectservice.Service) *ProjectGRPCHandler {
	return &ProjectGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "project.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *ProjectGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return projectv1connect.NewProjectServiceHandler(h, opts...)
}

// RegisterProject registers a project or returns existing project ID if already registered.
func (h *ProjectGRPCHandler) RegisterProject(
	ctx context.Context,
	req *connect.Request[projectv1.RegisterProjectReq],
) (*connect.Response[projectv1.RegisterProjectRes], error) {
	// Parse request
	msg := req.Msg
	params := projectservice.RegisterProjectParams{
		ConfiguredRemoteURL: msg.GetConfiguredRemoteUrl(),
		ActualRemoteURL:     msg.GetActualRemoteUrl(),
		ExistingProjectID:   msg.GetExistingProjectId(),
		Name:                msg.GetName(),
		IsGitProject:        msg.GetIsGitProject(),
		OrganizationID:      msg.GetOrganizationId(),
	}

	// Call service
	result, err := h.svc.RegisterProject(ctx, params)
	if err != nil {
		return nil, err
	}

	// Build response
	res := &projectv1.RegisterProjectRes{
		ProjectId:      result.ProjectID,
		IsNew:          result.IsNew,
		Name:           result.Name,
		IsGitProject:   result.IsGitProject,
		OrganizationId: result.OrganizationID,
	}

	return connect.NewResponse(res), nil
}

// Compile-time interface verification
var _ projectv1connect.ProjectServiceHandler = (*ProjectGRPCHandler)(nil)
