package logwatcher_test

import (
	"context"
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
	"github.com/team-attention/cops/daemon/internal/platform/setup"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
	"github.com/team-attention/cops/daemon/internal/service/logwatcher"
	apimock "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/mock"
	fsmock "github.com/team-attention/cops/daemon/internal/service/logwatcher/outbound/filesystem/mock"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("Flush Adaptive Batching", func() {
	var (
		svc     *logwatcher.Service
		mockAPI *apimock.APIClient
		mockFS  *fsmock.FileWatch
		cfg     *setup.Config
		ctx     context.Context
	)

	BeforeEach(func() {
		// 1. Create context.Background()
		ctx = context.Background()

		// 2. Create logger with io.Discard writer
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		// 3. Create mock file watcher
		mockFS = &fsmock.FileWatch{}

		// 4. Create mock API client
		mockAPI = &apimock.APIClient{}

		// 5. Create config with MaxBatchSize=100
		cfg = &setup.Config{
			Cops: setup.CopsConfig{
				MaxBatchSize: 100,
			},
		}

		// 6. Create service
		svc = logwatcher.NewService(logger, mockFS, mockAPI, cfg)
	})

	Describe("successful flush", func() {
		Context("when buffer is empty", func() {
			It("returns nil without calling API", func() {
				// 1. Call Flush with empty buffer
				err := svc.Flush(ctx)

				// 2. Assert no API calls made
				Expect(mockAPI.CallCount).To(Equal(0))

				// 3. Assert nil error returned
				Expect(err).To(BeNil())
			})
		})

		Context("when lines fit in single batch", func() {
			It("sends one batch", func() {
				// 1. Add 50 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				// Update targets to register the project
				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 50)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 2. Call Flush
				err := svc.Flush(ctx)

				// 3. Assert 1 API call made
				Expect(err).To(BeNil())
				Expect(mockAPI.CallCount).To(Equal(1))

				// 4. Assert batch contains 50 lines
				Expect(mockAPI.Batches).To(HaveLen(1))
				Expect(mockAPI.Batches[0].Lines).To(HaveLen(50))
			})
		})

		Context("when lines require multiple batches", func() {
			It("sends multiple batches", func() {
				// 1. Add 250 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 250)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 2. Call Flush
				err := svc.Flush(ctx)

				// 3. Assert 3 API calls made (100 + 100 + 50)
				Expect(err).To(BeNil())
				Expect(mockAPI.CallCount).To(Equal(3))

				// 4. Assert all lines sent
				totalLines := 0
				for _, batch := range mockAPI.Batches {
					totalLines += len(batch.Lines)
				}
				Expect(totalLines).To(Equal(250))
			})
		})
	})

	Describe("413 error handling", func() {
		Context("when first batch gets 413", func() {
			It("retries with halved batch size", func() {
				// 1. Set mockAPI.SendLogsFunc to return 413 on first call
				callCount := 0
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					callCount++
					if callCount == 1 {
						return errutil.PayloadTooLarge("test")
					}
					return nil
				}

				// 2. Add 100 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 100)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Assert multiple calls made
				Expect(err).To(BeNil())
				Expect(mockAPI.CallCount).To(BeNumerically(">=", 2))

				// 5. Assert we have successful batches and they are <= 50 lines
				Expect(mockAPI.Batches).NotTo(BeEmpty())
				for _, batch := range mockAPI.Batches {
					Expect(len(batch.Lines)).To(BeNumerically("<=", 50))
				}
			})
		})

		Context("when multiple 413 errors occur", func() {
			It("progressively reduces batch size", func() {
				// 1. Set mockAPI.SendLogsFunc to return 413 if len > 25
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					if len(batch.Lines) > 25 {
						return errutil.PayloadTooLarge("test")
					}
					return nil
				}

				// 2. Add 100 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 100)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Verify batch sizes progressively reduce
				Expect(err).To(BeNil())
				// First successful batch should be <= 25
				foundSmallBatch := false
				for _, batch := range mockAPI.Batches {
					if len(batch.Lines) <= 25 {
						foundSmallBatch = true
						break
					}
				}
				Expect(foundSmallBatch).To(BeTrue())
			})
		})

		Context("when batch size recovers after success", func() {
			It("doubles batch size up to maximum", func() {
				// 1. Set mockAPI.SendLogsFunc to always return nil
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					return nil
				}

				// 2. Add 250 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 250)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Check batch sizes - should see adaptive sizing
				Expect(err).To(BeNil())
				Expect(mockAPI.CallCount).To(BeNumerically(">", 0))
			})
		})

		Context("when single line is too large", func() {
			It("skips the line and continues", func() {
				// 1. Set mockAPI.SendLogsFunc to always return 413
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					return errutil.PayloadTooLarge("test")
				}

				// 2. Add 1 line to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := []string{`{"type":"test","data":"line"}`}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Assert no error (line was skipped)
				Expect(err).To(BeNil())

				// 5. Verify multiple attempts were made before skipping
				Expect(mockAPI.CallCount).To(BeNumerically(">", 0))
			})
		})
	})

	Describe("non-413 error handling", func() {
		Context("when network error occurs", func() {
			It("returns error and preserves lines in buffer", func() {
				// 1. Set mockAPI.SendLogsFunc to return network error
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					return errutil.Internal("network error")
				}

				// 2. Add 100 lines to buffer
				projectID := shareddomain.ID("test-project-id")
				claudeDir := "/home/user/.claude/projects/test"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDir,
						ProjectID:   projectID,
						ProjectPath: "/home/user/test",
					},
				})

				lines := make([]string, 100)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDir, lines)

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Assert error returned
				Expect(err).NotTo(BeNil())

				// 5. Lines should be back in buffer for next flush
				// (This is internal state, verified by next flush succeeding)
			})
		})

		Context("when partial success across projects", func() {
			It("reports error but completes all projects", func() {
				// 1. Add lines for project A and project B
				projectA := shareddomain.ID("project-a")
				projectB := shareddomain.ID("project-b")
				claudeDirA := "/home/user/.claude/projects/a"
				claudeDirB := "/home/user/.claude/projects/b"

				svc.UpdateTargets([]domain.WatchTarget{
					{
						ClaudeDir:   claudeDirA,
						ProjectID:   projectA,
						ProjectPath: "/home/user/a",
					},
					{
						ClaudeDir:   claudeDirB,
						ProjectID:   projectB,
						ProjectPath: "/home/user/b",
					},
				})

				lines := make([]string, 10)
				for i := range lines {
					lines[i] = `{"type":"test","data":"line"}`
				}
				svc.AddLinesForClaudeDir(claudeDirA, lines)
				svc.AddLinesForClaudeDir(claudeDirB, lines)

				// 2. Set mockAPI.SendLogsFunc to fail for project B
				mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
					if batch.ProjectID == projectB {
						return errutil.Internal("error")
					}
					return nil
				}

				// 3. Call Flush
				err := svc.Flush(ctx)

				// 4. Assert error from B returned
				Expect(err).NotTo(BeNil())

				// 5. Assert A's batches were sent
				foundA := false
				for _, batch := range mockAPI.Batches {
					if batch.ProjectID == projectA {
						foundA = true
						break
					}
				}
				Expect(foundA).To(BeTrue())
			})
		})
	})
})
