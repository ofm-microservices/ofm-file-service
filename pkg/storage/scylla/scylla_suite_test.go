package scylla

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScylla(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scylla Suite")
}
