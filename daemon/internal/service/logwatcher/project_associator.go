package logwatcher

import (
	"strings"

	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// projectAssociator resolves provider-specific identifiers to registered Cops project IDs.
// It maintains lookup tables for each provider's project identification scheme and
// the shared hierarchical path matching algorithm used across all path-based providers.
// This is a value type (not pointer) because it is rebuilt on every UpdateTargets call
// and only used within the mutex-protected Service.
type projectAssociator struct {
	// geminiHashToProjectID maps Gemini project hash -> ProjectID for registered projects.
	geminiHashToProjectID map[string]shareddomain.ID

	// projectPathToID maps absolute project path -> ProjectID for path-based matching
	// (used by Codex CLI which provides cwd and OpenCode which provides project path).
	projectPathToID map[string]shareddomain.ID

	// projectMappings stores project path -> mapping info with priority
	// for hierarchical path matching (shared across Claude, Codex, and OpenCode).
	projectMappings map[string]projectMapping
}

// newProjectAssociator creates a projectAssociator from registered project data.
// Returns a value type with initialized maps (never nil maps).
func newProjectAssociator(mappings map[string]projectMapping) projectAssociator {
	projectPaths := make([]string, 0, len(mappings))
	projectPathToID := make(map[string]shareddomain.ID, len(mappings))

	for path, m := range mappings {
		projectPaths = append(projectPaths, path)
		projectPathToID[path] = m.ProjectID
	}

	// Build Gemini hash lookup table
	hashToPath := pathutil.BuildGeminiHashToPathMap(projectPaths)
	geminiHashToProjectID := make(map[string]shareddomain.ID, len(hashToPath))
	for hash, path := range hashToPath {
		if id, ok := projectPathToID[path]; ok {
			geminiHashToProjectID[hash] = id
		}
	}

	// Copy projectMappings for hierarchical matching
	copiedMappings := make(map[string]projectMapping, len(mappings))
	for k, v := range mappings {
		copiedMappings[k] = v
	}

	return projectAssociator{
		geminiHashToProjectID: geminiHashToProjectID,
		projectPathToID:       projectPathToID,
		projectMappings:       copiedMappings,
	}
}

// resolveGemini resolves a Gemini CLI log directory to a registered project ID.
// Extracts the project hash from the directory path and looks it up.
func (a projectAssociator) resolveGemini(logDir string) shareddomain.ID {
	hash := pathutil.ExtractGeminiProjectHash(logDir)
	if hash == "" {
		return ""
	}
	return a.geminiHashToProjectID[hash]
}

// resolveByProjectPath resolves a project path to a registered project ID using
// hierarchical path matching: exact match first, then longest prefix match with priority.
// This is the single shared implementation used by Claude (after path decoding),
// Codex CWD resolution, and OpenCode project path resolution.
func (a projectAssociator) resolveByProjectPath(path string) shareddomain.ID {
	// Try exact match first
	if id, ok := a.projectPathToID[path]; ok {
		return id
	}

	// Find best match by priority and path length
	var bestMatch *projectMapping
	var bestMatchPath string

	for projectPath, mapping := range a.projectMappings {
		if !strings.HasPrefix(path, projectPath+"/") {
			continue
		}

		if bestMatch == nil {
			m := mapping
			bestMatch = &m
			bestMatchPath = projectPath
			continue
		}

		// Compare priority (lower is better)
		if mapping.Priority < bestMatch.Priority {
			m := mapping
			bestMatch = &m
			bestMatchPath = projectPath
		} else if mapping.Priority == bestMatch.Priority {
			// Same priority: prefer longer path (more specific match)
			if len(projectPath) > len(bestMatchPath) {
				m := mapping
				bestMatch = &m
				bestMatchPath = projectPath
			}
		}
	}

	if bestMatch != nil {
		return bestMatch.ProjectID
	}
	return ""
}

// resolveCodexCwd resolves a Codex CLI working directory path to a registered project ID.
// Delegates to resolveByProjectPath for hierarchical path matching.
func (a projectAssociator) resolveCodexCwd(cwd string) shareddomain.ID {
	return a.resolveByProjectPath(cwd)
}

// resolveOpenCode resolves an OpenCode project path to a registered project ID.
// Delegates to resolveByProjectPath for hierarchical path matching.
func (a projectAssociator) resolveOpenCode(projectPath string) shareddomain.ID {
	return a.resolveByProjectPath(projectPath)
}
