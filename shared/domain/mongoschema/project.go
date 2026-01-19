package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	ProjectCollectionName = "projects"
)

const (
	ProjectIDField             = "_id"
	ProjectNameField           = "name"
	ProjectPathField           = "path"
	ProjectIsGitProjectField   = "isGitProject"
	ProjectRegisteredAtField   = "registeredAt"
	ProjectGitBranchField      = "git_branch"
	ProjectRemoteURLField      = "remoteUrl"
	ProjectOrganizationIDField = "organizationId"
)

type Project struct {
	domain.Project `bson:",inline"`
	ID             bson.ObjectID `bson:"_id,omitempty"`
	OrganizationID bson.ObjectID `bson:"organizationId"`
}

func (s *Project) FromDomain(d *domain.Project) {
	if d == nil {
		return
	}

	s.Project = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.OrganizationID != "" {
		s.OrganizationID, _ = bson.ObjectIDFromHex(string(d.OrganizationID))
	}
}

func (s *Project) ToDomain() *domain.Project {
	if s == nil {
		return nil
	}

	s.Project.ID = domain.ID(s.ID.Hex())
	s.Project.OrganizationID = domain.ID(s.OrganizationID.Hex())

	return &s.Project
}
