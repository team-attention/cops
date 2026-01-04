package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/auth"
	"github.com/team-attention/cops/shared/domain"
	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
)

// AuthPublicGRPCHandler handles public auth gRPC endpoints (no authentication required).
type AuthPublicGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
	cfg    *config.Config
}

// NewAuthPublicGRPCHandler creates a new public auth gRPC handler.
func NewAuthPublicGRPCHandler(l *slog.Logger, svc *auth.Service, cfg *config.Config) *AuthPublicGRPCHandler {
	return &AuthPublicGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "auth.grpc.connectrpc.public")),
		cfg:    cfg,
	}
}

// GetHandler implements PublicConnectHandler interface.
func (h *AuthPublicGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return authv1connect.NewAuthServiceHandler(h, opts...)
}

// GoogleAuth handles Google OAuth authentication.
func (h *AuthPublicGRPCHandler) GoogleAuth(
	ctx context.Context,
	req *connect.Request[authv1.GoogleAuthReq],
) (*connect.Response[authv1.GoogleAuthRes], error) {
	params := auth.GoogleAuthParams{
		AuthorizationCode: req.Msg.AuthorizationCode,
		RedirectURI:       req.Msg.RedirectUri,
	}

	tokens, err := h.svc.GoogleAuth(ctx, params)
	if err != nil {
		h.logger.Error("GoogleAuth failed",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &authv1.GoogleAuthRes{
		Tokens: &authv1.TokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Unix(),
		},
	}

	return connect.NewResponse(res), nil
}

// RefreshToken handles token refresh.
func (h *AuthPublicGRPCHandler) RefreshToken(
	ctx context.Context,
	req *connect.Request[authv1.RefreshTokenReq],
) (*connect.Response[authv1.RefreshTokenRes], error) {
	tokens, err := h.svc.RefreshToken(ctx, req.Msg.RefreshToken)
	if err != nil {
		h.logger.Error("RefreshToken failed",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	res := &authv1.RefreshTokenRes{
		Tokens: &authv1.TokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Unix(),
		},
	}

	return connect.NewResponse(res), nil
}

// DeviceCode initiates device code flow.
func (h *AuthPublicGRPCHandler) DeviceCode(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeReq],
) (*connect.Response[authv1.DeviceCodeRes], error) {
	result, err := h.svc.DeviceCode(ctx)
	if err != nil {
		h.logger.Error("DeviceCode failed",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &authv1.DeviceCodeRes{
		DeviceCode:      result.DeviceCode,
		UserCode:        result.UserCode,
		VerificationUrl: result.VerificationURL,
		ExpiresIn:       int32(result.ExpiresIn),
		Interval:        int32(result.Interval),
	}

	return connect.NewResponse(res), nil
}

// DevicePoll polls for device code approval status.
func (h *AuthPublicGRPCHandler) DevicePoll(
	ctx context.Context,
	req *connect.Request[authv1.DevicePollReq],
) (*connect.Response[authv1.DevicePollRes], error) {
	result, err := h.svc.DevicePoll(ctx, req.Msg.DeviceCode)
	if err != nil {
		h.logger.Error("DevicePoll failed",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &authv1.DevicePollRes{
		Pending: result.Pending,
	}

	if !result.Pending && result.Tokens != nil {
		res.Tokens = &authv1.TokenPair{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresAt:    result.Tokens.ExpiresAt.Unix(),
		}
	}

	return connect.NewResponse(res), nil
}

// AuthPrivateGRPCHandler handles private auth gRPC endpoints (authentication required).
type AuthPrivateGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
}

// NewAuthPrivateGRPCHandler creates a new private auth gRPC handler.
func NewAuthPrivateGRPCHandler(l *slog.Logger, svc *auth.Service) *AuthPrivateGRPCHandler {
	return &AuthPrivateGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "auth.grpc.connectrpc.private")),
	}
}

// GetHandler implements PrivateConnectHandler interface.
func (h *AuthPrivateGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return authv1connect.NewAuthPrivateServiceHandler(h, opts...)
}

// DeviceCodeApprove approves a device code (requires authentication).
func (h *AuthPrivateGRPCHandler) DeviceCodeApprove(
	ctx context.Context,
	req *connect.Request[authv1.DeviceCodeApproveReq],
) (*connect.Response[authv1.DeviceCodeApproveRes], error) {
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	params := auth.DeviceCodeApproveParams{
		UserCode: req.Msg.UserCode,
		UserID:   domain.ID(userID),
	}

	err := h.svc.DeviceCodeApprove(ctx, params)
	if err != nil {
		h.logger.Error("DeviceCodeApprove failed",
			slog.String("userCode", req.Msg.UserCode),
			slog.Any("error", err),
		)

		errMsg := err.Error()
		code := connect.CodeInternal

		if strings.Contains(errMsg, "not found") {
			code = connect.CodeNotFound
		} else if strings.Contains(errMsg, "expired") {
			code = connect.CodeDeadlineExceeded
		} else if strings.Contains(errMsg, "already approved") {
			code = connect.CodeAlreadyExists
		}

		return nil, connect.NewError(code, err)
	}

	res := &authv1.DeviceCodeApproveRes{
		Success: true,
		Message: "Device approved successfully",
	}

	return connect.NewResponse(res), nil
}

// Compile-time interface verification.
var _ authv1connect.AuthServiceHandler = (*AuthPublicGRPCHandler)(nil)
var _ authv1connect.AuthPrivateServiceHandler = (*AuthPrivateGRPCHandler)(nil)
