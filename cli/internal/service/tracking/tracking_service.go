package tracking

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/samber/lo"

	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
	"github.com/team-attention/cops/cli/internal/platform/util/gitutil"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/shared/domain"
)

// AddProjectParams contains parameters for AddProject.
type AddProjectParams struct {
	Path  string
	NoGit bool
	Sync  bool
}

// Service provides tracking operations.
type Service struct {
	logger *slog.Logger

	configRepo config.ConfigPort
	parser     parser.ParserPort
	collector  api.CollectorPort
	project    api.ProjectPort
}

// NewService creates a new tracking service.
func NewService(
	l *slog.Logger,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	collector api.CollectorPort,
	project api.ProjectPort,
) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "tracking.service")),
		configRepo: configRepo,
		parser:     parser,
		collector:  collector,
		project:    project,
	}
}

// AddProject registers a new project for tracking.
func (s *Service) AddProject(ctx context.Context, params AddProjectParams) (*domain.Project, error) {
	// Expand and validate path
	absPath, err := pathutil.ExpandPath(params.Path)
	if err != nil {
		return nil, errutil.BadRequestf("invalid path: %v", err)
	}

	var projectPath string
	var isGitProject bool

	// Check if git project (unless --no-git flag)
	if !params.NoGit && gitutil.IsGitRepo(absPath) {
		// Find main repo path (not worktree)
		mainPath, err := gitutil.FindMainRepoPath(absPath)
		if err != nil {
			s.logger.Warn("failed to find main repo path, using current path",
				slog.Any("error", err))
			projectPath = absPath
			isGitProject = false
		} else {
			projectPath = mainPath
			isGitProject = true
		}
	} else {
		projectPath = absPath
		isGitProject = false
	}

	// Use directory name as project name
	name := filepath.Base(projectPath)

	// Determine project ID
	var projectID domain.ID
	var existingProjectID string

	// Check if local config exists
	if s.configRepo.LocalConfigExists(projectPath) {
		localConfig, err := s.configRepo.LoadLocalConfig(projectPath)
		if err != nil {
			return nil, errutil.Internalf("failed to load local config: %v", err)
		}
		existingProjectID = localConfig.ID.String()
	}

	// Get URLs (empty strings if not available)
	configuredURL := ""
	actualURL := ""
	if isGitProject {
		configuredURL, _ = gitutil.GetRemoteURL(projectPath)
		actualURL = gitutil.GetActualRemoteURL(projectPath)
	}

	// Always call API to register project
	result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
		ConfiguredRemoteURL: configuredURL,
		ActualRemoteURL:     actualURL,
		ExistingProjectID:   existingProjectID,
	})
	if err != nil {
		// If we have an existing local ID, use it
		if existingProjectID != "" {
			projectID = domain.ID(existingProjectID)
			s.logger.Warn("failed to register with API, using existing local ID",
				slog.Any("error", err),
				slog.String("id", projectID.String()))
		} else {
			// No existing ID and API unreachable - FAIL
			return nil, errutil.Internalf("cannot register project: API unreachable and no existing local ID: %v", err)
		}
	} else {
		projectID = result.ProjectID
		s.logger.Debug("registered project with API",
			slog.String("id", projectID.String()),
			slog.Bool("isNew", result.IsNew))
	}

	// Save local config with projectID
	localConfig := &config.LocalConfig{ID: projectID}
	if err := s.configRepo.SaveLocalConfig(projectPath, localConfig); err != nil {
		return nil, errutil.Internalf("failed to save local config: %v", err)
	}

	// Get Claude directory for this project
	claudeBaseDir, err := pathutil.DefaultClaudeDir()
	if err != nil {
		return nil, errutil.Internalf("failed to get Claude dir: %v", err)
	}
	claudeProjectDir := pathutil.GetClaudeProjectDir(claudeBaseDir, projectPath)

	// Create project
	project := &domain.Project{
		ProjectAbstract: domain.ProjectAbstract{
			ID:   projectID,
			Name: name,
			Path: projectPath,
		},
		IsGitProject: isGitProject,
		ClaudeDir:    claudeProjectDir,
		RegisteredAt: time.Now(),
	}

	// Check if project already in global registry
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return nil, errutil.Internalf("failed to load global config: %v", err)
	}

	// Check for existing project with same ID
	for _, p := range globalConfig.Projects {
		if p.ID == project.ID {
			s.logger.Info("project already registered",
				slog.String("id", project.ID.String()),
				slog.String("path", project.Path))
			return project, nil
		}
	}

	// Add to global registry
	globalConfig.Projects = append(globalConfig.Projects, project)
	if err := s.configRepo.SaveGlobalConfig(globalConfig); err != nil {
		return nil, errutil.Internalf("failed to save global config: %v", err)
	}

	s.logger.Info("project added",
		slog.String("id", project.ID.String()),
		slog.String("name", project.Name),
		slog.String("path", project.Path),
		slog.Bool("gitProject", project.IsGitProject))

	// Sync if requested
	if params.Sync {
		if err := s.SyncProject(ctx, project.ID); err != nil {
			s.logger.Warn("sync failed", slog.Any("error", err))
			// Don't fail the add operation
		}
	}

	return project, nil
}

