package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("URLValue", func() {

	DescribeTable("examples", func(value string, expected string) {
		u := new(httpclient.URLValue)
		err := u.Set(value)

		Expect(err).NotTo(HaveOccurred())
		Expect(u.String()).To(Equal(expected))
	},

		Entry("localhost", "localhost", "http://localhost"),
		Entry("example", "example.com", "http://example.com"),
		Entry("port", ":8080", "http://localhost:8080"),
		Entry("rooted", "/root", "/root"),
		Entry("empty", "", ""),
	)
})
