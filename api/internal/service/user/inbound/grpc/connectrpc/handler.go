package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/user"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	userv1 "github.com/team-attention/cops/shared/gen/grpcstub/user/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect"
)

// UserGRPCHandler handles gRPC requests for user service.
type UserGRPCHandler struct {
	svc    *user.Service
	logger *slog.Logger
}

// NewUserGRPCHandler creates a new user gRPC handler.
func NewUserGRPCHandler(l *slog.Logger, svc *user.Service) *UserGRPCHandler {
	return &UserGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "user.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *UserGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return userv1connect.NewUserServiceHandler(h, opts...)
}

// GetMe returns the authenticated user's information and organizations.
func (h *UserGRPCHandler) GetMe(
	ctx context.Context,
	req *connect.Request[userv1.GetMeReq],
) (*connect.Response[userv1.GetMeRes], error) {
	// 1. Extract userID from context (set by auth interceptor).
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("GetMe: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// 2. Call h.svc.GetMe to get user and organizations.
	result, err := h.svc.GetMe(ctx, userID)
	if err != nil {
		// 3a. If error contains "user not found", return connect.CodeNotFound.
		if strings.Contains(err.Error(), "user not found") {
			h.logger.Info("GetMe: user not found",
				slog.String("userID", userID),
			)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		// 3b. Otherwise, log error and return connect.CodeInternal.
		h.logger.Error("GetMe failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 4. Convert service result to protobuf response:
	// a. If result.User is not nil, create domainv1.User with fields mapped from result.User.
	var protoUser *domainv1.User
	if result.User != nil {
		protoUser = &domainv1.User{
			Id:        string(result.User.ID),
			Email:     result.User.Email,
			Name:      result.User.Name,
			AvatarUrl: result.User.ProfileImageURL,
		}
	}

	// b. For each item in result.Organizations, if userOrg.Organization is not nil,
	//    append domainv1.Organization with fields.
	//    Include current user's membership so frontend can determine role.
	var protoOrgs []*domainv1.Organization
	for _, userOrg := range result.Organizations {
		if userOrg.Organization != nil {
			protoOrgs = append(protoOrgs, &domainv1.Organization{
				Id:   string(userOrg.Organization.ID),
				Name: userOrg.Organization.Name,
				Slug: userOrg.Organization.Slug,
				Members: []*domainv1.OrganizationMember{
					{
						UserId: userID,
						Role:   string(userOrg.Role),
					},
				},
			})
		}
	}

	// 5. Create userv1.GetMeRes with User and Organizations fields.
	res := &userv1.GetMeRes{
		User:          protoUser,
		Organizations: protoOrgs,
	}

	// 6. Return connect.NewResponse with the response.
	return connect.NewResponse(res), nil
}

// DeleteAccount permanently deletes the authenticated user's account.
func (h *UserGRPCHandler) DeleteAccount(
	ctx context.Context,
	req *connect.Request[userv1.DeleteAccountReq],
) (*connect.Response[userv1.DeleteAccountRes], error) {
	// 1. Extract userID from context (set by auth interceptor).
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("DeleteAccount: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// 2. Get confirmation phrase from req.Msg.ConfirmationPhrase.
	confirmationPhrase := req.Msg.ConfirmationPhrase

	// 3. Call h.svc.DeleteAccount(ctx, userID, confirmationPhrase).
	result, err := h.svc.DeleteAccount(ctx, userID, confirmationPhrase)
	if err != nil {
		// 4a. If error contains "confirmation phrase", return connect.CodeInvalidArgument.
		if strings.Contains(err.Error(), "confirmation phrase") {
			h.logger.Info("DeleteAccount: invalid confirmation phrase",
				slog.String("userID", userID),
			)
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		// 4b. If error contains "user not found", return connect.CodeNotFound.
		if strings.Contains(err.Error(), "user not found") {
			h.logger.Info("DeleteAccount: user not found",
				slog.String("userID", userID),
			)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		// 4c. Otherwise, log error and return connect.CodeInternal.
		h.logger.Error("DeleteAccount failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 5. Create userv1.DeleteAccountRes with Success and Message.
	res := &userv1.DeleteAccountRes{
		Success: result.Success,
		Message: result.Message,
	}

	// 6. Return connect.NewResponse with the response.
	return connect.NewResponse(res), nil
}

// Interface verification
var _ userv1connect.UserServiceHandler = (*UserGRPCHandler)(nil)
