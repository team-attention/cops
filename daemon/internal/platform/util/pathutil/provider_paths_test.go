package pathutil_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
)

func TestPathutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pathutil Suite")
}

var _ = Describe("GeminiProjectHash", func() {
	It("returns a consistent 40-character hex string", func() {
		hash := pathutil.GeminiProjectHash("/Users/jayce/project")
		Expect(hash).To(HaveLen(40))
		Expect(hash).To(MatchRegexp("^[0-9a-f]{40}$"))
	})

	It("is deterministic for the same path", func() {
		hash1 := pathutil.GeminiProjectHash("/Users/jayce/project")
		hash2 := pathutil.GeminiProjectHash("/Users/jayce/project")
		Expect(hash1).To(Equal(hash2))
	})

	It("produces different hashes for different paths", func() {
		hash1 := pathutil.GeminiProjectHash("/Users/jayce/project-a")
		hash2 := pathutil.GeminiProjectHash("/Users/jayce/project-b")
		Expect(hash1).NotTo(Equal(hash2))
	})
})

var _ = Describe("BuildGeminiHashToPathMap", func() {
	It("builds a map with correct hash-to-path entries", func() {
		paths := []string{
			"/Users/jayce/project-a",
			"/Users/jayce/project-b",
			"/Users/jayce/project-c",
		}
		m := pathutil.BuildGeminiHashToPathMap(paths)
		Expect(m).To(HaveLen(3))

		for _, p := range paths {
			hash := pathutil.GeminiProjectHash(p)
			Expect(m).To(HaveKeyWithValue(hash, p))
		}
	})
})

var _ = Describe("ExtractGeminiProjectHash", func() {
	var geminiBaseDir string

	BeforeEach(func() {
		home, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		geminiBaseDir = filepath.Join(home, ".gemini", "tmp")
	})

	It("extracts hash from chats subdirectory path", func() {
		logDir := filepath.Join(geminiBaseDir, "abc123def456", "chats")
		hash := pathutil.ExtractGeminiProjectHash(logDir)
		Expect(hash).To(Equal("abc123def456"))
	})

	It("extracts hash from direct hash directory path", func() {
		logDir := filepath.Join(geminiBaseDir, "abc123def456")
		hash := pathutil.ExtractGeminiProjectHash(logDir)
		Expect(hash).To(Equal("abc123def456"))
	})

	It("returns empty string for non-Gemini path", func() {
		hash := pathutil.ExtractGeminiProjectHash("/some/other/path")
		Expect(hash).To(BeEmpty())
	})

	It("returns empty string for the base dir itself", func() {
		hash := pathutil.ExtractGeminiProjectHash(geminiBaseDir)
		Expect(hash).To(BeEmpty())
	})
})
