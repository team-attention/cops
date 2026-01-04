package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/organization"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	organizationv1 "github.com/team-attention/cops/shared/gen/grpcstub/organization/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/organization/v1/organizationv1connect"
)

// OrganizationGRPCHandler handles gRPC requests for organization service.
type OrganizationGRPCHandler struct {
	svc    *organization.Service
	logger *slog.Logger
}

// NewOrganizationGRPCHandler creates a new organization gRPC handler.
func NewOrganizationGRPCHandler(l *slog.Logger, svc *organization.Service) *OrganizationGRPCHandler {
	return &OrganizationGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "organization.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *OrganizationGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return organizationv1connect.NewOrganizationServiceHandler(h, opts...)
}

// CreateOrganization creates a new organization with the authenticated user as admin.
func (h *OrganizationGRPCHandler) CreateOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.CreateOrganizationReq],
) (*connect.Response[organizationv1.CreateOrganizationRes], error) {
	// Extract userID from context (set by auth interceptor).
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("CreateOrganization: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	result, err := h.svc.CreateOrganization(ctx, userID, req.Msg.Name, req.Msg.Slug)
	if err != nil {
		if strings.Contains(err.Error(), "slug") || strings.Contains(err.Error(), "name") || strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "characters") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if strings.Contains(err.Error(), "already taken") {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}

		h.logger.Error("CreateOrganization failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to protobuf
	protoOrg := &domainv1.Organization{
		Id:   string(result.Organization.ID),
		Name: result.Organization.Name,
		Slug: result.Organization.Slug,
	}

	res := &organizationv1.CreateOrganizationRes{
		Organization: protoOrg,
	}

	return connect.NewResponse(res), nil
}

// UpdateOrganization updates an organization's name and slug.
func (h *OrganizationGRPCHandler) UpdateOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.UpdateOrganizationReq],
) (*connect.Response[organizationv1.UpdateOrganizationRes], error) {
	// Extract userID from context (set by auth interceptor).
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("UpdateOrganization: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	result, err := h.svc.UpdateOrganization(ctx, userID, req.Msg.OrganizationId, req.Msg.Name, req.Msg.Slug)
	if err != nil {
		if strings.Contains(err.Error(), "admin role required") {
			h.logger.Info("UpdateOrganization: permission denied",
				slog.String("userID", userID),
				slog.String("organizationID", req.Msg.OrganizationId),
			)
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(err.Error(), "not a member") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(err.Error(), "slug") || strings.Contains(err.Error(), "name") || strings.Contains(err.Error(), "required") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		h.logger.Error("UpdateOrganization failed",
			slog.String("userID", userID),
			slog.String("organizationID", req.Msg.OrganizationId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to protobuf
	protoOrg := &domainv1.Organization{
		Id:   string(result.Organization.ID),
		Name: result.Organization.Name,
		Slug: result.Organization.Slug,
	}

	res := &organizationv1.UpdateOrganizationRes{
		Organization: protoOrg,
	}

	return connect.NewResponse(res), nil
}

// GetOrganizationMembers retrieves members with their details.
func (h *OrganizationGRPCHandler) GetOrganizationMembers(
	ctx context.Context,
	req *connect.Request[organizationv1.GetOrganizationMembersReq],
) (*connect.Response[organizationv1.GetOrganizationMembersRes], error) {
	// Extract userID from context.
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("GetOrganizationMembers: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	result, err := h.svc.GetOrganizationMembers(ctx, userID, req.Msg.OrganizationId)
	if err != nil {
		if strings.Contains(err.Error(), "not a member") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		h.logger.Error("GetOrganizationMembers failed",
			slog.String("userID", userID),
			slog.String("organizationID", req.Msg.OrganizationId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to protobuf
	var protoMembers []*organizationv1.OrganizationMemberWithDetails
	for _, member := range result.Members {
		protoMembers = append(protoMembers, &organizationv1.OrganizationMemberWithDetails{
			UserId:    member.UserID,
			Email:     member.Email,
			Name:      member.Name,
			AvatarUrl: member.AvatarURL,
			Role:      string(member.Role),
		})
	}

	res := &organizationv1.GetOrganizationMembersRes{
		Members: protoMembers,
	}

	return connect.NewResponse(res), nil
}

// UpdateMemberRole changes a member's role.
func (h *OrganizationGRPCHandler) UpdateMemberRole(
	ctx context.Context,
	req *connect.Request[organizationv1.UpdateMemberRoleReq],
) (*connect.Response[organizationv1.UpdateMemberRoleRes], error) {
	// Extract userID from context.
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("UpdateMemberRole: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	err := h.svc.UpdateMemberRole(ctx, userID, req.Msg.OrganizationId, req.Msg.UserId, req.Msg.Role)
	if err != nil {
		if strings.Contains(err.Error(), "admin role required") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(err.Error(), "last admin") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if strings.Contains(err.Error(), "role must be") || strings.Contains(err.Error(), "required") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		h.logger.Error("UpdateMemberRole failed",
			slog.String("userID", userID),
			slog.String("organizationID", req.Msg.OrganizationId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &organizationv1.UpdateMemberRoleRes{
		Success: true,
	}

	return connect.NewResponse(res), nil
}

// RemoveMember removes a member from the organization.
func (h *OrganizationGRPCHandler) RemoveMember(
	ctx context.Context,
	req *connect.Request[organizationv1.RemoveMemberReq],
) (*connect.Response[organizationv1.RemoveMemberRes], error) {
	// Extract userID from context.
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("RemoveMember: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	err := h.svc.RemoveMember(ctx, userID, req.Msg.OrganizationId, req.Msg.UserId)
	if err != nil {
		if strings.Contains(err.Error(), "admin role required") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(err.Error(), "last admin") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}

		h.logger.Error("RemoveMember failed",
			slog.String("userID", userID),
			slog.String("organizationID", req.Msg.OrganizationId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &organizationv1.RemoveMemberRes{
		Success: true,
	}

	return connect.NewResponse(res), nil
}

// LeaveOrganization removes the current user from the organization.
func (h *OrganizationGRPCHandler) LeaveOrganization(
	ctx context.Context,
	req *connect.Request[organizationv1.LeaveOrganizationReq],
) (*connect.Response[organizationv1.LeaveOrganizationRes], error) {
	// Extract userID from context.
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("LeaveOrganization: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Call service
	result, err := h.svc.LeaveOrganization(ctx, userID, req.Msg.OrganizationId)
	if err != nil {
		if strings.Contains(err.Error(), "sole admin") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if strings.Contains(err.Error(), "not a member") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}

		h.logger.Error("LeaveOrganization failed",
			slog.String("userID", userID),
			slog.String("organizationID", req.Msg.OrganizationId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &organizationv1.LeaveOrganizationRes{
		Success:            result.Success,
		IsLastOrganization: result.IsLastOrganization,
	}

	return connect.NewResponse(res), nil
}

// Interface verification
var _ organizationv1connect.OrganizationServiceHandler = (*OrganizationGRPCHandler)(nil)
