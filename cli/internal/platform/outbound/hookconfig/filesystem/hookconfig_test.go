package filesystem_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig"
	"github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig/filesystem"
)

func TestFilesystemHookConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filesystem HookConfig Suite")
}

var _ = Describe("FilesystemHookConfig", func() {
	var (
		logger     *slog.Logger
		tempDir    string
		projectDir string
		authPath   string
		ctx        context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		var err error
		tempDir, err = os.MkdirTemp("", "hookconfig-test")
		Expect(err).NotTo(HaveOccurred())

		projectDir = filepath.Join(tempDir, "project")
		Expect(os.MkdirAll(filepath.Join(projectDir, ".claude"), 0755)).To(Succeed())

		// Create auth path in temp directory
		authPath = filepath.Join(tempDir, ".cops", "auth.json")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("LoadHookSettings", func() {
		Context("when settings file exists with cops section", func() {
			BeforeEach(func() {
				settingsContent := `{
					"cops": {
						"enabled": true,
						"events": {
							"postToolUse": false,
							"sessionStart": true
						}
					}
				}`
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte(settingsContent), 0644)).To(Succeed())
			})

			It("loads the hook settings correctly", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				settings, err := adapter.LoadHookSettings(ctx, projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).NotTo(BeNil())
				Expect(settings.Enabled).To(BeTrue())
				Expect(settings.Events).NotTo(BeNil())
				Expect(*settings.Events.PostToolUse).To(BeFalse())
				Expect(*settings.Events.SessionStart).To(BeTrue())
			})
		})

		Context("when settings file exists without cops section", func() {
			BeforeEach(func() {
				settingsContent := `{
					"otherSettings": {
						"foo": "bar"
					}
				}`
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte(settingsContent), 0644)).To(Succeed())
			})

			It("returns nil without error", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				settings, err := adapter.LoadHookSettings(ctx, projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).To(BeNil())
			})
		})

		Context("when settings file does not exist", func() {
			It("returns nil without error", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				emptyDir := filepath.Join(tempDir, "empty")
				Expect(os.MkdirAll(emptyDir, 0755)).To(Succeed())

				settings, err := adapter.LoadHookSettings(ctx, emptyDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).To(BeNil())
			})
		})

		Context("when settings file contains invalid JSON", func() {
			BeforeEach(func() {
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte("invalid json"), 0644)).To(Succeed())
			})

			It("returns an error", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				settings, err := adapter.LoadHookSettings(ctx, projectDir)
				Expect(err).To(HaveOccurred())
				Expect(settings).To(BeNil())
			})
		})

		Context("when settings file has minimal cops section", func() {
			BeforeEach(func() {
				settingsContent := `{
					"cops": {
						"enabled": true
					}
				}`
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte(settingsContent), 0644)).To(Succeed())
			})

			It("loads with nil events config", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				settings, err := adapter.LoadHookSettings(ctx, projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(settings).NotTo(BeNil())
				Expect(settings.Enabled).To(BeTrue())
				Expect(settings.Events).To(BeNil())
			})
		})
	})

	Describe("LoadConfig", func() {
		Context("when hook is enabled but auth config is missing", func() {
			BeforeEach(func() {
				settingsContent := `{"cops": {"enabled": true}}`
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte(settingsContent), 0644)).To(Succeed())
			})

			It("returns ErrAPIKeyRequired", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				cfg, err := adapter.LoadConfig(ctx, projectDir)
				Expect(err).To(MatchError(hookconfig.ErrAPIKeyRequired))
				Expect(cfg).To(BeNil())
			})
		})

		Context("when hook is disabled", func() {
			BeforeEach(func() {
				settingsContent := `{"cops": {"enabled": false}}`
				settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
				Expect(os.WriteFile(settingsPath, []byte(settingsContent), 0644)).To(Succeed())
			})

			It("loads config successfully without auth", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				cfg, err := adapter.LoadConfig(ctx, projectDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.IsEnabled()).To(BeFalse())
			})
		})

		Context("when no settings file exists", func() {
			It("loads config with nil hook settings", func() {
				adapter := filesystem.NewFilesystemHookConfig(logger, authPath)

				emptyDir := filepath.Join(tempDir, "empty")
				Expect(os.MkdirAll(emptyDir, 0755)).To(Succeed())

				cfg, err := adapter.LoadConfig(ctx, emptyDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Hook).To(BeNil())
				Expect(cfg.IsEnabled()).To(BeFalse())
			})
		})
	})
})
