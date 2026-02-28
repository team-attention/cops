package session

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Provider Suite")
}

var _ = Describe("DetectProvider", func() {
	Context("when detecting OpenCode format", func() {
		It("detects OpenCode user message", func() {
			data := map[string]any{
				"id":        "msg_1",
				"sessionId": "s1",
				"role":      "user",
				"parts":     "[]",
				"model":     "",
				"createdAt": 1700000000.0,
				"updatedAt": 1700000000.0,
			}
			Expect(DetectProvider(data)).To(Equal(ProviderOpenCode))
		})

		It("detects OpenCode assistant message", func() {
			data := map[string]any{
				"id":        "msg_2",
				"sessionId": "s1",
				"role":      "assistant",
				"parts":     `[{"type":"text"}]`,
				"model":     "claude-sonnet-4-20250514",
				"createdAt": 1700000000.0,
				"updatedAt": 1700000000.0,
			}
			Expect(DetectProvider(data)).To(Equal(ProviderOpenCode))
		})

		It("rejects non-string parts field", func() {
			data := map[string]any{
				"sessionId": "s1",
				"role":      "user",
				"parts":     []any{},
			}
			Expect(DetectProvider(data)).To(Equal(ProviderUnknown))
		})

		It("rejects missing sessionId field", func() {
			data := map[string]any{
				"role":  "user",
				"parts": "[]",
			}
			Expect(DetectProvider(data)).To(Equal(ProviderUnknown))
		})

		It("rejects invalid role value", func() {
			data := map[string]any{
				"sessionId": "s1",
				"role":      "system",
				"parts":     "[]",
			}
			Expect(DetectProvider(data)).To(Equal(ProviderUnknown))
		})
	})

	Context("when detecting Claude Code format", func() {
		It("detects Claude Code user message", func() {
			data := map[string]any{
				"type":    "user",
				"message": map[string]any{},
			}
			Expect(DetectProvider(data)).To(Equal(ProviderClaudeCode))
		})

		It("detects Claude Code assistant message", func() {
			data := map[string]any{
				"type":    "assistant",
				"message": map[string]any{},
			}
			Expect(DetectProvider(data)).To(Equal(ProviderClaudeCode))
		})

		It("takes priority over OpenCode when both could match", func() {
			data := map[string]any{
				"type":      "user",
				"sessionId": "s1",
				"role":      "user",
				"parts":     "[]",
			}
			Expect(DetectProvider(data)).To(Equal(ProviderClaudeCode))
		})
	})

	Context("when detecting Gemini CLI format", func() {
		It("detects Gemini CLI session", func() {
			data := map[string]any{
				"sessionId": "s1",
				"messages":  []any{},
			}
			Expect(DetectProvider(data)).To(Equal(ProviderGeminiCLI))
		})

		It("does not confuse Gemini sessionId with OpenCode sessionId", func() {
			data := map[string]any{
				"sessionId": "s1",
				"messages":  []any{},
			}
			Expect(DetectProvider(data)).To(Equal(ProviderGeminiCLI))
		})
	})

	Context("when detecting unknown format", func() {
		It("returns unknown for unrecognized data", func() {
			data := map[string]any{
				"random": "data",
			}
			Expect(DetectProvider(data)).To(Equal(ProviderUnknown))
		})

		It("returns unknown for non-map data", func() {
			Expect(DetectProvider("not a map")).To(Equal(ProviderUnknown))
		})
	})
})
