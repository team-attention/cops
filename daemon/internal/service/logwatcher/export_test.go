package logwatcher

import shareddomain "github.com/team-attention/cops/shared/domain"

// ProjectMappingForTest exposes projectMapping for testing.
type ProjectMappingForTest struct {
	ProjectID      shareddomain.ID
	OrganizationID string
	Priority       WatchTargetPriority
}

// ProjectAssociatorForTest wraps projectAssociator for testing.
type ProjectAssociatorForTest struct {
	inner projectAssociator
}

// NewProjectAssociatorForTest creates a projectAssociator for testing.
func NewProjectAssociatorForTest(mappings map[string]ProjectMappingForTest) ProjectAssociatorForTest {
	if mappings == nil {
		return ProjectAssociatorForTest{
			inner: newProjectAssociator(make(map[string]projectMapping)),
		}
	}

	m := make(map[string]projectMapping, len(mappings))
	for path, tm := range mappings {
		m[path] = projectMapping{
			ProjectID:      tm.ProjectID,
			OrganizationID: tm.OrganizationID,
			Priority:       tm.Priority,
			PathLength:     len(path),
		}
	}

	return ProjectAssociatorForTest{
		inner: newProjectAssociator(m),
	}
}

// ResolveGemini exposes resolveGemini for testing.
func (a ProjectAssociatorForTest) ResolveGemini(logDir string) shareddomain.ID {
	return a.inner.resolveGemini(logDir)
}

// ResolveByProjectPath exposes resolveByProjectPath for testing.
func (a ProjectAssociatorForTest) ResolveByProjectPath(path string) shareddomain.ID {
	return a.inner.resolveByProjectPath(path)
}

// ResolveCodexCwd exposes resolveCodexCwd for testing.
func (a ProjectAssociatorForTest) ResolveCodexCwd(cwd string) shareddomain.ID {
	return a.inner.resolveCodexCwd(cwd)
}

// ResolveOpenCode exposes resolveOpenCode for testing.
func (a ProjectAssociatorForTest) ResolveOpenCode(projectPath string) shareddomain.ID {
	return a.inner.resolveOpenCode(projectPath)
}
