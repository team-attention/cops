package project

import (
	"log/slog"
	"sync"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/util/gitutil"
	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
)

// Service manages projects and generates watch targets.
type Service struct {
	logger       *slog.Logger
	watchTargets []domain.WatchTarget
	mu           sync.RWMutex
}

// NewService creates a new Project service.
func NewService(l *slog.Logger) *Service {
	return &Service{
		logger:       l.With(slog.String("name", "project.service")),
		watchTargets: []domain.WatchTarget{},
	}
}

// UpdateProjects updates the watch targets based on GlobalConfig.
func (s *Service) UpdateProjects(cfg domain.GlobalConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var targets []domain.WatchTarget

	for _, project := range cfg.Projects {
		if !project.Active {
			continue
		}

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

	s.watchTargets = targets
	s.logger.Info("updated watch targets",
		slog.Int("count", len(targets)),
	)

	return nil
}

// GetWatchTargets returns the current watch targets.
func (s *Service) GetWatchTargets() []domain.WatchTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.WatchTarget, len(s.watchTargets))
	copy(result, s.watchTargets)
	return result
}
