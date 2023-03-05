package wig_test

import (
	"crypto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wig", func() {

	It("has registered the hash functions", func() {
		// These need to be embedded into the wig executable
		// so that various --integrity hashes can be used
		hashes := []crypto.Hash{
			crypto.MD5,
			crypto.RIPEMD160,
			crypto.SHA1,
			crypto.SHA224,
			crypto.SHA256,
			crypto.SHA384,
			crypto.SHA512,
			crypto.SHA512_224,
			crypto.SHA512_256,
		}
		for _, h := range hashes {
			Expect(h.Available()).To(BeTrue(), "expected hash %v to be available", h)
		}
	})
})
