package api

import (
	"context"

	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
)

// GetMeResult contains the result of GetMe API call.
type GetMeResult struct {
	UserID        string
	Organizations []*domainv1.Organization
}

// UserAPIPort defines the interface for user API operations.
type UserAPIPort interface {
	// GetMe fetches the authenticated user's information and organizations.
	// Requires valid access token.
	GetMe(ctx context.Context, accessToken string) (*GetMeResult, error)
}
