package configwatcher

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/gitutil"
	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
)

// Service contains pure business logic for config watching.
// No goroutines, no event loops - just business logic.
type Service struct {
	logger     *slog.Logger
	pubsub     pubsub.WriterPort[[]domain.WatchTarget]
	configPath string
}

// NewService creates a new ConfigWatcher service.
func NewService(l *slog.Logger, ps pubsub.WriterPort[[]domain.WatchTarget], cfg *setup.Config) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "configwatcher.service")),
		pubsub:     ps,
		configPath: cfg.Cops.GlobalConfigPath,
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
			emptyConfig := &domain.GlobalConfig{Projects: []domain.ProjectConfig{}}
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
// This includes main project directories and git worktrees.
func (s *Service) buildWatchTargets(cfg *domain.GlobalConfig) []domain.WatchTarget {
	var targets []domain.WatchTarget

	for _, project := range cfg.Projects {
		// Add main project directory
		targets = append(targets, domain.WatchTarget{
			ProjectPath: project.Path,
			ClaudeDir:   pathutil.GetClaudeProjectDir(project.Path),
			Type:        domain.WatchTargetRoot,
		})

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
			for _, wt := range worktrees[1:] {
				targets = append(targets, domain.WatchTarget{
					ProjectPath: wt,
					ClaudeDir:   pathutil.GetClaudeProjectDir(wt),
					Type:        domain.WatchTargetWorktree,
				})
			}
		}
	}

	return targets
}
