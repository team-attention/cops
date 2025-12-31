package connectrpc_test

import (
	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("JSONL Parsing", func() {
	Describe("parsing valid JSONL lines", func() {
		Context("when all lines are valid JSON", func() {
			It("parses all lines successfully", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"inputTokens":10,"outputTokens":20,"cacheCreationInputTokens":0,"cacheReadInputTokens":0,"serviceTier":"standard"}}}`,
				}

				var records []shareddomain.Record
				var parseErrors []error

				for _, line := range lines {
					var record shareddomain.Record
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						parseErrors = append(parseErrors, err)
						continue
					}
					records = append(records, record)
				}

				Expect(records).To(HaveLen(2))
				Expect(parseErrors).To(BeEmpty())
				Expect(records[0].Type).To(Equal(shareddomain.RecordTypeUser))
				Expect(records[1].Type).To(Equal(shareddomain.RecordTypeMessage))
			})
		})
	})

	Describe("parsing invalid JSONL lines", func() {
		Context("when some lines contain invalid JSON", func() {
			It("skips invalid lines and parses valid ones", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`invalid json`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"inputTokens":10,"outputTokens":20,"cacheCreationInputTokens":0,"cacheReadInputTokens":0,"serviceTier":"standard"}}}`,
				}

				var records []shareddomain.Record
				var parseErrors []error

				for _, line := range lines {
					var record shareddomain.Record
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						parseErrors = append(parseErrors, err)
						continue
					}
					records = append(records, record)
				}

				Expect(records).To(HaveLen(2))
				Expect(parseErrors).To(HaveLen(1))
			})
		})
	})

	Describe("parsing JSONL with empty lines", func() {
		Context("when lines contain empty strings", func() {
			It("skips empty lines and parses valid ones", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					``,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"inputTokens":10,"outputTokens":20,"cacheCreationInputTokens":0,"cacheReadInputTokens":0,"serviceTier":"standard"}}}`,
				}

				var records []shareddomain.Record

				for _, line := range lines {
					if line == "" {
						continue
					}
					var record shareddomain.Record
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						continue
					}
					records = append(records, record)
				}

				Expect(records).To(HaveLen(2))
			})
		})
	})

	Describe("parsing record types", func() {
		Context("when parsing user record", func() {
			It("returns Record with UserRecord data", func() {
				userLine := `{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`

				var record shareddomain.Record
				err := sonic.Unmarshal([]byte(userLine), &record)

				Expect(err).NotTo(HaveOccurred())
				Expect(record.Type).To(Equal(shareddomain.RecordTypeUser))
				Expect(record.Data).To(BeAssignableToTypeOf(&shareddomain.UserRecord{}))
			})
		})

		Context("when parsing assistant record", func() {
			It("returns Record with AssistantRecord data", func() {
				assistantLine := `{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"inputTokens":10,"outputTokens":20,"cacheCreationInputTokens":0,"cacheReadInputTokens":0,"serviceTier":"standard"}}}`

				var record shareddomain.Record
				err := sonic.Unmarshal([]byte(assistantLine), &record)

				Expect(err).NotTo(HaveOccurred())
				Expect(record.Type).To(Equal(shareddomain.RecordTypeMessage))
				Expect(record.Data).To(BeAssignableToTypeOf(&shareddomain.AssistantRecord{}))
			})
		})

		Context("when parsing file-history-snapshot record", func() {
			It("returns Record with FileHistorySnapshotRecord data", func() {
				snapshotLine := `{"type":"file-history-snapshot","messageId":"m1","snapshot":{"messageId":"m1","trackedFileBackups":{}},"isSnapshotUpdate":false}`

				var record shareddomain.Record
				err := sonic.Unmarshal([]byte(snapshotLine), &record)

				Expect(err).NotTo(HaveOccurred())
				Expect(record.Type).To(Equal(shareddomain.RecordTypeFileHistorySnapshot))
				Expect(record.Data).To(BeAssignableToTypeOf(&shareddomain.FileHistorySnapshotRecord{}))
			})
		})
	})
})
