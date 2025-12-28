package connectrpc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConnectRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConnectRPC Handler Suite")
}
