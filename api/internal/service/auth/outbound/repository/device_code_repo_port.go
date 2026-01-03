package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// DeviceCodeRepositoryPort defines interface for device code data persistence.
type DeviceCodeRepositoryPort interface {
	// Create creates a new device code.
	Create(ctx context.Context, deviceCode *domain.DeviceCode) (*domain.DeviceCode, error)

	// GetByID retrieves device code by its secure ID (used for CLI polling).
	GetByID(ctx context.Context, id string) (*domain.DeviceCode, error)

	// GetByUserCode retrieves device code by its human-friendly user code.
	GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceCode, error)

	// Approve marks a device code as approved and links it to a user.
	// Returns error if device code is already approved, expired, or not found.
	Approve(ctx context.Context, userCode string, userID domain.ID) error
}
