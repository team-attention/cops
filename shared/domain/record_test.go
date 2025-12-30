package domain_test

import (
	"bufio"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

const assistantTestDataFile = "record_assistant.jsonl"
const userTestDataFile = "record_user.jsonl"
const fileHistoryTestDataFile = "record_file_history_snapshot.jsonl"

var _ = Describe("Record", func() {
	Describe("UnmarshalJSON", func() {
		Context("when unmarshaling assistant records", func() {
			It("parses assistant record with tool_use content from JSONL file", func() {
				// Read line 2 from record_assistant.jsonl (contains tool_use)
				file, err := os.Open(assistantTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				var line string
				for scanner.Scan() {
					lineNum++
					if lineNum == 2 {
						line = scanner.Text()
						break
					}
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(line).NotTo(BeEmpty())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type equals RecordTypeMessage
				Expect(record.Type).To(Equal(domain.RecordTypeMessage))

				// Assert Data is *AssistantRecord
				assistantRecord, ok := record.Data.(*domain.AssistantRecord)
				Expect(ok).To(BeTrue())

				// Assert RequestID is "req_011CWPJbziKELTTDQvopECdi"
				Expect(assistantRecord.RequestID).To(Equal("req_011CWPJbziKELTTDQvopECdi"))

				// Assert MessageMetadata fields are populated
				Expect(assistantRecord.ParentUUID).NotTo(BeNil())
				Expect(*assistantRecord.ParentUUID).To(Equal("13b58212-1f07-442f-a278-a62ed9500dd0"))
				Expect(assistantRecord.SessionID).To(Equal("2a9ad7a6-55c1-4cfd-b0f7-11357eb51ef4"))

				// Assert Message.Content contains tool_use block
				Expect(assistantRecord.Message.Content).To(HaveLen(1))
				Expect(assistantRecord.Message.Content[0].Type).To(Equal(domain.AssistantMessageContentTypeToolUse))
			})

			It("parses assistant record with thinking content from JSONL file", func() {
				// Read line 3 from record_assistant.jsonl (contains thinking)
				file, err := os.Open(assistantTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				var line string
				for scanner.Scan() {
					lineNum++
					if lineNum == 3 {
						line = scanner.Text()
						break
					}
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(line).NotTo(BeEmpty())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type equals RecordTypeMessage
				Expect(record.Type).To(Equal(domain.RecordTypeMessage))

				// Assert Data is *AssistantRecord
				assistantRecord, ok := record.Data.(*domain.AssistantRecord)
				Expect(ok).To(BeTrue())

				// Assert Message.Content contains thinking block
				Expect(assistantRecord.Message.Content).To(HaveLen(1))
				Expect(assistantRecord.Message.Content[0].Type).To(Equal(domain.AssistantMessageContentTypeTool))
			})
		})

		Context("when unmarshaling user records", func() {
			It("parses user record with meta flag from JSONL file", func() {
				// Read line 1 from record_user.jsonl (isMeta: true)
				file, err := os.Open(userTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				Expect(scanner.Scan()).To(BeTrue())
				line := scanner.Text()
				Expect(scanner.Err()).NotTo(HaveOccurred())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type equals RecordTypeUser
				Expect(record.Type).To(Equal(domain.RecordTypeUser))

				// Assert Data is *UserRecord
				userRecord, ok := record.Data.(*domain.UserRecord)
				Expect(ok).To(BeTrue())

				// Assert IsMeta is true
				Expect(userRecord.IsMeta).To(BeTrue())

				// Assert MessageMetadata fields are populated
				Expect(userRecord.SessionID).To(Equal("2a9ad7a6-55c1-4cfd-b0f7-11357eb51ef4"))
			})

			It("parses user record with thinkingMetadata from JSONL file", func() {
				// Read line 4 from record_user.jsonl (has thinkingMetadata)
				file, err := os.Open(userTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				var line string
				for scanner.Scan() {
					lineNum++
					if lineNum == 4 {
						line = scanner.Text()
						break
					}
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(line).NotTo(BeEmpty())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Data.ThinkingMetadata is not nil
				userRecord, ok := record.Data.(*domain.UserRecord)
				Expect(ok).To(BeTrue())
				Expect(userRecord.ThinkingMetadata).NotTo(BeNil())

				// Assert ThinkingMetadata.Level is "high"
				Expect(userRecord.ThinkingMetadata.Level).To(Equal("high"))
			})

			It("parses user record with todos from JSONL file", func() {
				// Read line 8 from record_user.jsonl (has todos)
				file, err := os.Open(userTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				var line string
				for scanner.Scan() {
					lineNum++
					if lineNum == 8 {
						line = scanner.Text()
						break
					}
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(line).NotTo(BeEmpty())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Data.Todos is not empty
				userRecord, ok := record.Data.(*domain.UserRecord)
				Expect(ok).To(BeTrue())
				Expect(userRecord.Todos).NotTo(BeEmpty())

				// Assert todo content and status fields
				Expect(userRecord.Todos[0].Content).To(Equal("테스트 할 일 항목 작성하기"))
				Expect(userRecord.Todos[0].Status).To(Equal("in_progress"))
			})
		})

		Context("when unmarshaling file-history-snapshot records", func() {
			It("parses file-history-snapshot record from JSONL file", func() {
				// Read line 1 from record_file_history_snapshot.jsonl
				file, err := os.Open(fileHistoryTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				Expect(scanner.Scan()).To(BeTrue())
				line := scanner.Text()
				Expect(scanner.Err()).NotTo(HaveOccurred())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type equals RecordTypeFileHistorySnapshot
				Expect(record.Type).To(Equal(domain.RecordTypeFileHistorySnapshot))

				// Assert Data is *FileHistorySnapshotRecord
				snapshotRecord, ok := record.Data.(*domain.FileHistorySnapshotRecord)
				Expect(ok).To(BeTrue())

				// Assert MessageID is populated
				Expect(snapshotRecord.MessageID).To(Equal("a8dcd15e-7df2-4956-a16d-867f69dc7b67"))

				// Assert Snapshot.MessageID matches
				Expect(snapshotRecord.Snapshot.MessageID).To(Equal("a8dcd15e-7df2-4956-a16d-867f69dc7b67"))

				// Assert IsSnapshotUpdate is false
				Expect(snapshotRecord.IsSnapshotUpdate).To(BeFalse())
			})

			It("parses file-history-snapshot update record from JSONL file", func() {
				// Read line 2 from record_file_history_snapshot.jsonl (isSnapshotUpdate: true)
				file, err := os.Open(fileHistoryTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				var line string
				for scanner.Scan() {
					lineNum++
					if lineNum == 2 {
						line = scanner.Text()
						break
					}
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
				Expect(line).NotTo(BeEmpty())

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Data is *FileHistorySnapshotRecord
				snapshotRecord, ok := record.Data.(*domain.FileHistorySnapshotRecord)
				Expect(ok).To(BeTrue())

				// Assert IsSnapshotUpdate is true
				Expect(snapshotRecord.IsSnapshotUpdate).To(BeTrue())

				// Assert Snapshot.TrackedFileBackups is not empty
				Expect(snapshotRecord.Snapshot.TrackedFileBackups).NotTo(BeEmpty())
			})
		})

		Context("when unmarshaling unknown record types", func() {
			It("stores unknown type as map[string]any and logs error", func() {
				// Create JSON with unknown type
				jsonData := []byte(`{"type":"future-type","customField":"value"}`)

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal(jsonData, &record)

				// Assert no error returned (permissive)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type equals "future-type"
				Expect(record.Type).To(Equal(domain.RecordType("future-type")))

				// Assert Data is map[string]any
				mapData, ok := record.Data.(map[string]any)
				Expect(ok).To(BeTrue())

				// Assert map contains "type" and "customField" keys
				Expect(mapData).To(HaveKey("type"))
				Expect(mapData).To(HaveKeyWithValue("customField", "value"))
			})

			It("stores record with missing type as map[string]any", func() {
				// Create JSON without type field
				jsonData := []byte(`{"field":"value"}`)

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal(jsonData, &record)

				// Assert no error returned (permissive)
				Expect(err).NotTo(HaveOccurred())

				// Assert Type is empty string
				Expect(record.Type).To(Equal(domain.RecordType("")))

				// Assert Data is map[string]any
				_, ok := record.Data.(map[string]any)
				Expect(ok).To(BeTrue())
			})
		})

		Context("when handling schema mismatches", func() {
			It("handles missing optional fields gracefully", func() {
				// Create minimal user JSON with only required fields
				jsonData := []byte(`{"type":"user","parentUuid":null,"isSidechain":false,"userType":"external","sessionId":"test","version":"1.0","gitBranch":"main","uuid":"test-uuid","timestamp":"2025-01-01T00:00:00Z","message":{"role":"user","content":"test"},"isMeta":false}`)

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal(jsonData, &record)
				Expect(err).NotTo(HaveOccurred())

				// Assert Data is *UserRecord
				userRecord, ok := record.Data.(*domain.UserRecord)
				Expect(ok).To(BeTrue())

				// Assert optional fields (ThinkingMetadata, Todos) are nil/empty
				Expect(userRecord.ThinkingMetadata).To(BeNil())
				Expect(userRecord.Todos).To(BeNil())
			})

			It("ignores extra fields not in struct", func() {
				// Create user JSON with extra unknown field
				jsonData := []byte(`{"type":"user","extraField":"ignored","parentUuid":null,"isSidechain":false,"userType":"external","sessionId":"test","version":"1.0","gitBranch":"main","uuid":"test-uuid","timestamp":"2025-01-01T00:00:00Z","message":{"role":"user","content":"test"},"isMeta":false}`)

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal(jsonData, &record)

				// Assert no error
				Expect(err).NotTo(HaveOccurred())

				// Assert Data is *UserRecord (extra field ignored)
				_, ok := record.Data.(*domain.UserRecord)
				Expect(ok).To(BeTrue())
			})
		})

		Context("when JSON is invalid", func() {
			It("returns error for malformed JSON", func() {
				// Create invalid JSON
				jsonData := []byte(`{not valid`)

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal(jsonData, &record)

				// Assert error is returned
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when marshaling typed records", func() {
			It("produces flat JSON for AssistantRecord", func() {
				// Create Record with Type=RecordTypeMessage, Data=&AssistantRecord{...}
				assistantRecord := &domain.AssistantRecord{
					RequestID: "test-request",
				}
				assistantRecord.SessionID = "test-session"
				assistantRecord.Version = "1.0"
				assistantRecord.GitBranch = "main"
				assistantRecord.UUID = "test-uuid"

				record := domain.Record{
					Type: domain.RecordTypeMessage,
					Data: assistantRecord,
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Parse result as map[string]any
				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				// Assert "type" field equals "assistant"
				Expect(result).To(HaveKeyWithValue("type", "assistant"))

				// Assert AssistantRecord fields present at top level
				Expect(result).To(HaveKeyWithValue("requestId", "test-request"))

				// Assert MessageMetadata fields present at top level
				Expect(result).To(HaveKeyWithValue("sessionId", "test-session"))
			})

			It("produces flat JSON for UserRecord", func() {
				// Create Record with Type=RecordTypeUser, Data=&UserRecord{...}
				userRecord := &domain.UserRecord{
					IsMeta: true,
				}
				userRecord.SessionID = "test-session"

				record := domain.Record{
					Type: domain.RecordTypeUser,
					Data: userRecord,
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Parse result as map[string]any
				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				// Assert "type" field equals "user"
				Expect(result).To(HaveKeyWithValue("type", "user"))

				// Assert UserRecord fields at top level
				Expect(result).To(HaveKeyWithValue("isMeta", true))
			})

			It("produces flat JSON for FileHistorySnapshotRecord", func() {
				// Create Record with Type=RecordTypeFileHistorySnapshot, Data=&FileHistorySnapshotRecord{...}
				snapshotRecord := &domain.FileHistorySnapshotRecord{
					MessageID:        "test-msg",
					IsSnapshotUpdate: false,
				}

				record := domain.Record{
					Type: domain.RecordTypeFileHistorySnapshot,
					Data: snapshotRecord,
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Parse result as map[string]any
				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				// Assert "type" field equals "file-history-snapshot"
				Expect(result).To(HaveKeyWithValue("type", "file-history-snapshot"))

				// Assert FileHistorySnapshotRecord fields at top level
				Expect(result).To(HaveKeyWithValue("messageId", "test-msg"))
				Expect(result).To(HaveKeyWithValue("isSnapshotUpdate", false))
			})
		})

		Context("when marshaling nil Data", func() {
			It("produces JSON with only type field", func() {
				// Create Record with Type=RecordTypeUser, Data=nil
				record := domain.Record{
					Type: domain.RecordTypeUser,
					Data: nil,
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Assert result equals {"type":"user"}
				Expect(string(jsonData)).To(Equal(`{"type":"user"}`))
			})
		})

		Context("when marshaling map[string]any Data", func() {
			It("produces flat JSON for unknown type stored as map", func() {
				// Create Record with Type="unknown", Data=map[string]any{"field":"value"}
				record := domain.Record{
					Type: "unknown",
					Data: map[string]any{"field": "value"},
				}

				// Marshal to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Parse result as map[string]any
				var result map[string]any
				err = json.Unmarshal(jsonData, &result)
				Expect(err).NotTo(HaveOccurred())

				// Assert result contains "type":"unknown" and "field":"value" at top level
				Expect(result).To(HaveKeyWithValue("type", "unknown"))
				Expect(result).To(HaveKeyWithValue("field", "value"))
			})
		})
	})

	Describe("Round-trip serialization", func() {
		Context("with real JSONL data", func() {
			It("preserves assistant record through marshal/unmarshal cycle", func() {
				// Read line from record_assistant.jsonl
				file, err := os.Open(assistantTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				Expect(scanner.Scan()).To(BeTrue())
				line := scanner.Text()

				// Unmarshal into domain.Record
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				// Marshal back to JSON
				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				// Unmarshal again into new domain.Record
				var record2 domain.Record
				err = json.Unmarshal(jsonData, &record2)
				Expect(err).NotTo(HaveOccurred())

				// Assert both Records have equal Type
				Expect(record2.Type).To(Equal(record.Type))

				// Assert both Records have same Data type (*AssistantRecord)
				assistantRecord1, ok1 := record.Data.(*domain.AssistantRecord)
				assistantRecord2, ok2 := record2.Data.(*domain.AssistantRecord)
				Expect(ok1).To(BeTrue())
				Expect(ok2).To(BeTrue())

				// Compare key fields
				Expect(assistantRecord2.RequestID).To(Equal(assistantRecord1.RequestID))
				Expect(assistantRecord2.Message.ID).To(Equal(assistantRecord1.Message.ID))
			})

			It("preserves user record through marshal/unmarshal cycle", func() {
				// Read line from record_user.jsonl
				file, err := os.Open(userTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				Expect(scanner.Scan()).To(BeTrue())
				line := scanner.Text()

				// Perform round-trip marshal/unmarshal
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				var record2 domain.Record
				err = json.Unmarshal(jsonData, &record2)
				Expect(err).NotTo(HaveOccurred())

				// Assert Records are equivalent
				Expect(record2.Type).To(Equal(record.Type))
				_, ok1 := record.Data.(*domain.UserRecord)
				_, ok2 := record2.Data.(*domain.UserRecord)
				Expect(ok1).To(BeTrue())
				Expect(ok2).To(BeTrue())
			})

			It("preserves file-history-snapshot record through marshal/unmarshal cycle", func() {
				// Read line from record_file_history_snapshot.jsonl
				file, err := os.Open(fileHistoryTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				Expect(scanner.Scan()).To(BeTrue())
				line := scanner.Text()

				// Perform round-trip marshal/unmarshal
				var record domain.Record
				err = json.Unmarshal([]byte(line), &record)
				Expect(err).NotTo(HaveOccurred())

				jsonData, err := json.Marshal(record)
				Expect(err).NotTo(HaveOccurred())

				var record2 domain.Record
				err = json.Unmarshal(jsonData, &record2)
				Expect(err).NotTo(HaveOccurred())

				// Assert Records are equivalent
				Expect(record2.Type).To(Equal(record.Type))
				_, ok1 := record.Data.(*domain.FileHistorySnapshotRecord)
				_, ok2 := record2.Data.(*domain.FileHistorySnapshotRecord)
				Expect(ok1).To(BeTrue())
				Expect(ok2).To(BeTrue())
			})
		})

		Context("with all JSONL files", func() {
			It("preserves all assistant records through round-trip", func() {
				// Read all lines from record_assistant.jsonl
				file, err := os.Open(assistantTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()

					// For each line, perform round-trip
					var record domain.Record
					err := json.Unmarshal([]byte(line), &record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: unmarshal failed", lineNum)

					jsonData, err := json.Marshal(record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: marshal failed", lineNum)

					var record2 domain.Record
					err = json.Unmarshal(jsonData, &record2)
					Expect(err).NotTo(HaveOccurred(), "Line %d: second unmarshal failed", lineNum)

					// Assert no errors and data preserved
					Expect(record2.Type).To(Equal(domain.RecordTypeMessage), "Line %d: type mismatch", lineNum)
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
			})

			It("preserves all user records through round-trip", func() {
				// Read all lines from record_user.jsonl
				file, err := os.Open(userTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()

					// For each line, perform round-trip
					var record domain.Record
					err := json.Unmarshal([]byte(line), &record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: unmarshal failed", lineNum)

					jsonData, err := json.Marshal(record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: marshal failed", lineNum)

					var record2 domain.Record
					err = json.Unmarshal(jsonData, &record2)
					Expect(err).NotTo(HaveOccurred(), "Line %d: second unmarshal failed", lineNum)

					// Assert no errors and data preserved
					Expect(record2.Type).To(Equal(domain.RecordTypeUser), "Line %d: type mismatch", lineNum)
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
			})

			It("preserves all file-history-snapshot records through round-trip", func() {
				// Read all lines from record_file_history_snapshot.jsonl
				file, err := os.Open(fileHistoryTestDataFile)
				Expect(err).NotTo(HaveOccurred())
				defer file.Close()

				scanner := bufio.NewScanner(file)
				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()

					// For each line, perform round-trip
					var record domain.Record
					err := json.Unmarshal([]byte(line), &record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: unmarshal failed", lineNum)

					jsonData, err := json.Marshal(record)
					Expect(err).NotTo(HaveOccurred(), "Line %d: marshal failed", lineNum)

					var record2 domain.Record
					err = json.Unmarshal(jsonData, &record2)
					Expect(err).NotTo(HaveOccurred(), "Line %d: second unmarshal failed", lineNum)

					// Assert no errors and data preserved
					Expect(record2.Type).To(Equal(domain.RecordTypeFileHistorySnapshot), "Line %d: type mismatch", lineNum)
				}
				Expect(scanner.Err()).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Integration with existing JSONL reading", func() {
		It("can parse entire assistant JSONL file", func() {
			// Open record_assistant.jsonl
			file, err := os.Open(assistantTestDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			// Create scanner
			scanner := bufio.NewScanner(file)
			recordCount := 0

			// For each line
			for scanner.Scan() {
				recordCount++
				line := scanner.Text()

				// Unmarshal into domain.Record
				var record domain.Record
				err := json.Unmarshal([]byte(line), &record)

				// Assert no error
				Expect(err).NotTo(HaveOccurred(), "Failed to parse line %d", recordCount)

				// Assert Type is RecordTypeMessage
				Expect(record.Type).To(Equal(domain.RecordTypeMessage))

				// Assert Data is *AssistantRecord
				_, ok := record.Data.(*domain.AssistantRecord)
				Expect(ok).To(BeTrue(), "Line %d: Data should be *AssistantRecord", recordCount)
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())

			// Assert correct number of records parsed (4 lines)
			Expect(recordCount).To(Equal(4))
		})

		It("can parse entire user JSONL file", func() {
			// Open record_user.jsonl
			file, err := os.Open(userTestDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			// Parse all lines into domain.Record slice
			var records []domain.Record
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				var record domain.Record
				err := json.Unmarshal(scanner.Bytes(), &record)
				Expect(err).NotTo(HaveOccurred())
				records = append(records, record)
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())

			// Assert correct number of records (8 lines)
			Expect(records).To(HaveLen(8))

			// Assert all have Type RecordTypeUser
			for i, record := range records {
				Expect(record.Type).To(Equal(domain.RecordTypeUser), "Line %d: type mismatch", i+1)
			}
		})

		It("can parse entire file-history-snapshot JSONL file", func() {
			// Open record_file_history_snapshot.jsonl
			file, err := os.Open(fileHistoryTestDataFile)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			// Parse all lines into domain.Record slice
			var records []domain.Record
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				var record domain.Record
				err := json.Unmarshal(scanner.Bytes(), &record)
				Expect(err).NotTo(HaveOccurred())
				records = append(records, record)
			}
			Expect(scanner.Err()).NotTo(HaveOccurred())

			// Assert correct number of records (7 lines)
			Expect(records).To(HaveLen(7))

			// Assert all have Type RecordTypeFileHistorySnapshot
			for i, record := range records {
				Expect(record.Type).To(Equal(domain.RecordTypeFileHistorySnapshot), "Line %d: type mismatch", i+1)
			}
		})
	})
})
