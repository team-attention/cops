package mock

import (
	"context"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// MembershipQuery represents a single IsMember query.
type MembershipQuery struct {
	UserID         string
	OrganizationID string
}

// OrganizationMemberRepository implements repository.OrganizationMemberRepositoryPort for testing.
type OrganizationMemberRepository struct {
	// IsMemberFunc is the behavior to execute when IsMember is called.
	IsMemberFunc func(ctx context.Context, userID, organizationID string) (bool, error)
	// CallCount tracks the number of IsMember calls.
	CallCount int
	// Queries records all membership queries.
	Queries []MembershipQuery
}

// IsMember implements repository.OrganizationMemberRepositoryPort.
func (m *OrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	m.CallCount++

	m.Queries = append(m.Queries, MembershipQuery{
		UserID:         userID,
		OrganizationID: organizationID,
	})

	if m.IsMemberFunc != nil {
		return m.IsMemberFunc(ctx, userID, organizationID)
	}

	return false, nil
}

// Compile-time interface verification.
var _ repository.OrganizationMemberRepositoryPort = (*OrganizationMemberRepository)(nil)
