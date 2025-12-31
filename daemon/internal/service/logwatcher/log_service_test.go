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
			It("parses user records correctly", func() {
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

						var record shareddomain.Record
						if err := sonic.Unmarshal([]byte(line), &record); err != nil {
							continue
						}

						if record.Type == shareddomain.RecordTypeUser {
							userRec, ok := record.Data.(*shareddomain.UserRecord)
							Expect(ok).To(BeTrue(), "User record should have UserRecord data")
							Expect(userRec.Message.Content).NotTo(BeEmpty(),
								"User message should have content")
							return // Found and validated at least one
						}
					}
				}
			})
		})

		Context("when processing assistant messages", func() {
			It("parses assistant records correctly", func() {
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

						var record shareddomain.Record
						if err := sonic.Unmarshal([]byte(line), &record); err != nil {
							continue
						}

						if record.Type == shareddomain.RecordTypeMessage {
							assistantRec, ok := record.Data.(*shareddomain.AssistantRecord)
							Expect(ok).To(BeTrue(), "Assistant record should have AssistantRecord data")
							Expect(assistantRec.Message.Model).NotTo(BeEmpty())
							// RequestID may be empty in some records
							foundAssistant = true
							break
						}
					}
					if foundAssistant {
						break
					}
				}

				if !foundAssistant {
					Skip("No assistant records found in JSONL files")
				}
			})
		})

		Context("when processing file-history-snapshot records", func() {
			It("parses snapshot records correctly", func() {
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

						var record shareddomain.Record
						if err := sonic.Unmarshal([]byte(line), &record); err != nil {
							continue
						}

						if record.Type == shareddomain.RecordTypeFileHistorySnapshot {
							snapshotRec, ok := record.Data.(*shareddomain.FileHistorySnapshotRecord)
							Expect(ok).To(BeTrue(), "Snapshot record should have FileHistorySnapshotRecord data")
							Expect(snapshotRec.MessageID).NotTo(BeEmpty())
							foundSnapshot = true
							break
						}
					}
					if foundSnapshot {
						break
					}
				}

				if !foundSnapshot {
					Skip("No file-history-snapshot records found in JSONL files")
				}
			})
		})
	})

	Describe("serialization round-trip", func() {
		It("preserves records through sonic.Marshal/Unmarshal", func() {
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

					var record shareddomain.Record
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						continue
					}

					if record.Data != nil {
						// Marshal the record
						recordBytes, err := sonic.Marshal(record)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(recordBytes)).NotTo(Equal(`""`),
							"Record should not serialize to empty string")

						// Unmarshal back
						var restored shareddomain.Record
						err = sonic.Unmarshal(recordBytes, &restored)
						Expect(err).NotTo(HaveOccurred())

						// Verify structure preserved
						Expect(restored.Type).To(Equal(record.Type))

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
