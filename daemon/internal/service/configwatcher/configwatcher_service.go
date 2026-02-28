package configwatcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/pkg/pubsub"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
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
	paths *setup.ExpandedPaths,
	localConfigPort localconfig.LocalConfigPort,
) *Service {
	return &Service{
		logger:          l.With(slog.String("name", "configwatcher.service")),
		pubsub:          ps,
		configPath:      paths.GlobalConfigPath,
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
			Provider:          domain.ProviderClaudeCode,
		})

		// Note: Subdirectories are no longer pre-walked here.
		// The log watcher watches ~/.claude/projects/ and its subdirectories directly,
		// then matches log file paths to registered projects using prefix matching.

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
					Provider:          domain.ProviderClaudeCode,
				})
			}
		}
	}

	// Discover non-Claude provider targets
	targets = append(targets, s.discoverGeminiTargets()...)
	targets = append(targets, s.discoverCodexTargets()...)
	targets = append(targets, s.discoverOpenCodeTargets()...)

	return targets
}

// discoverGeminiTargets discovers Gemini CLI log directories.
// Gemini stores session JSON files at ~/.gemini/tmp/{project_hash}/chats/session-*.json.
// Each project_hash directory becomes a WatchTarget.
func (s *Service) discoverGeminiTargets() []domain.WatchTarget {
	baseDir := pathutil.GetGeminiLogBaseDir()
	if baseDir == "" {
		return nil
	}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		s.logger.Debug("failed to read gemini base dir",
			slog.String("path", baseDir),
			slog.Any("error", err),
		)
		return nil
	}

	var targets []domain.WatchTarget

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip Gemini internal directories
		name := entry.Name()
		if name == "bin" || name == "garden" {
			continue
		}

		chatsDir := filepath.Join(baseDir, name, "chats")
		if _, err := os.Stat(chatsDir); err == nil {
			targets = append(targets, domain.WatchTarget{
				LogDir:   chatsDir,
				Provider: domain.ProviderGeminiCLI,
				Type:     domain.WatchTargetRoot,
			})
		}
	}

	// Also add the base dir itself to detect new project hashes
	targets = append(targets, domain.WatchTarget{
		LogDir:   baseDir,
		Provider: domain.ProviderGeminiCLI,
		Type:     domain.WatchTargetRoot,
	})

	return targets
}

// discoverCodexTargets discovers Codex CLI log directories.
// Codex stores JSONL files at ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl.
// The sessions base directory and its date subdirectories become WatchTargets.
func (s *Service) discoverCodexTargets() []domain.WatchTarget {
	baseDir := pathutil.GetCodexSessionsBaseDir()
	if baseDir == "" {
		return nil
	}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}

	var targets []domain.WatchTarget

	// Walk to find all leaf date directories (YYYY/MM/DD) and intermediate dirs
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}
		if d.IsDir() {
			targets = append(targets, domain.WatchTarget{
				LogDir:   path,
				Provider: domain.ProviderCodexCLI,
				Type:     domain.WatchTargetRoot,
			})
		}
		return nil
	})
	if err != nil {
		s.logger.Debug("failed to walk codex sessions dir",
			slog.String("path", baseDir),
			slog.Any("error", err),
		)
	}

	return targets
}

// discoverOpenCodeTargets discovers OpenCode log directory.
// OpenCode uses a SQLite DB at ~/.local/share/opencode/opencode.db.
// Returns a single WatchTarget pointing to the data directory.
func (s *Service) discoverOpenCodeTargets() []domain.WatchTarget {
	dbPath := pathutil.GetOpenCodeDBPath()
	if dbPath == "" {
		return nil
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}

	return []domain.WatchTarget{
		{
			LogDir:   pathutil.GetOpenCodeDataDir(),
			Provider: domain.ProviderOpenCode,
			Type:     domain.WatchTargetRoot,
		},
	}
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
