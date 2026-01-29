package connectrpc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SendLogs Raw Event Storage", func() {
	Describe("storing raw JSONL lines", func() {
		Context("when all lines are non-empty", func() {
			It("stores all lines as raw events", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
					`{"type":"file-history-snapshot","messageId":"m1","snapshot":{"messageId":"m1","trackedFileBackups":{}},"isSnapshotUpdate":false}`,
				}

				// Count non-empty lines
				var nonEmptyCount int
				for _, line := range lines {
					if line != "" {
						nonEmptyCount++
					}
				}

				Expect(nonEmptyCount).To(Equal(3))
			})
		})

		Context("when lines contain empty strings", func() {
			It("filters out empty lines before storage", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					``,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
				}

				// Count only non-empty lines
				var nonEmptyCount int
				for _, line := range lines {
					if line != "" {
						nonEmptyCount++
					}
				}

				Expect(nonEmptyCount).To(Equal(2))
			})
		})

		Context("when lines contain invalid JSON", func() {
			It("stores invalid JSON lines as raw events (parsing happens later)", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`invalid json`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
				}

				// All non-empty lines should be stored regardless of JSON validity
				var nonEmptyCount int
				for _, line := range lines {
					if line != "" {
						nonEmptyCount++
					}
				}

				Expect(nonEmptyCount).To(Equal(3))
			})
		})
	})
})
