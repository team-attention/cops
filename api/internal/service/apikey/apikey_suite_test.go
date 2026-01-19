package apikey_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPIKey(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIKey Service Suite")
}
