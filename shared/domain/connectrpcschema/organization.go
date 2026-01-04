package connectrpcschema

import (
	"github.com/team-attention/cops/shared/domain"
	domainv1 "github.com/team-attention/cops/shared/gen/grpcstub/domain/v1"
)

// OrganizationFromProto converts domainv1.Organization to domain.Organization.
// Members field is not populated as it's not needed for organization selection.
func OrganizationFromProto(pb *domainv1.Organization) *domain.Organization {
	if pb == nil {
		return nil
	}

	return &domain.Organization{
		ID:      domain.ID(pb.Id),
		Name:    pb.Name,
		Slug:    pb.Slug,
		Members: nil, // Not needed for organization selection
	}
}

// OrganizationsFromProto converts a slice of protobuf organizations to domain organizations.
func OrganizationsFromProto(pbs []*domainv1.Organization) []*domain.Organization {
	result := make([]*domain.Organization, len(pbs))
	for i, pb := range pbs {
		result[i] = OrganizationFromProto(pb)
	}
	return result
}
