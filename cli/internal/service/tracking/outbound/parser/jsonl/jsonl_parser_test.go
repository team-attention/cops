package jsonl_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl"
	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("JSONLParser", func() {
	var (
		parser *jsonl.JSONLParser
		tmpDir string
	)

	BeforeEach(func() {
		parser = jsonl.NewJSONLParser(slog.New(slog.NewTextHandler(io.Discard, nil)))
		var err error
		tmpDir, err = os.MkdirTemp("", "jsonl-parser-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("ParseSessionFiles", func() {
		Context("when directory does not exist", func() {
			It("returns empty slice without error", func() {
				transcripts, err := parser.ParseSessionFiles("/non/existent/path")
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(BeEmpty())
			})
		})

		Context("when directory has no JSONL files", func() {
			It("returns empty slice", func() {
				// Create a .txt file
				txtFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(txtFile, []byte("not a jsonl file"), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(BeEmpty())
			})
		})

		Context("when directory has valid JSONL files", func() {
			It("parses user transcripts", func() {
				jsonlContent := `{"type":"user","uuid":"test-uuid-1","sessionId":"test-session","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","isMeta":true}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeUser))

				userTranscript, ok := transcripts[0].Data.(*domain.UserTranscript)
				Expect(ok).To(BeTrue())
				Expect(userTranscript.UUID).To(Equal("test-uuid-1"))
				Expect(userTranscript.SessionID).To(Equal("test-session"))
				Expect(userTranscript.IsMeta).To(BeTrue())
			})

			It("parses assistant transcripts", func() {
				jsonlContent := `{"type":"assistant","uuid":"test-uuid-2","sessionId":"test-session","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","requestId":"req-123","message":{"model":"claude-3","role":"assistant","content":[]}}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeAssistant))

				assistantTranscript, ok := transcripts[0].Data.(*domain.AssistantTranscript)
				Expect(ok).To(BeTrue())
				Expect(assistantTranscript.RequestID).To(Equal("req-123"))
			})

			It("parses system transcripts", func() {
				jsonlContent := `{"type":"system","uuid":"test-uuid-3","sessionId":"test-session","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","subtype":"turn_duration","durationMs":1000}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeSystem))

				systemTranscript, ok := transcripts[0].Data.(*domain.SystemTranscript)
				Expect(ok).To(BeTrue())
				Expect(systemTranscript.Subtype).To(Equal(domain.SystemTranscriptSubtypeTurnDuration))
				Expect(systemTranscript.DurationMs).To(Equal(1000))
			})

			It("parses summary transcripts", func() {
				jsonlContent := `{"type":"summary","summary":"Test summary","leafUuid":"leaf-uuid-123"}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeSummary))

				summaryTranscript, ok := transcripts[0].Data.(*domain.SummaryTranscript)
				Expect(ok).To(BeTrue())
				Expect(summaryTranscript.Summary).To(Equal("Test summary"))
				Expect(summaryTranscript.LeafUUID).To(Equal("leaf-uuid-123"))
			})

			It("parses file-history-snapshot transcripts", func() {
				jsonlContent := `{"type":"file-history-snapshot","messageId":"msg-123","isSnapshotUpdate":false,"snapshot":{"trackedFileBackups":{}}}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeFileHistorySnapshot))

				snapshotTranscript, ok := transcripts[0].Data.(*domain.FileHistorySnapshotTranscript)
				Expect(ok).To(BeTrue())
				Expect(snapshotTranscript.MessageID).To(Equal("msg-123"))
				Expect(snapshotTranscript.IsSnapshotUpdate).To(BeFalse())
			})

			It("parses mixed transcript types from single file", func() {
				jsonlContent := `{"type":"user","uuid":"uuid-1","sessionId":"sess","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","isMeta":true}
{"type":"assistant","uuid":"uuid-2","sessionId":"sess","timestamp":"2025-01-01T00:00:01Z","version":"2.1.5","requestId":"req-1","message":{"model":"claude-3","role":"assistant","content":[]}}
{"type":"system","uuid":"uuid-3","sessionId":"sess","timestamp":"2025-01-01T00:00:02Z","version":"2.1.5","subtype":"turn_duration","durationMs":500}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(3))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeUser))
				Expect(transcripts[1].Type).To(Equal(domain.TranscriptTypeAssistant))
				Expect(transcripts[2].Type).To(Equal(domain.TranscriptTypeSystem))
			})

			It("aggregates transcripts from multiple files", func() {
				jsonlContent1 := `{"type":"user","uuid":"uuid-1","sessionId":"sess","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","isMeta":true}`
				jsonlContent2 := `{"type":"assistant","uuid":"uuid-2","sessionId":"sess","timestamp":"2025-01-01T00:00:01Z","version":"2.1.5","requestId":"req-1","message":{"model":"claude-3","role":"assistant","content":[]}}`

				err := os.WriteFile(filepath.Join(tmpDir, "file1.jsonl"), []byte(jsonlContent1), 0644)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(filepath.Join(tmpDir, "file2.jsonl"), []byte(jsonlContent2), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(2))
			})
		})

		Context("when handling malformed lines", func() {
			It("skips invalid JSON lines", func() {
				jsonlContent := `{"type":"user","uuid":"uuid-1","sessionId":"sess","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","isMeta":true}
not valid json
{"type":"assistant","uuid":"uuid-2","sessionId":"sess","timestamp":"2025-01-01T00:00:01Z","version":"2.1.5","requestId":"req-1","message":{"model":"claude-3","role":"assistant","content":[]}}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(2))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptTypeUser))
				Expect(transcripts[1].Type).To(Equal(domain.TranscriptTypeAssistant))
			})

			It("skips empty lines", func() {
				jsonlContent := `{"type":"user","uuid":"uuid-1","sessionId":"sess","timestamp":"2025-01-01T00:00:00Z","version":"2.1.5","isMeta":true}


{"type":"assistant","uuid":"uuid-2","sessionId":"sess","timestamp":"2025-01-01T00:00:01Z","version":"2.1.5","requestId":"req-1","message":{"model":"claude-3","role":"assistant","content":[]}}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(2))
			})
		})

		Context("when handling unknown transcript types", func() {
			It("stores unknown type data as map[string]any", func() {
				jsonlContent := `{"type":"future-type","customField":"value","anotherField":123}`
				jsonlFile := filepath.Join(tmpDir, "test.jsonl")
				err := os.WriteFile(jsonlFile, []byte(jsonlContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				transcripts, err := parser.ParseSessionFiles(tmpDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(transcripts).To(HaveLen(1))
				Expect(transcripts[0].Type).To(Equal(domain.TranscriptType("future-type")))

				mapData, ok := transcripts[0].Data.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(mapData).To(HaveKeyWithValue("customField", "value"))
				Expect(mapData).To(HaveKeyWithValue("anotherField", float64(123)))
			})
		})
	})
})
