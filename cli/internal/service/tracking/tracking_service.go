package tracking

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/samber/lo"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
	"github.com/team-attention/cops/cli/internal/platform/util/gitutil"
	"github.com/team-attention/cops/cli/internal/platform/util/pathutil"
	"github.com/team-attention/cops/cli/internal/service/daemon"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/api"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/config"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/shared/domain"
)

// AddProjectParams contains parameters for AddProject.
type AddProjectParams struct {
	Path           string
	Name           string
	NoGit          bool
	Sync           bool
	OrganizationID string
}

// LocalConfigInfo contains information about a detected local config.
type LocalConfigInfo struct {
	ProjectID         domain.ID
	OrganizationID    string
	ProjectPath       string
	AlreadyRegistered bool // True if project is already in ~/.cops/config.json
}

// ServerProjectInfo contains information about a project on the server.
type ServerProjectInfo struct {
	Found          bool
	ProjectID      string
	Name           string
	OrganizationID string
}

// Service provides tracking operations.
type Service struct {
	logger *slog.Logger

	authState  authstate.AuthStatePort
	configRepo config.ConfigPort
	parser     parser.ParserPort
	project    api.ProjectPort
	daemonSvc  *daemon.Service
}

// NewService creates a new tracking service.
func NewService(
	l *slog.Logger,
	authState authstate.AuthStatePort,
	configRepo config.ConfigPort,
	parser parser.ParserPort,
	project api.ProjectPort,
	daemonSvc *daemon.Service,
) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "tracking.service")),
		authState:  authState,
		configRepo: configRepo,
		parser:     parser,
		project:    project,
		daemonSvc:  daemonSvc,
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

	// Use provided name or fall back to directory name
	name := params.Name
	if name == "" {
		name = filepath.Base(projectPath)
	}

	// Determine project ID
	var projectID domain.ID
	var existingProjectID string

	// Check if local config exists
	if s.configRepo.LocalConfigExists(projectPath) {
		localConfig, err := s.configRepo.LoadLocalConfig(projectPath)
		if err != nil {
			return nil, errutil.Internalf("failed to load local config: %v", err)
		}
		existingProjectID = localConfig.ProjectID.String()
	}

	// Get URLs (empty strings if not available)
	configuredURL := ""
	actualURL := ""
	if isGitProject {
		configuredURL, _ = gitutil.GetRemoteURL(projectPath)
		actualURL = gitutil.GetActualRemoteURL(projectPath)
	}

	// Get access token for API authentication
	accessToken, err := s.authState.GetAccessToken(ctx)
	if err != nil {
		s.logger.Error("failed to get access token", slog.Any("error", err))
		return nil, errutil.Internalf("authentication failed: %v", err)
	}

	// Always call API to register project
	result, err := s.project.RegisterProject(ctx, accessToken, api.RegisterProjectParams{
		ConfiguredRemoteURL: configuredURL,
		ActualRemoteURL:     actualURL,
		ExistingProjectID:   existingProjectID,
		Name:                name,
		IsGitProject:        isGitProject,
		OrganizationID:      params.OrganizationID,
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
	localConfig := &config.LocalConfig{
		ProjectID:      projectID,
		OrganizationID: params.OrganizationID,
	}
	if err := s.configRepo.SaveLocalConfig(projectPath, localConfig); err != nil {
		return nil, errutil.Internalf("failed to save local config: %v", err)
	}

	// Create project
	project := &domain.Project{
		ProjectAbstract: domain.ProjectAbstract{
			ID:   projectID,
			Name: name,
			Path: projectPath,
		},
		IsGitProject: isGitProject,
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
// Requests the daemon to scan and upload log files for the specified project.
func (s *Service) SyncProject(ctx context.Context, projectID domain.ID) error {
	// Load global config to find project by ID
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return errutil.Internalf("failed to load global config: %v", err)
	}

	// Find project by ID
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

	// Load local config to get OrganizationID
	localConfig, err := s.configRepo.LoadLocalConfig(project.Path)
	if err != nil {
		return errutil.Internalf("failed to load local config: %v", err)
	}

	// Request daemon to scan logs
	return s.daemonSvc.RequestLogScan(ctx, daemon.RequestLogScanParams{
		ProjectID:      projectID.String(),
		ProjectPath:    project.Path,
		OrganizationID: localConfig.OrganizationID,
	})
}

// FindLocalConfig checks if local config exists and whether the project is in global registry.
// Returns LocalConfigInfo with AlreadyRegistered=true if project exists in global registry.
// Returns LocalConfigInfo with AlreadyRegistered=false if local config exists but not registered.
// Returns nil if local config doesn't exist.
func (s *Service) FindLocalConfig(targetPath string) (*LocalConfigInfo, error) {
	// 1. Expand and validate the target path
	absPath, err := pathutil.ExpandPath(targetPath)
	if err != nil {
		return nil, errutil.BadRequestf("invalid path: %v", err)
	}

	// 2. Check if local config exists at the target path
	if !s.configRepo.LocalConfigExists(absPath) {
		// No local config found
		return nil, nil
	}

	// 3. Load the local config
	localConfig, err := s.configRepo.LoadLocalConfig(absPath)
	if err != nil {
		return nil, errutil.Internalf("failed to load local config: %v", err)
	}

	// 4. Load global config
	globalConfig, err := s.configRepo.LoadGlobalConfig()
	if err != nil {
		return nil, errutil.Internalf("failed to load global config: %v", err)
	}

	// 5. Check if any project in global config has the same ID as local config's ProjectID
	alreadyRegistered := false
	for _, p := range globalConfig.Projects {
		if p.ID == localConfig.ProjectID {
			alreadyRegistered = true
			break
		}
	}

	// 6. Return LocalConfigInfo with AlreadyRegistered flag
	return &LocalConfigInfo{
		ProjectID:         localConfig.ProjectID,
		OrganizationID:    localConfig.OrganizationID,
		ProjectPath:       absPath,
		AlreadyRegistered: alreadyRegistered,
	}, nil
}

// CheckProjectOnServer verifies if a project with given ID exists on the server within the specified organization.
// Uses both projectID and organizationID to ensure accurate matching and security.
// Returns ServerProjectInfo with Found=false if project doesn't exist, belongs to different org, or API call fails.
func (s *Service) CheckProjectOnServer(ctx context.Context, projectID string, organizationID string) (*ServerProjectInfo, error) {
	// Get access token from authState
	accessToken, err := s.authState.GetAccessToken(ctx)
	if err != nil {
		s.logger.Error("failed to get access token", slog.Any("error", err))
		return nil, errutil.Internalf("authentication failed: %v", err)
	}

	// Call project.GetProjectByIDAndOrg
	result, err := s.project.GetProjectByIDAndOrg(ctx, accessToken, projectID, organizationID)
	if err != nil {
		// Graceful degradation: treat API failures as "not found"
		s.logger.Warn("failed to check project on server, treating as not found",
			slog.String("projectID", projectID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err))
		return &ServerProjectInfo{Found: false}, nil
	}

	// Convert API result to ServerProjectInfo
	return &ServerProjectInfo{
		Found:          result.Found,
		ProjectID:      result.ProjectID,
		Name:           result.Name,
		OrganizationID: result.OrganizationID,
	}, nil
}