// ListProjects returns all registered projects with their worktrees.
func (s *Service) ListProjects(ctx context.Context) ([]*domain.ProjectWithWorktrees, error) {
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return nil, errutil.Internalf("failed to load global config: %v", err)
	}

	result := make([]*domain.ProjectWithWorktrees, 0, len(globalConfig.Projects))

	for _, project := range globalConfig.Projects {
		projectWithWorktrees := &domain.ProjectWithWorktrees{
			Project: *project,
		}

		// Discover worktrees for git projects
		if project.IsGitProject {
			worktrees, err := gitutil.ListWorktrees(project.Path)
			if err != nil {
				s.logger.Warn("failed to list worktrees",
					slog.String("project", project.Path),
					slog.Any("error", err))
			} else {
				projectWithWorktrees.Worktrees = worktrees
			}
		}

		result = append(result, projectWithWorktrees)
	}

	return result, nil
}

// RemoveProject removes a project from tracking.
func (s *Service) RemoveProject(ctx context.Context, projectID domain.ID) error {
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return errutil.Internalf("failed to load global config: %v", err)
	}

	// Filter out the project
	filtered := make([]*domain.Project, 0, len(globalConfig.Projects))
	found := false
	for _, p := range globalConfig.Projects {
		if p.ID != projectID {
			filtered = append(filtered, p)
		} else {
			found = true
		}
	}

	if !found {
		return errutil.NotFoundf("project not found: %s", projectID)
	}

	globalConfig.Projects = filtered
	if err := s.configRepo.SaveGlobalConfig(globalConfig); err != nil {
		return errutil.Internalf("failed to save global config: %v", err)
	}

	s.logger.Info("project removed", slog.String("id", projectID.String()))
	return nil
}

// RemoveProjectByPathParams contains parameters for RemoveProjectByPath.
type RemoveProjectByPathParams struct {
	Path string
}

// RemoveProjectByPath removes a project from tracking by its path.
// This only removes from local configs (global config + local .cops/ directory).
// Does NOT delete Claude Code session logs.
// Does NOT communicate with API server.
func (s *Service) RemoveProjectByPath(ctx context.Context, params RemoveProjectByPathParams) error {
	// Expand and validate path
	absPath, err := pathutil.ExpandPath(params.Path)
	if err != nil {
		return errutil.BadRequestf("invalid path: %v", err)
	}

	// Delete local config first (graceful if not exists)
	if err := s.configRepo.DeleteLocalConfig(absPath); err != nil {
		s.logger.Warn("failed to delete local config, continuing with global config removal",
			slog.String("path", absPath),
			slog.Any("error", err))
		// Continue with global config update
	}

	// Load global config
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return errutil.Internalf("failed to load global config: %v", err)
	}

	// Filter out project with matching path
	filtered := lo.Filter(globalConfig.Projects, func(p *domain.Project, _ int) bool {
		return p.Path != absPath
	})
	found := len(filtered) < len(globalConfig.Projects)

	// Save global config if project was found
	if found {
		globalConfig.Projects = filtered
		if err := s.configRepo.SaveGlobalConfig(globalConfig); err != nil {
			return errutil.Internalf("failed to save global config: %v", err)
		}
		s.logger.Info("project removed from tracking",
			slog.String("path", absPath))
	} else {
		s.logger.Info("project not in global config, local config deleted if existed",
			slog.String("path", absPath))
	}

	return nil
}

// SyncProject syncs session records for a project to the collector.
func (s *Service) SyncProject(ctx context.Context, projectID domain.ID) error {
	// Find project
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return errutil.Internalf("failed to load global config: %v", err)
	}

	var project *domain.Project
	for _, p := range globalConfig.Projects {
		if p.ID == projectID {
			project = p
			break
		}
	}

	if project == nil {
		return errutil.NotFoundf("project not found: %s", projectID)
	}

	// Parse session files
	records, err := s.parser.ParseSessionFiles(project.ClaudeDir)
	if err != nil {
		return errutil.Internalf("failed to parse sessions: %v", err)
	}

	if len(records) == 0 {
		s.logger.Info("no records to sync",
			slog.String("projectId", projectID.String()))
		return nil
	}

	// Send to collector
	if err := s.collector.SendRecords(ctx, project, records); err != nil {
		return errutil.Internalf("Collector server must be running for sync: %v", err)
	}

	s.logger.Info("project synced",
		slog.String("projectId", projectID.String()),
		slog.Int("records", len(records)))

	return nil
}
