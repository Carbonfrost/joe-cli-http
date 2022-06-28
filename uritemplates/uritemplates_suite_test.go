package uritemplates_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUritemplates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Uritemplates Suite")
}
