package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
	"github.com/team-attention/cops/api/internal/service/user"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
	userv1 "github.com/team-attention/cops/shared/gen/grpcstub/user/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/user/v1/userv1connect"
)

// UserGRPCHandler handles gRPC requests for user service.
type UserGRPCHandler struct {
	svc    *user.Service
	logger *slog.Logger
	cfg    *config.Config
}

// NewUserGRPCHandler creates a new user gRPC handler.
func NewUserGRPCHandler(l *slog.Logger, svc *user.Service, cfg *config.Config) *UserGRPCHandler {
	return &UserGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "user.grpc.connectrpc")),
		cfg:    cfg,
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
	// 1. Extract Authorization header from request.
	authHeader := req.Header().Get("Authorization")

	// 2. Validate header exists and has "Bearer " prefix.
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		h.logger.Warn("GetMe: missing or invalid authorization header")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing or invalid authorization header"))
	}

	// 3. Extract token string by trimming "Bearer " prefix.
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// 4. Create jwtutil.Config from h.cfg.JWT fields.
	jwtCfg := &jwtutil.Config{
		SecretKey:            h.cfg.JWT.SecretKey,
		AccessTokenDuration:  h.cfg.JWT.AccessTokenDuration,
		RefreshTokenDuration: h.cfg.JWT.RefreshTokenDuration,
		Issuer:               h.cfg.JWT.Issuer,
	}

	// 5. Call jwtutil.ValidateAccessToken to get userID.
	userID, err := jwtutil.ValidateAccessToken(jwtCfg, tokenString)
	if err != nil {
		h.logger.Warn("GetMe: invalid access token",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid access token"))
	}

	// 6. Call h.svc.GetMe with userID.
	result, err := h.svc.GetMe(ctx, userID)
	if err != nil {
		// If error contains "user not found", return connect.CodeNotFound.
		if strings.Contains(err.Error(), "user not found") {
			h.logger.Info("GetMe: user not found",
				slog.String("userID", userID),
			)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		// If other error, log error and return connect.CodeInternal.
		h.logger.Error("GetMe failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// 7. Convert service result to protobuf response:
	// a. Map domain.User to domainv1.User
	var protoUser *domainv1.User
	if result.User != nil {
		protoUser = &domainv1.User{
			Id:        string(result.User.ID),
			Email:     result.User.Email,
			Name:      result.User.Name,
			AvatarUrl: result.User.ProfileImageURL,
		}
	}

	// b. Map each UserOrganization to domainv1.Organization
	var protoOrgs []*domainv1.Organization
	for _, userOrg := range result.Organizations {
		if userOrg.Organization != nil {
			protoOrgs = append(protoOrgs, &domainv1.Organization{
				Id:   string(userOrg.Organization.ID),
				Name: userOrg.Organization.Name,
				Slug: userOrg.Organization.Slug,
				// Note: Members field intentionally not populated for GetMe response
			})
		}
	}

	res := &userv1.GetMeRes{
		User:          protoUser,
		Organizations: protoOrgs,
	}

	// 8. Return connect.NewResponse with the response.
	return connect.NewResponse(res), nil
}

// Interface verification
var _ userv1connect.UserServiceHandler = (*UserGRPCHandler)(nil)
