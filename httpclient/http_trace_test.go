package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TraceLevel", func() {

	DescribeTable("examples", func(value httpclient.TraceLevel, expected string) {
		actual := httpclient.TraceLevel(value).String()
		Expect(actual).To(Equal(expected))
	},
		Entry("named composite", httpclient.TraceOn, "on"),
		Entry("composite", httpclient.TraceOn|httpclient.TraceTLS, "on,tls"),
		Entry("unknown composite", httpclient.TraceOn|1024, "on,1024"),

		Entry("debug", httpclient.TraceDebug, "debug"),
		Entry("verbose", httpclient.TraceVerbose, "verbose"),
		Entry("connections", httpclient.TraceConnections, "connections"),
		Entry("requestHeaders", httpclient.TraceRequestHeaders, "requestHeaders"),
		Entry("dNS", httpclient.TraceDNS, "dns"),
		Entry("tLS", httpclient.TraceTLS, "tls"),
		Entry("http1xx", httpclient.TraceHTTP1XX, "http1xx"),
		Entry("requestBody", httpclient.TraceRequestBody, "requestBody"),
		Entry("off", httpclient.TraceOff, "off"),
	)
})