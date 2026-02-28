package logwatcher_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/daemon/internal/platform/util/pathutil"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
)

var _ = Describe("ProjectAssociator", func() {
	var (
		geminiBase string
	)

	BeforeEach(func() {
		home, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		geminiBase = filepath.Join(home, ".gemini", "tmp")
	})

	Describe("resolveGemini", func() {
		It("resolves a matching Gemini hash to project ID", func() {
			projectPath := "/Users/jayce/my-project"
			hash := pathutil.GeminiProjectHash(projectPath)

			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				projectPath: {ProjectID: "proj-1", Priority: 1},
			})

			logDir := filepath.Join(geminiBase, hash, "chats")
			result := assoc.ResolveGemini(logDir)
			Expect(string(result)).To(Equal("proj-1"))
		})

		It("returns empty ID for unregistered Gemini hash", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/registered": {ProjectID: "proj-1", Priority: 1},
			})

			logDir := filepath.Join(geminiBase, "unknownhash1234567890abcdef1234567890ab", "chats")
			result := assoc.ResolveGemini(logDir)
			Expect(string(result)).To(BeEmpty())
		})

		It("returns empty ID for non-Gemini path", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveGemini("/some/random/path")
			Expect(string(result)).To(BeEmpty())
		})
	})

	Describe("resolveByProjectPath", func() {
		It("returns exact match project ID", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveByProjectPath("/Users/jayce/project")
			Expect(string(result)).To(Equal("proj-1"))
		})

		It("returns parent project ID for subdirectory match", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveByProjectPath("/Users/jayce/project/src/internal")
			Expect(string(result)).To(Equal("proj-1"))
		})

		It("returns empty ID for unregistered path", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveByProjectPath("/Users/other/unknown")
			Expect(string(result)).To(BeEmpty())
		})

		It("prefers main project over worktree when both match by priority", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project":            {ProjectID: "main-proj", Priority: 1},
				"/Users/jayce/project/.worktrees": {ProjectID: "wt-proj", Priority: 2},
			})

			// Main project has higher priority (lower number = better) so it wins
			result := assoc.ResolveByProjectPath("/Users/jayce/project/.worktrees/feature/src")
			Expect(string(result)).To(Equal("main-proj"))
		})

		It("prefers longer path at same priority", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce":         {ProjectID: "short-proj", Priority: 1},
				"/Users/jayce/project": {ProjectID: "long-proj", Priority: 1},
			})

			result := assoc.ResolveByProjectPath("/Users/jayce/project/src/main.go")
			Expect(string(result)).To(Equal("long-proj"))
		})
	})

	Describe("resolveCodexCwd", func() {
		It("delegates to resolveByProjectPath", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveCodexCwd("/Users/jayce/project")
			Expect(string(result)).To(Equal("proj-1"))
		})
	})

	Describe("resolveOpenCode", func() {
		It("delegates to resolveByProjectPath", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(map[string]logwatcher.ProjectMappingForTest{
				"/Users/jayce/project": {ProjectID: "proj-1", Priority: 1},
			})

			result := assoc.ResolveOpenCode("/Users/jayce/project")
			Expect(string(result)).To(Equal("proj-1"))
		})
	})

	Describe("zero-value associator", func() {
		It("returns empty ID for all resolution methods", func() {
			assoc := logwatcher.NewProjectAssociatorForTest(nil)

			Expect(string(assoc.ResolveGemini(filepath.Join(geminiBase, "somehash", "chats")))).To(BeEmpty())
			Expect(string(assoc.ResolveByProjectPath("/any/path"))).To(BeEmpty())
			Expect(string(assoc.ResolveCodexCwd("/any/cwd"))).To(BeEmpty())
			Expect(string(assoc.ResolveOpenCode("/any/path"))).To(BeEmpty())
		})
	})
})
