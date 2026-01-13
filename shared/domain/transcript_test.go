package domain_test

import (
	"bufio"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

const (
	transcriptUserTestDataFile      = "transcript_user.jsonl"
	transcriptAssistantTestDataFile = "transcript_assistant.jsonl"
	transcriptSystemTestDataFile    = "transcript_system.jsonl"
	transcriptSummaryTestDataFile   = "transcript_summary.jsonl"
	transcriptSnapshotTestDataFile  = "transcript_snapshot.jsonl"
)

// readLineFromFile reads a specific line number from a file (1-indexed)
func readLineFromFile(filename string, lineNum int) string {
	file, err := os.Open(filename)
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine == lineNum {
			return scanner.Text()
		}
	}
	Expect(scanner.Err()).NotTo(HaveOccurred())
	return ""
}

var _ = Describe("Transcript", func() {
	Describe("UnmarshalJSON", func() {
		Context("when unmarshaling user transcripts", func() {
			It("parses user with isMeta=true from JSONL file", func() {
				line := readLineFromFile(transcriptUserTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptTypeUser))
				userTranscript, ok := transcript.Data.(*domain.UserTranscript)
				Expect(ok).To(BeTrue())
				Expect(userTranscript.IsMeta).To(BeTrue())
				Expect(userTranscript.ParentUUID).To(BeNil())
				Expect(userTranscript.SessionID).To(Equal("888509f0-1291-4727-828b-cad270a4829d"))
				Expect(userTranscript.Version).To(Equal("2.1.5"))
			})

			It("parses user with parentUuid from JSONL file", func() {
				line := readLineFromFile(transcriptUserTestDataFile, 2)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				userTranscript, ok := transcript.Data.(*domain.UserTranscript)
				Expect(ok).To(BeTrue())
				Expect(userTranscript.ParentUUID).NotTo(BeNil())
				Expect(*userTranscript.ParentUUID).To(Equal("89e6d6b4-03c6-4500-9fcd-e2ea83aec443"))
			})

			It("parses user with thinkingMetadata triggers from JSONL file", func() {
				line := readLineFromFile(transcriptUserTestDataFile, 3)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				userTranscript, ok := transcript.Data.(*domain.UserTranscript)
				Expect(ok).To(BeTrue())
				Expect(userTranscript.ThinkingMetadata).NotTo(BeNil())
				Expect(userTranscript.ThinkingMetadata.Level).To(Equal("high"))
				Expect(userTranscript.ThinkingMetadata.Disabled).To(BeFalse())
				Expect(userTranscript.ThinkingMetadata.Triggers).To(HaveLen(1))
				Expect(userTranscript.ThinkingMetadata.Triggers[0].Text).To(Equal("ultrathink"))
				Expect(userTranscript.ThinkingMetadata.Triggers[0].Start).To(Equal(18))
				Expect(userTranscript.ThinkingMetadata.Triggers[0].End).To(Equal(28))
				Expect(userTranscript.Slug).To(Equal("parallel-imagining-fern"))
			})

			It("parses user with tool_result and sourceToolAssistantUUID from JSONL file", func() {
				line := readLineFromFile(transcriptUserTestDataFile, 4)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				userTranscript, ok := transcript.Data.(*domain.UserTranscript)
				Expect(ok).To(BeTrue())
				Expect(userTranscript.SourceToolAssistantUUID).NotTo(BeNil())
				Expect(*userTranscript.SourceToolAssistantUUID).To(Equal("fef0d544-9d45-4b00-9acc-48e5727d52f9"))
			})
		})

		Context("when unmarshaling assistant transcripts", func() {
			It("parses assistant with thinking block from JSONL file", func() {
				line := readLineFromFile(transcriptAssistantTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptTypeAssistant))
				assistantTranscript, ok := transcript.Data.(*domain.AssistantTranscript)
				Expect(ok).To(BeTrue())
				Expect(assistantTranscript.RequestID).To(Equal("req_011CX31WPjfCNu6JVyzFvPgw"))
				Expect(assistantTranscript.Message.Model).To(Equal("claude-opus-4-5-20251101"))
				Expect(assistantTranscript.Message.Content).To(HaveLen(1))
				Expect(assistantTranscript.Message.Content[0].Type).To(Equal("thinking"))
				Expect(assistantTranscript.Message.Content[0].Thinking).NotTo(BeNil())
				Expect(*assistantTranscript.Message.Content[0].Thinking).To(Equal("Test thinking content"))
				Expect(assistantTranscript.Message.Content[0].Signature).NotTo(BeNil())
				Expect(*assistantTranscript.Message.Content[0].Signature).To(Equal("test-signature"))
			})

			It("parses assistant with text block from JSONL file", func() {
				line := readLineFromFile(transcriptAssistantTestDataFile, 2)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				assistantTranscript, ok := transcript.Data.(*domain.AssistantTranscript)
				Expect(ok).To(BeTrue())
				Expect(assistantTranscript.Message.Content).To(HaveLen(1))
				Expect(assistantTranscript.Message.Content[0].Type).To(Equal("text"))
				Expect(assistantTranscript.Message.Content[0].Text).NotTo(BeNil())
				Expect(*assistantTranscript.Message.Content[0].Text).To(Equal("Test response text"))
				Expect(assistantTranscript.Message.StopReason).NotTo(BeNil())
				Expect(*assistantTranscript.Message.StopReason).To(Equal("end_turn"))
			})

			It("parses assistant with tool_use block from JSONL file", func() {
				line := readLineFromFile(transcriptAssistantTestDataFile, 3)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				assistantTranscript, ok := transcript.Data.(*domain.AssistantTranscript)
				Expect(ok).To(BeTrue())
				Expect(assistantTranscript.Message.Content).To(HaveLen(1))
				Expect(assistantTranscript.Message.Content[0].Type).To(Equal("tool_use"))
				Expect(assistantTranscript.Message.Content[0].ID).NotTo(BeNil())
				Expect(*assistantTranscript.Message.Content[0].ID).To(Equal("toolu_01QEnMDvBScoNEV5fMDvCcpe"))
				Expect(assistantTranscript.Message.Content[0].Name).NotTo(BeNil())
				Expect(*assistantTranscript.Message.Content[0].Name).To(Equal("Task"))
				Expect(assistantTranscript.Message.Content[0].Input).NotTo(BeNil())
				Expect(assistantTranscript.Message.Content[0].Input["subagent_type"]).To(Equal("Explore"))
				Expect(assistantTranscript.Message.StopReason).NotTo(BeNil())
				Expect(*assistantTranscript.Message.StopReason).To(Equal("tool_use"))
			})

			It("parses assistant usage correctly from JSONL file", func() {
				line := readLineFromFile(transcriptAssistantTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				assistantTranscript := transcript.Data.(*domain.AssistantTranscript)
				Expect(assistantTranscript.Message.Usage).NotTo(BeNil())
				Expect(assistantTranscript.Message.Usage.InputTokens).To(Equal(10))
				Expect(assistantTranscript.Message.Usage.OutputTokens).To(Equal(1))
				Expect(assistantTranscript.Message.Usage.CacheCreationInputTokens).To(Equal(6405))
				Expect(assistantTranscript.Message.Usage.CacheReadInputTokens).To(Equal(26808))
				Expect(assistantTranscript.Message.Usage.CacheCreation).NotTo(BeNil())
				Expect(assistantTranscript.Message.Usage.CacheCreation.Ephemeral5mInputTokens).To(Equal(6405))
			})
		})

		Context("when unmarshaling system transcripts", func() {
			It("parses system with turn_duration subtype from JSONL file", func() {
				line := readLineFromFile(transcriptSystemTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptTypeSystem))
				systemTranscript, ok := transcript.Data.(*domain.SystemTranscript)
				Expect(ok).To(BeTrue())
				Expect(systemTranscript.Subtype).To(Equal(domain.SystemTranscriptSubtypeTurnDuration))
				Expect(systemTranscript.DurationMs).To(Equal(294122))
				Expect(systemTranscript.IsMeta).To(BeFalse())
				Expect(systemTranscript.ParentUUID).NotTo(BeNil())
			})

			It("parses second system entry from JSONL file", func() {
				line := readLineFromFile(transcriptSystemTestDataFile, 2)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				systemTranscript, ok := transcript.Data.(*domain.SystemTranscript)
				Expect(ok).To(BeTrue())
				Expect(systemTranscript.DurationMs).To(Equal(353398))
			})
		})

		Context("when unmarshaling summary transcripts", func() {
			It("parses summary from JSONL file", func() {
				line := readLineFromFile(transcriptSummaryTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptTypeSummary))
				summaryTranscript, ok := transcript.Data.(*domain.SummaryTranscript)
				Expect(ok).To(BeTrue())
				Expect(summaryTranscript.Summary).To(Equal("CLI Device Token Authentication Workflow Design"))
				Expect(summaryTranscript.LeafUUID).To(Equal("1822ee49-fcaa-4fb6-a1b7-7c90a413a124"))
			})

			It("parses all summaries from JSONL file", func() {
				file, err := os.Open(transcriptSummaryTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				count := 0
				for scanner.Scan() {
					count++
					var transcript domain.Transcript
					err := json.Unmarshal(scanner.Bytes(), &transcript)
					Expect(err).NotTo(HaveOccurred())
					Expect(transcript.Type).To(Equal(domain.TranscriptTypeSummary))

					summaryTranscript, ok := transcript.Data.(*domain.SummaryTranscript)
					Expect(ok).To(BeTrue())
					Expect(summaryTranscript.Summary).NotTo(BeEmpty())
					Expect(summaryTranscript.LeafUUID).NotTo(BeEmpty())
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(count).To(Equal(3))
			})
		})

		Context("when unmarshaling file-history-snapshot transcripts", func() {
			It("parses snapshot with isSnapshotUpdate=false from JSONL file", func() {
				line := readLineFromFile(transcriptSnapshotTestDataFile, 1)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptTypeFileHistorySnapshot))
				snapshotTranscript, ok := transcript.Data.(*domain.FileHistorySnapshotTranscript)
				Expect(ok).To(BeTrue())
				Expect(snapshotTranscript.MessageID).To(Equal("38c9c54d-3492-4cb3-a0bb-f7e8cbcfd758"))
				Expect(snapshotTranscript.IsSnapshotUpdate).To(BeFalse())
				Expect(snapshotTranscript.Snapshot.TrackedFileBackups).To(BeEmpty())
			})

			It("parses snapshot with isSnapshotUpdate=true and trackedFileBackups from JSONL file", func() {
				line := readLineFromFile(transcriptSnapshotTestDataFile, 2)
				Expect(line).NotTo(BeEmpty())

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred())

				snapshotTranscript, ok := transcript.Data.(*domain.FileHistorySnapshotTranscript)
				Expect(ok).To(BeTrue())
				Expect(snapshotTranscript.IsSnapshotUpdate).To(BeTrue())
				Expect(snapshotTranscript.Snapshot.TrackedFileBackups).NotTo(BeEmpty())
				Expect(snapshotTranscript.Snapshot.TrackedFileBackups).To(HaveKey("/Users/jayce/.claude/plans/test.md"))

				backup := snapshotTranscript.Snapshot.TrackedFileBackups["/Users/jayce/.claude/plans/test.md"]
				Expect(backup.Version).To(Equal(1))
				Expect(backup.BackupFileName).To(BeNil())
			})
		})

		Context("when unmarshaling unknown transcript types", func() {
			It("stores unknown type as map[string]any", func() {
				jsonData := []byte(`{"type":"future-type","customField":"value"}`)

				var transcript domain.Transcript
				err := json.Unmarshal(jsonData, &transcript)
				Expect(err).NotTo(HaveOccurred())

				Expect(transcript.Type).To(Equal(domain.TranscriptType("future-type")))
				mapData, ok := transcript.Data.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(mapData).To(HaveKeyWithValue("customField", "value"))
			})
		})
	})

	Describe("MarshalJSON", func() {
		It("produces flat JSON for UserTranscript", func() {
			userTranscript := &domain.UserTranscript{
				IsMeta: true,
			}
			userTranscript.SessionID = "test-session"
			userTranscript.UUID = "test-uuid"

			transcript := domain.Transcript{
				Type: domain.TranscriptTypeUser,
				Data: userTranscript,
			}

			jsonData, err := json.Marshal(transcript)
			Expect(err).NotTo(HaveOccurred())

			var result map[string]any
			err = json.Unmarshal(jsonData, &result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(HaveKeyWithValue("type", "user"))
			Expect(result).To(HaveKeyWithValue("isMeta", true))
			Expect(result).To(HaveKeyWithValue("sessionId", "test-session"))
			Expect(result).To(HaveKeyWithValue("uuid", "test-uuid"))
		})

		It("produces flat JSON for AssistantTranscript", func() {
			assistantTranscript := &domain.AssistantTranscript{
				RequestID: "test-request",
			}
			assistantTranscript.SessionID = "test-session"

			transcript := domain.Transcript{
				Type: domain.TranscriptTypeAssistant,
				Data: assistantTranscript,
			}

			jsonData, err := json.Marshal(transcript)
			Expect(err).NotTo(HaveOccurred())

			var result map[string]any
			err = json.Unmarshal(jsonData, &result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(HaveKeyWithValue("type", "assistant"))
			Expect(result).To(HaveKeyWithValue("requestId", "test-request"))
		})

		It("produces flat JSON for SummaryTranscript", func() {
			summaryTranscript := &domain.SummaryTranscript{
				Summary:  "Test summary",
				LeafUUID: "test-leaf-uuid",
			}

			transcript := domain.Transcript{
				Type: domain.TranscriptTypeSummary,
				Data: summaryTranscript,
			}

			jsonData, err := json.Marshal(transcript)
			Expect(err).NotTo(HaveOccurred())

			var result map[string]any
			err = json.Unmarshal(jsonData, &result)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(HaveKeyWithValue("type", "summary"))
			Expect(result).To(HaveKeyWithValue("summary", "Test summary"))
			Expect(result).To(HaveKeyWithValue("leafUuid", "test-leaf-uuid"))
		})

		It("produces JSON with only type field when Data is nil", func() {
			transcript := domain.Transcript{
				Type: domain.TranscriptTypeUser,
				Data: nil,
			}

			jsonData, err := json.Marshal(transcript)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(jsonData)).To(Equal(`{"type":"user"}`))
		})
	})

	Describe("Round-trip serialization", func() {
		It("preserves user transcripts through round-trip", func() {
			file, err := os.Open(transcriptUserTestDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred(), "Line %d: unmarshal failed", lineNum)
				Expect(transcript.Type).To(Equal(domain.TranscriptTypeUser), "Line %d: type mismatch", lineNum)

				jsonData, err := json.Marshal(transcript)
				Expect(err).NotTo(HaveOccurred(), "Line %d: marshal failed", lineNum)

				var transcript2 domain.Transcript
				err = json.Unmarshal(jsonData, &transcript2)
				Expect(err).NotTo(HaveOccurred(), "Line %d: second unmarshal failed", lineNum)
				Expect(transcript2.Type).To(Equal(domain.TranscriptTypeUser), "Line %d: type mismatch after round-trip", lineNum)

				user1 := transcript.Data.(*domain.UserTranscript)
				user2 := transcript2.Data.(*domain.UserTranscript)
				Expect(user2.UUID).To(Equal(user1.UUID))
				Expect(user2.SessionID).To(Equal(user1.SessionID))
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())
		})

		It("preserves assistant transcripts through round-trip", func() {
			file, err := os.Open(transcriptAssistantTestDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				var transcript domain.Transcript
				err := json.Unmarshal([]byte(line), &transcript)
				Expect(err).NotTo(HaveOccurred(), "Line %d: unmarshal failed", lineNum)

				jsonData, err := json.Marshal(transcript)
				Expect(err).NotTo(HaveOccurred(), "Line %d: marshal failed", lineNum)

				var transcript2 domain.Transcript
				err = json.Unmarshal(jsonData, &transcript2)
				Expect(err).NotTo(HaveOccurred(), "Line %d: second unmarshal failed", lineNum)

				assistant1 := transcript.Data.(*domain.AssistantTranscript)
				assistant2 := transcript2.Data.(*domain.AssistantTranscript)
				Expect(assistant2.RequestID).To(Equal(assistant1.RequestID))
				Expect(assistant2.Message.Model).To(Equal(assistant1.Message.Model))
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())
		})

		It("preserves all transcript types through round-trip", func() {
			testFiles := map[string]domain.TranscriptType{
				transcriptUserTestDataFile:      domain.TranscriptTypeUser,
				transcriptAssistantTestDataFile: domain.TranscriptTypeAssistant,
				transcriptSystemTestDataFile:    domain.TranscriptTypeSystem,
				transcriptSummaryTestDataFile:   domain.TranscriptTypeSummary,
				transcriptSnapshotTestDataFile:  domain.TranscriptTypeFileHistorySnapshot,
			}

			for filename, expectedType := range testFiles {
				file, err := os.Open(filename)
				Expect(err).NotTo(HaveOccurred())

				scanner := bufio.NewScanner(file)
				scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()

					var transcript domain.Transcript
					err := json.Unmarshal([]byte(line), &transcript)
					Expect(err).NotTo(HaveOccurred(), "File %s Line %d: unmarshal failed", filename, lineNum)
					Expect(transcript.Type).To(Equal(expectedType), "File %s Line %d: type mismatch", filename, lineNum)

					jsonData, err := json.Marshal(transcript)
					Expect(err).NotTo(HaveOccurred(), "File %s Line %d: marshal failed", filename, lineNum)

					var transcript2 domain.Transcript
					err = json.Unmarshal(jsonData, &transcript2)
					Expect(err).NotTo(HaveOccurred(), "File %s Line %d: second unmarshal failed", filename, lineNum)
					Expect(transcript2.Type).To(Equal(expectedType), "File %s Line %d: type mismatch after round-trip", filename, lineNum)
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				file.Close()
			}
		})
	})
})
