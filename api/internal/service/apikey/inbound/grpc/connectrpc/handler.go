package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/platform/interceptor"
	"github.com/team-attention/cops/api/internal/service/apikey"
	"github.com/team-attention/cops/shared/domain"
	apikeyv1 "github.com/team-attention/cops/shared/gen/grpcstub/apikey/v1"
	"github.com/team-attention/cops/shared/gen/grpcstub/apikey/v1/apikeyv1connect"
)

// APIKeyGRPCHandler handles gRPC requests for API key service.
type APIKeyGRPCHandler struct {
	svc    *apikey.Service
	logger *slog.Logger
}

// NewAPIKeyGRPCHandler creates a new API key gRPC handler.
func NewAPIKeyGRPCHandler(l *slog.Logger, svc *apikey.Service) *APIKeyGRPCHandler {
	return &APIKeyGRPCHandler{
		svc:    svc,
		logger: l.With(slog.String("name", "apikey.grpc.connectrpc")),
	}
}

// GetHandler implements ConnectHandler interface.
func (h *APIKeyGRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	return apikeyv1connect.NewAPIKeyServiceHandler(h, opts...)
}

// IssueAPIKey creates a new API key for the authenticated user.
func (h *APIKeyGRPCHandler) IssueAPIKey(
	ctx context.Context,
	req *connect.Request[apikeyv1.IssueAPIKeyReq],
) (*connect.Response[apikeyv1.IssueAPIKeyRes], error) {
	// Extract userID from context (set by auth interceptor)
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("IssueAPIKey: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Build IssueAPIKeyParams from request
	params := apikey.IssueAPIKeyParams{
		UserID:        userID,
		Name:          req.Msg.Name,
		ExpiresInDays: req.Msg.ExpiresInDays,
	}

	// Call service.IssueAPIKey
	result, err := h.svc.IssueAPIKey(ctx, params)
	if err != nil {
		h.logger.Error("IssueAPIKey failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert result to proto response
	keyInfo := domainAPIKeyToProto(result.KeyInfo)

	res := &apikeyv1.IssueAPIKeyRes{
		ApiKey:  result.APIKey,
		KeyInfo: keyInfo,
	}

	return connect.NewResponse(res), nil
}

// ListAPIKeys retrieves all API keys for the authenticated user.
func (h *APIKeyGRPCHandler) ListAPIKeys(
	ctx context.Context,
	req *connect.Request[apikeyv1.ListAPIKeysReq],
) (*connect.Response[apikeyv1.ListAPIKeysRes], error) {
	// Extract userID from context (set by auth interceptor)
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("ListAPIKeys: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Build ListAPIKeysParams from request
	params := apikey.ListAPIKeysParams{
		UserID:         userID,
		IncludeRevoked: req.Msg.IncludeRevoked,
	}

	// Call service.ListAPIKeys
	keys, err := h.svc.ListAPIKeys(ctx, params)
	if err != nil {
		h.logger.Error("ListAPIKeys failed",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert result to proto response
	protoKeys := make([]*apikeyv1.APIKeyInfo, 0, len(keys))
	for _, key := range keys {
		protoKeys = append(protoKeys, domainAPIKeyToProto(key))
	}

	res := &apikeyv1.ListAPIKeysRes{
		Keys: protoKeys,
	}

	return connect.NewResponse(res), nil
}

// RevokeAPIKey revokes an API key.
func (h *APIKeyGRPCHandler) RevokeAPIKey(
	ctx context.Context,
	req *connect.Request[apikeyv1.RevokeAPIKeyReq],
) (*connect.Response[apikeyv1.RevokeAPIKeyRes], error) {
	// Extract userID from context (set by auth interceptor)
	userID := interceptor.UserIDFromContext(ctx)
	if userID == "" {
		h.logger.Warn("RevokeAPIKey: user not authenticated")
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	// Build RevokeAPIKeyParams from request
	params := apikey.RevokeAPIKeyParams{
		UserID: userID,
		KeyID:  req.Msg.KeyId,
	}

	// Call service.RevokeAPIKey
	err := h.svc.RevokeAPIKey(ctx, params)
	if err != nil {
		h.logger.Error("RevokeAPIKey failed",
			slog.String("userID", userID),
			slog.String("keyID", req.Msg.KeyId),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &apikeyv1.RevokeAPIKeyRes{
		Success: true,
	}

	return connect.NewResponse(res), nil
}

// ValidateAPIKey validates an API key.
// This endpoint does NOT require authentication (used by interceptor).
func (h *APIKeyGRPCHandler) ValidateAPIKey(
	ctx context.Context,
	req *connect.Request[apikeyv1.ValidateAPIKeyReq],
) (*connect.Response[apikeyv1.ValidateAPIKeyRes], error) {
	// Call service.ValidateAPIKey
	result, err := h.svc.ValidateAPIKey(ctx, req.Msg.ApiKey)
	if err != nil {
		h.logger.Error("ValidateAPIKey failed",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := &apikeyv1.ValidateAPIKeyRes{
		Valid:        result.Valid,
		UserId:       result.UserID,
		ErrorMessage: result.ErrorMessage,
	}

	return connect.NewResponse(res), nil
}

// domainAPIKeyToProto converts domain.APIKey to apikeyv1.APIKeyInfo.
func domainAPIKeyToProto(key *domain.APIKey) *apikeyv1.APIKeyInfo {
	if key == nil {
		return nil
	}

	info := &apikeyv1.APIKeyInfo{
		Id:        string(key.ID),
		UserId:    string(key.UserID),
		Name:      key.Name,
		KeyPrefix: key.KeyPrefix,
		CreatedAt: key.CreatedAt.Unix(),
	}

	if key.LastUsedAt != nil {
		info.LastUsedAt = key.LastUsedAt.Unix()
	}

	if key.RevokedAt != nil {
		info.RevokedAt = key.RevokedAt.Unix()
	}

	if key.ExpiresAt != nil {
		info.ExpiresAt = key.ExpiresAt.Unix()
	}

	return info
}

// Interface verification
var _ apikeyv1connect.APIKeyServiceHandler = (*APIKeyGRPCHandler)(nil)
