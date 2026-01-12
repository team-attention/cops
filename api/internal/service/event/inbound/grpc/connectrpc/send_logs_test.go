package connectrpc_test

import (
	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("SendLogs JSONL Parsing", func() {
	Describe("parsing valid JSONL lines", func() {
		Context("when all lines are valid JSON", func() {
			It("parses all lines successfully", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
					`{"type":"file-history-snapshot","messageId":"m1","snapshot":{"messageId":"m1","trackedFileBackups":{}},"isSnapshotUpdate":false}`,
				}

				var transcripts []shareddomain.Transcript
				var parseErrors []error

				for _, line := range lines {
					var transcript shareddomain.Transcript
					if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
						parseErrors = append(parseErrors, err)
						continue
					}
					transcripts = append(transcripts, transcript)
				}

				Expect(transcripts).To(HaveLen(3))
				Expect(parseErrors).To(BeEmpty())
				Expect(transcripts[0].Type).To(Equal(shareddomain.TranscriptTypeUser))
				Expect(transcripts[1].Type).To(Equal(shareddomain.TranscriptTypeAssistant))
				Expect(transcripts[2].Type).To(Equal(shareddomain.TranscriptTypeFileHistorySnapshot))
			})
		})
	})

	Describe("parsing invalid JSONL lines", func() {
		Context("when some lines contain invalid JSON", func() {
			It("skips invalid lines and parses valid ones", func() {
				lines := []string{
					`{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`,
					`invalid json`,
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
				}

				var transcripts []shareddomain.Transcript
				var parseErrors []error

				for _, line := range lines {
					var transcript shareddomain.Transcript
					if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
						parseErrors = append(parseErrors, err)
						continue
					}
					transcripts = append(transcripts, transcript)
				}

				Expect(transcripts).To(HaveLen(2))
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
					`{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`,
				}

				var transcripts []shareddomain.Transcript

				for _, line := range lines {
					if line == "" {
						continue
					}
					var transcript shareddomain.Transcript
					if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
						continue
					}
					transcripts = append(transcripts, transcript)
				}

				Expect(transcripts).To(HaveLen(2))
			})
		})
	})

	Describe("parsing all transcript types", func() {
		Context("when parsing user transcript", func() {
			It("returns Transcript with UserTranscript data", func() {
				userLine := `{"type":"user","sessionId":"s1","uuid":"1","timestamp":"2024-01-01T00:00:00Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","message":{"role":"user","content":"test"},"isMeta":false}`

				var transcript shareddomain.Transcript
				err := sonic.Unmarshal([]byte(userLine), &transcript)

				Expect(err).NotTo(HaveOccurred())
				Expect(transcript.Type).To(Equal(shareddomain.TranscriptTypeUser))
				Expect(transcript.Data).To(BeAssignableToTypeOf(&shareddomain.UserTranscript{}))
			})
		})

		Context("when parsing assistant transcript", func() {
			It("returns Transcript with AssistantTranscript data", func() {
				assistantLine := `{"type":"assistant","sessionId":"s1","uuid":"2","timestamp":"2024-01-01T00:00:01Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","requestId":"r1","message":{"model":"claude","id":"m1","type":"message","role":"assistant","content":[],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"service_tier":"standard"}}}`

				var transcript shareddomain.Transcript
				err := sonic.Unmarshal([]byte(assistantLine), &transcript)

				Expect(err).NotTo(HaveOccurred())
				Expect(transcript.Type).To(Equal(shareddomain.TranscriptTypeAssistant))
				Expect(transcript.Data).To(BeAssignableToTypeOf(&shareddomain.AssistantTranscript{}))
			})
		})

		Context("when parsing file-history-snapshot transcript", func() {
			It("returns Transcript with FileHistorySnapshotTranscript data", func() {
				snapshotLine := `{"type":"file-history-snapshot","messageId":"m1","snapshot":{"messageId":"m1","trackedFileBackups":{}},"isSnapshotUpdate":false}`

				var transcript shareddomain.Transcript
				err := sonic.Unmarshal([]byte(snapshotLine), &transcript)

				Expect(err).NotTo(HaveOccurred())
				Expect(transcript.Type).To(Equal(shareddomain.TranscriptTypeFileHistorySnapshot))
				Expect(transcript.Data).To(BeAssignableToTypeOf(&shareddomain.FileHistorySnapshotTranscript{}))
			})
		})

		Context("when parsing system transcript", func() {
			It("returns Transcript with SystemTranscript data", func() {
				systemLine := `{"type":"system","sessionId":"s1","uuid":"3","timestamp":"2024-01-01T00:00:02Z","isSidechain":false,"userType":"external","version":"1.0","gitBranch":"main","subtype":"turn_duration","durationMs":1234,"isMeta":true}`

				var transcript shareddomain.Transcript
				err := sonic.Unmarshal([]byte(systemLine), &transcript)

				Expect(err).NotTo(HaveOccurred())
				Expect(transcript.Type).To(Equal(shareddomain.TranscriptTypeSystem))
				Expect(transcript.Data).To(BeAssignableToTypeOf(&shareddomain.SystemTranscript{}))
			})
		})

		Context("when parsing summary transcript", func() {
			It("returns Transcript with SummaryTranscript data", func() {
				summaryLine := `{"type":"summary","summary":"This is a conversation summary","leafUuid":"leaf-uuid-123"}`

				var transcript shareddomain.Transcript
				err := sonic.Unmarshal([]byte(summaryLine), &transcript)

				Expect(err).NotTo(HaveOccurred())
				Expect(transcript.Type).To(Equal(shareddomain.TranscriptTypeSummary))
				Expect(transcript.Data).To(BeAssignableToTypeOf(&shareddomain.SummaryTranscript{}))
			})
		})
	})
})
