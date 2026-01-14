package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// InvitationRepositoryPort defines interface for invitation persistence.
type InvitationRepositoryPort interface {
	// Create persists a new invitation.
	// Returns the created invitation with generated ID.
	Create(ctx context.Context, invitation *domain.Invitation) (*domain.Invitation, error)

	// GetByToken retrieves an invitation by its secure token.
	// Returns nil, nil if not found.
	GetByToken(ctx context.Context, token string) (*domain.Invitation, error)

	// GetByID retrieves an invitation by ID.
	// Returns nil, nil if not found.
	GetByID(ctx context.Context, id string) (*domain.Invitation, error)

	// GetByEmailAndOrg checks if an invitation already exists for email in org.
	// Returns nil, nil if not found.
	GetByEmailAndOrg(ctx context.Context, email, organizationID string) (*domain.Invitation, error)

	// ListByOrganization retrieves all pending invitations for an organization.
	ListByOrganization(ctx context.Context, organizationID string) ([]*domain.Invitation, error)

	// UpdateStatus updates an invitation's status.
	// Sets AcceptedAt if status is "accepted".
	UpdateStatus(ctx context.Context, id string, status domain.InvitationStatus) error

	// Delete removes an invitation permanently.
	Delete(ctx context.Context, id string) error
}
