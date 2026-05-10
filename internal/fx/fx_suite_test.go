package appfx

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFX(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FX Suite")
}
