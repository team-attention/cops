package domain

// GlobalConfig represents ~/.cops/config.json structure.
type GlobalConfig struct {
	Projects []ProjectConfig `json:"projects"`
}

// ProjectConfig represents a project entry in GlobalConfig.
type ProjectConfig struct {
	Path         string `json:"path"`           // Project root directory
	Name         string `json:"name,omitempty"` // Display name (optional)
	IsGitProject bool   `json:"isGitProject"`   // CLI determines this when adding
}
