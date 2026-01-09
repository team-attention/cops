package hookconfig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/cli/internal/platform/outbound/hookconfig"
)

func TestHookConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hook Config Suite")
}

var _ = Describe("Config", func() {
	Describe("IsEventEnabled", func() {
		Context("when Hook is nil", func() {
			It("returns false for all events", func() {
				cfg := &hookconfig.Config{Hook: nil}

				Expect(cfg.IsEventEnabled(hookconfig.EventTypePostToolUse)).To(BeFalse())
				Expect(cfg.IsEventEnabled(hookconfig.EventTypeSessionStart)).To(BeFalse())
			})
		})

		Context("when Hook is disabled", func() {
			It("returns false for all events", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: false},
				}

				for _, eventType := range hookconfig.AllEventTypes() {
					Expect(cfg.IsEventEnabled(eventType)).To(BeFalse())
				}
			})
		})

		Context("when Hook is enabled with no event config", func() {
			It("returns true for all events", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: true},
					Auth: &hookconfig.AuthConfig{APIKey: "test-key"},
				}

				for _, eventType := range hookconfig.AllEventTypes() {
					Expect(cfg.IsEventEnabled(eventType)).To(BeTrue())
				}
			})
		})

		Context("when specific events are disabled", func() {
			It("returns false for disabled events and true for others", func() {
				disabled := false
				enabled := true
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{
						Enabled: true,
						Events: &hookconfig.EventConfig{
							PostToolUse:  &disabled,
							SessionStart: &enabled,
						},
					},
					Auth: &hookconfig.AuthConfig{APIKey: "test-key"},
				}

				Expect(cfg.IsEventEnabled(hookconfig.EventTypePostToolUse)).To(BeFalse())
				Expect(cfg.IsEventEnabled(hookconfig.EventTypeSessionStart)).To(BeTrue())
				// Events not explicitly set default to enabled
				Expect(cfg.IsEventEnabled(hookconfig.EventTypeNotification)).To(BeTrue())
			})
		})

		Context("with unknown event type", func() {
			It("returns true (default enabled)", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: true},
					Auth: &hookconfig.AuthConfig{APIKey: "test-key"},
				}

				Expect(cfg.IsEventEnabled(hookconfig.EventType("UnknownEvent"))).To(BeTrue())
			})
		})
	})

	Describe("IsEnabled", func() {
		It("returns false when Hook is nil", func() {
			cfg := &hookconfig.Config{Hook: nil}
			Expect(cfg.IsEnabled()).To(BeFalse())
		})

		It("returns false when Hook.Enabled is false", func() {
			cfg := &hookconfig.Config{
				Hook: &hookconfig.HookSettings{Enabled: false},
			}
			Expect(cfg.IsEnabled()).To(BeFalse())
		})

		It("returns true when Hook.Enabled is true", func() {
			cfg := &hookconfig.Config{
				Hook: &hookconfig.HookSettings{Enabled: true},
			}
			Expect(cfg.IsEnabled()).To(BeTrue())
		})
	})

	Describe("Validate", func() {
		Context("when Hook is enabled but API key is missing", func() {
			It("returns ErrAPIKeyRequired with nil Auth", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: true},
					Auth: nil,
				}

				Expect(cfg.Validate()).To(MatchError(hookconfig.ErrAPIKeyRequired))
			})

			It("returns ErrAPIKeyRequired with empty API key", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: true},
					Auth: &hookconfig.AuthConfig{APIKey: ""},
				}

				Expect(cfg.Validate()).To(MatchError(hookconfig.ErrAPIKeyRequired))
			})
		})

		Context("when Hook is enabled and API key is present", func() {
			It("returns nil", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: true},
					Auth: &hookconfig.AuthConfig{APIKey: "test-key"},
				}

				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Context("when Hook is disabled", func() {
			It("returns nil even without API key", func() {
				cfg := &hookconfig.Config{
					Hook: &hookconfig.HookSettings{Enabled: false},
					Auth: nil,
				}

				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Context("when Hook is nil", func() {
			It("returns nil", func() {
				cfg := &hookconfig.Config{Hook: nil}

				Expect(cfg.Validate()).To(Succeed())
			})
		})
	})
})

var _ = Describe("AllEventTypes", func() {
	It("returns all 7 event types", func() {
		eventTypes := hookconfig.AllEventTypes()
		Expect(eventTypes).To(HaveLen(7))
		Expect(eventTypes).To(ContainElements(
			hookconfig.EventTypePostToolUse,
			hookconfig.EventTypeNotification,
			hookconfig.EventTypeUserPromptSubmit,
			hookconfig.EventTypeStop,
			hookconfig.EventTypeSubagentStop,
			hookconfig.EventTypeSessionStart,
			hookconfig.EventTypeSessionEnd,
		))
	})
})
