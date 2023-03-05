package wig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wig Suite")
}
