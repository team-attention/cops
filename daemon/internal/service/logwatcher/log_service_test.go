package logwatcher_test

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("LogService Integration", func() {
	var (
		claudeProjectsDir string
		jsonlFiles        []string
	)

	BeforeEach(func() {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		claudeProjectsDir = filepath.Join(homeDir, ".claude", "projects")

		// Find JSONL files
		matches, err := filepath.Glob(filepath.Join(claudeProjectsDir, "*", "*.jsonl"))
		if err != nil || len(matches) == 0 {
			Skip("No JSONL files found in ~/.claude/projects/")
		}
		jsonlFiles = matches
	})

	Describe("parsing real JSONL files", func() {
		Context("when processing user messages", func() {
			It("parses user transcripts correctly", func() {
				for _, jsonlFile := range jsonlFiles {
					file, err := os.Open(jsonlFile)
					if err != nil {
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							continue
						}

						var transcript shareddomain.Transcript
						if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
							continue
						}

						if transcript.Type == shareddomain.TranscriptTypeUser {
							userTranscript, ok := transcript.Data.(*shareddomain.UserTranscript)
							Expect(ok).To(BeTrue(), "User transcript should have UserTranscript data")
							Expect(userTranscript.Message.Content).NotTo(BeNil(),
								"User message should have content")
							return // Found and validated at least one
						}
					}
				}
			})
		})

		Context("when processing assistant messages", func() {
			It("parses assistant transcripts correctly", func() {
				foundAssistant := false

				for _, jsonlFile := range jsonlFiles {
					file, err := os.Open(jsonlFile)
					if err != nil {
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							continue
						}

						var transcript shareddomain.Transcript
						if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
							continue
						}

						if transcript.Type == shareddomain.TranscriptTypeAssistant {
							assistantTranscript, ok := transcript.Data.(*shareddomain.AssistantTranscript)
							Expect(ok).To(BeTrue(), "Assistant transcript should have AssistantTranscript data")
							Expect(assistantTranscript.Message.Model).NotTo(BeEmpty())
							// RequestID may be empty in some transcripts
							foundAssistant = true
							break
						}
					}
					if foundAssistant {
						break
					}
				}

				if !foundAssistant {
					Skip("No assistant transcripts found in JSONL files")
				}
			})
		})

		Context("when processing file-history-snapshot transcripts", func() {
			It("parses snapshot transcripts correctly", func() {
				foundSnapshot := false

				for _, jsonlFile := range jsonlFiles {
					file, err := os.Open(jsonlFile)
					if err != nil {
						continue
					}
					defer file.Close()

					scanner := bufio.NewScanner(file)
					buf := make([]byte, 0, 64*1024)
					scanner.Buffer(buf, 1024*1024)

					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							continue
						}

						var transcript shareddomain.Transcript
						if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
							continue
						}

						if transcript.Type == shareddomain.TranscriptTypeFileHistorySnapshot {
							snapshotTranscript, ok := transcript.Data.(*shareddomain.FileHistorySnapshotTranscript)
							Expect(ok).To(BeTrue(), "Snapshot transcript should have FileHistorySnapshotTranscript data")
							Expect(snapshotTranscript.MessageID).NotTo(BeEmpty())
							foundSnapshot = true
							break
						}
					}
					if foundSnapshot {
						break
					}
				}

				if !foundSnapshot {
					Skip("No file-history-snapshot transcripts found in JSONL files")
				}
			})
		})
	})

	Describe("serialization round-trip", func() {
		It("preserves transcripts through sonic.Marshal/Unmarshal", func() {
			for _, jsonlFile := range jsonlFiles {
				file, err := os.Open(jsonlFile)
				if err != nil {
					continue
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 1024*1024)

				testedCount := 0
				for scanner.Scan() && testedCount < 10 {
					line := scanner.Text()
					if line == "" {
						continue
					}

					var transcript shareddomain.Transcript
					if err := sonic.Unmarshal([]byte(line), &transcript); err != nil {
						continue
					}

					if transcript.Data != nil {
						// Marshal the transcript
						transcriptBytes, err := sonic.Marshal(transcript)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(transcriptBytes)).NotTo(Equal(`""`),
							"Transcript should not serialize to empty string")

						// Unmarshal back
						var restored shareddomain.Transcript
						err = sonic.Unmarshal(transcriptBytes, &restored)
						Expect(err).NotTo(HaveOccurred())

						// Verify structure preserved
						Expect(restored.Type).To(Equal(transcript.Type))

						testedCount++
					}
				}

				if testedCount > 0 {
					return // Successfully tested
				}
			}
		})
	})
})
