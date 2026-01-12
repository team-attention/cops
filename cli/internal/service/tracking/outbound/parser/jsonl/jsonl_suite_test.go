package jsonl_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJSONLParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JSONLParser Suite")
}
