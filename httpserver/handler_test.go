// Copyright 2025 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package httpserver_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"github.com/Carbonfrost/joe-cli-http/httpclient/expr"
	"github.com/Carbonfrost/joe-cli-http/httpserver"
	"github.com/Carbonfrost/joe-cli-http/httpserver/httpserverfakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("NewPingHandler", func() {

	It("writes out ping message", func() {
		recorder := httptest.NewRecorder()
		p := httpserver.NewPingHandler()
		p.ServeHTTP(recorder, nil)

		Expect(recorder.Body.String()).To(Equal("ping\n"))
	})
})

var _ = Describe("NewRequestLogger", func() {

	var (
		recorder  *httptest.ResponseRecorder
		accessLog string
		output    bytes.Buffer
	)

	JustBeforeEach(func() {
		recorder = httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "https://example.com/ring", nil)

		handler := httpserver.NewRequestLogger(accessLog, &output, httpserver.NewPingHandler())
		output.Reset()
		handler.ServeHTTP(recorder, request)
	})

	Context("when using the default access log format", func() {

		BeforeEach(func() {
			accessLog = ""
		})

		It("writes out the expected log message", func() {
			Expect(removeANSICodes(output.String())).To(And(
				ContainSubstring(`"GET /ring HTTP/1.1"`),
				MatchRegexp(`\[\d{2}/[a-zA-Z]{3}/\d{4} \d{2}:\d{2}:\d{2}]`),
			))
		})

		It("writes out color for method", func() {
			Expect(derefANSICodes(output.String())).To(And(
				ContainSubstring(`"{reverse}{blue}GET{reset}`),
			))
		})

		It("writes out color for status", func() {
			Expect(derefANSICodes(output.String())).To(And(
				ContainSubstring("{reverse}{green}200 OK{reset}"),
			))
		})

	})

	Context("when containing custom access log string", func() {

		BeforeEach(func() {
			accessLog = "%(go.version)"
		})

		It("writes out the expected log message", func() {
			Expect(output.String()).To(MatchRegexp(`go\d`))
		})
	})
})

var _ = Describe("NewHeaderMiddleware", func() {

	It("sets up header with name", func() {
		recorder := httptest.NewRecorder()

		p := httpserver.NewHeaderMiddleware("Server", "Albatross")
		handler := p(httpserver.NewPingHandler())
		handler.ServeHTTP(recorder, nil)

		Expect(recorder.Header()).To(HaveKeyWithValue("Server", []string{"Albatross"}))
	})

	It("adds additional headers", func() {
		recorder := httptest.NewRecorder()

		handler := httpserver.NewPingHandler()
		handler = httpserver.NewHeaderMiddleware("Server", "A")(handler)
		handler = httpserver.NewHeaderMiddleware("Server", "B")(handler)
		handler.ServeHTTP(recorder, nil)

		Expect(recorder.Header()).To(HaveKeyWithValue("Server", []string{"B", "A"}))
	})
})

var _ = Describe("ExpandRequest", func() {

	DescribeTable("examples", func(text string, expected types.GomegaMatcher) {
		request, _ := http.NewRequest("GET", "https://example.com/whistle", nil)
		e := expr.Compile(text)

		ww := new(httpserverfakes.FakeWrapResponseWriter)
		ww.BytesWrittenReturns(800)
		ww.HeaderReturns(http.Header{
			http.CanonicalHeaderKey("X-Request-ID"): []string{"732"},
		})
		ww.StatusReturns(429)
		expander := httpserver.ExpandRequest(request, ww)
		Expect(e.Expand(expander)).To(expected)
	},
		Entry("bytesWritten", "%(bytesWritten)", Equal("800")),
		Entry("method", "%(method)", Equal("GET")),
		Entry("protocol", "%(protocol)", Equal("HTTP/1.1")),
		Entry("status", "%(status)", Equal("429 Too Many Requests")),
		Entry("statusCode", "%(statusCode)", Equal("429")),
		Entry("urlPath", "%(urlPath)", Equal("/whistle")),
		Entry("header", "%(header)", Equal("X-Request-Id: 732\r\n")),
		Entry("header direct name", "%(header.X-Request-ID)", Equal("732")),
		Entry("header non-canonical name", "%(header.x-request-id)", Equal("732")),
		Entry("header camel name", "%(header.xRequestId)", Equal("732")),
	)
})

var ansiPattern = regexp.MustCompile(`\x1B\[[0-9;]*m`)

func removeANSICodes(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func derefANSICodes(s string) string {
	for _, k := range []string{"reset", "blue", "reverse", "magenta", "green"} {
		s = strings.ReplaceAll(
			s,
			expr.ExpandColors(k).(string),
			fmt.Sprintf("{%s}", k))
	}
	return s
}
