package logwatcher_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogwatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logwatcher Suite")
}
