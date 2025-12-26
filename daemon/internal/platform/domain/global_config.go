package domain

import (
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// GlobalConfig represents ~/.cops/config.json structure.
type GlobalConfig struct {
	Projects []*shareddomain.Project `json:"projects"`
}
