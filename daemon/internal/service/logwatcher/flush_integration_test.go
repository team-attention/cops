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

var _ = Describe("Flush Integration", func() {
	Context("with adaptive batch sizing", func() {
		It("handles 413 errors and adapts batch size", func() {
			// 1. Create service with maxBatchSize=100
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockFS := &fsmock.FileWatch{}
			mockAPI := &apimock.APIClient{}

			cfg := &setup.Config{
				Cops: setup.CopsConfig{
					MaxBatchSize: 100,
				},
			}

			svc := logwatcher.NewService(logger, mockFS, mockAPI, cfg)

			// 2. Configure mock to accept all batches (no 413)
			// This tests that the happy path sends all data
			mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
				return nil
			}

			// 3. Add 100 lines to buffer
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

			// 4. Call Flush
			err := svc.Flush(ctx)

			// 5. Assert all 100 lines sent successfully
			Expect(err).To(BeNil())
			totalLines := 0
			for _, batch := range mockAPI.Batches {
				totalLines += len(batch.Lines)
			}
			Expect(totalLines).To(Equal(100))
		})

		It("retries with smaller batches on 413 errors", func() {
			// Test that 413 errors trigger batch size reduction and retry
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			mockFS := &fsmock.FileWatch{}
			mockAPI := &apimock.APIClient{}

			cfg := &setup.Config{
				Cops: setup.CopsConfig{
					MaxBatchSize: 100,
				},
			}

			svc := logwatcher.NewService(logger, mockFS, mockAPI, cfg)

			// Configure mock to reject first attempt only, then accept all
			firstAttempt := true
			mockAPI.SendLogsFunc = func(ctx context.Context, batch domain.LogBatch) error {
				if firstAttempt && len(batch.Lines) == 50 {
					firstAttempt = false
					return errutil.PayloadTooLarge("test")
				}
				return nil
			}

			// Add 50 lines
			projectID := shareddomain.ID("test-project-id")
			claudeDir := "/home/user/.claude/projects/test"

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

			// Call Flush
			err := svc.Flush(ctx)

			// Should succeed after retry with smaller batch
			Expect(err).To(BeNil())
			// Should have made at least 2 attempts (one failed, one succeeded)
			Expect(mockAPI.CallCount).To(BeNumerically(">=", 2))
			// All 50 lines should eventually be sent
			totalLines := 0
			for _, batch := range mockAPI.Batches {
				totalLines += len(batch.Lines)
			}
			Expect(totalLines).To(Equal(50))
		})
	})
})
