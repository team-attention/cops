package codexutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/daemon/internal/platform/util/codexutil"
)

func TestCodexutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Codexutil Suite")
}

var _ = Describe("ExtractCwd", func() {
	It("extracts cwd from a valid session_meta line", func() {
		line := `{"type":"session_meta","payload":{"cwd":"/home/user/project"}}`
		cwd := codexutil.ExtractCwd(line)
		Expect(cwd).To(Equal("/home/user/project"))
	})

	It("returns empty for non session_meta line", func() {
		line := `{"type":"message","content":"hello"}`
		cwd := codexutil.ExtractCwd(line)
		Expect(cwd).To(BeEmpty())
	})

	It("returns empty for session_meta without cwd", func() {
		line := `{"type":"session_meta","payload":{}}`
		cwd := codexutil.ExtractCwd(line)
		Expect(cwd).To(BeEmpty())
	})

	It("returns empty for invalid JSON", func() {
		cwd := codexutil.ExtractCwd("not json")
		Expect(cwd).To(BeEmpty())
	})
})

var _ = Describe("ExtractCwdFromLines", func() {
	It("finds cwd in the third line of a batch", func() {
		lines := []string{
			`{"type":"message","content":"hello"}`,
			`{"type":"event","data":"something"}`,
			`{"type":"session_meta","payload":{"cwd":"/home/user/project"}}`,
		}
		cwd := codexutil.ExtractCwdFromLines(lines)
		Expect(cwd).To(Equal("/home/user/project"))
	})

	It("returns empty when no session_meta in batch", func() {
		lines := []string{
			`{"type":"message","content":"hello"}`,
			`{"type":"event","data":"something"}`,
			`{"type":"response","data":"world"}`,
			`{"type":"message","content":"bye"}`,
			`{"type":"done"}`,
		}
		cwd := codexutil.ExtractCwdFromLines(lines)
		Expect(cwd).To(BeEmpty())
	})

	It("returns the first cwd found when multiple exist", func() {
		lines := []string{
			`{"type":"session_meta","payload":{"cwd":"/first/project"}}`,
			`{"type":"session_meta","payload":{"cwd":"/second/project"}}`,
		}
		cwd := codexutil.ExtractCwdFromLines(lines)
		Expect(cwd).To(Equal("/first/project"))
	})
})
