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
