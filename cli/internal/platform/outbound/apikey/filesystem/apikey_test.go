package filesystem_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/cli/internal/platform/outbound/apikey/filesystem"
	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
)

func TestFilesystemAPIKey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filesystem APIKey Suite")
}

var _ = Describe("FilesystemAPIKey", func() {
	var (
		tempDir     string
		originalEnv string
		hasEnv      bool
	)

	BeforeEach(func() {
		// 1. Create temp directory for test files.
		var err error
		tempDir, err = os.MkdirTemp("", "apikey-test-*")
		Expect(err).NotTo(HaveOccurred())

		// 2. Save and clear existing env var.
		originalEnv, hasEnv = os.LookupEnv(filesystem.EnvAPIKey)
		os.Unsetenv(filesystem.EnvAPIKey)
	})

	AfterEach(func() {
		// 1. Restore original env var.
		if hasEnv {
			os.Setenv(filesystem.EnvAPIKey, originalEnv)
		} else {
			os.Unsetenv(filesystem.EnvAPIKey)
		}

		// 2. Clean up temp directory.
		os.RemoveAll(tempDir)
	})

	// Helper to create adapter with custom auth path
	createAdapter := func(authPath string) *filesystem.FilesystemAPIKeyWithPath {
		l := slog.New(slog.NewTextHandler(io.Discard, nil))
		return filesystem.NewFilesystemAPIKeyWithPath(l, authPath)
	}

	// Helper to write auth.json file
	writeAuthJSON := func(authPath string, content interface{}) {
		dir := filepath.Dir(authPath)
		Expect(os.MkdirAll(dir, 0700)).To(Succeed())
		data, err := json.Marshal(content)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(authPath, data, 0600)).To(Succeed())
	}

	Describe("GetAPIKey", func() {
		Context("when COPS_API_KEY environment variable is set", func() {
			It("returns the env var value", func() {
				// 1. Set env var.
				os.Setenv(filesystem.EnvAPIKey, "env-test-key-123")

				// 2. Create adapter (file path doesn't matter).
				adapter := createAdapter(filepath.Join(tempDir, ".cops", "auth.json"))

				// 3. Call GetAPIKey.
				key, err := adapter.GetAPIKey(context.Background())

				// 4. Assert env var value returned, no error.
				Expect(err).NotTo(HaveOccurred())
				Expect(key).To(Equal("env-test-key-123"))
			})

			It("prioritizes env var over file", func() {
				// 1. Set env var.
				os.Setenv(filesystem.EnvAPIKey, "env-priority-key")

				// 2. Create auth.json with different key.
				authPath := filepath.Join(tempDir, ".cops", "auth.json")
				writeAuthJSON(authPath, map[string]string{"apiKey": "file-key"})

				// 3. Create adapter and call GetAPIKey.
				adapter := createAdapter(authPath)
				key, err := adapter.GetAPIKey(context.Background())

				// 4. Assert env var value returned (not file value).
				Expect(err).NotTo(HaveOccurred())
				Expect(key).To(Equal("env-priority-key"))
			})
		})

		Context("when no env var and file exists with valid key", func() {
			It("returns the API key from file", func() {
				// 1. Create auth.json with valid key.
				authPath := filepath.Join(tempDir, ".cops", "auth.json")
				writeAuthJSON(authPath, map[string]string{"apiKey": "file-api-key-456"})

				// 2. Create adapter and call GetAPIKey.
				adapter := createAdapter(authPath)
				key, err := adapter.GetAPIKey(context.Background())

				// 3. Assert file key returned, no error.
				Expect(err).NotTo(HaveOccurred())
				Expect(key).To(Equal("file-api-key-456"))
			})
		})

		Context("when no env var and file does not exist", func() {
			It("returns NotFound error", func() {
				// 1. Create adapter with non-existent path.
				authPath := filepath.Join(tempDir, ".cops", "auth.json")
				adapter := createAdapter(authPath)

				// 2. Call GetAPIKey.
				key, err := adapter.GetAPIKey(context.Background())

				// 3. Assert NotFound error, empty key.
				Expect(err).To(HaveOccurred())
				Expect(errutil.IsNotFound(err)).To(BeTrue())
				Expect(key).To(BeEmpty())
			})
		})

		Context("when no env var and file contains invalid JSON", func() {
			It("returns BadRequest error", func() {
				// 1. Create auth.json with invalid JSON.
				authPath := filepath.Join(tempDir, ".cops", "auth.json")
				dir := filepath.Dir(authPath)
				Expect(os.MkdirAll(dir, 0700)).To(Succeed())
				Expect(os.WriteFile(authPath, []byte("{invalid json}"), 0600)).To(Succeed())

				// 2. Create adapter and call GetAPIKey.
				adapter := createAdapter(authPath)
				key, err := adapter.GetAPIKey(context.Background())

				// 3. Assert BadRequest error, empty key.
				Expect(err).To(HaveOccurred())
				Expect(errutil.IsBadRequest(err)).To(BeTrue())
				Expect(key).To(BeEmpty())
			})
		})

		Context("when no env var and file has empty apiKey", func() {
			It("returns BadRequest error", func() {
				// 1. Create auth.json with empty apiKey.
				authPath := filepath.Join(tempDir, ".cops", "auth.json")
				writeAuthJSON(authPath, map[string]string{"apiKey": ""})

				// 2. Create adapter and call GetAPIKey.
				adapter := createAdapter(authPath)
				key, err := adapter.GetAPIKey(context.Background())

				// 3. Assert BadRequest error, empty key.
				Expect(err).To(HaveOccurred())
				Expect(errutil.IsBadRequest(err)).To(BeTrue())
				Expect(key).To(BeEmpty())
			})
		})
	})
})
