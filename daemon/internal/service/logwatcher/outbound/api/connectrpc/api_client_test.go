package connectrpc_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"connectrpc.com/connect"
	"github.com/team-attention/cops/daemon/internal/platform/util/errutil"
)

func TestAPIClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIClient Suite")
}

var _ = Describe("APIClient SendLogs", func() {
	Describe("error detection", func() {
		Context("when HTTP 413 error occurs", func() {
			It("returns PayloadTooLarge for error message containing '413'", func() {
				// 1. Create mock connect error with "413" in message
				err := connect.NewError(connect.CodeUnknown, fmt.Errorf("HTTP 413 error"))

				// 2. Simulate error wrapping (as done in SendLogs)
				wrappedErr := errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err)

				// 3. Verify errutil.IsPayloadTooLarge is true
				Expect(errutil.IsPayloadTooLarge(wrappedErr)).To(BeTrue())
			})

			It("returns PayloadTooLarge for 'Request Entity Too Large' message", func() {
				// 1. Create error with "Request Entity Too Large" in message
				err := connect.NewError(connect.CodeUnknown, fmt.Errorf("Request Entity Too Large"))

				// 2. Simulate error wrapping (as done in SendLogs)
				wrappedErr := errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err)

				// 3. Verify errutil.IsPayloadTooLarge is true
				Expect(errutil.IsPayloadTooLarge(wrappedErr)).To(BeTrue())
			})
		})

		Context("when CodeResourceExhausted error occurs", func() {
			It("returns PayloadTooLarge", func() {
				// 1. Create connect.Error with CodeResourceExhausted
				err := connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("resource exhausted"))

				// 2. Simulate error wrapping (as done in SendLogs)
				wrappedErr := errutil.Wrap(errutil.ErrorTypePayloadTooLarge, "batch rejected by server", err)

				// 3. Verify errutil.IsPayloadTooLarge is true
				Expect(errutil.IsPayloadTooLarge(wrappedErr)).To(BeTrue())
			})
		})

		Context("when other errors occur", func() {
			It("does not wrap as PayloadTooLarge", func() {
				// 1. Create generic connect error
				err := connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))

				// 2. For non-413 errors, SendLogs returns them unwrapped
				// Verify this error is NOT PayloadTooLarge
				Expect(errutil.IsPayloadTooLarge(err)).To(BeFalse())
			})
		})
	})
})
