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
					`{"uuid":"1","type":"user","sessionId":"s1","timestamp":"2024-01-01T00:00:00Z"}`,
					`{"uuid":"2","type":"assistant","sessionId":"s1","timestamp":"2024-01-01T00:00:01Z"}`,
				}

				var records []shareddomain.SessionRecord
				var parseErrors []error

				for _, line := range lines {
					var record shareddomain.SessionRecord
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						parseErrors = append(parseErrors, err)
						continue
					}
					records = append(records, record)
				}

				Expect(records).To(HaveLen(2))
				Expect(parseErrors).To(BeEmpty())
				Expect(records[0].UUID).To(Equal("1"))
				Expect(records[0].Type).To(Equal(shareddomain.SessionTypeUser))
			})
		})
	})

	Describe("parsing invalid JSONL lines", func() {
		Context("when some lines contain invalid JSON", func() {
			It("skips invalid lines and parses valid ones", func() {
				lines := []string{
					`{"uuid":"1","type":"user"}`,
					`invalid json`,
					`{"uuid":"2","type":"assistant"}`,
				}

				var records []shareddomain.SessionRecord
				var parseErrors []error

				for _, line := range lines {
					var record shareddomain.SessionRecord
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
					`{"uuid":"1","type":"user"}`,
					``,
					`{"uuid":"2","type":"assistant"}`,
				}

				var records []shareddomain.SessionRecord

				for _, line := range lines {
					if line == "" {
						continue
					}
					var record shareddomain.SessionRecord
					if err := sonic.Unmarshal([]byte(line), &record); err != nil {
						continue
					}
					records = append(records, record)
				}

				Expect(records).To(HaveLen(2))
			})
		})
	})

	Describe("parsing message content", func() {
		Context("when content is a text string", func() {
			It("parses text content correctly", func() {
				textLine := `{"uuid":"1","type":"user","message":{"role":"user","content":"hello"}}`

				var record shareddomain.SessionRecord
				err := sonic.Unmarshal([]byte(textLine), &record)

				Expect(err).NotTo(HaveOccurred())
				Expect(record.Message).NotTo(BeNil())
				Expect(record.Message.Content).NotTo(BeNil())
				Expect(record.Message.Content.IsBlocks).To(BeFalse())
				Expect(record.Message.Content.Text).NotTo(BeNil())
				Expect(*record.Message.Content.Text).To(Equal("hello"))
			})
		})

		Context("when content is an array of blocks", func() {
			It("parses content blocks correctly", func() {
				blockLine := `{"uuid":"1","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"t1","name":"read","input":{}}]}}`

				var record shareddomain.SessionRecord
				err := sonic.Unmarshal([]byte(blockLine), &record)

				Expect(err).NotTo(HaveOccurred())
				Expect(record.Message).NotTo(BeNil())
				Expect(record.Message.Content).NotTo(BeNil())
				Expect(record.Message.Content.IsBlocks).To(BeTrue())
				Expect(record.Message.Content.Blocks).To(HaveLen(2))
			})
		})
	})
})
