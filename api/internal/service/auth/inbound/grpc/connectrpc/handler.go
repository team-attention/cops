package connectrpc

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/team-attention/cops/shared/gen/grpcstub/auth/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/auth/v1/authv1connect"
	"github.com/team-attention/cops/api/internal/service/auth"
)

type AuthGRPCHandler struct {
	svc    *auth.Service
	logger *slog.Logger
}

func NewAuthGRPCHandler(l *slog.Logger, svc *auth.Service) *AuthGRPCHandler {
	return &AuthGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "auth.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *AuthGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return authv1connect.NewAuthServiceHandler(h, opts...)
}

func (h *AuthGRPCHandler) GoogleAuth(
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

func (h *AuthGRPCHandler) DeviceCode(
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

func (h *AuthGRPCHandler) DevicePoll(
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

func (h *AuthGRPCHandler) RefreshToken(
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

var _ authv1connect.AuthServiceHandler = (*AuthGRPCHandler)(nil)
