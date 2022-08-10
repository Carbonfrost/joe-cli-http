package httpclient_test

import (
	"crypto/tls"
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CipherSuites", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected httpclient.CipherSuites) {
				cs := new(httpclient.CipherSuites)
				for _, a := range args {
					err := cs.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(*cs).To(Equal(expected))
			},
			Entry(
				"simple",
				[]string{"TLS_RSA_WITH_AES_128_CBC_SHA"},
				httpclient.CipherSuites([]uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA}),
			),
		)
	})
})

var _ = Describe("CurveID", func() {

	Describe("Set", func() {
		DescribeTable("examples",
			func(args []string, expected httpclient.CurveIDs) {
				cs := new(httpclient.CurveIDs)
				for _, a := range args {
					err := cs.Set(a)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(*cs).To(Equal(expected))
			},
			Entry("P256", []string{"P256"}, httpclient.CurveIDs([]tls.CurveID{tls.CurveP256})),
			Entry("P384", []string{"P384"}, httpclient.CurveIDs([]tls.CurveID{tls.CurveP384})),
			Entry("P521", []string{"P521"}, httpclient.CurveIDs([]tls.CurveID{tls.CurveP521})),
			Entry("X25519", []string{"X25519"}, httpclient.CurveIDs([]tls.CurveID{tls.X25519})),
			Entry("multi", []string{"P256", "X25519"}, httpclient.CurveIDs([]tls.CurveID{tls.CurveP256, tls.X25519})),
		)
	})
})
