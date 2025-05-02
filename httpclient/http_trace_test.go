// Copyright 2023 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpclient_test

import (
	"github.com/Carbonfrost/joe-cli-http/httpclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TraceLevel", func() {

	Describe("String", func() {

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
			Entry("responseHeaders", httpclient.TraceResponseHeaders, "responseHeaders"),
			Entry("DNS", httpclient.TraceDNS, "dns"),
			Entry("TLS", httpclient.TraceTLS, "tls"),
			Entry("http1xx", httpclient.TraceHTTP1XX, "http1xx"),
			Entry("redirects", httpclient.TraceRedirects, "redirects"),
			Entry("requestBody", httpclient.TraceRequestBody, "requestBody"),
			Entry("responseStatus", httpclient.TraceResponseStatus, "responseStatus"),
			Entry("off", httpclient.TraceOff, "off"),
		)

	})

	Describe("Set", func() {

		DescribeTable("examples", func(value string, expected httpclient.TraceLevel) {
			var actual httpclient.TraceLevel
			err := actual.Set(value)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("named composite", "on", httpclient.TraceOn),
			Entry("composite", "on,tls", httpclient.TraceOn|httpclient.TraceTLS),
			Entry("debug", "debug", httpclient.TraceDebug),
			Entry("verbose", "verbose", httpclient.TraceVerbose),
			Entry("connections", "connections", httpclient.TraceConnections),
			Entry("requestHeaders", "requestHeaders", httpclient.TraceRequestHeaders),
			Entry("responseHeaders", "responseHeaders", httpclient.TraceResponseHeaders),
			Entry("DNS", "dns", httpclient.TraceDNS),
			Entry("TLS", "tls", httpclient.TraceTLS),
			Entry("http1xx", "http1xx", httpclient.TraceHTTP1XX),
			Entry("redirects", "redirects", httpclient.TraceRedirects),
			Entry("requestBody", "requestBody", httpclient.TraceRequestBody),
			Entry("responseStatus", "responseStatus", httpclient.TraceResponseStatus),
			Entry("off", "off", httpclient.TraceOff),
		)

	})
})
