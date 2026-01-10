package configwatcher

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/dirutil"
	"github.com/team-attention/cops/daemon/internal/platform/util/gitutil"
	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
	"github.com/team-attention/cops/daemon/internal/service/configwatcher/outbound/localconfig"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// Service contains pure business logic for config watching.
// No goroutines, no event loops - just business logic.
type Service struct {
	logger          *slog.Logger
	pubsub          pubsub.WriterPort[[]domain.WatchTarget]
	configPath      string
	localConfigPort localconfig.LocalConfigPort
}

// NewService creates a new ConfigWatcher service.
func NewService(
	l *slog.Logger,
	ps pubsub.WriterPort[[]domain.WatchTarget],
	cfg *setup.Config,
	localConfigPort localconfig.LocalConfigPort,
) *Service {
	return &Service{
		logger:          l.With(slog.String("name", "configwatcher.service")),
		pubsub:          ps,
		configPath:      cfg.Cops.GlobalConfigPath,
		localConfigPort: localConfigPort,
	}
}

// HandleConfigChange handles a config file change event.
// This is called by the Inbound handler when the file changes.
func (s *Service) HandleConfigChange(path string) error {
	cfg, err := s.loadConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	targets := s.buildWatchTargets(cfg)

	s.logger.Info("config loaded and targets built",
		slog.Int("projects", len(cfg.Projects)),
		slog.Int("targets", len(targets)),
	)

	return s.pubsub.Publish(targets)
}

// loadConfig loads and parses the global config file.
func (s *Service) loadConfig(path string) (*domain.GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty config file if it doesn't exist
			emptyConfig := &domain.GlobalConfig{Projects: []*shareddomain.Project{}}
			if err := s.saveConfig(path, emptyConfig); err != nil {
				return nil, fmt.Errorf("failed to create config file: %w", err)
			}
			s.logger.Info("created empty config file", slog.String("path", path))
			return emptyConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg domain.GlobalConfig
	if err := sonic.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// saveConfig saves the global config to file.
func (s *Service) saveConfig(path string, cfg *domain.GlobalConfig) error {
	data, err := sonic.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// buildWatchTargets builds watch targets from global config.
// This includes main project directories, git worktrees, and their subdirectories.
// Projects and worktrees without local config are skipped.
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
	var targets []domain.WatchTarget

	// First pass: collect all main project paths for priority checking
	mainProjectPaths := make(map[string]bool)
	for _, project := range cfg.Projects {
		if project != nil {
			mainProjectPaths[project.Path] = true
		}
	}

	for _, project := range cfg.Projects {
		if project == nil {
			continue
		}

		// Load local config - skip if not found
		localCfg, err := s.loadLocalConfig(project.Path)
		if err != nil {
			s.logger.Warn("skipping project without local config (project not registered)",
				slog.String("path", project.Path),
				slog.Any("error", err),
			)
			continue
		}

		// Skip projects without OrganizationID
		if localCfg.OrganizationID == "" {
			s.logger.Warn("skipping project without organization ID (run 'cops add' to re-register)",
				slog.String("path", project.Path),
			)
			continue
		}

		// Add main project directory
		targets = append(targets, domain.WatchTarget{
			ProjectPath:       project.Path,
			ClaudeDir:         pathutil.GetClaudeProjectDir(project.Path),
			Type:              domain.WatchTargetRoot,
			ProjectID:         localCfg.ProjectID,
			OrganizationID:    localCfg.OrganizationID,
			ParentProjectPath: "",
		})

		// Add subdirectories (excluding those that are main projects)
		subDirs, err := dirutil.WalkDirectories(project.Path)
		if err != nil {
			s.logger.Warn("failed to walk subdirectories",
				slog.String("path", project.Path),
				slog.Any("error", err),
			)
		} else {
			for _, subDir := range subDirs {
				if subDir == project.Path {
					continue // Skip root itself
				}
				// Skip if this subdirectory is a registered main project
				if mainProjectPaths[subDir] {
					continue
				}

				targets = append(targets, domain.WatchTarget{
					ProjectPath:       subDir,
					ClaudeDir:         pathutil.GetClaudeProjectDir(subDir),
					Type:              domain.WatchTargetSubdirectory,
					ProjectID:         localCfg.ProjectID,
					OrganizationID:    localCfg.OrganizationID,
					ParentProjectPath: project.Path,
				})
			}
		}

		// Add worktrees if git project
		if project.IsGitProject {
			worktrees, err := gitutil.GetWorktrees(project.Path)
			if err != nil {
				s.logger.Warn("failed to get worktrees",
					slog.String("path", project.Path),
					slog.Any("error", err),
				)
				continue
			}

			// Skip first element (main repo) as it's already added
			// Each worktree reads its own local config
			for _, wt := range worktrees[1:] {
				worktreeCfg, err := s.loadLocalConfig(wt)
				if err != nil {
					s.logger.Warn("skipping worktree without local config (worktree not registered)",
						slog.String("worktree", wt),
						slog.String("parentProject", project.Path),
						slog.Any("error", err),
					)
					continue
				}

				// Skip worktrees without OrganizationID
				if worktreeCfg.OrganizationID == "" {
					s.logger.Warn("skipping worktree without organization ID (run 'cops add' to re-register)",
						slog.String("worktree", wt),
					)
					continue
				}

				targets = append(targets, domain.WatchTarget{
					ProjectPath:       wt,
					ClaudeDir:         pathutil.GetClaudeProjectDir(wt),
					Type:              domain.WatchTargetWorktree,
					ProjectID:         worktreeCfg.ProjectID,
					OrganizationID:    worktreeCfg.OrganizationID,
					ParentProjectPath: project.Path,
				})

				// Add worktree subdirectories
				wtSubDirs, err := dirutil.WalkDirectories(wt)
				if err != nil {
					continue
				}
				for _, subDir := range wtSubDirs {
					if subDir == wt {
						continue
					}
					if mainProjectPaths[subDir] {
						continue
					}

					targets = append(targets, domain.WatchTarget{
						ProjectPath:       subDir,
						ClaudeDir:         pathutil.GetClaudeProjectDir(subDir),
						Type:              domain.WatchTargetSubdirectory,
						ProjectID:         worktreeCfg.ProjectID,
						OrganizationID:    worktreeCfg.OrganizationID,
						ParentProjectPath: wt,
					})
				}
			}
		}
	}

	return targets
}

// loadLocalConfig loads the full LocalConfig from the local config file.
// Returns error if config file is not found or cannot be read.
func (s *Service) loadLocalConfig(projectPath string) (*localconfig.LocalConfig, error) {
	localCfg, err := s.localConfigPort.LoadLocalConfig(projectPath)
	if err != nil {
		return nil, err
	}
	return localCfg, nil
}
