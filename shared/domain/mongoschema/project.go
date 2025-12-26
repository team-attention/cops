package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	ProjectCollectionName = "projects"
)

const (
	ProjectIDField           = "_id"
	ProjectNameField         = "name"
	ProjectPathField         = "path"
	ProjectIsGitProjectField = "isGitProject"
	ProjectClaudeDirField    = "claudeDir"
	ProjectRegisteredAtField = "registeredAt"
	ProjectGitBranchField    = "git_branch"
	ProjectWorktreesField    = "worktrees"
	ProjectRemoteURLField    = "remoteUrl"
)

type Project struct {
	domain.Project `bson:",inline"`
	ID             bson.ObjectID `bson:"_id,omitempty"`
}

func (s *Project) FromDomain(d *domain.Project) {
	if d == nil {
		return
	}

	s.Project = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}
}

func (s *Project) ToDomain() *domain.Project {
	if s == nil {
		return nil
	}

	s.Project.ID = domain.ID(s.ID.Hex())

	return &s.Project
}
